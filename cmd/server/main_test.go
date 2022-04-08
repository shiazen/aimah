package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestService(t *testing.T) {

var TestIMS = InMemoryStore{gaugeMetrics: map[string]float64{}, counterMetrics: map[string]int64{}}

	type want struct {
		code        int
		response    string
		contentType string
	}

	tests := []struct {
		name        string
		query       string
		method      string
		body        string
		contentType string
		want        want
	}{
		{
			name:   "GET metrics LIST test (empty)",
			query:  "/",
			method: "GET",
			want:   want{code: 200, response: "<html><body></body></html>", contentType: "text/html"},
		},
		{
			name:   "POST metrics WRITE test",
			query:  "/update/gauge/RandomValue/2",
			method: "POST",
			want:   want{code: 200, contentType: "text/plain"}},
		{
			name:   "GET metrics READ test",
			query:  "/value/gauge/RandomValue",
			method: "GET",
			want:   want{code: 200, response: "2", contentType: "text/plain"},
		},
		{
			name:   "GET metrics LIST test",
			query:  "/",
			method: "GET",
			want:   want{code: 200, response: "<html><body><p>RandomValue 2</p></body></html>", contentType: "text/html"},
		},
		{
			name:   "POST metrics READ JSON gauge test #1",
			query:  "/value/",
			method: "POST",
			body:   "{\"id\": \"RandomValue\", \"type\": \"gauge\"}",
			want:   want{code: 200, response: "{\"id\":\"RandomValue\",\"type\":\"gauge\",\"value\":2}", contentType: "application/json"},
		},
		{
			name:        "POST metrics WRITE JSON counter test #1",
			query:       "/update/",
			contentType: "application/json",
			method:      "POST",
			body:        "{\"id\": \"RandomValue\", \"type\": \"gauge\", \"value\": 0.7}",
			want:        want{code: 200, contentType: "text/plain"},
		},
		{
			name:   "POST metrics READ JSON gauge test #2",
			query:  "/value/",
			method: "POST",
			body:   "{\"id\": \"RandomValue\", \"type\": \"gauge\"}",
			want:   want{code: 200, response: "{\"id\":\"RandomValue\",\"type\":\"gauge\",\"value\":0.7}", contentType: "application/json"},
		},
		{
			name:        "POST metrics WRITE JSON counter test #1",
			query:       "/update/",
			contentType: "application/json",
			method:      "POST",
			body:        "{\"id\": \"TestCounter\", \"type\": \"counter\", \"delta\": 777}",
			want:        want{code: 200, contentType: "text/plain"},
		},
		{
			name:        "POST metrics WRITE JSON counter test #2",
			query:       "/update/",
			contentType: "application/json",
			method:      "POST",
			body:        "{\"id\": \"TestCounter\", \"type\": \"counter\", \"delta\": 111}",
			want:        want{code: 200, contentType: "text/plain"},
		},
		{
			name:   "POST metrics READ JSON counter test #2",
			query:  "/value/",
			method: "POST",
			body:   "{\"id\": \"TestCounter\", \"type\": \"counter\"}",
			want:   want{code: 200, response: "{\"id\":\"TestCounter\",\"type\":\"counter\",\"delta\":888}", contentType: "application/json"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.query, strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h := http.Handler(TestIMS.service())
			h.ServeHTTP(w, request)
			res := w.Result()

			if res.StatusCode != tt.want.code {
				t.Errorf("Expected status code %d, got %d", tt.want.code, w.Code)
			}

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(resBody) != tt.want.response {
				t.Errorf("Expected body \"%s\", got \"%s\"", tt.want.response, w.Body.String())
			}

			// заголовок ответа
			if res.Header.Get("Content-Type") != tt.want.contentType {
				t.Errorf("Expected Content-Type %s, got %s", tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}
