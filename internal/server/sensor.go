package server

import (
	. "fe/internal/common"
)

type Sensor struct {
	Id     UUID     `json:"id"`
	Groups []*Group `json:"groups"`
	*RemoteHttpServer
}

func (server *Server) NewSensor(uuid UUID, ip IP) *Sensor {
	sensor := &Sensor{
		Id:     uuid,
		Groups: make([]*Group, 0),
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", server.HttpLogger),
		},
	}
	server.sensors.Store(uuid, sensor)
	return sensor
}

func (s *Sensor) RemoveFromGroup(g *Group) {

	//todo if server is not in the group
	for idx, val := range g.Sensors {
		if val == s {
			g.Sensors = append(g.Sensors[:idx], g.Sensors[idx+1:]...)
		}
	}

	for idx, val := range s.Groups {
		if val == g {
			s.Groups = append(s.Groups[:idx], s.Groups[idx+1:]...)
		}
	}
}

func (s *Sensor) SubmitTask(taskId UUID, batchParams BatchParams, samplingParams SamplingParams, schema string, encryptionParams FEEncryptionParams) (statusCode int, responseBody []byte, e error) {
	//method := "POST"
	url := "/task"
	body := SubmitTaskRequest{
		TaskId:             taskId,
		BatchParams:        batchParams,
		SamplingParams:     samplingParams,
		Schema:             schema,
		FEEncryptionParams: encryptionParams,
	}

	return s.POST(url, body)
}
