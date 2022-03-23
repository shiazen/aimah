package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var memStats = [27]string{"Alloc", "BuckHashSys", "Frees",
	"GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle",
	"HeapInuse", "HeapObjects", "HeapReleased", "HeapSys",
	"LastGC", "Lookups", "MCacheInuse", "MCacheSys",
	"MSpanInuse", "MSpanSys", "Mallocs", "NextGC",
	"NumForcedGC", "NumGC", "OtherSys", "PauseTotalNs",
	"StackInuse", "StackSys", "Sys", "RandomValue"}

var ms runtime.MemStats
var Counter int64
var RandomValue float64

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

var config = map[string]string{
	"ADDRESS":         "127.0.0.1:8080",
	"POLL_INTERVAL":   "2",
	"REPORT_INTERVAL": "10",
}

func main() {
	rand.Seed(time.Now().UnixNano())
	client := &http.Client{}

	jsonPtr := flag.Bool("j", true, "talk to server in json")
	flag.Parse()
	//fmt.Println(*jsonPtr)

	for k := range config {
		if val, ok := os.LookupEnv(k); ok {
			config[k] = val
		}
	}
	//for k, v := range config { fmt.Printf("%s -> %s\n", k, v) }

	serverAddress := config["ADDRESS"]
	pollInterval, err := strconv.Atoi(config["POLL_INTERVAL"])
	if err != nil {
		log.Fatal(err)
	}
	reportInterval, err := strconv.Atoi(config["REPORT_INTERVAL"])
	if err != nil {
		log.Fatal(err)
	}

	TickerPoll := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer TickerPoll.Stop()
	TickerSend := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer TickerSend.Stop()

	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {}()

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
			var query string
			var Payload []byte

			for i, varName := range memStats {
				varValue := getField(&ms, memStats[i])
				varType := "gauge"
				if *jsonPtr {
					Payload = jsonify(Metrics{ID: varName, MType: varType, Value: &varValue})
					query = fmt.Sprintf("http://%v/update/", serverAddress)
					// fmt.Println(string(Payload))
					sendStuff(client, query, Payload, "application/json")
				} else {
					query = fmt.Sprintf("http://%v/update/%v/%v/%v", serverAddress, varType, varName, varValue)
					sendStuff(client, query, Payload, "text/plain")
				}
			}
			if *jsonPtr {
				Payload = jsonify(Metrics{ID: "PollCount", MType: "counter", Delta: &Counter})
				// fmt.Println(string(Payload))
				query = fmt.Sprintf("http://%v/update/", serverAddress)
				sendStuff(client, query, Payload, "application/json")
			} else {
				query = fmt.Sprintf("http://%v/update/%v/%v/%v", serverAddress, "counter", "PollCount", Counter)
				sendStuff(client, query, Payload, "text/plain")
			}
		}
	}
}

func getField(v *runtime.MemStats, field string) float64 {
	r := reflect.ValueOf(v)
	f := reflect.Indirect(r).FieldByName(field)
	if field == "GCCPUFraction" {
		return (f.Float())
	} else if field == "RandomValue" {
		return RandomValue
	}
	return float64(f.Uint())
}

func sendStuff(c *http.Client, q string, b []byte, h string) {
	request, err := http.NewRequest(http.MethodPost, q, bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Content-Type", h)
	//request.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()
}

func jsonify(m Metrics) []byte {
	p, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	return p
}
