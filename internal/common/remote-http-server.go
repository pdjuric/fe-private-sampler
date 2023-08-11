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
func (httpClient *RemoteHttpServer) POST(url string, body any) (statusCode int, responseBody []byte, e error) {
	payload, _ := json.Marshal(body)
	httpClient.Logger.Info("POST %s body: %v", httpClient.IP.String()+url, string(payload))

	e = fmt.Errorf("http error")

	// Create a request with the payload
	req, err := http.NewRequest("POST", httpClient.IP.String()+url, bytes.NewBuffer(payload))
	if err != nil {
		httpClient.Logger.Error("error during creating request: %s", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		httpClient.Logger.Error("error during sending request: %s", err)
		return
	}
	defer resp.Body.Close() // todo does this need to be somewhere else ?

	statusCode = resp.StatusCode
	if statusCode != 204 {
		if responseBody, err = getResponseBody(resp); err != nil {
			// todo does it raise exception if the response is empty?
			httpClient.Logger.Error("error during creating request: %s", err)
			return
		}
	}

	//httpLogger.Infof("server: %s, ip: %s, status code: %d", s.Uuid, s.Ip, status)
	return

}

func getResponseBody(resp *http.Response) (body []byte, err error) {
	body = make([]byte, 0)
	_, err = resp.Body.Read(body)
	return
}
