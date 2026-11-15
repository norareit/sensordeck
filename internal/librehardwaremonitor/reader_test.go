package librehardwaremonitor

import (
	"bytes"
	"compress/gzip"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/norareit/sensordeck/internal/sensors"
)

func everything(string) bool { return true }

// serve starts a server whose handler is given. Handlers that wait end when the request ends, and the connections are
// closed when the test ends.
func serve(test *testing.T, handler http.HandlerFunc) (*httptest.Server, *url.URL) {
	test.Helper()
	server := httptest.NewServer(handler)
	test.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})
	address, err := url.Parse(server.URL + "/data.json")
	if err != nil {
		test.Fatal(err)
	}
	return server, address
}

func answering(body []byte) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(body)
	}
}

func waitForTheClient(request *http.Request) { <-request.Context().Done() }

func oneDocument(test *testing.T) []byte {
	return document(test,
		hardwareGroup(group("Temperatures",
			temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C"))),
		hardwareGroup(group("Temperatures", temperature("/nvme/0/temperature/0", "Temperature", "35.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/memory/dimm/0/temperature/0", "DIMM #0", "36.0 °C"))))
}

func TestADocumentIsReadWithOneGetOfThePathAndQuery(test *testing.T) {
	var method, path, query atomic.Value
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		method.Store(request.Method)
		path.Store(request.URL.Path)
		query.Store(request.URL.RawQuery)
		answering(oneDocument(test))(writer, request)
	})
	address.RawQuery = "a=b"
	got, err := NewReader(address, time.Second).Read(context.Background())
	if err != nil {
		test.Fatal(err)
	}
	want := []sensors.Reading{reading("cpu.temp", 46.9), reading("ram.1.temp", 36), reading("nvme.1.temp", 35)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	if method.Load() != "GET" || path.Load() != "/data.json" || query.Load() != "a=b" {
		test.Errorf("request %v %v ?%v", method.Load(), path.Load(), query.Load())
	}
}

func TestAStatusOtherThan200IsAFailureNamedByItsCodeAndText(test *testing.T) {
	for status, want := range map[int]string{401: "status 401 Unauthorized", 404: "status 404 Not Found", 500: "status 500 Internal Server Error"} {
		_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(status)
			_, _ = writer.Write(oneDocument(test))
		})
		got, err := NewReader(address, time.Second).Read(context.Background())
		if err == nil || err.Error() != want || got != nil {
			test.Errorf("%d: got %v, %v", status, got, err)
		}
	}
}

func TestABodyThatIsNoJSONIsAFailure(test *testing.T) {
	_, address := serve(test, answering([]byte("<html>no</html>")))
	_, err := NewReader(address, time.Second).Read(context.Background())
	if err != ErrNotJSON {
		test.Errorf("error %v", err)
	}
}

func TestABodyThatIsNoTreeIsAFailure(test *testing.T) {
	_, address := serve(test, answering([]byte(`{"a": 1}`)))
	_, err := NewReader(address, time.Second).Read(context.Background())
	if err != ErrNotTree {
		test.Errorf("error %v", err)
	}
}

func TestAServerThatAnswersAfterTheTimeLimitFailsTheReadingWithinTheLimit(test *testing.T) {
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-request.Context().Done():
		}
		answering(oneDocument(test))(writer, request)
	})
	assertFailsWithin(test, NewReader(address, 50*time.Millisecond), context.Background())
}

func TestAServerThatHoldsTheBodyOpenFailsTheReadingWithinTheLimit(test *testing.T) {
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "100000")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"Children":`))
		writer.(http.Flusher).Flush()
		waitForTheClient(request)
	})
	assertFailsWithin(test, NewReader(address, 50*time.Millisecond), context.Background())
}

func assertFailsWithin(test *testing.T, reader *Reader, stop context.Context) {
	test.Helper()
	started := time.Now()
	got, err := reader.Read(stop)
	if err == nil || got != nil {
		test.Errorf("got %v, %v", got, err)
	}
	if took := time.Since(started); took > 100*time.Millisecond {
		test.Errorf("the reading took %v", took)
	}
}

func TestAStopWhileWaitingEndsTheReadingAtOnce(test *testing.T) {
	arrived := make(chan struct{})
	var once sync.Once
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		once.Do(func() { close(arrived) })
		waitForTheClient(request)
	})
	stop, cancel := context.WithCancel(context.Background())
	go func() {
		<-arrived
		cancel()
	}()
	assertFailsWithin(test, NewReader(address, 10*time.Second), stop)
}

func TestTheUserInformationOfTheURLIsSentAsBasicAuthentication(test *testing.T) {
	var user, password atomic.Value
	var given atomic.Bool
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		name, secret, ok := request.BasicAuth()
		user.Store(name)
		password.Store(secret)
		given.Store(ok)
		answering(oneDocument(test))(writer, request)
	})
	address.User = url.UserPassword("monitor", "s3cret")
	if _, err := NewReader(address, time.Second).Read(context.Background()); err != nil {
		test.Fatal(err)
	}
	if !given.Load() || user.Load() != "monitor" || password.Load() != "s3cret" {
		test.Errorf("got %v %v %v", given.Load(), user.Load(), password.Load())
	}
}

func TestAGzipBodyIsRead(test *testing.T) {
	var packed bytes.Buffer
	writer := gzip.NewWriter(&packed)
	_, _ = writer.Write(oneDocument(test))
	_ = writer.Close()
	var accepted atomic.Value
	_, address := serve(test, func(response http.ResponseWriter, request *http.Request) {
		accepted.Store(request.Header.Get("Accept-Encoding"))
		response.Header().Set("Content-Encoding", "gzip")
		_, _ = response.Write(packed.Bytes())
	})
	got, err := NewReader(address, time.Second).Read(context.Background())
	if err != nil || len(got) != 3 {
		test.Errorf("got %v, %v", got, err)
	}
	if !strings.Contains(accepted.Load().(string), "gzip") {
		test.Errorf("Accept-Encoding %q", accepted.Load())
	}
}

func TestTwoReadingsUseOneConnection(test *testing.T) {
	var opened atomic.Int32
	server := httptest.NewUnstartedServer(answering(oneDocument(test)))
	server.Config.ConnState = func(connection net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	server.Start()
	test.Cleanup(server.Close)
	address, _ := url.Parse(server.URL + "/data.json")
	reader := NewReader(address, time.Second)
	for range 2 {
		if _, err := reader.Read(context.Background()); err != nil {
			test.Fatal(err)
		}
	}
	if opened.Load() != 1 {
		test.Errorf("%d connections", opened.Load())
	}
}

func TestARefusedConnectionNamesNeitherTheMethodNorTheURLOfTheRequest(test *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	address, _ := url.Parse(server.URL + "/data.json")
	server.Close()
	_, err := NewReader(address, time.Second).Read(context.Background())
	if err == nil || strings.Contains(err.Error(), "GET") || strings.Contains(err.Error(), "http://") || strings.Contains(err.Error(), "data.json") {
		test.Errorf("error %v", err)
	}
}

func TestNoProxyIsUsedWhateverTheEnvironmentSays(test *testing.T) {
	var proxied atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { proxied.Add(1) }))
	defer proxy.Close()
	test.Setenv("HTTP_PROXY", proxy.URL)
	test.Setenv("http_proxy", proxy.URL)
	test.Setenv("NO_PROXY", "")
	test.Setenv("no_proxy", "")
	_, address := serve(test, answering(oneDocument(test)))
	reader := NewReader(address, time.Second)
	if _, err := reader.Read(context.Background()); err != nil {
		test.Fatal(err)
	}
	if proxied.Load() != 0 {
		test.Errorf("the proxy saw %d requests", proxied.Load())
	}
	// Go never proxies a loopback address, so the part above cannot fail alone.
	if reader.client.Transport.(*http.Transport).Proxy != nil {
		test.Error("the transport has a proxy")
	}
}

func TestWantsAsksForTheNamesAMachineCanHave(test *testing.T) {
	for _, name := range []string{"cpu.temp", "cpu.clock", "cpu.ccd1.temp", "cpu.ccd64.temp", "ram.1.temp", "ram.64.temp", "nvme.1.temp", "nvme.64.temp"} {
		if !Wants(func(asked string) bool { return asked == name }) {
			test.Errorf("%s is not wanted", name)
		}
	}
	for _, name := range []string{"cpu.util", "ram.used", "net.up", "gpu.temp", "commander.1.temp", "nvme.65.temp", "ram.65.temp", "cpu.ccd65.temp"} {
		if Wants(func(asked string) bool { return asked == name }) {
			test.Errorf("%s is wanted", name)
		}
	}
}

func TestAReadingThatWantsNoMonitorMetricMakesNoRequestAndNoReport(test *testing.T) {
	var requests atomic.Int32
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		answering(oneDocument(test))(writer, request)
	})
	reported := 0
	source := Source{Reader: NewReader(address, time.Second), Report: func([]sensors.Reading, error) { reported++ }}
	got := source.Read(context.Background(), func(name string) bool { return name == "cpu.util" || name == "net.up" })
	if len(got) != 0 || requests.Load() != 0 || reported != 0 {
		test.Errorf("got %v, %d requests, %d reports", got, requests.Load(), reported)
	}
}

func TestOneReadingOfSeveralMetricsMakesOneRequestAndGivesOnlyTheWantedOnes(test *testing.T) {
	var requests atomic.Int32
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		answering(oneDocument(test))(writer, request)
	})
	var reported []sensors.Reading
	source := Source{Reader: NewReader(address, time.Second), Report: func(found []sensors.Reading, err error) {
		if err != nil {
			test.Error(err)
		}
		reported = found
	}}
	wanted := func(name string) bool { return name == "cpu.temp" || name == "nvme.1.temp" || name == "ram.1.temp" }
	got := source.Read(context.Background(), wanted)
	want := []sensors.Reading{reading("cpu.temp", 46.9), reading("ram.1.temp", 36), reading("nvme.1.temp", 35)}
	if !reflect.DeepEqual(got, want) || requests.Load() != 1 {
		test.Errorf("got %v, %d requests", got, requests.Load())
	}
	if len(reported) != 3 {
		test.Errorf("reported %v", reported)
	}
	if only := source.Read(context.Background(), func(name string) bool { return name == "nvme.1.temp" }); len(only) != 1 {
		test.Errorf("got %v", only)
	}
}

func TestTheReportIsGivenEveryReadingOfTheDocumentWhateverIsWanted(test *testing.T) {
	_, address := serve(test, answering(oneDocument(test)))
	var reported []sensors.Reading
	source := Source{Reader: NewReader(address, time.Second), Report: func(found []sensors.Reading, err error) { reported = found }}
	source.Read(context.Background(), func(name string) bool { return name == "cpu.temp" })
	if len(reported) != 3 {
		test.Errorf("reported %v", reported)
	}
}

func TestTheReportIsGivenTheErrorOfAFailure(test *testing.T) {
	_, address := serve(test, func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(http.StatusUnauthorized) })
	var failure error
	source := Source{Reader: NewReader(address, time.Second), Report: func(found []sensors.Reading, err error) { failure = err }}
	got := source.Read(context.Background(), everything)
	if len(got) != 0 || failure == nil || failure.Error() != "status 401 Unauthorized" {
		test.Errorf("got %v, %v", got, failure)
	}
}
