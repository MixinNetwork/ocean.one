package main

import (
	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/bugsnag/bugsnag-go"
)

func setupBugsnag() {
	logger := &bugsnagLogger{}
	bugsnag.Configure(bugsnag.Configuration{
		APIKey:              config.BugsnagAPIKey,
		AppVersion:          config.BuildVersion,
		ReleaseStage:        config.Environment,
		NotifyReleaseStages: []string{"development", "staging", "production"},
		PanicHandler:        func() {},
		Logger:              logger,
	})
}

type bugsnagLogger struct{}

func (logger *bugsnagLogger) Printf(format string, v ...interface{}) {
}
