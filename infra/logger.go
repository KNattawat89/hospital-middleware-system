package infra

import (
	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"go.uber.org/zap"
)

func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	if cfg.App.Debug {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
