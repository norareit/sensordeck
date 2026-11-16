package custommetrics

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/config"
	"github.com/norareit/sensordeck/internal/sensors"
)

type clock struct {
	mutex sync.Mutex
	at    time.Time
}

func (current *clock) now() time.Time {
	current.mutex.Lock()
	defer current.mutex.Unlock()
	return current.at
}

func (current *clock) advance(by time.Duration) {
	current.mutex.Lock()
	defer current.mutex.Unlock()
	current.at = current.at.Add(by)
}

type lines struct {
	mutex sync.Mutex
	text  strings.Builder
}

func (written *lines) Write(data []byte) (int, error) {
	written.mutex.Lock()
	defer written.mutex.Unlock()
	return written.text.Write(data)
}

func (written *lines) String() string {
	written.mutex.Lock()
	defer written.mutex.Unlock()
	return written.text.String()
}

func everything(string) bool { return true }

func commandMetric(name, text string, every time.Duration) config.Metric {
	return config.Metric{Name: name, Command: text, Every: every, Scale: 1}
}

func newSet(definitions ...config.Metric) (*Set, *clock, *lines) {
	current := &clock{at: time.Unix(1000, 0)}
	written := &lines{}
	return NewSet(definitions, "", "/", current.now, written), current, written
}

func readValues(set *Set) map[string]float64 {
	values := map[string]float64{}
	for _, each := range set.Read(context.Background(), everything) {
		values[each.Name] = each.Value
	}
	return values
}

func eventually(test *testing.T, set *Set, name string, want float64) {
	test.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if value, found := readValues(set)[name]; found && value == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	test.Fatalf("%s never had the value %v, it has %v", name, want, readValues(set))
}

func TestARoundDoesNotWaitForACommand(test *testing.T) {
	set, _, _ := newSet(commandMetric("a.b", "sleep 1; echo 5", time.Minute))
	defer set.Close()
	started := time.Now()
	if values := readValues(set); len(values) != 0 {
		test.Errorf("values %v before the first run finished", values)
	}
	if time.Since(started) > 500*time.Millisecond {
		test.Errorf("the round lasted %v", time.Since(started))
	}
	eventually(test, set, "a.b", 5)
}

func TestACommandRunsAgainOnlyAfterEveryAndNotWhileInProgress(test *testing.T) {
	counter := filepath.Join(test.TempDir(), "count")
	set, current, _ := newSet(commandMetric("a.b", "echo x >> "+counter+"; sleep 0.3; wc -l < "+counter, 10*time.Second))
	defer set.Close()
	readValues(set)
	current.advance(time.Second)
	readValues(set)
	eventually(test, set, "a.b", 1)
	current.advance(5 * time.Second)
	if values := readValues(set); values["a.b"] != 1 {
		test.Errorf("values %v within every", values)
	}
	current.advance(10 * time.Second)
	readValues(set)
	current.advance(10 * time.Second)
	readValues(set)
	eventually(test, set, "a.b", 2)
}

func TestAFailedRunClearsTheValueWithItsLine(test *testing.T) {
	flag := filepath.Join(test.TempDir(), "flag")
	set, current, written := newSet(commandMetric("a.b", "if [ -e "+flag+" ]; then exit 2; fi; echo 4", 2*time.Second))
	defer set.Close()
	readValues(set)
	eventually(test, set, "a.b", 4)
	writeFile(test, flag, "")
	current.advance(3 * time.Second)
	deadline := time.Now().Add(5 * time.Second)
	for len(readValues(set)) != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if want := "metric a.b: the command exited with status 2\n"; written.String() != want {
		test.Errorf("written %q, want %q", written.String(), want)
	}
}

func TestAMetricThatIsNotWantedIsNotRead(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "marker")
	set, _, _ := newSet(commandMetric("a.b", "echo 1; : > "+marker, time.Minute))
	defer set.Close()
	set.Read(context.Background(), func(name string) bool { return name != "a.b" })
	time.Sleep(200 * time.Millisecond)
	if _, found := readFileText(marker); found {
		test.Error("the command was started for a metric that was not wanted")
	}
	// Reading it now starts the command, which shows that the first read did not.
	eventually(test, set, "a.b", 1)
}

func TestReplaceKeepsAnUnchangedDefinition(test *testing.T) {
	file := writeFile(test, filepath.Join(test.TempDir(), "f"), "x")
	kept := config.Metric{Name: "f.f", File: file, Scale: 1}
	set, _, written := newSet(kept, commandMetric("c.c", "sleep 1; echo 3", time.Minute))
	defer set.Close()
	readValues(set)
	set.Replace([]config.Metric{kept, commandMetric("c.c", "sleep 1; echo 3", time.Minute)})
	if got := written.String(); got != "metric f.f: no number in "+file+"\n" {
		test.Errorf("written %q", got)
	}
	readValues(set)
	if got := written.String(); strings.Count(got, "\n") != 1 {
		test.Errorf("the streak was not kept: %q", got)
	}
	eventually(test, set, "c.c", 3)
	set.Replace([]config.Metric{commandMetric("c.c", "sleep 1; echo 3", time.Minute)})
	if values := readValues(set); values["c.c"] != 3 {
		test.Errorf("the value was not kept: %v", values)
	}
}

func TestReplaceStartsAChangedDefinitionAndKillsItsRun(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "marker")
	set, _, _ := newSet(commandMetric("c.c", "sleep 1; : > "+marker+"; echo 3", time.Minute))
	defer set.Close()
	readValues(set)
	set.Replace([]config.Metric{commandMetric("c.c", "echo 8", time.Minute)})
	eventually(test, set, "c.c", 8)
	time.Sleep(1500 * time.Millisecond)
	if values := readValues(set); values["c.c"] != 8 {
		test.Errorf("values %v", values)
	}
	if _, found := readFileText(marker); found {
		test.Error("the run of the old definition was not killed")
	}
}

func TestReplaceKillsTheRunOfARemovedDefinition(test *testing.T) {
	marker := filepath.Join(test.TempDir(), "marker")
	set, _, _ := newSet(commandMetric("c.c", "sleep 1; : > "+marker+"; echo 3", time.Minute))
	defer set.Close()
	readValues(set)
	set.Replace(nil)
	time.Sleep(1500 * time.Millisecond)
	if _, found := readFileText(marker); found {
		test.Error("the run of the removed definition was not killed")
	}
}

func TestCloseKillsARunInProgress(test *testing.T) {
	set, _, _ := newSet(commandMetric("c.c", "sleep 30", time.Minute))
	readValues(set)
	time.Sleep(100 * time.Millisecond)
	started := time.Now()
	set.Close()
	if time.Since(started) > time.Second {
		test.Errorf("Close lasted %v", time.Since(started))
	}
}

var _ sensors.Source = (*Set)(nil)
