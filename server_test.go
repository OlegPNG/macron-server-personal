package main

import (
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestConfigLoad(t *testing.T) {
    testString := `
    [Server]
    auth_type = "password"
    password = "foobar"
`
    expected := Config {
	Server: ServerConfig{
	    AuthType: "password",
	    Password: "foobar",
	},
    }
    var cfg Config 
    err := toml.Unmarshal([]byte(testString), &cfg)
    if err != nil {
	t.Fatalf("Error unmarshalling config string: %v", err)
    }
    if cfg != expected {
	t.Fatalf("got=%v, expected=%v", cfg, expected)
    }

}
