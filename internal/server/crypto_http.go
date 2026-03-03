package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// jsonMarshal wraps json.Marshal
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// jsonUnmarshal wraps json.Unmarshal
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// httpPost makes a simple POST request and returns the response body
func httpPost(url, contentType string, body []byte) ([]byte, error) {
	resp, err := http.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
