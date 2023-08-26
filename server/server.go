package server

import (
	. "fe/common"
	"fmt"
	"sync"
)

type Server struct {
	customers sync.Map
	sensors   sync.Map
	tasks     sync.Map

	Authority *Authority
	*Host[Task]
}

func InitServer() *Server {
	server := &Server{}
	server.Host = InitHost[Task](ServerLogDir, ServerLogFilename, ServerTaskDaemonChanSize, server.GetEndpoints())
	return server
}

func (server *Server) IsAuthoritySet() bool {
	return server.Authority != nil
}

func (server *Server) GetCustomer(uuid UUID) (*Customer, error) {
	customer, exists := server.customers.Load(uuid)
	if !exists {
		return nil, fmt.Errorf("customer with id %s not found", uuid)
	}

	return customer.(*Customer), nil
}

func (server *Server) AddSensorToCustomer(uuid UUID, ip IP, customer *Customer) {
	sensor, exists := server.sensors.Load(uuid)
	if !exists {
		sensor = server.NewSensor(uuid, ip)
		server.sensors.Store(uuid, sensor)
	}

	customer.AddSensor(sensor.(*Sensor))
}

func (server *Server) AddTask(task *Task) {
	server.tasks.Store(task.Id, task)
}

func (server *Server) GetTask(taskId UUID) (*Task, error) {
	taskAny, exists := server.tasks.Load(taskId)
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskId)
	}

	return taskAny.(*Task), nil
}
