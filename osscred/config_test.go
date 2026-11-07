package osscred

import "testing"

func TestConfig_Validate(t *testing.T) {
	cfg := Config{AccessKeyID: "a", AccessKeySecret: "b", RoleARN: "r"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.SessionName != defaultSessionName {
		t.Errorf("SessionName = %q", cfg.SessionName)
	}
}

func TestConfig_Validate_MissingFields(t *testing.T) {
	var cfg Config
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.SessionName == "" {
		t.Fatal("expected default session name")
	}
}
