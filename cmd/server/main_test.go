package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// there are no actual tests

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func TestPing(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/does/not/matter", nil)

	ping(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected %v but for %v\n", http.StatusOK, res.StatusCode)
	}
}

