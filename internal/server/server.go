package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/google/uuid"
	"sync"
)

type Server struct {
	groups          sync.Map // map[uuid.UUID]*Group
	sensors         sync.Map //map[uuid.UUID]*Sensor
	taskDaemonQueue chan *Task

	tasks sync.Map //map[uuid.UUID]*Sensor
	*HttpServer
}

var logger *Logger // todo this logger should be removed, along with SetLogger

func InitServer() *Server {
	return &Server{
		taskDaemonQueue: make(chan *Task, 15),
	}
}

func (s *Server) GetGroup(uuid uuid.UUID) (*Group, error) {
	group, exists := s.groups.Load(uuid)
	if !exists {
		return nil, fmt.Errorf("group with id %s not found", uuid)
	}

	return group.(*Group), nil
}

func (s *Server) AddSensorToGroup(uuid uuid.UUID, ip IP, group *Group) {
	sensor, exists := s.sensors.Load(uuid)
	if !exists {
		sensor = s.NewSensor(uuid, ip)
	}

	group.AddSensor(sensor.(*Sensor))
}

func (s *Server) AddTask(task *Task) {
	s.tasks.Store(task.Uuid.String(), task) // fixme task.Uuid didn't work !
	s.taskDaemonQueue <- task
}

func (s *Server) GetTaskChannel() *chan *Task {
	return &s.taskDaemonQueue
}

func SetLogger(newLogger *Logger) {
	logger = newLogger
}
