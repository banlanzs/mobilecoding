package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws"}, false},
		{"missing port", Config{AuthToken: "abc", Workspace: "/tmp/ws"}, true},
		{"missing token", Config{Port: "8443", Workspace: "/tmp/ws"}, true},
		{"missing workspace", Config{Port: "8443", AuthToken: "abc"}, true},
		{"mtls optional ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", MTLS: "optional"}, false},
		{"mtls required ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", MTLS: "required"}, false},
		{"mtls none rejected", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", MTLS: "none"}, true},
		{"mtls invalid", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", MTLS: "bogus"}, true},
		{"launch mode managed ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", LaunchMode: "managed"}, false},
		{"launch mode remote control ok", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", LaunchMode: "remote-control"}, false},
		{"launch mode invalid", Config{Port: "8443", AuthToken: "abc", Workspace: "/tmp/ws", LaunchMode: "bogus"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.cfg.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	c := Config{}.WithDefaults()
	if c.Port != "8443" {
		t.Errorf("default port = %q, want 8443", c.Port)
	}
	if c.LogLevel != "info" {
		t.Errorf("default log level = %q, want info", c.LogLevel)
	}
	if c.MTLS != "optional" {
		t.Errorf("default mtls = %q, want optional", c.MTLS)
	}
	if c.LaunchMode != "managed" {
		t.Errorf("default launch mode = %q, want managed", c.LaunchMode)
	}
}
