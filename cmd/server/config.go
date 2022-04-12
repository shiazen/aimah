package main

import (
	"flag"
	"os"
	"strings"
)

var ConfigMap = map[string]string{
	"ADDRESS":        "127.1:8080",
	"RESTORE":        "true",
	"STORE_INTERVAL": "300",
	"STORE_FILE":     "/tmp/devops-metrics-db.json",
	"KEY":            "",
	"DATABASE_DSN":   "",
}

func PopulateConfig(c *map[string]string) {
	positional := make(map[string]*string)
	for k := range ConfigMap {
		letter := strings.ToLower(k[0:1])
		if k == "STORE_FILE" {
			letter = "f"
		} else if k == "STORE_INTERVAL" {
			letter = "i"
		}
		positional[k] = flag.String(letter, ConfigMap[k], k)
	}
	flag.Parse()

	for k := range ConfigMap {
		if positional[k] != nil {
			ConfigMap[k] = *positional[k]
		}
		if val, ok := os.LookupEnv(k); ok {
			ConfigMap[k] = val
		}
	}
}
