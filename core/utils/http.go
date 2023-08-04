package utils

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
)

func HttpGetWithHeaders(url string, header map[string]string) (http.Header, string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return req.Header, "", err
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if nil != resp.Body {
		defer resp.Body.Close()
	} else {
		return resp.Header, "{}", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, "", err
	}

	return resp.Header, string(body), err
}
func HttpPostWithHeaders(url string, jsonStr string, header map[string]string) (http.Header, string, error) {
	var json = []byte(jsonStr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json))
	if err != nil {
		return req.Header, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return resp.Header, "", err
	}
	if (nil != resp) && (nil != resp.Body) {
		defer resp.Body.Close()
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, "", err
	}
	return resp.Header, string(body), err
}

// CheckHttpHeader verify http header support cross-origin
func CheckHttpHeader(responseHeader http.Header, domain string, expectHeader map[string]string) bool {
	compare := true
	for headerKey, headerValue := range expectHeader {
		respValues := responseHeader.Get(headerKey)
		if respValues == "" {
			log.Errorf("headerKey: %s, respValues: %s, expectValue: %s", headerKey, respValues, headerValue)
			compare = false
			continue
		}
		headerValue = strings.TrimSpace(headerValue)
		respValues = strings.TrimSpace(respValues)

		if respValues != headerValue && !strings.Contains(respValues, "*") && !strings.Contains(respValues, domain) {
			log.Errorf("headerKey: %s, respValues: %s, expectValue: %s", headerKey, respValues, headerValue)
			compare = false
		}
	}
	return compare
}