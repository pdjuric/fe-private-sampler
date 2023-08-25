package common

// todo optional and required parameters in all requests

type RegisterSensorRequest struct {
	SensorId UUID `json:"sensorId"`
	IP
}

type CreateTaskRequest struct {
	GroupId   UUID `json:"groupId"`
	BatchSize int  `json:"batchSize"` // no need for BatchParams, it can be derived
	Start     int  `json:"start"`     // timestamp when server resets for the first time and starts measuring
	Duration  int  `json:"duration"`
	RateId    UUID `json:"rateId"`
}

type CreateAuthorityTaskRequest struct {
	Id        UUID
	SensorIds []UUID
	BatchParams
	MaxRateValue   int `json:"maxCoeffValue"`
	MaxSampleValue int `json:"maxSampleValue"`
}

type SubmitSensorTaskRequest struct {
	TaskId UUID `json:"id"`
	SamplingParams
	AuthorityIP IP `json:"authorityIP"`
}
