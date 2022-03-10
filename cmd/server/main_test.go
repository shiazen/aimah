package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)


func TestService(t *testing.T) {

datData.counterMetrics = make(map[string]Counter)
datData.gaugeMetrics = make(map[string]Gauge)

	type want struct {
		code        int
		response    string
		contentType string
	}

	tests := []struct {
		name   string
		query  string
		method string
		want   want
	}{
		{
			name:  "metrics list test #1 (empty)",
			query: "/", //meaningless here
			method:	"GET",
			want: want{
				code:        200,
				response:    "<html><body></body></html>",
				contentType: "text/html",
			},
		},
		{
			name:  "metrics post test #1",
			query: "/update/gauge/RandomValue/2",
			method:	"POST",
			want: want{
				code:        200,
				response:    "",
				contentType: "text/plain",
			},
		},
		{
			name:  "metrics get test #1",
			query: "/value/gauge/RandomValue",
			method:	"GET",
			want: want{
				code:        200,
				response:    "2",
				contentType: "text/plain",
			},
		},
		{
			name:  "metrics list test #2",
			query: "/", //meaningless here
			method:	"GET",
			want: want{
				code:        200,
				response:    "<html><body><p>RandomValue 2</p></body></html>",
				contentType: "text/html",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.query, nil)
			w := httptest.NewRecorder()
			h := http.Handler(service())
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
