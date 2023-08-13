package common

import (
	"fmt"
	_uuid "github.com/google/uuid"
	"net"
)

// todo optional and required parameters in all requests

type RegisterSensorRequest struct {
	SensorId UUID `json:"sensorId"`
	IP
}

type SubmitCipherRequest struct {
	//TaskId   UUID 		`json:"taskId"` // sent through path param
	SensorId UUID     `json:"sensorId"`
	BatchIdx int      `json:"batchIdx"`
	Cipher   FECipher `json:"cipher"`
}

type CreateTaskRequest struct {
	GroupId   UUID `json:"groupId"`
	BatchSize int  `json:"batchSize"` // no need for BatchParams, it can be derived
	SamplingParams
	CoefficientsByPeriod []int `json:"coefficientsByPeriod" default:"nil" required:"true"`
}

type SubmitTaskRequest struct {
	TaskId UUID `json:"id"`
	BatchParams
	SamplingParams

	Schema string // needed for recognizing FEEncryptionParams concrete type
	FEEncryptionParams
}

type IP struct {
	Schema string `json:"schema"`
	IPv4   net.IP `json:"ipv4"`
	Port   string `json:"port"`
}

func (ip IP) String() string {
	return ip.Schema + "://" + ip.IPv4.String() + ":" + ip.Port
}

type UUID string

func (uuid UUID) Verify() bool {
	_, err := _uuid.Parse(string(uuid))
	return err == nil
}

func (uuid UUID) IsNil() bool {
	return string(uuid) == ""
}

func NewUUID() UUID {
	return UUID(_uuid.New().String())
}

func NewUUIDFromString(uuidString string) (UUID, error) {
	uuid := UUID(uuidString)
	if !uuid.Verify() {
		return UUID(""), fmt.Errorf("invalid UUID: %s", uuidString)
	} else {
		return uuid, nil
	}
}
