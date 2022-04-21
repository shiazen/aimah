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
	OnErrorFail(err)
	err = json.Unmarshal(byteStreamBody, &m)
	OnErrorFail(err)
	return m
}

func DeJSONifyArray(body *io.ReadCloser) []Metrics {
	m := []Metrics{}
	byteStreamBody, err := io.ReadAll(*body)
	OnErrorFail(err)
	err = json.Unmarshal(byteStreamBody, &m)
	OnErrorFail(err)
	return m
}

func (m *Metrics) JSON() []byte {
	p, err := json.Marshal(m)
	OnErrorFail(err)
	return p
}

func (m *Metrics) HashCheck() bool {
	var ServerSideHash string
	switch m.MType {
	case "gauge":
		ServerSideHash = MkHash(fmt.Sprintf("%s:gauge:%f:%s", m.ID, *m.Value, ConfigMap["KEY"]))
	case "counter":
		ServerSideHash = MkHash(fmt.Sprintf("%s:counter:%d:%s", m.ID, *m.Delta, ConfigMap["KEY"]))
	}
	return m.Hash == ServerSideHash
}

func MkHash(d string) string {
	h := sha256.Sum256([]byte(d))
	return fmt.Sprintf("%x", string(h[:]))
}

//func (m *Metrics) Validate() bool {
//	switch m.MType {
//	case "gauge": return true
//	case "counter": return true
//	default: return false
//	} return true
//}
