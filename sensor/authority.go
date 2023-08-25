package sensor

import (
	. "fe/common"
)

type Authority struct {
	*RemoteHttpServer
}

func (a *Authority) GetEncryptionParams(taskId UUID, sensorId UUID) (FEEncryptionParams, error) {
	//method := "GET"
	url := "/encryption/" + string(taskId) + "/" + string(sensorId)

	_, responseBody, err := a.GET(url)
	if err != nil {
		return nil, err
	}

	feEncryptionParams, err := Decode(responseBody)
	return feEncryptionParams, nil
}
