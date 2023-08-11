package common

import (
	"github.com/google/uuid"
	"net"
)

// todo optional and required parameters in all requests

type RegisterSensorRequest struct {
	SensorId string `json:"sensorId"`
	IP
}

// todo for now, this is avoideede altogether, but needs to be used for Multi
type SubmitBatchRequest struct {
	TaskId uuid.UUID `json:"id"` // todo may be removed, it's send through query param
	// todo  sensor id hashed with task id !!!
	BatchIdx int `json:"batchIdx"`
	// todo add FE cipher
}

type CreateTaskRequest struct {
	GroupId   uuid.UUID `json:"groupId"`
	BatchSize int       `json:"batchSize"` // no need for BatchParams, it can be derived
	SamplingParams
	CoefficientsByPeriod []int `json:"coefficientsByPeriod" default:"nil" required:"true"`
}

type SubmitTaskRequest struct {
	TaskId uuid.UUID `json:"id"`
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
