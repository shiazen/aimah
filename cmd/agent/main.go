package main

import (
	"fmt"
	"runtime"
	"net/http"
	"math/rand"
	"time"
	"reflect"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var memStats = [26]string{ "Alloc", "BuckHashSys", "Frees",
		"GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle",
		"HeapInuse", "HeapObjects", "HeapReleased", "HeapSys",
		"LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC",
		"NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs",
		"StackInuse", "StackSys", "Sys"}

var ms runtime.MemStats
var Counter int64
var RandomValue float64

func main() {
	rand.Seed(time.Now().UnixNano())
	client := &http.Client{}

	var pollInterval = 2
	var reportInterval = 10
	serverAddress := "http://127.0.0.1"
	serverPort := 8080

	TickerPoll := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer TickerPoll.Stop()
	TickerSend := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer TickerSend.Stop()

	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() { }()

	for {
		select {
			case datSignal := <-SigChan:
			switch datSignal {
				case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
					fmt.Printf("\nSignal %v triggered.\n", datSignal)
					os.Exit(0)
				default:
					fmt.Printf("\nSignal %v triggered.\n", datSignal)
					os.Exit(1)
			}
			case <-TickerPoll.C:
				runtime.ReadMemStats(&ms)
				RandomValue = rand.Float64()
				Counter++
			case <-TickerSend.C:
				//fmt.Println(Counter, RandomValue)
				for i, varName := range memStats {
					varValue := getField(&ms, memStats[i])
					varType := "gauge"
					query := fmt.Sprintf("%v:%v/update/%v/%v/%v", serverAddress, serverPort, varType, varName, varValue)
					sendStuff(client, query)
				}
				sendStuff(client, fmt.Sprintf("%v:%v/update/%v/%v/%v", serverAddress, serverPort, "gauge", "RandomValue", RandomValue))
				sendStuff(client, fmt.Sprintf("%v:%v/update/%v/%v/%v", serverAddress, serverPort, "counter", "PollCount", Counter))
		}
	}
}

func getField(v *runtime.MemStats, field string) reflect.Value {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	return f
}

func sendStuff (c *http.Client, q string) {
	request, err := http.NewRequest(http.MethodPost, q, nil)
	if err != nil { log.Fatal(err) }
	request.Header.Set("Content-Type", "text/plain")
	resp, err := c.Do(request)
	if err != nil { log.Fatal(err) }
	resp.Body.Close()
}
