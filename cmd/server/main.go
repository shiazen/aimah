package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"encoding/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Metrics struct {
	ID	string	`json:"id"`	// имя метрики
	MType string	`json:"type"`	// параметр, принимающий значение gauge или counter
	Delta *int64	`json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

//type Gauge float64
//type Counter int64

type InMemoryStore struct {
	gaugeMetrics	map[string]float64
	counterMetrics map[string]int64
}

var datData InMemoryStore

func main() {

	datData.gaugeMetrics = make(map[string]float64)
	datData.counterMetrics = make(map[string]int64)

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
	r.Post("/update/", PostUpdateJson)
	r.Post("/value/", PostValueJson)

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
	//if r.Header.Get("Content-type") == "text/plain" { fmt.Printf("text/plain") }
	//if r.Header.Get("Content-type") == "text/plain" { fmt.Println("OKAY") }
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

func PostUpdateJson(w http.ResponseWriter, r *http.Request) {

	MetricsJson := Metrics{}
	//err := json.NewDecoder(r.Body).Decode(&MetricsJson)
	body, err := io.ReadAll(r.Body)
	if err != nil { log.Fatal(err) }
	err = json.Unmarshal(body, &MetricsJson)
//	fmt.Println(MetricsJson.ID, MetricsJson.MType)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//if err := json.Unmarshal(r.Body, &MetricJson); err != nil { panic(err) }
	switch MetricsJson.MType {
		case "gauge":
			if MetricsJson.Value != nil {
				datData.gaugeMetrics[MetricsJson.ID] = *MetricsJson.Value
			}
		case "counter":
			if MetricsJson.Delta != nil {
				datData.counterMetrics[MetricsJson.ID] += *MetricsJson.Delta
			}
		default:
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

func PostValueJson(w http.ResponseWriter, r *http.Request) {

	MetricsJson := Metrics{}
	body, err := io.ReadAll(r.Body)
	if err != nil { log.Fatal(err) }
	err = json.Unmarshal(body, &MetricsJson)
	if (err != nil) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch MetricsJson.MType {
		case "gauge":
			if _, ok := datData.gaugeMetrics[MetricsJson.ID]; ok {
				w.Write([]byte(fmt.Sprintf("%v", datData.gaugeMetrics[MetricsJson.ID])))
			}
		case "counter":
			if _, ok := datData.counterMetrics[MetricsJson.ID]; ok {
				w.Write([]byte(fmt.Sprintf("%v", datData.counterMetrics[MetricsJson.ID])))
			}
		default:
			http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}

func DeJsonify (body *io.Reader) Metrics {
	theMetrics := Metrics{}
	bytestreamBody, err := io.ReadAll(*body)
	if err != nil { log.Fatal(err) }
	err = json.Unmarshal(bytestreamBody, &theMetrics)
	return theMetrics
}
