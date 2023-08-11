package sensor

import (
	. "fe/internal/common"
	"github.com/google/uuid"
)

type Sensor struct {
	Uuid   uuid.UUID  `json:"id"`
	Group  *uuid.UUID // sensor can be at most one group -> // todo on the server
	Server *Server

	*HttpServer

	// task daemon
	taskChan          *chan *Task // todo close
	taskDaemonStateFn func() RunnableState
	taskDaemonStopFn  func()
}

func InitSensor() *Sensor {
	taskChan := make(chan *Task, 15)
	return &Sensor{
		Uuid:     uuid.New(),
		taskChan: &taskChan, //make(chan *Task, 15),			// todo wtf to do with this chan
	}
}

func (s *Sensor) AddTask(task *Task) {
	*s.taskChan <- task // todo wtf to do with this chan
}

func (s *Sensor) GetTaskChannel() *chan *Task {
	return s.taskChan // todo wtf to do with this chan
}
