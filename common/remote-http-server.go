package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RemoteHttpServer struct {
	IP
	Logger *Logger
}

// POST ; url should not include schema, ip address and port
func (httpClient *RemoteHttpServer) POST(url string, body any, contentType string) (int, []byte, error) {
	if contentType == BodyJSON {
		body, _ = json.Marshal(body)
		httpClient.Logger.Info("POST %s body: %s", httpClient.IP.String()+url, body)
	} else if contentType != BodyOctetStream {
		panic("ergkjne")
	}

	// Create a request with the payload
	req, err := http.NewRequest("POST", httpClient.IP.String()+url, bytes.NewBuffer(body.([]byte)))
	if err != nil {
		httpClient.Logger.Error("error during creating request: %s", err)
		return 0, nil, fmt.Errorf("error during creating http request")
	}

	if contentType == "json" {

		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		httpClient.Logger.Error("error during sending request: %s", err)
		return 0, nil, fmt.Errorf("error during sending http request")
	}

	// get status code & read response body
	statusCode := resp.StatusCode
	responseBody, err := getResponseBody(resp)
	if err != nil {
		httpClient.Logger.Error("error during reading response body: %s", err)
		return statusCode, nil, fmt.Errorf("error during reading http response")
	}

	httpClient.Logger.Info("POST %s -> %d %s ", httpClient.IP.String()+url, statusCode, string(responseBody))
	return statusCode, responseBody, nil
}

// GET ; url should not include schema, ip address and port
func (httpClient *RemoteHttpServer) GET(url string) (int, []byte, error) {
	httpClient.Logger.Info("GET %s", httpClient.IP.String()+url)

	// Create a request with the payload
	req, err := http.NewRequest("GET", httpClient.IP.String()+url, nil)
	if err != nil {
		httpClient.Logger.Error("error during creating request: %s", err)
		return 0, nil, fmt.Errorf("error during creating http request")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		httpClient.Logger.Error("error during sending request: %s", err)
		return 0, nil, fmt.Errorf("error during sending http request")
	}

	// get status code & read response body
	statusCode := resp.StatusCode
	responseBody, err := getResponseBody(resp)
	if err != nil {
		httpClient.Logger.Error("error during reading response body: %s", err)
		return statusCode, nil, fmt.Errorf("error during reading http response")
	}

	httpClient.Logger.Info("GET %s -> %d %s ", httpClient.IP.String()+url, statusCode, string(responseBody))
	return statusCode, responseBody, nil
}

func getResponseBody(resp *http.Response) (body []byte, err error) {
	bytees, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return bytees, nil
}
