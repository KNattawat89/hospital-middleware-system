package main

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// @title Hospital Middleware System
// @description API documentation for Hospital Middleware System
// @version 1
// @host localhost:8080
// @BasePath /
// @produce json
// @consumes json
func main() {
	Setup(
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
	).Run()
}
