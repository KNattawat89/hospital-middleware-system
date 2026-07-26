package staff

import (
	"errors"
	"net/http"

	"github.com/KNattawat89/hospital-middleware-system/infra/utils"
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
	group := router.Group("/staff")
	group.POST("/create", h.create)
	group.POST("/login", h.login)
}

type createRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

// create godoc
// @Summary Create a new hospital staff member
// @Description Creates a staff login scoped to a single hospital.
// @Tags staff
// @Accept json
// @Produce json
// @Param body body createRequest true "Staff credentials"
// @Success 201 {object} map[string]any
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /staff/create [post]
func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.BadRequestErrResponse)
		return
	}

	staffRecord, err := h.service.Create(CreateInput{
		Username: req.Username,
		Password: req.Password,
		Hospital: req.Hospital,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrHospitalNotFound):
			c.JSON(http.StatusNotFound, utils.NotFoundErrResponse)
		case errors.Is(err, ErrUsernameTaken):
			c.JSON(http.StatusConflict, utils.ConflictErrResponse)
		default:
			c.JSON(http.StatusInternalServerError, utils.InternalServerErrorErrResponse)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       staffRecord.ID,
		"username": staffRecord.Username,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

// login godoc
// @Summary Staff login
// @Description Authenticates a staff member scoped to a hospital and issues a JWT token pair.
// @Tags staff
// @Accept json
// @Produce json
// @Param body body loginRequest true "Login credentials"
// @Success 200 {object} auth.TokenPair
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Router /staff/login [post]
func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utils.BadRequestErrResponse)
		return
	}

	pair, err := h.service.Login(LoginInput{
		Username: req.Username,
		Password: req.Password,
		Hospital: req.Hospital,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, utils.UnauthorizedErrResponse)
		} else {
			c.JSON(http.StatusInternalServerError, utils.InternalServerErrorErrResponse)
		}
		return
	}

	c.JSON(http.StatusOK, pair)
}
