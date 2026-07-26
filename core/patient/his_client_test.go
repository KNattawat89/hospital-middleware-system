package patient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHISClient_SearchByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/patient/search/1234567890123" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(HISPatient{
				FirstNameTh: "สมชาย",
				NationalID:  "1234567890123",
			})
		}))
		defer server.Close()

		client := NewHISClient()
		got, err := client.SearchByID(server.URL, "1234567890123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.FirstNameTh != "สมชาย" {
			t.Fatalf("unexpected response: %+v", got)
		}
	})

	t.Run("not found maps to ErrPatientNotFoundUpstream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewHISClient()
		_, err := client.SearchByID(server.URL, "0000000000000")
		if err != ErrPatientNotFoundUpstream {
			t.Fatalf("expected ErrPatientNotFoundUpstream, got %v", err)
		}
	})

	t.Run("non-200 status returns an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewHISClient()
		_, err := client.SearchByID(server.URL, "x")
		if err == nil {
			t.Fatalf("expected an error for a 500 response")
		}
	})

	t.Run("malformed JSON returns an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewHISClient()
		_, err := client.SearchByID(server.URL, "x")
		if err == nil {
			t.Fatalf("expected a decode error")
		}
	})

	t.Run("unreachable host returns an error", func(t *testing.T) {
		client := NewHISClient()
		_, err := client.SearchByID("http://127.0.0.1:1", "x")
		if err == nil {
			t.Fatalf("expected a connection error")
		}
	})
}
