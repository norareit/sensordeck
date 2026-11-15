package librehardwaremonitor

import (
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/norareit/sensordeck/internal/sensors"
)

// node is a node of the tree in the shape of "What is known" of the spec. Its fields are left as JSON, so that a test
// can give a field the wrong type.
type node map[string]json.RawMessage

// quoted is a JSON string.
func quoted(text string) json.RawMessage {
	encoded, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return encoded
}

// array is a JSON array of the nodes.
func array(nodes ...node) json.RawMessage {
	encoded, err := json.Marshal(append([]node{}, nodes...))
	if err != nil {
		panic(err)
	}
	return encoded
}

func group(text string, children ...node) node {
	return node{"id": json.RawMessage(`1`), "Text": quoted(text), "Min": quoted(""), "Value": quoted(""), "Max": quoted(""),
		"ImageURL": quoted(""), "Children": array(children...)}
}

func sensor(id, kind, text, value string) node {
	return node{"id": json.RawMessage(`2`), "Text": quoted(text), "Min": quoted(value), "Value": quoted(value), "Max": quoted(value),
		"ImageURL": quoted(""), "SensorId": quoted(id), "Type": quoted(kind), "Children": array()}
}

func temperature(id, text, value string) node { return sensor(id, "Temperature", text, value) }
func clock(id, text, value string) node       { return sensor(id, "Clock", text, value) }

// document wraps the hardware in the root and the computer.
func document(test *testing.T, hardware ...node) []byte {
	test.Helper()
	root := node{"id": json.RawMessage(`0`), "Text": quoted("Sensor"), "Min": quoted("Min"), "Value": quoted("Value"),
		"Max": quoted("Max"), "ImageURL": quoted(""), "Children": array(group("DESKTOP", hardware...))}
	body, err := json.Marshal(root)
	if err != nil {
		test.Fatal(err)
	}
	return body
}

func mapped(test *testing.T, hardware ...node) []sensors.Reading {
	test.Helper()
	got, err := Map(document(test, hardware...))
	if err != nil {
		test.Fatal(err)
	}
	return got
}

func reading(name string, value float64) sensors.Reading {
	return sensors.Reading{Name: name, Value: value}
}

// hardwareGroup is a hardware node that holds the sensors directly, which is enough for the mapping, since it looks at
// every depth.
func hardwareGroup(sensorNodes ...node) node { return group("hardware", sensorNodes...) }

func TestAnAMDCPUGivesTheControlTemperatureTheCCDsAndTheHighestCoreClock(test *testing.T) {
	got := mapped(test, hardwareGroup(
		group("Temperatures",
			temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C"),
			temperature("/amdcpu/0/temperature/3", "CCD1 (Tdie)", "41.5 °C"),
			temperature("/amdcpu/0/temperature/4", "CCD2 (Tdie)", "43.0 °C"),
			temperature("/amdcpu/0/temperature/5", "CCDs Max (Tdie)", "43.0 °C"),
			temperature("/amdcpu/0/temperature/6", "CCDs Average (Tdie)", "42.3 °C")),
		group("Clocks",
			clock("/amdcpu/0/clock/1", "Core #1", "4425.4 MHz"),
			clock("/amdcpu/0/clock/2", "Core #2", "-"),
			clock("/amdcpu/0/clock/3", "Core #3", "4300.0 MHz"),
			clock("/amdcpu/0/clock/4", "Core #4", "3900.1 MHz"),
			clock("/amdcpu/0/clock/5", "Core #1 (Effective)", "5100.0 MHz"),
			clock("/amdcpu/0/clock/6", "Bus Speed", "100.0 MHz"),
			clock("/amdcpu/0/clock/7", "Cores (Average)", "6000.0 MHz"))))
	want := []sensors.Reading{
		reading("cpu.temp", 46.9), reading("cpu.ccd1.temp", 41.5), reading("cpu.ccd2.temp", 43), reading("cpu.clock", 4425.4),
	}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestTctlAloneWhenTheCPUHasAnOffset(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures",
		temperature("/amdcpu/0/temperature/0", "Core (Tdie)", "40.0 °C"),
		temperature("/amdcpu/0/temperature/1", "Core (Tctl)", "50.0 °C"))))
	if want := []sensors.Reading{reading("cpu.temp", 50)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestTctlTdieIsPreferredToTctlWhateverTheOrder(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures",
		temperature("/amdcpu/0/temperature/0", "CPU Package", "30.0 °C"),
		temperature("/amdcpu/0/temperature/1", "Core (Tctl)", "50.0 °C"),
		temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "60.0 °C"))))
	if want := []sensors.Reading{reading("cpu.temp", 60)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestAnIntelCPUGivesThePackageTemperatureAndTheHighestCoreClock(test *testing.T) {
	got := mapped(test, hardwareGroup(
		group("Temperatures",
			temperature("/intelcpu/0/temperature/0", "Core Max", "70.0 °C"),
			temperature("/intelcpu/0/temperature/1", "CPU Package", "65.0 °C")),
		group("Clocks",
			clock("/intelcpu/0/clock/0", "Bus Speed", "100.0 MHz"),
			clock("/intelcpu/0/clock/1", "P-Core #1", "5000.0 MHz"),
			clock("/intelcpu/0/clock/2", "E-Core #1", "3800.0 MHz"))))
	want := []sensors.Reading{reading("cpu.temp", 65), reading("cpu.clock", 5000)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestACPUOfOneCoreGivesItsClock(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Clocks", clock("/intelcpu/0/clock/1", "CPU Core", "2400.0 MHz"))))
	if want := []sensors.Reading{reading("cpu.clock", 2400)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestTheAMDCPUIsTakenWhenTheDocumentHasBoth(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures", temperature("/intelcpu/0/temperature/0", "CPU Package", "65.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/amdcpu/0/temperature/0", "Core (Tctl/Tdie)", "45.0 °C"))))
	if want := []sensors.Reading{reading("cpu.temp", 45)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestAnAMDCPUWithoutAControlSensorDoesNotFallBackToTheIntelOne(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures", temperature("/intelcpu/0/temperature/0", "CPU Package", "65.0 °C"))),
		hardwareGroup(group("Clocks", clock("/amdcpu/0/clock/1", "Core #1", "4000.0 MHz"))))
	if want := []sensors.Reading{reading("cpu.clock", 4000)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestAnotherCPUPackageGivesNothing(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures", temperature("/amdcpu/1/temperature/0", "Core (Tctl/Tdie)", "45.0 °C"))))
	if len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestTheValueIsReadAsRDoesTwentySays(test *testing.T) {
	cases := []struct {
		name, kind, value string
		want              float64
		ok                bool
	}{
		{"point", "Temperature", "46.9 °C", 46.9, true},
		{"comma", "Temperature", "46,9 °C", 46.9, true},
		{"whole", "Temperature", "47 °C", 47, true},
		{"fahrenheit", "Temperature", "116.0 °F", 46.666666666666664, true},
		{"fahrenheit freezing", "Temperature", "32,0 °F", 0, true},
		{"minus", "Temperature", "-5.5 °C", -5.5, true},
		{"unicode minus", "Temperature", "−5.5 °C", -5.5, true},
		{"unicode minus comma", "Temperature", "−5,5 °C", -5.5, true},
		{"spaces around", "Temperature", "  46.9 °C ", 46.9, true},
		{"two spaces inside", "Temperature", "46.9  °C", 46.9, true},
		{"clock", "Clock", "4425.4 MHz", 4425.4, true},
		{"clock comma", "Clock", "4425,4 MHz", 4425.4, true},
		{"dash", "Temperature", "-", 0, false},
		{"empty", "Temperature", "", 0, false},
		{"not a number", "Temperature", "NaN °C", 0, false},
		{"no space", "Temperature", "46.9°C", 0, false},
		{"no unit", "Temperature", "46.9", 0, false},
		{"clock unit for a temperature", "Temperature", "46.9 MHz", 0, false},
		{"temperature unit for a clock", "Clock", "46.9 °C", 0, false},
		{"another type", "Load", "46.9 %", 0, false},
		{"group separator", "Clock", "1,234.5 MHz", 0, false},
		{"two decimal separators", "Clock", "1.234,5 MHz", 0, false},
		{"plus sign", "Temperature", "+46.9 °C", 0, false},
		{"separator without digits", "Temperature", "46. °C", 0, false},
	}
	for _, each := range cases {
		id, text := "/amdcpu/0/temperature/0", "Core (Tctl/Tdie)"
		if each.kind == "Clock" {
			id, text = "/amdcpu/0/clock/1", "Core #1"
		}
		body := document(test, hardwareGroup(sensor(id, each.kind, text, each.value)))
		got, err := Map(body)
		if err != nil {
			test.Fatal(err)
		}
		if each.ok {
			if len(got) != 1 || got[0].Value < each.want-1e-9 || got[0].Value > each.want+1e-9 {
				test.Errorf("%s: got %v, want %v", each.name, got, each.want)
			}
		} else if len(got) != 0 {
			test.Errorf("%s: got %v, want nothing", each.name, got)
		}
	}
}

func TestAClockWithoutAValueIsPassedOverAndTheHighestOfTheRestIsGiven(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Clocks",
		clock("/amdcpu/0/clock/1", "Core #1", "-"),
		clock("/amdcpu/0/clock/2", "Core #2", "3000.0 MHz"))))
	if want := []sensors.Reading{reading("cpu.clock", 3000)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
	got = mapped(test, hardwareGroup(group("Clocks", clock("/amdcpu/0/clock/1", "Core #1", "-"))))
	if len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestDrivesAreNumberedByTheirNumberAndKeepItWithoutATemperature(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures", temperature("/nvme/10/temperature/0", "Temperature", "55.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/nvme/0/temperature/0", "Temperature", "35.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/nvme/2/temperature/1", "Temperature 1", "40.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/nvme/0/temperature/1", "Temperature 1", "99.0 °C"))))
	want := []sensors.Reading{reading("nvme.1.temp", 35), reading("nvme.3.temp", 55)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestTheTextOfADriveSensorIsNotLookedAt(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures", temperature("/nvme/0/temperature/0", "Renamed by the user", "35.0 °C"))))
	if want := []sensors.Reading{reading("nvme.1.temp", 35)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestOtherDriveSensorsAndOtherHardwareGiveNothing(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures",
			temperature("/nvme/0/temperature/1", "Warning Temperature", "70.0 °C"),
			temperature("/nvme/0/temperature/2", "Temperature 1", "40.0 °C"),
			temperature("/scsi/1/temperature/0", "Temperature", "30.0 °C"),
			temperature("/gpu-nvidia/0/temperature/0", "GPU Core", "50.0 °C"),
			temperature("/lpcio/nct6798d/0/temperature/0", "Temperature #1", "28.0 °C"))))
	if len(got) != 0 {
		test.Errorf("got %v", got)
	}
}

func TestAHardwareThatIsNoDriveNumberIsPassedOver(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures",
		temperature("/nvme/id_error/temperature/0", "Temperature", "30.0 °C"),
		temperature("/nvme/1/temperature/0", "Temperature", "31.0 °C"))))
	if want := []sensors.Reading{reading("nvme.1.temp", 31)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestModulesWithAThermalSensorAreNumberedAndOthersTakeNoNumber(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures",
			temperature("/memory/dimm/3/temperature/0", "DIMM #3", "38.0 °C"),
			temperature("/memory/dimm/3/temperature/1", "Thermal Sensor Low Limit", "0.0 °C"),
			temperature("/memory/dimm/3/temperature/2", "Thermal Sensor High Limit", "85.0 °C"))),
		hardwareGroup(group("Timings", sensor("/memory/dimm/2/timing/0", "Timing", "CL", "40"))),
		hardwareGroup(group("Temperatures",
			temperature("/memory/dimm/1/temperature/0", "DIMM #1", "36.5 °C"),
			temperature("/memory/dimm/1/temperature/3", "Thermal Sensor Critical Limit", "95.0 °C"))))
	want := []sensors.Reading{reading("ram.1.temp", 36.5), reading("ram.2.temp", 38)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestAModuleWithoutAValueKeepsItsNumber(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures",
		temperature("/memory/dimm/1/temperature/0", "DIMM #1", "-"),
		temperature("/memory/dimm/2/temperature/0", "DIMM #2", "39.0 °C"))))
	if want := []sensors.Reading{reading("ram.2.temp", 39)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestNodesOfAnotherShapeAreIgnoredAndTheRestIsRead(test *testing.T) {
	odd := sensor("/amdcpu/0/temperature/9", "Temperature", "Core #9", "50.0 °C")
	odd["Value"] = json.RawMessage(`50`) // a field of another type: no sensor
	notArray := group("Odd")
	notArray["Children"] = quoted("none")
	withNumbers := temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C")
	withNumbers["RawValue"] = json.RawMessage(`46.9`) // master writes numbers
	withNumbers["RawMin"] = json.RawMessage(`null`)
	withNumbers["HardwareId"] = quoted("/amdcpu/0")
	stringRaw := clock("/amdcpu/0/clock/1", "Core #1", "4000.0 MHz")
	stringRaw["RawValue"] = quoted("4000.0 MHz") // v0.9.6 writes strings
	short := sensor("/amdcpu/0/clock/2", "Clock", "Core #2", "9000.0 MHz")
	short["SensorId"] = quoted("/amdcpu/clock")
	noDigits := sensor("/amdcpu/0/clock/x", "Clock", "Core #3", "9000.0 MHz")
	noType := clock("/amdcpu/0/clock/4", "Core #4", "9000.0 MHz")
	delete(noType, "Type")
	root := node{"Version": quoted("0.9.7"), "Text": quoted("Sensor"), "Children": array(
		group("DESKTOP", hardwareGroup(
			notArray, odd, withNumbers,
			group("board", group("chip", group("Clocks", stringRaw, short, noDigits, noType))))))}
	body, err := json.Marshal(root)
	if err != nil {
		test.Fatal(err)
	}
	got, err := Map(body)
	if err != nil {
		test.Fatal(err)
	}
	want := []sensors.Reading{reading("cpu.temp", 46.9), reading("cpu.clock", 4000)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestANodeWhoseChildrenAreNotAnArrayStillCountsAsASensor(test *testing.T) {
	cpu := temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C")
	cpu["Children"] = json.RawMessage(`null`)
	got := mapped(test, hardwareGroup(cpu))
	if want := []sensors.Reading{reading("cpu.temp", 46.9)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestNumbersAboveTheLimitGiveNothing(test *testing.T) {
	nodes := []node{temperature("/amdcpu/0/temperature/0", "Core (Tctl/Tdie)", "45.0 °C")}
	nodes = append(nodes,
		temperature("/amdcpu/0/temperature/1", "CCD64 (Tdie)", "50.0 °C"),
		temperature("/amdcpu/0/temperature/2", "CCD65 (Tdie)", "51.0 °C"))
	for number := 0; number < 65; number++ {
		nodes = append(nodes, temperature("/nvme/"+strconv.Itoa(number)+"/temperature/0", "Temperature", "30.0 °C"))
	}
	got := mapped(test, hardwareGroup(nodes...))
	names := map[string]bool{}
	for _, each := range got {
		names[each.Name] = true
	}
	if !names["cpu.ccd64.temp"] || names["cpu.ccd65.temp"] || !names["nvme.64.temp"] || names["nvme.65.temp"] {
		test.Errorf("got %v", got)
	}
}

func TestCCDNamesNeedANumberWithoutLeadingZeros(test *testing.T) {
	got := mapped(test, hardwareGroup(group("Temperatures",
		temperature("/amdcpu/0/temperature/1", "CCD0 (Tdie)", "40.0 °C"),
		temperature("/amdcpu/0/temperature/2", "CCD01 (Tdie)", "40.0 °C"),
		temperature("/amdcpu/0/temperature/3", "CCD3 (Tdie)", "41.0 °C"))))
	if want := []sensors.Reading{reading("cpu.ccd3.temp", 41)}; !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}

func TestTheReadingsComeInTheOrderOfTheTable(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures", temperature("/nvme/0/temperature/0", "Temperature", "35.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/memory/dimm/0/temperature/0", "DIMM #0", "36.0 °C"))),
		hardwareGroup(
			group("Clocks", clock("/amdcpu/0/clock/1", "Core #1", "4000.0 MHz")),
			group("Temperatures",
				temperature("/amdcpu/0/temperature/4", "CCD2 (Tdie)", "42.0 °C"),
				temperature("/amdcpu/0/temperature/3", "CCD1 (Tdie)", "41.0 °C"),
				temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C"))))
	var names []string
	for _, each := range got {
		names = append(names, each.Name)
	}
	want := []string{"cpu.temp", "cpu.ccd1.temp", "cpu.ccd2.temp", "cpu.clock", "ram.1.temp", "nvme.1.temp"}
	if !reflect.DeepEqual(names, want) {
		test.Errorf("got %v", names)
	}
}

func TestADocumentThatHoldsNoMetricGivesNoReadingAndNoError(test *testing.T) {
	got, err := Map(document(test, hardwareGroup(group("Load", sensor("/gpu-nvidia/0/load/0", "Load", "GPU Core", "5.0 %")))))
	if err != nil || len(got) != 0 {
		test.Errorf("got %v, %v", got, err)
	}
	got, err = Map([]byte(`{"Children": []}`))
	if err != nil || len(got) != 0 {
		test.Errorf("got %v, %v", got, err)
	}
}

func TestADocumentWithoutSensorIdGivesNoMetric(test *testing.T) {
	old := temperature("/amdcpu/0/temperature/2", "Core (Tctl/Tdie)", "46.9 °C")
	delete(old, "SensorId")
	got, err := Map(document(test, hardwareGroup(old)))
	if err != nil || len(got) != 0 {
		test.Errorf("got %v, %v", got, err)
	}
}

func TestEachFailureThatTheBodyDecides(test *testing.T) {
	cases := []struct {
		name, body string
		want       error
	}{
		{"empty", "", ErrNotJSON},
		{"text", "<html>", ErrNotJSON},
		{"truncated", `{"Children": [`, ErrNotJSON},
		{"two values", `{"Children": []}{"Children": []}`, ErrNotJSON},
		{"a value and text", `{"Children": []} x`, ErrNotJSON},
		{"array", `[]`, ErrNotTree},
		{"string", `"data"`, ErrNotTree},
		{"null", `null`, ErrNotTree},
		{"no children", `{"Text": "Sensor"}`, ErrNotTree},
		{"children is a string", `{"Children": "none"}`, ErrNotTree},
		{"children is null", `{"Children": null}`, ErrNotTree},
		{"children is an object", `{"Children": {}}`, ErrNotTree},
	}
	for _, each := range cases {
		got, err := Map([]byte(each.body))
		if !errors.Is(err, each.want) || got != nil {
			test.Errorf("%s: got %v, %v", each.name, got, err)
		}
	}
	if ErrNotJSON.Error() != "the answer is not JSON" || ErrNotTree.Error() != "the answer is not the tree of data.json" {
		test.Error("the texts of the errors changed")
	}
}

func TestNumbersAreOrderedByValueWithoutOverflow(test *testing.T) {
	got := mapped(test,
		hardwareGroup(group("Temperatures", temperature("/nvme/99999999999999999999999/temperature/0", "Temperature", "55.0 °C"))),
		hardwareGroup(group("Temperatures", temperature("/nvme/9/temperature/0", "Temperature", "33.0 °C"))))
	want := []sensors.Reading{reading("nvme.1.temp", 33), reading("nvme.2.temp", 55)}
	if !reflect.DeepEqual(got, want) {
		test.Errorf("got %v", got)
	}
}
