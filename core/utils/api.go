package utils

import (
	"bytes"
	"encoding/hex"
	"net/http"
)

func GetNonce(endpoint string, header map[string]string) (*http.Header, string, error) {
	respHeader, response, err := HttpGetWithHeaders(endpoint+"/auth/request_nonce", header)
	return respHeader, response, err
}
func OptionsGetNonce(endpoint string, header map[string]string) (*http.Response, string, error) {
	respHeader, response, err := HttpOptionsWithHeaders(endpoint+"/auth/request_nonce", header)
	return respHeader, response, err
}

func ConvertToString(targetStrung []byte) string {
	var buf bytes.Buffer
	buf.Write(targetStrung)
	return "0x" + hex.EncodeToString(buf.Bytes())
}
