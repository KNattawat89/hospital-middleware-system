package patient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrPatientNotFoundUpstream is returned when the HIS has no record for the
// given id. It's not a failure of the search itself - the local DB is still
// queried with whatever the staff already has on file.
var ErrPatientNotFoundUpstream = errors.New("patient not found upstream")

// HISPatient mirrors Hospital A's response body exactly (field names and
// all), so the adapter's JSON tags follow the upstream contract, not our
// internal model's column names.
type HISPatient struct {
	FirstNameTh  string `json:"first_name_th"`
	MiddleNameTh string `json:"middle_name_th"`
	LastNameTh   string `json:"last_name_th"`
	FirstNameEn  string `json:"first_name_en"`
	MiddleNameEn string `json:"middle_name_en"`
	LastNameEn   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

// HISClient calls a hospital's own Hospital Information System. Hospital A
// is the only implementation today; onboarding another hospital means a new
// adapter behind this same interface, not a new public route.
type HISClient interface {
	SearchByID(baseURL, id string) (*HISPatient, error)
}

func NewHISClient() HISClient {
	return &hisClient{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

type hisClient struct {
	httpClient *http.Client
}

func (c *hisClient) SearchByID(baseURL, id string) (*HISPatient, error) {
	endpoint := fmt.Sprintf("%s/patient/search/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(id))

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("call HIS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrPatientNotFoundUpstream
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HIS returned status %d", resp.StatusCode)
	}

	var result HISPatient
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode HIS response: %w", err)
	}

	return &result, nil
}
