package apierror

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"gorm.io/gorm"
)

func TestClassifyHTTPStatus(t *testing.T) {
	wrapped := fmt.Errorf("login: %w", ErrUnauthorized)
	if ClassifyHTTPStatus(wrapped) != http.StatusUnauthorized {
		t.Fatal("wrapped unauthorized")
	}
	if ClassifyHTTPStatus(ErrValidation) != http.StatusBadRequest {
		t.Fatal("validation")
	}
	if ClassifyHTTPStatus(gorm.ErrRecordNotFound) != http.StatusNotFound {
		t.Fatal("not found")
	}
	if ClassifyHTTPStatus(errors.New("db down")) != http.StatusInternalServerError {
		t.Fatal("default")
	}
}

func TestJSONBody(t *testing.T) {
	b, err := json.Marshal(JSONBody(errors.New("missing authorization header")))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"error":"missing authorization header"}` {
		t.Fatalf("got %s", b)
	}
}
