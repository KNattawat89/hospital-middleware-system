package infra

import (
	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"github.com/KNattawat89/hospital-middleware-system/infra/web"
	"go.uber.org/fx"
)

var Modules = fx.Module(
	"infrastructure",
	fx.Provide(
		config.NewConfig,
		NewLogger,
	),
	web.Module,

	fx.Invoke(
		fx.Annotate(web.Setup, fx.ParamTags(``, `group:"routes"`)),
		web.Start,
	),
)
