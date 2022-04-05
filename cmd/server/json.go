package main

import (
	"encoding/json"
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
