package coscred

import "testing"

func TestConfig_Validate(t *testing.T) {
	cfg := Config{
		AppID: "1", SecretID: "s", SecretKey: "k", Bucket: "b",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Region != defaultRegion {
		t.Errorf("Region = %q", cfg.Region)
	}
	if cfg.DurationSeconds != defaultDurationSeconds {
		t.Errorf("DurationSeconds = %d", cfg.DurationSeconds)
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
	if cfg.Region == "" || cfg.DurationSeconds <= 0 {
		t.Fatalf("defaults: %+v", cfg)
	}
}
