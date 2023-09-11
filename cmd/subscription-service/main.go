package main

import (
	"github.com/bugfixes/go-bugfixes/logs"
	"github.com/k8sdeploy/subscription-service/internal/config"
	"github.com/k8sdeploy/subscription-service/internal/service"
)

var (
	BuildVersion = "dev"
	BuildHash    = "none"
	ServiceName  = "agent-service"
)

func main() {
	logs.Local().Infof("Starting %s", ServiceName)
	logs.Local().Infof("Version: %s, Hash: %s", BuildVersion, BuildHash)

	cfg, err := config.Build()
	if err != nil {
		_ = logs.Errorf("unable to build config: %v", err)
		return
	}

	if err := service.NewService(*cfg).Start(); err != nil {
		_ = logs.Errorf("unable to start service: %v", err)
		return
	}
}
