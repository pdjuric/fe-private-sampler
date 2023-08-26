package common

import "time"

const (
	FHMultiIPESecLevel              = 1
	MaxParallelSubmissionsPerSensor = 3 //
	ServerLogDir                    = "server-logs"
	SensorLogDir                    = "sensor-logs"
	ServerLogFilename               = "server"
	SensorLogFilename               = "sensor"
	ServerTaskDaemonChanSize        = 15
	SensorTaskChanSize              = 15
	SensorSamplingChanSizeCoeff     = 2
	SensorEncryptionChanSizeCoeff   = 1
	DecryptionParamsPollingInterval = 5 * time.Second
	SchemaParamsPollingInterval     = 10 * time.Second
	EncryptionParamsPollingInterval = 10 * time.Second
	AuthorityLogDir                 = "authority-logs"
	AuthorityLogFilename            = "authority"
)
