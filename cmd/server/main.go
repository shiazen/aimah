package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {

	PopulateConfig(&ConfigMap)

	var IMS = InMemoryStore{gaugeMetrics: map[string]float64{}, counterMetrics: map[string]int64{}}

	server := &http.Server{Addr: ConfigMap["ADDRESS"], Handler: IMS.service()}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// file storage/restore
	if ConfigMap["STORE_FILE"] != "" {
		if restoreb, err := strconv.ParseBool(ConfigMap["RESTORE"]); err == nil {
			if restoreb {
				if JSONFileContent, err := os.ReadFile(ConfigMap["STORE_FILE"]); err == nil {
					IMS.PopulateInMemoryStore(JSONFileContent)
				} else {
					log.Print(err)
				}
			}
		} else {
			OnErrorFail(err)
		}

		// --- json file store ticker
		var storeInterval time.Duration
		if tmpStoreInterval, err := strconv.Atoi(ConfigMap["STORE_INTERVAL"]); err == nil {
			storeInterval = time.Duration(tmpStoreInterval) * time.Second
		} else if tmpStoreInterval, err := time.ParseDuration(ConfigMap["STORE_INTERVAL"]); err == nil {
			storeInterval = tmpStoreInterval
		} else {
			OnErrorFail(err)
		}

		if storeInterval > 0 {
			TickerStore := time.NewTicker(storeInterval)
			defer TickerStore.Stop()
			go func() {
				for {
					<-TickerStore.C
					IMS.StoreData(ConfigMap["STORE_FILE"])
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

		if ConfigMap["STORE_FILE"] != "" {
			IMS.StoreData(ConfigMap["STORE_FILE"])
		}

		err := server.Shutdown(shutdownCtx)
		OnErrorFail(err)
		serverStopCtx()
	}()

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-serverCtx.Done()
	// -------- ------------------------
}

func OnErrorFail(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func OnErrorProceed(e error) {
	if e != nil {
		log.Print(e)
	}
}
