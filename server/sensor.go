package server

import (
	. "fe/common"
)

type Sensor struct {
	Id        UUID        `json:"id"`
	Customers []*Customer `json:"customers"`
	*RemoteHttpServer
}

func (server *Server) NewSensor(uuid UUID, ip IP) *Sensor {
	sensor := &Sensor{
		Id:        uuid,
		Customers: make([]*Customer, 0),
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", server.HttpLogger),
		},
	}
	server.sensors.Store(uuid, sensor)
	return sensor
}

func (s *Sensor) RemoveFromCustomer(g *Customer) {
	// ignores if the sensor does not belong to the customer

	for idx, val := range g.Sensors {
		if val == s {
			g.Sensors = append(g.Sensors[:idx], g.Sensors[idx+1:]...)
		}
	}

	for idx, val := range s.Customers {
		if val == g {
			s.Customers = append(s.Customers[:idx], s.Customers[idx+1:]...)
		}
	}
}

func (s *Sensor) SubmitTask(taskId UUID, samplingParams SamplingParams, authorityIp IP) (statusCode int, responseBody []byte, e error) {
	//method := "POST"
	url := "/task"
	body := SensorTaskRequest{
		TaskId:         taskId,
		SamplingParams: samplingParams,
		AuthorityIP:    authorityIp,
	}

	return s.POST(url, body, BodyJSON)
}
