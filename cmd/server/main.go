package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type InMemoryStore struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

var datData = &InMemoryStore{gaugeMetrics: map[string]float64{}, counterMetrics: map[string]int64{}}

func main() {

	server := &http.Server{Addr: "127.0.0.1:8080", Handler: service()}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-serverCtx.Done()
}

func service() http.Handler {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	r.Get("/", MetricList)
	r.Get("/{action}/{type}/{name}", MetricGet)
	r.Post("/update/{type}/{name}/{value}", MetricPost)
	r.Post("/update/", PostUpdateJSON)
	r.Post("/value/", PostValueJSON)

	return r
}

func MetricList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	w.Write([]byte("<html><body>"))
	for k, v := range datData.gaugeMetrics {
		w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
	}
	for k, v := range datData.counterMetrics {
		w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
	}
	w.Write([]byte("</body></html>"))
}

func MetricGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if metricAction := chi.URLParam(r, "action"); metricAction == "value" {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		switch metricType {
		case "gauge":
			if _, ok := datData.gaugeMetrics[metricName]; ok {
				w.Write([]byte(fmt.Sprintf("%v", datData.gaugeMetrics[metricName])))
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		case "counter":
			if _, ok := datData.counterMetrics[metricName]; ok {
				w.Write([]byte(fmt.Sprintf("%v\n", datData.counterMetrics[metricName])))
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		default:
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		}
	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func MetricPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	rawMetricValue := chi.URLParam(r, "value")
	switch metricType {
	case "gauge":
		metricValue, err := strconv.ParseFloat(rawMetricValue, 64)
		if err == nil {
			datData.gaugeMetrics[metricName] = metricValue
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "counter":
		metricValue, err := strconv.ParseInt(rawMetricValue, 10, 64)
		if err == nil {
			metricValue += datData.counterMetrics[metricName]
			datData.counterMetrics[metricName] = metricValue
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

func PostUpdateJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	MetricsJSON := DeJSONify(&r.Body)

	switch MetricsJSON.MType {
	case "gauge":
		if MetricsJSON.Value != nil {
			datData.gaugeMetrics[MetricsJSON.ID] = *MetricsJSON.Value
		}
	case "counter":
		if MetricsJSON.Delta != nil {
			datData.counterMetrics[MetricsJSON.ID] += *MetricsJSON.Delta
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

func PostValueJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	MetricsJSON := DeJSONify(&r.Body)

	switch MetricsJSON.MType {
	case "gauge":
		if val, ok := datData.gaugeMetrics[MetricsJSON.ID]; ok {
			MetricsJSON.Value = &val
			w.Write(jsonify(MetricsJSON))
			// MetricsJSON.Value = nil
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	case "counter":
		if val, ok := datData.counterMetrics[MetricsJSON.ID]; ok {
			MetricsJSON.Delta = &val
			w.Write(jsonify(MetricsJSON))
			// MetricsJSON.Delta = nil
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

func DeJSONify(body *io.ReadCloser) Metrics {
	theMetrics := Metrics{}
	byteStreamBody, err := io.ReadAll(*body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(byteStreamBody, &theMetrics)
	if err != nil {
		log.Fatal(err)
	}
	return theMetrics
}

func jsonify(m Metrics) []byte {
	p, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	return p
}
