package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Setup registers every route contributed to the "routes" fx group onto the engine,
// plus the embedded swagger UI.
func Setup(engine *gin.Engine, routes []Route) {
	registerSwagger(engine)

	for _, route := range routes {
		route.Register(engine)
	}
}

// Start boots the HTTP server on cfg.App.Port and wires it to the fx lifecycle.
func Start(lc fx.Lifecycle, engine *gin.Engine, cfg *config.Config, log *zap.Logger) {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.App.Port),
		Handler: engine,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("listen %s: %w", srv.Addr, err)
			}

			log.Info("starting http server", zap.String("addr", srv.Addr))
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error("http server stopped unexpectedly", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
