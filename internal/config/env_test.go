package config

import (
	"os"
	"reflect"
	"testing"
)

func Test_parseEnv(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		want    Config
		isEqual bool
	}{
		{
			"positive test #1",
			Config{Endpoint: "localhost:11"},
			Config{Endpoint: "localhost:11"},
			true,
		},
		{
			"negative test #2",
			Config{Endpoint: "localhost:11"},
			Config{Endpoint: "localhost:154"},
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if len(test.cfg.Endpoint) > 0 {
				if err := os.Setenv("ADDRESS", test.cfg.Endpoint); err != nil {
					t.Errorf("ошибка setenv %s", err.Error())
				}
			}

			if got := parseEnv(); reflect.DeepEqual(got, test.want) != test.isEqual {
				t.Errorf("parseEnv() = %v, want %v", got, test.want)
			}
		})
	}
}
