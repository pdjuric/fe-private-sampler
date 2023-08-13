package sensor

import (
	"encoding/json"
	. "fe/internal/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

//todo do i need to replace any more maps with sync.Map

// POST /task
// body: groupId, start, measuringPeriod, submittingPeriod
func (sensor *Sensor) submitTaskEndpoint(c *gin.Context) {
	// get JSON from request body; can't use BindJSON as we're unmarshalling twice!
	jsonData, _ := c.GetRawData()

	// Parse SubmitTaskRequest
	// schema type is verified here, so no need to check id anywhere later ! (FE .. creation)
	var taskRequest SubmitTaskRequest
	err := json.Unmarshal(jsonData, &taskRequest)
	if err != nil {
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	task := sensor.NewTask(&taskRequest)

	sensor.AddTask(task)
	sensor.SendTaskToDaemon(task)

	msg := fmt.Sprintf("task %s added successfully", task.Id)
	sensor.HttpLogger.Info(msg)
	c.String(http.StatusAccepted, msg)
}

func (sensor *Sensor) setServerEndpoint(c *gin.Context) {
	var ip IP

	if err := c.BindJSON(&ip); err != nil {
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	newServer := sensor.NewServer(ip)

	// assert that Sensor doesn't already have Server
	if sensor.Server == nil {
		sensor.Server = newServer
		msg := fmt.Sprintf("server %s set successfully", ip)
		sensor.HttpLogger.Info(msg)
		c.String(http.StatusOK, msg)

	} else if sensor.Server.IP.String() == newServer.IP.String() {
		msg := fmt.Sprintf("server %s is already set", sensor.Server.IP.String())
		sensor.HttpLogger.Info(msg)
		c.String(http.StatusOK, msg)

	} else {
		err := fmt.Errorf("could not set server %s, as server %s is already set", newServer.IP.String(), sensor.Server.IP.String())
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}

}

func (sensor *Sensor) setGroupEndpoint(c *gin.Context) {
	var data struct {
		GroupId UUID `json:"id"`
	}

	if err := c.BindJSON(&data); err != nil {
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// assert that Sensor doesn't already have GroupId
	if sensor.GroupId.IsNil() {
		sensor.GroupId = data.GroupId
		msg := fmt.Sprintf("group %s set successfully", data.GroupId)
		sensor.HttpLogger.Info(msg)
		c.String(http.StatusOK, msg)

	} else if sensor.GroupId == data.GroupId {
		msg := fmt.Sprintf("group %s is already set", sensor.GroupId)
		sensor.HttpLogger.Info(msg)
		c.String(http.StatusOK, msg)

	} else {
		err := fmt.Errorf("could not set group %s, as group %s is already set", data.GroupId, sensor.GroupId)
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
	}
}

func (sensor *Sensor) registerSensorEndpoint(c *gin.Context) {
	// assert that the Server is already set
	if sensor.Server == nil {
		err := fmt.Errorf("server must be set before sensor registration")
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// assert that the GroupId is already set
	if sensor.GroupId.IsNil() {
		err := fmt.Errorf("group must be set before sensor registration")
		sensor.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	sensor.Server.Register(sensor)
	c.String(200, "??")
}

func (sensor *Sensor) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"POST", "/server", sensor.setServerEndpoint},
		{"POST", "/group", sensor.setGroupEndpoint},
		{"POST", "/task", sensor.submitTaskEndpoint},
		{"GET", "/register", sensor.registerSensorEndpoint},
	}
}
