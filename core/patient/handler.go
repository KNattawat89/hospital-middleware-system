package patient

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/KNattawat89/hospital-middleware-system/infra/auth"
	"github.com/KNattawat89/hospital-middleware-system/infra/utils"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	auth    *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, auth: authService}
}

// Register implements web.Route so fx wires this handler onto the Gin engine.
func (h *Handler) Register(router gin.IRouter) {
	group := router.Group("/patient")
	group.Use(h.auth.Middleware())
	group.POST("/search", h.search)
}

type searchRequest struct {
	NationalID  *string `json:"national_id"`
	PassportID  *string `json:"passport_id"`
	FirstName   *string `json:"first_name"`
	MiddleName  *string `json:"middle_name"`
	LastName    *string `json:"last_name"`
	DateOfBirth *string `json:"date_of_birth"`
	PhoneNumber *string `json:"phone_number"`
	Email       *string `json:"email"`
}

// search godoc
// @Summary Search patients
// @Description Searches patients scoped to the logged-in staff's hospital. All fields are optional.
// @Tags patient
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body searchRequest false "Search filters"
// @Success 200 {array} model.Patient
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Router /patient/search [post]
func (h *Handler) search(c *gin.Context) {
	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, utils.BadRequestErrResponse)
		return
	}

	var dob *time.Time
	if req.DateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.BadRequestErrResponse)
			return
		}
		dob = &parsed
	}

	hospitalID := c.GetString(auth.ContextHospitalIDKey)

	patients, err := h.service.Search(SearchInput{
		HospitalID:  hospitalID,
		NationalID:  req.NationalID,
		PassportID:  req.PassportID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		DateOfBirth: dob,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.InternalServerErrorErrResponse)
		return
	}

	c.JSON(http.StatusOK, patients)
}
