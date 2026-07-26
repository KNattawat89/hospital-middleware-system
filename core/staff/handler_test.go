package staff

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KNattawat89/hospital-middleware-system/infra/auth"
	"github.com/KNattawat89/hospital-middleware-system/infra/db/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter(service *Service) *gin.Engine {
	router := gin.New()
	h := NewHandler(service)
	h.Register(router)
	return router
}

func TestHandler_Create(t *testing.T) {
	hospitalID := uuid.New()

	t.Run("success returns 201", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return nil, gorm.ErrRecordNotFound
			},
			createStaff: func(staffRecord *model.Staff) error { return nil },
		}
		router := newTestRouter(&Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(
			`{"username":"nurse001","password":"s3cret!","hospital":"HOSPITAL_A"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing field returns 400", func(t *testing.T) {
		router := newTestRouter(&Service{repo: &fakeRepo{}, tokens: &fakeTokenIssuer{}, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(
			`{"username":"nurse001","hospital":"HOSPITAL_A"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unknown hospital returns 404", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return nil, gorm.ErrRecordNotFound
			},
		}
		router := newTestRouter(&Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(
			`{"username":"nurse001","password":"s3cret!","hospital":"UNKNOWN"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("duplicate username returns 409", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return &model.Staff{}, nil
			},
		}
		router := newTestRouter(&Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/create", strings.NewReader(
			`{"username":"nurse001","password":"s3cret!","hospital":"HOSPITAL_A"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandler_Login(t *testing.T) {
	hospitalID := uuid.New()
	staffID := uuid.New()
	hashed := hashPassword(t, "correct-password")

	t.Run("success returns 200", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return &model.Staff{ID: &staffID, PasswordHashed: &hashed}, nil
			},
			touchLastLoginAt: func(id string) error { return nil },
		}
		tokens := &fakeTokenIssuer{pair: &auth.TokenPair{AccessToken: "access", RefreshToken: "refresh"}}
		router := newTestRouter(&Service{repo: repo, tokens: tokens, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/login", strings.NewReader(
			`{"username":"nurse001","password":"correct-password","hospital":"HOSPITAL_A"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "access") {
			t.Fatalf("expected access token in response, got %s", w.Body.String())
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		repo := &fakeRepo{
			findHospitalByCode: func(code string) (*model.Hospital, error) {
				return &model.Hospital{ID: &hospitalID, Code: &code}, nil
			},
			findByHospitalAndUsername: func(hospitalID, username string) (*model.Staff, error) {
				return &model.Staff{ID: &staffID, PasswordHashed: &hashed}, nil
			},
		}
		router := newTestRouter(&Service{repo: repo, tokens: &fakeTokenIssuer{}, log: zap.NewNop()})

		req := httptest.NewRequest(http.MethodPost, "/staff/login", strings.NewReader(
			`{"username":"nurse001","password":"wrong-password","hospital":"HOSPITAL_A"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}
