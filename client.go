package main

import (
    "encoding/json"
    "io/ioutil"
    "net/http"
    "strings"
    "time"
)

type ResponseData struct {
    Boolean int                    `json:"boolean"`
    Message string                 `json:"message"`
    Data    map[string]interface{} `json:"data"`
}

type HttpClient struct {
    client *http.Client
}

func NewHttpClient(conf HttpConf) *HttpClient {
    client := &http.Client{
        Timeout: time.Duration(conf.Timeout) * time.Second,
    }

    return &HttpClient{client: client}
}

func (h *HttpClient) Post(url, data string) (ResponseData, error) {
    request, err := http.NewRequest("POST", url, strings.NewReader(data))
    if err != nil {
        return ResponseData{}, err
    }

    defer func() {
        _ = request.Body.Close()
    }()

    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    response, err := h.client.Do(request)
    if err != nil {
        return ResponseData{}, err
    }

    responseBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return ResponseData{}, err
    }

    var responseData ResponseData
    err = json.Unmarshal(responseBytes, &responseData)
    if err != nil {
        return ResponseData{}, err
    }

    return responseData, nil
}
