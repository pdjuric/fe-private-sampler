package sensor

import (
	. "fe/internal/common"
	"fmt"
	"github.com/google/uuid"
)

type Server struct {
	*RemoteHttpServer
}

func (sensor *Sensor) NewServer(ip IP) *Server {
	return &Server{
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", sensor.HttpLogger),
		},
	}
}

func (s *Server) Register(sensor *Sensor) {
	//method := "POST"
	url := "/group/" + sensor.Group.String() + "/sensor"
	body := RegisterSensorRequest{
		SensorId: sensor.Uuid.String(),
		IP:       sensor.IP,
	}

	statusCode, responseBody, err := s.POST(url, body)
	if err != nil {
		return
	}

	fmt.Printf("status code: %d\n", statusCode)
	fmt.Printf("response body: %s\n", responseBody)

}

func (s *Server) SubmitBatch(taskId uuid.UUID, cipher any) {
	//method := "POST"
	url := "/task/" + taskId.String() + "/data"
	body := cipher

	statusCode, responseBody, err := s.POST(url, body)
	if err != nil {
		return
	}

	fmt.Printf("status code: %d\n", statusCode)
	fmt.Printf("response body: %s\n", responseBody)

}
