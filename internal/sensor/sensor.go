package sensor

import (
	. "fe/internal/common"
)

type Sensor struct {
	Id      UUID `json:"id"`
	GroupId UUID // sensor can be at most one group -> // todo on the server		//can this be nil
	Server  *Server

	*Host[Task]
}

func InitSensor() *Sensor {
	sensor := &Sensor{
		Id: NewUUID(),
	}
	sensor.Host = InitHost[Task](SensorLogDir, SensorLogFilename, SensorTaskChanSize, sensor.GetEndpoints())
	return sensor
}

func (s *Sensor) AddTask(task *Task) {
	// todo
}
