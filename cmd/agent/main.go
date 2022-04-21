package main

import (
	"bytes"
	"crypto/sha256"
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

var ConfigMap = map[string]string{
	"ADDRESS":         "127.1:8080",
	"POLL_INTERVAL":   "2",
	"REPORT_INTERVAL": "10",
	"KEY":             "",
	"JSON":            "false",
	"BULK":            "true",
}

func main() {
	rand.Seed(time.Now().UnixNano())
	client := &http.Client{}

	//jsonPtr := flag.Bool("j", false, "talk to server in json")
	//BulkPtr := flag.Bool("b", true, "sent metrics as jsons in bulk")
	positional := make(map[string]*string)
	for k := range ConfigMap {
		letter := strings.ToLower(k[0:1])
		positional[k] = flag.String(letter, ConfigMap[k], k)
	}
	//fmt.Println(*jsonPtr,*BulkPtr)
	flag.Parse() // this flips SOME booleans for some reason
	//fmt.Println(*jsonPtr,*BulkPtr)

	for k := range ConfigMap {
		if positional[k] != nil {
			ConfigMap[k] = *positional[k]
		}
		if val, ok := os.LookupEnv(k); ok {
			ConfigMap[k] = val
		}
	}
	//for k, v := range ConfigMap { fmt.Printf("%s -> %s\n", k, v) }

	// had a bad time with boolean flags, so
	bulkJSON, err := strconv.ParseBool(ConfigMap["BULK"])
	OnErrorFail(err)
	sendJSON, err := strconv.ParseBool(ConfigMap["JSON"])
	OnErrorFail(err)

	serverAddress := ConfigMap["ADDRESS"]
	pollInterval, err := strconv.Atoi(ConfigMap["POLL_INTERVAL"])
	OnErrorFail(err)
	reportInterval, err := strconv.Atoi(ConfigMap["REPORT_INTERVAL"])
	OnErrorFail(err)

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
			if !(sendJSON || bulkJSON) {
				for i, varName := range memStats {
					varValue := getField(&ms, memStats[i])
					query := fmt.Sprintf("http://%v/update/%v/%v/%v", serverAddress, "gauge", varName, varValue)
					sendStuff(client, query, nil, "text/plain")
				}
				query := fmt.Sprintf("http://%v/update/%v/%v/%v", serverAddress, "counter", "PollCount", Counter)
				sendStuff(client, query, nil, "text/plain")
			} else {
				var mj []Metrics
				for i, varName := range memStats {
					varValue := getField(&ms, memStats[i])

					mj = append(mj, Metrics{ID: varName, MType: "gauge", Value: &varValue})
					if ConfigMap["KEY"] != "" {
						mj[i].BuildHash()
					}

					if !bulkJSON && sendJSON {
						query := fmt.Sprintf("http://%v/update/", serverAddress)
						sendStuff(client, query, mj[i].JSON(), "application/json")
					}
				}

				mj = append(mj, Metrics{ID: "PollCount", MType: "counter", Delta: &Counter})
				if ConfigMap["KEY"] != "" {
					mj[len(mj)-1].BuildHash()
				}
				if !bulkJSON && sendJSON {
					query := fmt.Sprintf("http://%v/update/", serverAddress)
					sendStuff(client, query, mj[len(mj)-1].JSON(), "application/json")
				} else {
					p, err := json.Marshal(mj)
					OnErrorFail(err)
					query := fmt.Sprintf("http://%v/updates/", serverAddress)
					sendStuff(client, query, p, "application/json")
				}
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
	OnErrorFail(err)
	request.Header.Set("Content-Type", h)
	resp, err := c.Do(request)
	OnErrorFail(err)
	resp.Body.Close()
}

func jsonify(m Metrics) []byte {
	p, err := json.Marshal(m)
	OnErrorFail(err)
	return p
}

func hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	// fmt.Println(data) // DEVINFO
	// fmt.Printf("%x\n",string(hash[:])) // DEVINFO
	return fmt.Sprintf("%x", string(hash[:]))
}

func (m *Metrics) BuildHash() bool {
	switch m.MType {
	case "gauge":
		m.Hash = hash(fmt.Sprintf("%s:%s:%f:%s", m.ID, m.MType, *m.Value, ConfigMap["KEY"]))
	case "counter":
		m.Hash = hash(fmt.Sprintf("%s:%s:%d:%s", m.ID, m.MType, *m.Delta, ConfigMap["KEY"]))
	default:
		return false
	}
	return true
}

func (m *Metrics) JSON() []byte {
	p, err := json.Marshal(m)
	OnErrorFail(err)
	return p
}

func OnErrorFail(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
