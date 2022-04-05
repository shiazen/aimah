package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func service() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(gzipHandle)

	r.Get("/", MetricList)
	r.Get("/json", MetricListJSON)
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

func MetricListJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.Write(ExtractFromInMemoryStore(datData))
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
	err := InsertInMemoryStore(&MetricsJSON, datData)
	if err != nil {
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
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	case "counter":
		if val, ok := datData.counterMetrics[MetricsJSON.ID]; ok {
			MetricsJSON.Delta = &val
			w.Write(jsonify(MetricsJSON))
		} else {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	default:
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	}
}
