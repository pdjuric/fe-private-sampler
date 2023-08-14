package sensor

import (
	. "fe/internal/common"
	"fmt"
	"sync"
)

type Sensor struct {
	Id      UUID `json:"id"`
	GroupId UUID // sensor can be at most one group -> // todo on the server		//can this be nil
	Server  *Server
	tasks   sync.Map

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
	s.tasks.Store(task.Id, task)
}

func (s *Sensor) GetTask(taskId UUID) (*Task, error) {
	taskAny, exists := s.tasks.Load(taskId)
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskId)
	}

	return taskAny.(*Task), nil
}
