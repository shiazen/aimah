package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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

	// file storage/restore
	if config["STORE_FILE"] != "" {

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

		if config["STORE_FILE"] != "" {
			JSONByteArray := ExtractFromInMemoryStore(datData)
			err := os.WriteFile(config["STORE_FILE"], JSONByteArray, 0644)
			check(err)
		}

		err := server.Shutdown(shutdownCtx)
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

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
