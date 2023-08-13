package sensor

import (
	. "fe/internal/common"
	"fmt"
	"net/http"
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
	url := "/group/" + string(sensor.GroupId) + "/sensor"
	body := RegisterSensorRequest{
		SensorId: sensor.Id,
		IP:       *sensor.IP,
	}

	statusCode, responseBody, err := s.POST(url, body)
	if err != nil {
		return
	}

	fmt.Printf("status code: %d\n", statusCode)
	fmt.Printf("response body: %s\n", responseBody)

}

func (s *Server) SubmitCipher(taskId UUID, sensorId UUID, batchIdx int, cipher FECipher) error {
	//method := "POST"
	url := "/task/" + string(taskId) + "/data"
	body := SubmitCipherRequest{
		SensorId: sensorId,
		BatchIdx: batchIdx,
		Cipher:   cipher,
	}

	statusCode, _, err := s.POST(url, body)
	if err != nil {
		return err
	}

	// successful submission returns status code http.StatusAccepted
	if statusCode != http.StatusAccepted {
		return fmt.Errorf("wrong status code, expected %d, got %d", http.StatusAccepted, statusCode)
	}
	return err
}
