package utils

import (
	"bytes"
	"encoding/hex"
	"net/http"
)

func GetNonce(userAddress string, endpoint string) (*http.Header, string, error) {
	header := make(map[string]string)
	header["X-Gnfd-User-Address"] = userAddress
	header["X-Gnfd-App-Domain"] = "https://greenfield.bnbchain.org/"
	respHeader, response, err := HttpGetWithHeaders(endpoint+"/auth/request_nonce", header)
	return respHeader, response, err
}

func ConvertToString(targetStrung []byte) string {
	var buf bytes.Buffer
	buf.Write(targetStrung)
	return "0x" + hex.EncodeToString(buf.Bytes())
}
