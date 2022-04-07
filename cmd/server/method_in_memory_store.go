package main

import (
	"encoding/json"
	"errors"
	"log"
)

//type InMemoryStore struct {
	//gaugeMetrics	map[string]float64
	//counterMetrics	map[string]int64
//}

func (ims *InMemoryStore) InsertInMemoryStore (m *Metrics) error {
	switch m.MType {
	case "gauge":
		if m.Value != nil {
			ims.gaugeMetrics[m.ID] = *m.Value
		} else {
			return errors.New("type not specified")
		}
	case "counter":
		if m.Delta != nil {
			ims.counterMetrics[m.ID] += *m.Delta
		} else {
			return errors.New("type not specified")
		}
	default:
		return errors.New("unknown type specified")
	}
	return nil
}

func (ims *InMemoryStore) ExtractFromInMemoryStore() []byte {
	var mj []Metrics

	for k := range ims.gaugeMetrics {
		var m Metrics
		var g []float64
		g = append(g, ims.gaugeMetrics[k])
		m.MType = "gauge"
		m.ID = k
		m.Value = &g[len(g)-1]
		mj = append(mj, m)
	}

	for k := range ims.counterMetrics {
		var m Metrics
		var c []int64
		c = append(c, ims.counterMetrics[k])
		m.MType = "counter"
		m.ID = k
		m.Delta = &c[len(c)-1]
		mj = append(mj, m)
	}
	p, err := json.Marshal(mj)
	check(err)
	return p
}

func (ims *InMemoryStore) PopulateInMemoryStore(j []byte) {
	var mj []*Metrics
	err := json.Unmarshal(j, &mj)
	if err == nil {
		for k := range mj {
			ims.InsertInMemoryStore(mj[k])
		}
	} else {
		log.Print(err)
	}
}
