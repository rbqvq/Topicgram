package utils

import (
	. "Topicgram/common"
	"Topicgram/pkg/proxy"
	"bytes"
	"io"
	"net/http"
	"time"
)

const defaultTimeout = time.Minute

var (
	defaultClient = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			DialContext:       proxy.DialContext,
			ForceAttemptHTTP2: true,
			DisableKeepAlives: true,
			TLSClientConfig:   TLSConfig,
		},
	}

	BotClient = &http.Client{
		Transport: &http.Transport{
			DialContext:       proxy.DialContext,
			ForceAttemptHTTP2: true,
			TLSClientConfig:   TLSConfig,
		},
	}
)

// Curl use global proxy settings
func Curl(method, url string, data []byte, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return 0, nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	response, err := defaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	resp, err := io.ReadAll(response.Body)
	return response.StatusCode, resp, err
}
