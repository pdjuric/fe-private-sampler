package sensor

import (
	. "fe/common"
	"fmt"
	"net/http"
)

type Server struct {
	*RemoteHttpServer
}

// NewServer creates new remote server with the remote IP address.
func (sensor *Sensor) NewServer(ip IP) *Server {
	return &Server{
		RemoteHttpServer: &RemoteHttpServer{
			IP:     ip,
			Logger: GetLogger("http client", sensor.HttpLogger),
		},
	}
}

// Register registers the sensor to the server.
func (s *Server) Register(sensor *Sensor) {
	//method := "POST"
	url := "/customer/" + string(sensor.CustomerId) + "/sensor"
	body := RegisterSensorRequest{
		SensorId: sensor.Id,
		IP:       *sensor.IP,
	}

	statusCode, responseBody, err := s.POST(url, body, BodyJSON)
	if err != nil {
		return
	}

	fmt.Printf("status code: %d\n", statusCode)
	fmt.Printf("response body: %s\n", responseBody)

}

// SubmitCipher sends encrypted data to the server.
func (s *Server) SubmitCipher(taskId UUID, sensorId UUID, cipher FECipher) error {
	//method := "POST"
	url := "/task/" + string(taskId) + "/" + string(sensorId)
	data, err := Encode(cipher)
	if err != nil {
		return err
	}

	statusCode, _, err := s.POST(url, data, BodyOctetStream)
	if err != nil {
		return err
	}

	// successful submission returns status code http.StatusAccepted
	if statusCode != http.StatusAccepted {
		return fmt.Errorf("wrong status code, expected %d, got %d", http.StatusAccepted, statusCode)
	}
	return err
}
