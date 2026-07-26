package staff

import (
	"github.com/KNattawat89/hospital-middleware-system/infra/web"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"core/staff",
	fx.Provide(
		NewRepo,
		NewService,
		fx.Annotate(
			NewHandler,
			fx.As(new(web.Route)),
			fx.ResultTags(`group:"routes"`),
		),
	),
)
