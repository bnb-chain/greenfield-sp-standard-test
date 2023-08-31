package utils

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
)

func HttpGetWithHeaders(url string, header map[string]string) (*http.Header, string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if nil != resp.Body {
		defer resp.Body.Close()
	} else {
		return nil, "{}", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return &resp.Header, string(body), err
}
func HttpPostWithHeaders(url string, jsonStr string, header map[string]string) (*http.Header, string, error) {
	var json = []byte(jsonStr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	if (nil != resp) && (nil != resp.Body) {
		defer resp.Body.Close()
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return &resp.Header, string(body), err
}

// CheckHttpHeader verify http header support cross-origin
func CheckHttpHeader(responseHeader http.Header, domain string, expectHeader map[string]string) bool {
	compare := true
	for headerKey, headerValue := range expectHeader {
		respValues := responseHeader.Get(headerKey)
		if strings.Contains(respValues, domain) {
			return true
		}
		if respValues == "" {
			log.Errorf("headerKey: %s, respValues: %s, expectValue: %s", headerKey, respValues, headerValue)
			compare = false
			continue
		}
		headerValue = strings.TrimSpace(headerValue)
		respValues = strings.TrimSpace(respValues)

		if respValues != headerValue && !strings.Contains(respValues, "*") {
			log.Errorf("headerKey: %s, respValues: %s, expectValue: %s", headerKey, respValues, headerValue)
			compare = false
		}
	}
	return compare
}
