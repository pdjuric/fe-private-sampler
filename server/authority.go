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

func (a *Authority) GenerateFESchemaParams(taskId UUID, sensorIds []UUID, batchParams BatchParams, MaxRateValue int, MaxSampleValue int) (FESchemaParams, error) {
	url := "/task"
	body := CreateAuthorityTaskRequest{
		Id:             taskId,
		SensorIds:      sensorIds,
		BatchParams:    batchParams,
		MaxRateValue:   MaxRateValue,
		MaxSampleValue: MaxSampleValue,
	}

	statusCode, responseBody, _ := a.POST(url, body, BodyJSON)
	if statusCode == http.StatusBadRequest {
		var kvMap map[string]string
		_ = json.Unmarshal(responseBody, &kvMap)

		return nil, fmt.Errorf(kvMap["error"])
	}

	data, err := Decode(responseBody)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (a *Authority) SendCoefficients(taskId UUID, coeffs []int) (UUID, error) {
	url := "/decryption/" + string(taskId)
	data, err := Encode(coeffs)
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
