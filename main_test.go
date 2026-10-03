package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestHealth(t *testing.T) {
	tests := []struct {
		name     string
		db       fakeDB
		wantCode int
		wantBody string
	}{
		{"database up", fakeDB{}, http.StatusOK, `{"status":"ok","version":"dev","database":"up"}`},
		{"database down", fakeDB{errors.New("connection refused")}, http.StatusServiceUnavailable, `{"status":"degraded","version":"dev","database":"down"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newRouter(tt.db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
				t.Fatalf("body = %s, want %s", got, tt.wantBody)
			}
		})
	}
}

func TestHealthRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	newRouter(fakeDB{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/health", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
