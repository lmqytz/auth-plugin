package main

import (
	"net/http"
	"strings"
	"time"
)

type HttpClient struct {
	client *http.Client
}

func NewHttpClient(conf HttpConf) *HttpClient {
	client := &http.Client{
		Timeout: time.Duration(conf.Timeout) * time.Second,
	}

	return &HttpClient{client: client}
}

func (h *HttpClient) Post(url, data string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = req.Body.Close()
	}()

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}