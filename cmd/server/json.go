package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func DeJSONify(body *io.ReadCloser) Metrics {
	m := Metrics{}
	byteStreamBody, err := io.ReadAll(*body)
	check(err)
	err = json.Unmarshal(byteStreamBody, &m)
	check(err)
	return m
}

func (m *Metrics) jsonify() []byte {
	p, err := json.Marshal(m)
	check(err)
	return p
}

func (m *Metrics) HashCheck() bool {
	var ServerSideHash string
	switch m.MType {
	case "gauge":
		ServerSideHash = hash(fmt.Sprintf("%s:gauge:%f:%s", m.ID, *m.Value, config["KEY"]))
	case "counter":
		ServerSideHash = hash(fmt.Sprintf("%s:counter:%d:%s", m.ID, *m.Delta, config["KEY"]))
	}
	return m.Hash == ServerSideHash
}

func hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", string(hash[:]))
}

//func (m *Metrics) Validate() bool {
//	switch m.MType {
//	case "gauge": return true
//	case "counter": return true
//	default: return false
//	} return true
//}
