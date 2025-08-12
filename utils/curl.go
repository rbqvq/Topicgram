package utils

import (
	. "Topicgram/common"
	"bytes"
	"io"
	"net/http"
	"time"

	"Topicgram/pkg/proxy"
)

var (
	defaultTransport = &http.Transport{
		DialContext:       proxy.DialContext,
		ForceAttemptHTTP2: true,
		DisableKeepAlives: true,
		TLSClientConfig:   TLSConfig,
	}

	BotClient = &http.Client{
		Timeout:   60 * time.Second,
		Transport: defaultTransport,
	}
)

// Curl use global proxy settings
func Curl(method, url string, data []byte, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return 000, nil, err
	}

	if len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: defaultTransport,
	}

	response, err := client.Do(req)
	if err != nil {
		return 000, nil, err
	}

	defer response.Body.Close()

	resp, err := io.ReadAll(response.Body)
	return response.StatusCode, resp, err
}
