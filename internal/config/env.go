package config

import (
	"github.com/caarlos0/env/v6"
	"log"
)

func parseEnv() Config {

	var config Config

	err := env.Parse(&config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}
