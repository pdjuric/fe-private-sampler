package server

import (
	"encoding/json"
	. "fe/common"
	"fmt"
	"net/http"
)

type Authority struct {
	*RemoteHttpServer
}

func (server *Server) NewAuthority(ip IP) *Authority {
	return &Authority{
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", server.HttpLogger),
		},
	}
}

func (a *Authority) SubmitTask(taskId UUID, sensorIds []UUID, batchParams BatchParams, MaxTariffValue int, MaxSampleValue int, EnableEncryption bool) error {
	url := "/task"
	body := AuthorityTaskRequest{
		Id:               taskId,
		SensorIds:        sensorIds,
		BatchParams:      batchParams,
		MaxTariffValue:   MaxTariffValue,
		MaxSampleValue:   MaxSampleValue,
		EnableEncryption: EnableEncryption,
	}

	statusCode, responseBody, _ := a.POST(url, body, BodyJSON)
	if statusCode == http.StatusBadRequest {
		var kvMap map[string]string
		_ = json.Unmarshal(responseBody, &kvMap)

		return fmt.Errorf(kvMap["error"])
	}

	return nil
}

func (a *Authority) SendRates(taskId UUID, rates []int) (UUID, error) {
	url := "/rates/" + string(taskId)
	data, err := Encode(rates)
	if err != nil {
		return "", err
	}

	_, responseBody, err := a.POST(url, data, BodyOctetStream)
	decryptionParamsId, err := NewUUIDFromString(string(responseBody))
	if err != nil {
		a.Logger.Err(err)
		return "", nil
	}

	return decryptionParamsId, nil
}

func (a *Authority) FetchSchemaParamsStatus(taskId UUID) (*string, error) {
	url := "/schema-status/" + string(taskId)
	statusCode, responseBody, err := a.GET(url)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		a.Logger.Err(fmt.Errorf("status code: %d, expected %d", statusCode, http.StatusOK))
		return nil, fmt.Errorf("status code: %d", statusCode)
	}

	status := string(responseBody)

	return &status, nil
}

func (a *Authority) FetchDecryptionParamsStatus(taskId UUID, decryptionParamsId UUID) (*string, error) {
	url := "/decryption-status/" + string(taskId) + "/" + string(decryptionParamsId)
	statusCode, responseBody, err := a.GET(url)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		a.Logger.Err(fmt.Errorf("status code: %d, expected %d", statusCode, http.StatusOK))
		return nil, fmt.Errorf("status code: %d", statusCode)
	}

	status := string(responseBody)

	return &status, nil

}

func (a *Authority) FetchDecryptionParams(taskId UUID, decryptionParamsId UUID) (FEDecryptionParams, error) {
	url := "/decryption/" + string(taskId) + "/" + string(decryptionParamsId)
	statusCode, responseBody, err := a.GET(url)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		a.Logger.Err(fmt.Errorf("status code: %d, expected %d", statusCode, http.StatusOK))
		return nil, fmt.Errorf("status code: %d", statusCode)
	}

	data, err := Decode(responseBody)
	if err != nil {
		return nil, err
	}

	return data, nil
}
