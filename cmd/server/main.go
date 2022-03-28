package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type InMemoryStore struct {
	gaugeMetrics   map[string]float64
	counterMetrics map[string]int64
}

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

var config = map[string]string{
	"ADDRESS":        "127.1:8080",
	"RESTORE":        "true",
	"STORE_INTERVAL": "300",
	"STORE_FILE":     "/tmp/devops-metrics-db.json",
}

var datData = &InMemoryStore{gaugeMetrics: map[string]float64{}, counterMetrics: map[string]int64{}}

func main() {

	positional := make(map[string]*string)
	for k := range config {
		letter := strings.ToLower(k[0:1]) // made sense for agent
		if k == "STORE_FILE" {            // why not FILE_STORAGE_PATH eh
			//letter = strings.ToLower(k[6:7])
			letter = "f"
		} else if k == "STORE_INTERVAL" {
			letter = "i"
		}
		positional[k] = flag.String(letter, config[k], k)
	}
	flag.Parse()

	for k := range config {
		if positional[k] != nil {
			config[k] = *positional[k]
		}
		if val, ok := os.LookupEnv(k); ok {
			config[k] = val
		}
	}

	server := &http.Server{Addr: config["ADDRESS"], Handler: service()}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	if restoreb, err := strconv.ParseBool(config["RESTORE"]); err == nil {
		if restoreb {
			if JSONFile, err := os.ReadFile(config["STORE_FILE"]); err == nil {
				PopulateInMemoryStore(JSONFile, datData)
			} else {
				log.Print(err)
			}
		}
	} else {
		check(err)
	}

	// --- json file store ticker
	var storeInterval time.Duration
	if tmpStoreInterval, err := strconv.Atoi(config["STORE_INTERVAL"]); err == nil {
		storeInterval = time.Duration(tmpStoreInterval) * time.Second
	} else if tmpStoreInterval, err := time.ParseDuration(config["STORE_INTERVAL"]); err == nil {
		storeInterval = tmpStoreInterval
	} else {
		check(err)
	}

	if storeInterval > 0 {
		TickerStore := time.NewTicker(storeInterval)
		defer TickerStore.Stop()
		go func() {
			for {
				<-TickerStore.C
				JSONByteArray := ExtractFromInMemoryStore(datData)
				err := os.WriteFile(config["STORE_FILE"], JSONByteArray, 0644)
				check(err)
			}
		}()
	}
	// ----------------- ----------

	//----- chi examples/graceful copypaste
	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-SigChan
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()
		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		JSONByteArray := ExtractFromInMemoryStore(datData)
		err := os.WriteFile(config["STORE_FILE"], JSONByteArray, 0644)
		check(err)

		err = server.Shutdown(shutdownCtx)
		check(err)
		serverStopCtx()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-serverCtx.Done()
	// -------- ------------------------
}

func service() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	//r.Use(middleware.Logger)
	//r.Use(gzipHandle)

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
	check(err)
	err = json.Unmarshal(byteStreamBody, &theMetrics)
	check(err)
	return theMetrics
}

func jsonify(m Metrics) []byte {
	p, err := json.Marshal(m)
	check(err)
	return p
}

func check(e error) {
	if e != nil {
		// panic(e)
		log.Fatal(e)
	}
}

// ------- gzipWriter copy paste
type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

// ------- -----------------------

func InsertInMemoryStore(m *Metrics, s *InMemoryStore) error {
	switch m.MType {
	case "gauge":
		if m.Value != nil {
			s.gaugeMetrics[m.ID] = *m.Value
		} else {
			return errors.New("type not specified")
		}
	case "counter":
		if m.Delta != nil {
			s.counterMetrics[m.ID] += *m.Delta
		} else {
			return errors.New("type not specified")
		}
	default:
		return errors.New("unknown type specified")
	}
	return nil
}

func ExtractFromInMemoryStore(ims *InMemoryStore) []byte {
	var mj []Metrics

	for k := range ims.gaugeMetrics {
		var m Metrics
		var g []float64
		g = append(g, ims.gaugeMetrics[k])
		m.MType = "gauge"
		m.ID = k
		m.Value = &g[len(g)-1]
		mj = append(mj, m)
	}

	for k := range ims.counterMetrics {
		var m Metrics
		var c []int64
		c = append(c, ims.counterMetrics[k])
		m.MType = "counter"
		m.ID = k
		m.Delta = &c[len(c)-1]
		mj = append(mj, m)
	}
	p, err := json.Marshal(mj)
	check(err)
	return p
}

func PopulateInMemoryStore(j []byte, ims *InMemoryStore) {
	var mj []*Metrics
	err := json.Unmarshal(j, &mj)
	if err == nil {
		for k := range mj {
			InsertInMemoryStore(mj[k], ims)
		}
	} else {
		log.Print(err)
	}
}
