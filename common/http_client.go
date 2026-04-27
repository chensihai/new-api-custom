package common

import "net/http"

var globalHttpClient *http.Client

func SetGlobalHttpClient(client *http.Client) {
	globalHttpClient = client
}

func GetGlobalHttpClient() *http.Client {
	if globalHttpClient != nil {
		return globalHttpClient
	}
	return http.DefaultClient
}
