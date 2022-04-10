package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (ims *InMemoryStore) service() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(gzipHandle)

	r.Get("/", ims.ListMetricsPLAIN)
	r.Get("/json", ims.ListMetricsJSON)
	r.Get("/{action}/{type}/{name}", ims.ValueViaGetPLAIN)
	r.Post("/update/{type}/{name}/{value}", ims.UpdateViaPostPLAIN)
	r.Post("/update/", ims.UpdateViaPostJSON)
	r.Post("/value/", ims.ValueViaPostJSON)

	return r
}

func (ims *InMemoryStore) ListMetricsPLAIN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	w.Write([]byte("<html><body>"))
	for k, v := range ims.gaugeMetrics {
		w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
	}
	for k, v := range ims.counterMetrics {
		w.Write([]byte(fmt.Sprintf("<p>%v %v</p>", k, v)))
	}
	w.Write([]byte("</body></html>"))
}

func (ims *InMemoryStore) ListMetricsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.Write(ims.ExtractFromInMemoryStore())
}

func (ims *InMemoryStore) ValueViaGetPLAIN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if metricAction := chi.URLParam(r, "action"); metricAction == "value" {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		switch metricType {
		case "gauge":
			if _, ok := ims.gaugeMetrics[metricName]; ok {
				w.Write([]byte(fmt.Sprintf("%v", ims.gaugeMetrics[metricName])))
			} else {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			}
		case "counter":
			if _, ok := ims.counterMetrics[metricName]; ok {
				w.Write([]byte(fmt.Sprintf("%v\n", ims.counterMetrics[metricName])))
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

func (ims *InMemoryStore) UpdateViaPostPLAIN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	rawMetricValue := chi.URLParam(r, "value")
	switch metricType {
	case "gauge":
		metricValue, err := strconv.ParseFloat(rawMetricValue, 64)
		if err == nil {
			ims.gaugeMetrics[metricName] = metricValue
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "counter":
		metricValue, err := strconv.ParseInt(rawMetricValue, 10, 64)
		if err == nil {
			metricValue += ims.counterMetrics[metricName]
			ims.counterMetrics[metricName] = metricValue
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}

	if config["STORE_INTERVAL"] == "0" {
		ims.StoreData(config["STORE_FILE"])
	}
}

func (ims *InMemoryStore) UpdateViaPostJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	
	var err error

	MetricsFromJSON := DeJSONify(&r.Body)
	if MetricsFromJSON.HashCheck() {
	err = ims.InsertInMemoryStore(&MetricsFromJSON)
	} else { http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest) }

	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
	if config["STORE_INTERVAL"] == "0" {
		ims.StoreData(config["STORE_FILE"])
	}
}

func (ims *InMemoryStore) ValueViaPostJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	MetricsFromJSON := DeJSONify(&r.Body)

	switch MetricsFromJSON.MType {
	case "gauge":
		if val, ok := ims.gaugeMetrics[MetricsFromJSON.ID]; ok {
			MetricsFromJSON.Value = &val
			w.Write(MetricsFromJSON.jsonify())
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	case "counter":
		if val, ok := ims.counterMetrics[MetricsFromJSON.ID]; ok {
			MetricsFromJSON.Delta = &val
			w.Write(MetricsFromJSON.jsonify())
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}
