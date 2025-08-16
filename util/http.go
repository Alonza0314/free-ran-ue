package util

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type HttpResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

func SendHttpRequest(uri string, method string, headers map[string]string, data []byte) (*HttpResponse, error) {
	req, err := http.NewRequest(method, uri, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if err := resp.Body.Close(); err != nil {
		return nil, fmt.Errorf("failed to close response body: %v", err)
	}

	return &HttpResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}
