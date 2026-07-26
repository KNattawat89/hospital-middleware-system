package web

import "github.com/gin-gonic/gin"

// Route is implemented by handlers that want to register endpoints on the
// engine. Handlers contribute a Route to fx's "routes" group so infra/web
// stays decoupled from core.
type Route interface {
	Register(router gin.IRouter)
}
