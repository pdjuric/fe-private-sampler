package sensor

import (
	"encoding/json"
	. "fe/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// POST /task
// body: customerId, start, measuringPeriod, submittingPeriod
func (sensor *Sensor) submitTaskEndpoint(c *gin.Context) (ResponseType, int, any) {
	// get JSON from request body; can't use BindJSON as we're unmarshalling twice!
	jsonData, _ := c.GetRawData()

	// Parse SensorTaskRequest
	// schema type is verified here, so no need to check id anywhere later ! (FE .. creation)
	var taskRequest SensorTaskRequest
	err := json.Unmarshal(jsonData, &taskRequest)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	task := sensor.NewTask(&taskRequest)

	sensor.SendTaskToDaemon(task)

	msg := fmt.Sprintf("task %s added successfully", task.Id)
	sensor.HttpLogger.Info(msg)

	return StringResponse, http.StatusAccepted, msg
}

func (sensor *Sensor) setServerEndpoint(c *gin.Context) (ResponseType, int, any) {
	var ip IP

	if err := c.BindJSON(&ip); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	newServer := sensor.NewServer(ip)

	// assert that Sensor doesn't already have Server
	if sensor.Server == nil {
		sensor.Server = newServer
		msg := fmt.Sprintf("server %s set successfully", ip)
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else if sensor.Server.IP.String() == newServer.IP.String() {
		msg := fmt.Sprintf("server %s is already set", sensor.Server.IP.String())
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("could not set server %s, as server %s is already set", newServer.IP.String(), sensor.Server.IP.String())
	}

}

func (sensor *Sensor) setCustomerEndpoint(c *gin.Context) (ResponseType, int, any) {
	var data struct {
		CustomerId UUID `json:"id"`
	}

	if err := c.BindJSON(&data); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// assert that Sensor doesn't already have CustomerId
	if sensor.CustomerId.IsNil() {
		sensor.CustomerId = data.CustomerId
		msg := fmt.Sprintf("customer %s set successfully", data.CustomerId)
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else if sensor.CustomerId == data.CustomerId {
		msg := fmt.Sprintf("customer %s is already set", sensor.CustomerId)
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("could not set customer %s, as customer %s is already set", data.CustomerId, sensor.CustomerId)
	}
}

func (sensor *Sensor) registerSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	// assert that the Server is already set
	if sensor.Server == nil {
		return ErrorResponse, http.StatusBadRequest, "server must be set before sensor registration"
	}

	// assert that the CustomerId is already set
	if sensor.CustomerId.IsNil() {
		return ErrorResponse, http.StatusBadRequest, "customer must be set before sensor registration"
	}

	sensor.Server.Register(sensor)
	return NoResponse, http.StatusNoContent, nil
}

func (sensor *Sensor) getSamplesEndpoint(c *gin.Context) (ResponseType, int, any) {
	// get task uuid
	taskIdString := c.Param("id")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := sensor.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return JSONResponse, http.StatusOK, task.GetSamples()
}

func (sensor *Sensor) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"POST", "/server", sensor.setServerEndpoint},
		{"POST", "/customer", sensor.setCustomerEndpoint},
		{"POST", "/task", sensor.submitTaskEndpoint},
		{"GET", "/register", sensor.registerSensorEndpoint},
		{"GET", "/task/:id/samples", sensor.getSamplesEndpoint},
	}
}
