package config

import (
	"github.com/bugfixes/go-bugfixes/logs"
	"github.com/caarlos0/env/v8"
)

type K8sDeploy struct {
	MinimumAgents              int `json:"minimum_agents" envDefault:"2"`
	MinimumGrandfatheredAgents int `json:"minimum_grandfathered_agents" envDefault:"10"`
}

func BuildK8sDeploy(c *Config) error {
	cfg := &K8sDeploy{}

	if err := env.Parse(cfg); err != nil {
		return logs.Errorf("error parsing k8sdeploy: %v", err)
	}
	c.K8sDeploy = *cfg
	return nil
}
