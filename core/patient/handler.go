package patient

import (
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register implements web.Route so fx wires this handler onto the Gin engine.
func (h *Handler) Register(router gin.IRouter) {
	// group := router.Group("/patients")
}
