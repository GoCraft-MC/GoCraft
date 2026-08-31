package gocraft

import "testing"

type noopPlugin struct{}

func (*noopPlugin) OnLoad(Context) error { return nil }
func (*noopPlugin) OnEnable() error      { return nil }
func (*noopPlugin) OnDisable() error     { return nil }

func TestParseRunnerOptions(t *testing.T) {
	options, err := parseRunnerOptions([]string{"--sock", "runtime.sock", "--abi", "1"})
	if err != nil || options.socket != "runtime.sock" || options.abi != 1 {
		t.Fatalf("parseRunnerOptions() = %#v, %v", options, err)
	}
	for _, arguments := range [][]string{
		{"--abi", "1"},
		{"--sock", "runtime.sock", "--abi", "2"},
		{"--unknown"},
	} {
		if _, err := parseRunnerOptions(arguments); err == nil {
			t.Fatalf("parseRunnerOptions(%v) accepted invalid input", arguments)
		}
	}
}

func TestValidateMetadata(t *testing.T) {
	valid := Metadata{ID: "example", Version: "1.0.0", APIVersion: CurrentVersion}
	if err := validateMetadata(valid, &noopPlugin{}); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.APIVersion++
	if err := validateMetadata(invalid, &noopPlugin{}); err == nil {
		t.Fatal("unsupported API version was accepted")
	}
	if err := validateMetadata(valid, nil); err == nil {
		t.Fatal("nil plugin was accepted")
	}
}
