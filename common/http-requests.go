package common

type RegisterSensorRequest struct {
	SensorId UUID `json:"sensorId"`
	IP
}

type ServerTaskRequest struct {
	CustomerId       UUID `json:"customerId"`
	Start            int  `json:"start"` // timestamp when server resets for the first time and starts measuring
	Duration         int  `json:"duration"`
	TariffId         UUID `json:"tariffId"`
	EnableEncryption bool `json:"enableEncryption"`
}

type AuthorityTaskRequest struct {
	Id        UUID
	SensorIds []UUID
	BatchParams
	MaxTariffValue   int  `json:"maxRateValue"`
	MaxSampleValue   int  `json:"maxSampleValue"`
	EnableEncryption bool `json:"enableEncryption"`
}

type SensorTaskRequest struct {
	TaskId UUID `json:"id"`
	SamplingParams
	AuthorityIP IP `json:"authorityIP"`
}
