package main

import (
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	goreleaserPath = ".goreleaser.yaml"
	releasePath    = ".github/workflows/release.yml"
	ciPath         = ".github/workflows/ci.yml"
)

var (
	exactActionTag  = regexp.MustCompile(`@v\d+\.\d+\.\d+$`)
	exactToolTag    = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	suiteCommands   = []string{"go test ./...", "go vet ./...", "gofmt -l .", "GOOS=windows GOARCH=amd64 go build ./...", "GOOS=windows GOARCH=amd64 go vet ./..."}
	wantedPlatforms = []string{"linux/amd64", "linux/arm64", "windows/amd64"}
)

// parseYAML parses the file into a value of the type the caller names.
func parseYAML[Document goreleaserConfig | workflow](test *testing.T, path string) Document {
	test.Helper()
	var document Document
	content, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	if err := yaml.Unmarshal(content, &document); err != nil {
		test.Fatalf("%s: %v", path, err)
	}
	return document
}

type goreleaserConfig struct {
	Version int    `yaml:"version"`
	Dist    string `yaml:"dist"`
	Builds  []struct {
		Main    string   `yaml:"main"`
		Env     []string `yaml:"env"`
		Flags   []string `yaml:"flags"`
		Ldflags []string `yaml:"ldflags"`
		Goos    []string `yaml:"goos"`
		Goarch  []string `yaml:"goarch"`
		Ignore  []struct {
			Goos   string `yaml:"goos"`
			Goarch string `yaml:"goarch"`
		} `yaml:"ignore"`
	} `yaml:"builds"`
	Archives []struct {
		Formats         []string `yaml:"formats"`
		NameTemplate    string   `yaml:"name_template"`
		Files           []string `yaml:"files"`
		FormatOverrides []struct {
			Goos    string   `yaml:"goos"`
			Formats []string `yaml:"formats"`
		} `yaml:"format_overrides"`
	} `yaml:"archives"`
	Checksum struct {
		Algorithm    string `yaml:"algorithm"`
		NameTemplate string `yaml:"name_template"`
	} `yaml:"checksum"`
	Release struct {
		Draft bool `yaml:"draft"`
	} `yaml:"release"`
	Changelog struct {
		Disable bool `yaml:"disable"`
	} `yaml:"changelog"`
}

func TestTheGoreleaserConfigurationBuildsTheArchivesOfTheSpec(test *testing.T) {
	config := parseYAML[goreleaserConfig](test, goreleaserPath)
	if config.Version != 2 {
		test.Errorf("version %d", config.Version)
	}
	if len(config.Builds) != 1 {
		test.Fatalf("%d builds", len(config.Builds))
	}
	build := config.Builds[0]
	if build.Main != "." {
		test.Errorf("main %q", build.Main)
	}
	if !slices.Contains(build.Env, "CGO_ENABLED=0") {
		test.Errorf("env %v", build.Env)
	}
	if !slices.Contains(build.Flags, "-trimpath") {
		test.Errorf("flags %v", build.Flags)
	}
	if !reflect.DeepEqual(build.Ldflags, []string{"-s -w -X main.version=v{{.Version}}"}) {
		test.Errorf("ldflags %q", build.Ldflags)
	}
	var platforms []string
	for _, goos := range build.Goos {
		for _, goarch := range build.Goarch {
			ignored := slices.ContainsFunc(build.Ignore, func(pair struct {
				Goos   string `yaml:"goos"`
				Goarch string `yaml:"goarch"`
			}) bool {
				return pair.Goos == goos && pair.Goarch == goarch
			})
			if !ignored {
				platforms = append(platforms, goos+"/"+goarch)
			}
		}
	}
	slices.Sort(platforms)
	if !reflect.DeepEqual(platforms, wantedPlatforms) {
		test.Errorf("platforms %v", platforms)
	}

	if len(config.Archives) != 1 {
		test.Fatalf("%d archives", len(config.Archives))
	}
	archive := config.Archives[0]
	wantFiles := []string{"README.md", "LICENSE", "THIRD_PARTY_LICENSES", "udev/70-sensordeck.rules"}
	if !reflect.DeepEqual(archive.Files, wantFiles) {
		test.Errorf("files %v", archive.Files)
	}
	if !reflect.DeepEqual(archive.Formats, []string{"tar.gz"}) {
		test.Errorf("formats %v", archive.Formats)
	}
	if len(archive.FormatOverrides) != 1 || archive.FormatOverrides[0].Goos != "windows" || !reflect.DeepEqual(archive.FormatOverrides[0].Formats, []string{"zip"}) {
		test.Errorf("format overrides %+v", archive.FormatOverrides)
	}
	if archive.NameTemplate != "" {
		test.Errorf("the archive sets a name template %q", archive.NameTemplate)
	}
	if config.Checksum.Algorithm != "sha256" || config.Checksum.NameTemplate != "" {
		test.Errorf("checksum %+v", config.Checksum)
	}
	if !config.Release.Draft {
		test.Error("the release is not a draft")
	}
	if !config.Changelog.Disable {
		test.Error("the changelog is not disabled")
	}
	if config.Dist == "" || strings.TrimSuffix(config.Dist, "/") == "dist" {
		test.Errorf("dist %q", config.Dist)
	}
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
}

type workflowJob struct {
	Uses        string            `yaml:"uses"`
	Needs       string            `yaml:"needs"`
	RunsOn      string            `yaml:"runs-on"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []workflowStep    `yaml:"steps"`
}

// pushTrigger holds the keys of the push trigger, each kept as it is so that a key the spec does not name is seen.
type pushTrigger map[string]yaml.Node

type workflow struct {
	// On keeps each trigger as a node, because a trigger may be empty and the keys of push are checked on their own.
	On          map[string]yaml.Node `yaml:"on"`
	Permissions map[string]string    `yaml:"permissions"`
	Jobs        map[string]workflowJob
}

func checkExactActions(test *testing.T, path string, job workflowJob) {
	test.Helper()
	for _, step := range job.Steps {
		if step.Uses != "" && !exactActionTag.MatchString(step.Uses) {
			test.Errorf("%s: %q is not pinned to an exact release tag", path, step.Uses)
		}
	}
}

func triggerNames(triggers map[string]yaml.Node) []string {
	var names []string
	for name := range triggers {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func TestTheReleaseWorkflowRunsTheSuiteBeforeGoreleaser(test *testing.T) {
	release := parseYAML[workflow](test, releasePath)
	if !reflect.DeepEqual(triggerNames(release.On), []string{"push"}) {
		test.Errorf("triggers %v", triggerNames(release.On))
	}
	var push pushTrigger
	pushNode := release.On["push"]
	if err := pushNode.Decode(&push); err != nil {
		test.Fatal(err)
	}
	tags := push["tags"]
	var tagPatterns []string
	if err := tags.Decode(&tagPatterns); err != nil {
		test.Fatal(err)
	}
	if len(push) != 1 || !reflect.DeepEqual(tagPatterns, []string{"v*"}) {
		test.Errorf("push keys %d, tags %v", len(push), tagPatterns)
	}
	if release.Permissions == nil || len(release.Permissions) != 0 {
		test.Errorf("top-level permissions %v", release.Permissions)
	}

	var suiteName, releaseName string
	for name, job := range release.Jobs {
		if job.Uses != "" {
			suiteName = name
		} else {
			releaseName = name
		}
	}
	suite, releasing := release.Jobs[suiteName], release.Jobs[releaseName]
	if suite.Uses != "./"+ciPath {
		test.Errorf("the suite job uses %q", suite.Uses)
	}
	if _, err := os.Stat(ciPath); err != nil {
		test.Error(err)
	}
	if !reflect.DeepEqual(suite.Permissions, map[string]string{"contents": "read"}) {
		test.Errorf("suite permissions %v", suite.Permissions)
	}
	if releasing.Needs != suiteName {
		test.Errorf("needs %v, want %q", releasing.Needs, suiteName)
	}
	if !reflect.DeepEqual(releasing.Permissions, map[string]string{"contents": "write"}) {
		test.Errorf("release permissions %v", releasing.Permissions)
	}
	checkExactActions(test, releasePath, releasing)
	var fullHistory, goVersionFile, goreleaser bool
	for _, step := range releasing.Steps {
		switch {
		case strings.HasPrefix(step.Uses, "actions/checkout@"):
			fullHistory = step.With["fetch-depth"] == "0"
		case strings.HasPrefix(step.Uses, "actions/setup-go@"):
			goVersionFile = step.With["go-version-file"] == "go.mod"
		case strings.HasPrefix(step.Uses, "goreleaser/goreleaser-action@"):
			goreleaser = step.With["args"] == "release --clean" && exactToolTag.MatchString(step.With["version"])
		}
	}
	if !fullHistory || !goVersionFile || !goreleaser {
		test.Errorf("full history %v, go version from go.mod %v, goreleaser release --clean at an exact version %v", fullHistory, goVersionFile, goreleaser)
	}
}

func TestTheCIWorkflowRunsTheSuite(test *testing.T) {
	ci := parseYAML[workflow](test, ciPath)
	if want := []string{"pull_request", "push", "workflow_call"}; !reflect.DeepEqual(triggerNames(ci.On), want) {
		test.Errorf("triggers %v", triggerNames(ci.On))
	}
	if !reflect.DeepEqual(ci.Permissions, map[string]string{"contents": "read"}) {
		test.Errorf("permissions %v", ci.Permissions)
	}
	if len(ci.Jobs) != 2 {
		test.Fatalf("%d jobs", len(ci.Jobs))
	}
	var suite, windows workflowJob
	for name, job := range ci.Jobs {
		checkExactActions(test, ciPath, job)
		switch job.RunsOn {
		case "ubuntu-24.04":
			suite = job
		case "windows-2025":
			windows = job
		default:
			test.Errorf("job %s runs on %q", name, job.RunsOn)
		}
	}
	checkSuiteJob(test, suite)
	checkWindowsJob(test, suite, windows)
}

func hasStep(job workflowJob, matches func(step workflowStep) bool) bool {
	return slices.ContainsFunc(job.Steps, matches)
}

func setsUpGoFromTheModule(step workflowStep) bool {
	return strings.HasPrefix(step.Uses, "actions/setup-go@") && step.With["go-version-file"] == "go.mod"
}

func checkSuiteJob(test *testing.T, suite workflowJob) {
	test.Helper()
	if !hasStep(suite, setsUpGoFromTheModule) {
		test.Error("no step of the suite job sets up Go from go.mod")
	}
	for _, command := range suiteCommands {
		if !hasStep(suite, func(step workflowStep) bool { return strings.Contains(step.Run, command) }) {
			test.Errorf("no step runs %q", command)
		}
	}
}

// checkWindowsJob checks the job on Windows: it checks out and sets up Go as the suite job does, and builds, vets and
// tests the three packages written for Windows, each as a step of its own.
func checkWindowsJob(test *testing.T, suite, windows workflowJob) {
	test.Helper()
	for _, action := range []func(step workflowStep) bool{
		func(step workflowStep) bool { return strings.HasPrefix(step.Uses, "actions/checkout@") },
		setsUpGoFromTheModule,
	} {
		inSuite := slices.IndexFunc(suite.Steps, action)
		inWindows := slices.IndexFunc(windows.Steps, action)
		if inSuite < 0 || inWindows < 0 || suite.Steps[inSuite].Uses != windows.Steps[inWindows].Uses {
			test.Error("the job on Windows does not use the actions of the suite job at the same version")
		}
	}
	for _, command := range []string{"go build ./...", "go vet ./...", "go test ./internal/winhid ./internal/commandercore ./internal/windowssensors"} {
		if !hasStep(windows, func(step workflowStep) bool { return strings.TrimSpace(step.Run) == command }) {
			test.Errorf("no step of the job on Windows runs exactly %q", command)
		}
	}
}
