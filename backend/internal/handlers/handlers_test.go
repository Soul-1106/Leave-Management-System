package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"leave-management-backend/internal/services"
)

func TestRespondMapsClientErrorWithoutLeakingCause(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := &services.Error{
		Status:  http.StatusConflict,
		Message: "request conflicts with existing leave",
		Err:     errors.New("pq: sensitive database detail"),
	}
	respond(recorder, nil, err)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusConflict)
	}
	if strings.Contains(recorder.Body.String(), "pq:") {
		t.Fatal("response leaked the internal error")
	}
}

func TestRespondHidesUnknownErrors(t *testing.T) {
	recorder := httptest.NewRecorder()
	respond(recorder, nil, errors.New("database password failed"))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("got status %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(recorder.Body.String(), "internal server error") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
