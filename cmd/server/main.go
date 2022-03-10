package main

import (
	"context"
	"fmt"
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

type Gauge float64
type Counter int64

type InMemoryStore struct {
	gaugeMetrics   map[string]Gauge
	counterMetrics map[string]Counter
}

var datData InMemoryStore

func main() {

	datData.counterMetrics = make(map[string]Counter)
	datData.gaugeMetrics = make(map[string]Gauge)

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
	r.Post("/{action}/{type}/{name}/{value}", MetricPost)

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
	w.Header().Set("content-type", "text/plain")
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
	w.Header().Set("content-type", "text/plain")
	if metricAction := chi.URLParam(r, "action"); metricAction == "update" {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		rawMetricValue := chi.URLParam(r, "value")
		switch metricType {
		case "gauge":
			metricValue, err := strconv.ParseFloat(rawMetricValue, 64)
			if err == nil {
				datData.gaugeMetrics[metricName] = Gauge(metricValue)
			} else {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			}
		case "counter":
			metricValue, err := strconv.ParseInt(rawMetricValue, 10, 64)
			if err == nil {
				datData.counterMetrics[metricName] += Counter(metricValue)
			} else {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			}
		default:
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		}
	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}
