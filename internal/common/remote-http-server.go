package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RemoteHttpServer struct {
	IP
	Logger *Logger
}

// POST ; url should not include schema, ip address and port
func (httpClient *RemoteHttpServer) POST(url string, body any) (int, []byte, error) {
	payload, _ := json.Marshal(body)
	httpClient.Logger.Info("POST %s body: %s", httpClient.IP.String()+url, string(payload))

	// Create a request with the payload
	req, err := http.NewRequest("POST", httpClient.IP.String()+url, bytes.NewBuffer(payload))
	if err != nil {
		httpClient.Logger.Error("error during creating request: %s", err)
		return 0, nil, fmt.Errorf("error during creating http request")
	}

	req.Header.Set("Content-Type", "application/json")

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

func getResponseBody(resp *http.Response) (body []byte, err error) {
	body = make([]byte, 0)
	bodyLen, err := resp.Body.Read(body)
	resp.Body.Close()
	if err != nil && bodyLen != 0 {
		return nil, err
	}
	return body, nil

}
