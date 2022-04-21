package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	PopulateConfig(&ConfigMap) // config.go

	var IMS = InMemoryStore{gaugeMetrics: map[string]float64{}, counterMetrics: map[string]int64{}, saverchan: make(chan struct{})}

	StoreSignalChannel := IMS.StoreStuff()
	if StoreSignalChannel {
		//IMS.RestoreFromFile()
		LaunchStoreTicker(ConfigMap["STORE_INTERVAL"], IMS.saverchan)
	}

	server := &http.Server{Addr: ConfigMap["ADDRESS"], Handler: IMS.service()}
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

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

		// save on exit
		if StoreSignalChannel {
			IMS.saverchan <- struct{}{}
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
