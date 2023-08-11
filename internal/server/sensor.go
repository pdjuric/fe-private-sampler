package server

import (
	. "fe/internal/common"
	"github.com/google/uuid"
)

type Sensor struct {
	Uuid   uuid.UUID `json:"id"`
	Groups []*Group  `json:"groups"`
	*RemoteHttpServer
}

func (s *Server) NewSensor(uuid uuid.UUID, ip IP) *Sensor {
	sensor := &Sensor{
		Uuid:   uuid,
		Groups: make([]*Group, 0),
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", s.HttpLogger),
		},
	}
	s.sensors.Store(uuid, sensor)
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

func (s *Sensor) SubmitTask(taskRequest *SubmitTaskRequest) (statusCode int, responseBody []byte, e error) {
	//method := "POST"
	url := "/task"

	return s.POST(url, taskRequest)

}
