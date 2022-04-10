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
	"strings"
	"syscall"
	"time"
	"crypto/sha256"
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
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

var config = map[string]string{
	"ADDRESS":         "127.1:8080",
	"POLL_INTERVAL":   "2",
	"REPORT_INTERVAL": "10",
	"KEY":	"",
}

func main() {
	rand.Seed(time.Now().UnixNano())
	client := &http.Client{}

	jsonPtr := flag.Bool("j", true, "talk to server in json")
	//flag.Parse()
	//fmt.Println(*jsonPtr)

	positional := make(map[string]*string)
	for k := range config {
		letter := strings.ToLower(k[0:1])
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
					if ( config["KEY"] != "" ) {
						varHash := hash(fmt.Sprintf("%s:gauge:%f:%s", varName, varValue, config["KEY"]))
						//fmt.Printf("%s:gauge:%f:%s\n", varName, varValue, config["KEY"]) // DEVINFO
						Payload = jsonify(Metrics{ID: varName, MType: varType, Value: &varValue, Hash: varHash})
					} else {
						Payload = jsonify(Metrics{ID: varName, MType: varType, Value: &varValue})
					}
					query = fmt.Sprintf("http://%v/update/", serverAddress)
					// fmt.Println(string(Payload)) // DEVINFO
					sendStuff(client, query, Payload, "application/json")
				} else {
					query = fmt.Sprintf("http://%v/update/%v/%v/%v", serverAddress, varType, varName, varValue)
					sendStuff(client, query, Payload, "text/plain")
				}
			}
			if *jsonPtr {
				if config["KEY"] != "" {
					varHash := hash(fmt.Sprintf("%s:counter:%d:%s", "PollCount", Counter, config["KEY"]))
					//fmt.Printf("%s:counter:%d:%s\n", "PollCount", Counter, config["KEY"]) // DEVINFO
					Payload = jsonify(Metrics{ID: "PollCount", MType: "counter", Delta: &Counter, Hash: varHash})
				} else {
					Payload = jsonify(Metrics{ID: "PollCount", MType: "counter", Delta: &Counter})
				}
				// fmt.Println(string(Payload)) // DEVINFO
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

func hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	// fmt.Println(data) // DEVINFO
	// fmt.Printf("%x\n",string(hash[:])) // DEVINFO
	return fmt.Sprintf("%x", string(hash[:]))
}
