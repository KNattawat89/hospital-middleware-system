package main

import (
	"github.com/KNattawat89/hospital-middleware-system/core"

	"github.com/KNattawat89/hospital-middleware-system/infra"
	"github.com/KNattawat89/hospital-middleware-system/infra/db"
	"go.uber.org/fx"
)

func Setup(options ...fx.Option) *fx.App {
	return fx.New(
		fx.Options(options...),
		core.Modules,
		infra.Modules,
		db.Module,
	)
}
