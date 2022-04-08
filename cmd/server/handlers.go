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

	r.Get("/", ims.MetricList)
	r.Get("/json", ims.MetricListJSON)
	r.Get("/{action}/{type}/{name}", ims.MetricGet)
	r.Post("/update/{type}/{name}/{value}", ims.MetricPost)
	r.Post("/update/", ims.PostUpdateJSON)
	r.Post("/value/", ims.PostValueJSON)

	return r
}

func (ims *InMemoryStore) MetricList(w http.ResponseWriter, r *http.Request) {
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

func (ims *InMemoryStore) MetricListJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.Write(ims.ExtractFromInMemoryStore())
}

func (ims *InMemoryStore) MetricGet(w http.ResponseWriter, r *http.Request) {
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

func (ims *InMemoryStore) MetricPost(w http.ResponseWriter, r *http.Request) {
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

func (ims *InMemoryStore) PostUpdateJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	MetricsJSON := DeJSONify(&r.Body)
	err := ims.InsertInMemoryStore(&MetricsJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
	if config["STORE_INTERVAL"] == "0" {
		ims.StoreData(config["STORE_FILE"])
	}
}

func (ims *InMemoryStore) PostValueJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	MetricsJSON := DeJSONify(&r.Body)

	switch MetricsJSON.MType {
	case "gauge":
		if val, ok := ims.gaugeMetrics[MetricsJSON.ID]; ok {
			MetricsJSON.Value = &val
			w.Write(jsonify(MetricsJSON))
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	case "counter":
		if val, ok := ims.counterMetrics[MetricsJSON.ID]; ok {
			MetricsJSON.Delta = &val
			w.Write(jsonify(MetricsJSON))
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}
