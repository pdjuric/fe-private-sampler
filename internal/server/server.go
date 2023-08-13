package server

import (
	. "fe/internal/common"
	"fmt"
	"sync"
)

type Server struct {
	groups  sync.Map
	sensors sync.Map
	tasks   sync.Map

	*Host[Task]
}

func InitServer() *Server {
	server := &Server{}
	server.Host = InitHost[Task](ServerLogDir, ServerLogFilename, ServerTaskDaemonChanSize, server.GetEndpoints())
	return server
}

func (server *Server) GetGroup(uuid UUID) (*Group, error) {
	group, exists := server.groups.Load(uuid)
	if !exists {
		return nil, fmt.Errorf("group with id %s not found", uuid)
	}

	return group.(*Group), nil
}

func (server *Server) AddSensorToGroup(uuid UUID, ip IP, group *Group) {
	sensor, exists := server.sensors.Load(uuid)
	if !exists {
		sensor = server.NewSensor(uuid, ip)
	}

	group.AddSensor(sensor.(*Sensor))
}

func (server *Server) AddTask(task *Task) {
	server.tasks.Store(task.Id, task)
}
