package config

import (
	"github.com/bugfixes/go-bugfixes/logs"
	"github.com/caarlos0/env/v8"
	ConfigBuilder "github.com/keloran/go-config"
)

type Config struct {
	K8sDeploy
	ConfigBuilder.Config
}

func Build() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, logs.Error(err)
	}

	c, err := ConfigBuilder.Build(ConfigBuilder.Local, ConfigBuilder.Vault, ConfigBuilder.Mongo)
	if err != nil {
		return nil, logs.Errorf("config builder: %s", err)
	}
	cfg.Config = *c

	if err := BuildK8sDeploy(cfg); err != nil {
		return nil, logs.Error(err)
	}

	return cfg, nil
}
