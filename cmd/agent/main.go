package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type gauge float64
type counter int64

type Payload struct {
	Alloc,
	BuckHashSys,
	Frees,
	GCCPUFraction,
	GCSys,
	HeapAlloc,
	HeapIdle,
	HeapInuse,
	HeapObjects,
	HeapReleased,
	HeapSys,
	LastGC,
	Lookups,
	MCacheInuse,
	MCacheSys,
	MSpanInuse,
	MSpanSys,
	Mallocs,
	NextGC,
	NumForcedGC,
	NumGC,
	OtherSys,
	PauseTotalNs,
	StackInuse,
	StackSys,
	Sys,
	RandomValue gauge // — обновляемое рандомное значение.

	PollCount counter // — счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime (на каждый pollInterval — см. ниже).
}

func poller(dat_random float64, dat_counter *int64) Payload {

	var pld Payload

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	pld.Alloc = gauge(ms.Alloc)
	pld.BuckHashSys = gauge(ms.BuckHashSys)
	pld.Frees = gauge(ms.Frees)
	pld.GCCPUFraction = gauge(ms.GCCPUFraction)
	pld.GCSys = gauge(ms.GCSys)
	pld.HeapAlloc = gauge(ms.HeapAlloc)
	pld.HeapIdle = gauge(ms.HeapIdle)
	pld.HeapInuse = gauge(ms.HeapInuse)
	pld.HeapObjects = gauge(ms.HeapObjects)
	pld.HeapReleased = gauge(ms.HeapReleased)
	pld.HeapSys = gauge(ms.HeapSys)
	pld.LastGC = gauge(ms.LastGC)
	pld.Lookups = gauge(ms.Lookups)
	pld.MCacheInuse = gauge(ms.MCacheInuse)
	pld.MCacheSys = gauge(ms.MCacheSys)
	pld.MSpanInuse = gauge(ms.MSpanInuse)
	pld.MSpanSys = gauge(ms.MSpanSys)
	pld.Mallocs = gauge(ms.Mallocs)
	pld.NextGC = gauge(ms.NextGC)
	pld.NumForcedGC = gauge(ms.NumForcedGC)
	pld.NumGC = gauge(ms.NumGC)
	pld.OtherSys = gauge(ms.OtherSys)
	pld.PauseTotalNs = gauge(ms.PauseTotalNs)
	pld.StackInuse = gauge(ms.StackInuse)
	pld.StackSys = gauge(ms.StackSys)
	pld.Sys = gauge(ms.Sys)

	*dat_counter++
	pld.PollCount = counter(*dat_counter)
	pld.RandomValue = gauge(dat_random)

	return pld
}

// payload should be shared between
// polling and sending ticker loops
var DatPayload Payload

// poll counter
var cnt int64 = 0

func main() {

	// по умолчанию на адрес: 127.0.0.1, порт: 8080
	server_address := "http://127.0.0.1"
	server_port := 8080

	// randomness happens
	rand.Seed(time.Now().UnixNano())

	client := &http.Client{}

	// По умолчанию приложение должно обновлять метрики из пакета runtime с заданной частотой:
	var pollInterval = 2
	// По умолчанию приложение должно отправлять метрики на сервер с заданной частотой:
	var reportInterval = 10

	TickerPoll := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer TickerPoll.Stop()
	TickerSend := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer TickerSend.Stop()

	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
	}()

	for {
		select {
		case dat_signal := <-SigChan:
			switch dat_signal {
			// Агент должен штатно завершаться по сигналам: syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT.
			// eh?
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				//fmt.Printf("\nSignal %v triggered.\n", dat_signal)
				os.Exit(0)
			default:
				fmt.Printf("\nSignal %v triggered.\n", dat_signal)
				os.Exit(1)
			}
		//case t := <-TickerPoll.C:
		case <-TickerPoll.C:
			// fmt.Println("polling ticker: ", t)
			DatPayload = poller(rand.Float64(), &cnt)
		//case t := <-TickerSend.C:
		case <-TickerSend.C:
			//fmt.Println("sending ticker: ", t)

			e := reflect.ValueOf(&DatPayload).Elem()

			// here be sending loop
			for i := 0; i < e.NumField(); i++ {
				varName := e.Type().Field(i).Name              // NumGC
				varValue := e.Field(i).Interface()             // 0
				varType := e.Type().Field(i).Type.String()     // main.gauge
				varType = strings.TrimPrefix(varType, "main.") // gauge
				// в формате: http://<АДРЕС_СЕРВЕРА>/update/<ТИП_МЕТРИКИ>/<ИМЯ_МЕТРИКИ>/<ЗНАЧЕНИЕ_МЕТРИКИ>
				query := fmt.Sprintf("%v:%v/update/%v/%v/%v", server_address, server_port, varType, varName, varValue)
				// Метрики нужно отправлять по протоколу HTTP, методом POST:
				request, err := http.NewRequest(http.MethodPost, query, nil) //bytes.NewBuffer(dat_payload))
				if err != nil {
					log.Fatal(err)
				}
				request.Header.Set("Content-Type", "text/plain")
				resp, err := client.Do(request)
				if err != nil {
					log.Fatal(err)
				}
				defer resp.Body.Close()
				//fmt.Printf("\tmetric: %v; status_code: %v\n", varName, resp.StatusCode)
			}
		}
	}
}
