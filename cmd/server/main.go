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

var dat_data InMemoryStore

func main() {

	dat_data.counterMetrics = make(map[string]Counter)
	dat_data.gaugeMetrics = make(map[string]Gauge)

	server := &http.Server{Addr: "127.0.0.1:8080", Handler: service()}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig
		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)
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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("<html><body>")))
		for k, v := range dat_data.gaugeMetrics {
			w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
		}
		for k, v := range dat_data.counterMetrics {
			w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
		}
		w.Write([]byte(fmt.Sprintf("</body></html>")))
	})

	r.Get("/value/gauge/{name}", func(w http.ResponseWriter, r *http.Request) {
		metricName := chi.URLParam(r, "name")
		if _, ok := dat_data.gaugeMetrics[metricName]; ok {
			w.Write([]byte(fmt.Sprintf("%v\n", dat_data.gaugeMetrics[metricName])))
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	})
	r.Get("/value/counter/{name}", func(w http.ResponseWriter, r *http.Request) {
		metricName := chi.URLParam(r, "name")
		if _, ok := dat_data.counterMetrics[metricName]; ok {
			w.Write([]byte(fmt.Sprintf("%v\n", dat_data.counterMetrics[metricName])))
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	})

	r.Post("/update/counter/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
		metricName := chi.URLParam(r, "name")
		rawMetricValue := chi.URLParam(r, "value")
		metricValue, _ := strconv.ParseInt(rawMetricValue, 10, 64)

		dat_data.counterMetrics[metricName] = Counter(metricValue)

		//fmt.Printf("name: %v;\tr_val: %v;\tc_val: %v\n", metricName, rawMetricValue, metricValue)
		//fmt.Printf("data stored: %v\n", dat_data.counterMetrics[metricName] )
	})
	r.Post("/update/gauge/{name}/{value}", func(w http.ResponseWriter, r *http.Request) {
		metricName := chi.URLParam(r, "name")
		rawMetricValue := chi.URLParam(r, "value")
		metricValue, _ := strconv.ParseFloat(rawMetricValue, 64)

		dat_data.gaugeMetrics[metricName] += Gauge(metricValue)

		//fmt.Printf("name: %v;\tr_val: %v;\tc_val: %v\n", metricName, rawMetricValue, metricValue)
		//fmt.Printf("data stored: %v\n", dat_data.gaugeMetrics[metricName] )
	})

	return r
}
