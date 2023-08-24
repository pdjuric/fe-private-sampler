package common

import "time"

const (
	FHMultiIPESecLevel              = 1
	SensorMaxParallelSubmitBatches  = 3 // todo rename
	ServerLogDir                    = "server-logs"
	SensorLogDir                    = "sensor-logs"
	ServerLogFilename               = "server"
	SensorLogFilename               = "sensor"
	ServerTaskDaemonChanSize        = 15
	SensorTaskChanSize              = 15
	SensorSamplingChanSizeCoeff     = 2
	SensorEncryptionChanSizeCoeff   = 1 // todo ???? (was /2 => rendez-vous occurred when BatchesPerSensor = 1)
	DecryptionParamsPollingInterval = 5 * time.Second
)
