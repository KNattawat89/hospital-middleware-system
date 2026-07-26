package patient

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KNattawat89/hospital-middleware-system/infra/auth"
	"github.com/KNattawat89/hospital-middleware-system/infra/config"
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAuthService(t *testing.T, accessTTL, refreshTTL string) *auth.Service {
	t.Helper()
	s, err := auth.NewService(&config.Config{
		Jwt: config.JwtConfig{
			Secret:          "test-secret",
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: refreshTTL,
		},
	})
	if err != nil {
		t.Fatalf("build auth service: %v", err)
	}
	return s
}

func newTestRouter(service *Service, authService *auth.Service) *gin.Engine {
	router := gin.New()
	h := NewHandler(service, authService)
	h.Register(router)
	return router
}

func TestHandler_Search_RequiresAuth(t *testing.T) {
	authService := newAuthService(t, "15m", "168h")
	router := newTestRouter(&Service{repo: &fakeRepo{}, his: &fakeHISClient{}, log: zap.NewNop()}, authService)

	t.Run("missing token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("garbage token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer not-a-real-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("expired token with no valid refresh returns 401", func(t *testing.T) {
		expiredAuth := newAuthService(t, "-1s", "-1s")
		pair, err := expiredAuth.GenerateTokenPair(uuid.NewString(), uuid.NewString())
		if err != nil {
			t.Fatalf("generate token pair: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandler_Search_ScopesToStaffHospital(t *testing.T) {
	authService := newAuthService(t, "15m", "168h")
	hospitalID := uuid.New()

	var gotHospitalID string
	repo := &fakeRepo{
		searchPatients: func(filters SearchFilters) ([]*model.Patient, error) {
			gotHospitalID = filters.HospitalID
			return []*model.Patient{}, nil
		},
	}
	router := newTestRouter(&Service{repo: repo, his: &fakeHISClient{}, log: zap.NewNop()}, authService)

	pair, err := authService.GenerateTokenPair(uuid.NewString(), hospitalID.String())
	if err != nil {
		t.Fatalf("generate token pair: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/patient/search", strings.NewReader(
		`{"last_name":"Somchai","hospital":"a-different-hospital-should-be-ignored"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotHospitalID != hospitalID.String() {
		t.Fatalf("expected search scoped to token's hospital %s, got %s", hospitalID.String(), gotHospitalID)
	}
}
