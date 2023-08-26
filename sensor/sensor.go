package sensor

import (
	. "fe/common"
	"fmt"
	"sync"
)

type Sensor struct {
	Id         UUID `json:"id"`
	CustomerId UUID // todo can this be nil on the server?
	Server     *Server
	tasks      sync.Map

	*Host[Task]
}

func InitSensor() *Sensor {
	sensor := &Sensor{
		Id: NewUUID(),
	}
	sensor.Host = InitHost[Task](SensorLogDir, SensorLogFilename, SensorTaskChanSize, sensor.GetEndpoints())
	return sensor
}

func (sensor *Sensor) AddTask(task *Task) {
	sensor.tasks.Store(task.Id, task)
}

func (sensor *Sensor) GetTask(taskId UUID) (*Task, error) {
	taskAny, exists := sensor.tasks.Load(taskId)
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskId)
	}

	return taskAny.(*Task), nil
}
