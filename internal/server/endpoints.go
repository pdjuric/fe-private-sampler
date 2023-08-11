package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"github.com/gin-gonic/gin"
	"net/http"
)

// addSensorEndpoint creates new Sensor (or fetches existing) and adds it to the specified Group
// @endpoint /groups/:id/sensors [POST]
func (server *Server) addSensorEndpoint(c *gin.Context) {
	//region param parsing
	groupUuidString := c.Param("id")

	var body RegisterSensorRequest
	if err := c.BindJSON(&body); err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	//endregion

	groupUuid, logErr, _ := ParseUuid(groupUuidString)
	if logErr != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group uuid"})
		return
	}

	group, err := server.GetGroup(*groupUuid)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	sensorUuid, logErr, err := ParseUuid(body.SensorId)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		//todo or this c.AbortWithError(http.StatusBadRequest, err) ???
		return
	}

	server.AddSensorToGroup(*sensorUuid, body.IP, group)

	// todo what should be the response
	c.Status(http.StatusNoContent)
}

func (server *Server) removeSensorEndpoint(c *gin.Context) {
	//region param parsing
	groupUuidString := c.Param("id")

	var body struct {
		SensorId string `json:"sensorId"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	//endregion

	groupUuid, logErr, err := ParseUuid(groupUuidString)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := server.GetGroup(*groupUuid)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sensorUuid, logErr, err := ParseUuid(body.SensorId)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// todo move to server
	sensor, exists := server.sensors.Load(*sensorUuid)
	if !exists {
		//todo return error
	}

	// ongoing tasks won't be affected!
	sensor.(*Sensor).RemoveFromGroup(group)

	// todo what should be the response
	c.Status(http.StatusNoContent)
}

func (server *Server) createGroupEndpoint(ctxt *gin.Context) {
	group := server.AddGroup()

	// return group uuid
	server.HttpLogger.Info("created group %s [%s]", group.Uuid, group)
	ctxt.JSON(http.StatusOK, gin.H{"id": group.Uuid})
}

// POST /group/:id
func (server *Server) getGroupDetailsEndpoint(c *gin.Context) {
	//region param parsing
	uuidString := c.Param("id")
	//endregion

	uuid, logErr, err := ParseUuid(uuidString)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(*uuid)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, group)
}

// GET /group/:id/lock
func (server *Server) lockGroupEndpoint(c *gin.Context) {
	uuidString := c.Param("id")
	uuid, logErr, err := ParseUuid(uuidString)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(*uuid)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group.Lock()

	c.Status(http.StatusNoContent)
}

func (server *Server) unlockGroupEndpoint(c *gin.Context) {
	uuidString := c.Param("id")
	uuid, logErr, err := ParseUuid(uuidString)
	if err != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := server.GetGroup(*uuid)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group.Unlock()

	c.Status(http.StatusNoContent)
}

// addTaskEndpoint creates a new Task based on CreateTaskRequest and sends it to the TaskDaemon chan
// @endpoint /group/:id/task [POST]
func (server *Server) addTaskEndpoint(c *gin.Context) {

	// Parse the JSON data from the request body into the CreateTaskRequest struct
	var taskRequest CreateTaskRequest
	if err := c.BindJSON(&taskRequest); err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	//region asserts
	errors := make([]error, 0)

	// there should be exactly SampleCount coefficients
	if len(taskRequest.CoefficientsByPeriod) != taskRequest.SampleCount {
		errors = append(errors, fmt.Errorf("there should be exactly SampleCount coefficients (expected %d, got %d)", taskRequest.SampleCount, len(taskRequest.CoefficientsByPeriod)))
	}

	// submission frequency must be a divisor of sample count
	if taskRequest.SampleCount%taskRequest.BatchSize != 0 {
		errors = append(errors, fmt.Errorf("batch size must be a divisor of sample count (%d mod %d != 0)", taskRequest.SampleCount, taskRequest.BatchSize))
	}

	// todo assert >=1 period
	// todo assert start is in the future

	if len(errors) > 0 {
		for _, err := range errors {
			server.HttpLogger.Err(err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errors})
		return
	}

	//endregion

	// get Group
	group, err := server.GetGroup(taskRequest.GroupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// create new Task
	task := NewTaskFromTaskRequest(taskRequest)
	if err = task.SetSensors(group); err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// send task to TaskDaemon
	server.AddTask(task)
	server.HttpLogger.Info("task %s sent to task daemon", task.Uuid)

	c.String(http.StatusAccepted, "task %s added", task.Uuid)
}

func (server *Server) removeTaskEndpoint(c *gin.Context) {
	// todo
}

func (server *Server) getTaskDetailsEndpoint(c *gin.Context) {
	// todo
	//    returns task status
	//    how many submissions from each server occurred, and when is the next submission
}

func (server *Server) submitEncryptedBatchEndpoint(c *gin.Context) {

	// get task uuid
	taskUuidString := c.Param("id")
	taskUuid, logErr, _ := ParseUuid(taskUuidString)
	if logErr != nil {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task uuid"})
		return
	}

	// get task
	taskAny, exists := server.tasks.Load(taskUuid.String())
	if !exists {
		server.HttpLogger.Err(logErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "task does not exist"}) // todo add uuid
		return
	}

	task := taskAny.(*Task)

	if task.decryptionKey == nil {
		server.HttpLogger.Error("decryption key is not derived")
		// todo derive key now
	}

	// fixme do this in a separate goroutine

	switch task.GetSchemaName() {
	case SchemaFHIPE:

		// deserialize cipher
		cipher := new(SingleFECipher)
		err := c.BindJSON(cipher)
		if err != nil {
			server.HttpLogger.Err(err)
			return
		}

		// generate schema
		schema := fullysec.NewFHIPEFromParams(task.FEParams.(*SingleFEParams).Params)

		res, err := schema.Decrypt(*cipher, task.decryptionKey.(SingleFEDecryptionKey))
		if err != nil {
			fmt.Print(err)
			return
		}

		task.result = res
		fmt.Printf("task %s result %s\n", task.Uuid, res.String())

	case SchemaFHMultiIPE:
		cipher := new(MultiFECipher)
		err := c.BindJSON(cipher)
		if err != nil {
			server.HttpLogger.Err(err)
			return
		}

		// todo complete

	}

}

func (server *Server) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"POST", "/group", server.createGroupEndpoint},
		{"GET", "/group/:id", server.getGroupDetailsEndpoint},

		{"GET", "/group/:id/lock", server.lockGroupEndpoint},
		{"GET", "/group/:id/unlock", server.unlockGroupEndpoint},

		{"POST", "/group/:id/sensor", server.addSensorEndpoint},
		{"DELETE", "/group/:id/sensor", server.removeSensorEndpoint},

		{"POST", "/task", server.addTaskEndpoint},
		{"DELETE", "/task/:id", server.removeTaskEndpoint},
		{"GET", "/task/:id", server.getTaskDetailsEndpoint},
		{"POST", "/task/:id/data", server.submitEncryptedBatchEndpoint},
	}
}
