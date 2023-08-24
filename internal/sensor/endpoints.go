package sensor

import (
	"encoding/json"
	. "fe/internal/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

//todo do I need to replace any more maps with sync.Map

// POST /task
// body: groupId, start, measuringPeriod, submittingPeriod
func (sensor *Sensor) submitTaskEndpoint(c *gin.Context) (ResponseType, int, any) {
	// get JSON from request body; can't use BindJSON as we're unmarshalling twice!
	jsonData, _ := c.GetRawData()

	// Parse SubmitSensorTaskRequest
	// schema type is verified here, so no need to check id anywhere later ! (FE .. creation)
	var taskRequest SubmitSensorTaskRequest
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

func (sensor *Sensor) setGroupEndpoint(c *gin.Context) (ResponseType, int, any) {
	var data struct {
		GroupId UUID `json:"id"`
	}

	if err := c.BindJSON(&data); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// assert that Sensor doesn't already have GroupId
	if sensor.GroupId.IsNil() {
		sensor.GroupId = data.GroupId
		msg := fmt.Sprintf("group %s set successfully", data.GroupId)
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else if sensor.GroupId == data.GroupId {
		msg := fmt.Sprintf("group %s is already set", sensor.GroupId)
		sensor.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("could not set group %s, as group %s is already set", data.GroupId, sensor.GroupId)
	}
}

func (sensor *Sensor) registerSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	// assert that the Server is already set
	if sensor.Server == nil {
		return ErrorResponse, http.StatusBadRequest, "server must be set before sensor registration"
	}

	// assert that the GroupId is already set
	if sensor.GroupId.IsNil() {
		return ErrorResponse, http.StatusBadRequest, "group must be set before sensor registration"
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
		{"POST", "/group", sensor.setGroupEndpoint},
		{"POST", "/task", sensor.submitTaskEndpoint},
		{"GET", "/register", sensor.registerSensorEndpoint},
		{"GET", "/task/:id/samples", sensor.getSamplesEndpoint},
	}
}
