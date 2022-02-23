package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

//        "github.com/go-chi/chi/v5"
 //       "github.com/go-chi/chi/v5/middleware"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func TestPing(t *testing.T) {
	// create resp writer and request to pass
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/does/not/matter", nil)

	// call your endpoint
	ping(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected %v but for %v\n", http.StatusOK, res.StatusCode)
	}
}

