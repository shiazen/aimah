package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
)

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

func hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", string(hash[:]))
}

func (m *Metrics) HashCheck() bool {
	var LocalHash string
	switch m.MType {
	case "gauge":
		LocalHash = hash(fmt.Sprintf("%s:gauge:%f:%s", m.ID, *m.Value, config["KEY"]))
	case "counter":
		LocalHash = hash(fmt.Sprintf("%s:counter:%d:%s", m.ID, *m.Delta, config["KEY"]))
	}
	if m.Hash == LocalHash {
		return true
	}
return false
}

//func (m *Metrics) Validate() bool {
//	switch m.MType {
//	case "gauge": return true
//	case "counter": return true
//	default: return false
//	} return true
//}
