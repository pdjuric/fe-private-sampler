package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// addSensorEndpoint creates new Sensor (or fetches existing) and adds it to the specified Group
// @endpoint /groups/:id/sensors [POST]
func (server *Server) addSensorEndpoint(c *gin.Context) {
	//region param parsing
	groupIdString := c.Param("id")

	var body RegisterSensorRequest
	if err := c.BindJSON(&body); err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	//endregion

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	if !body.SensorId.Verify() {
		err = fmt.Errorf("invalid uuid %s", body.SensorId)
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		//todo or this c.AbortWithError(http.StatusBadRequest, err) ???
		return
	}

	server.AddSensorToGroup(body.SensorId, body.IP, group)

	// todo what should be the response
	c.Status(http.StatusNoContent)
}

func (server *Server) removeSensorEndpoint(c *gin.Context) {
	//region param parsing
	groupIdString := c.Param("id")

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

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	sensorId, err := NewUUIDFromString(body.SensorId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// todo move to server
	sensor, exists := server.sensors.Load(sensorId)
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
	groupIdString := c.Param("id")
	//endregion

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, group)
}

// GET /group/:id/lock
func (server *Server) lockGroupEndpoint(c *gin.Context) {
	groupIdString := c.Param("id")
	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group.Lock()

	c.Status(http.StatusNoContent)
}

func (server *Server) unlockGroupEndpoint(c *gin.Context) {
	groupIdString := c.Param("id")
	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
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

	// todo assert that maxvalue fits in int64
	// todo assert >=1 period
	// todo assert SampleCount > 0
	// todo assert t.BatchSize > 0
	// todo assert start is in the future

	if len(errors) > 0 {
		for _, err := range errors {
			server.HttpLogger.Err(err)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errors})
		return
	}

	//endregion

	// get GroupId
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
	server.SendTaskToDaemon(task)
	server.HttpLogger.Info("task %s sent to task daemon", task.Id)

	c.String(http.StatusAccepted, "%s", task.Id)
}

func (server *Server) removeTaskEndpoint(c *gin.Context) {
	// todo
}

func (server *Server) getTaskDetailsEndpoint(c *gin.Context) {
	// todo
	//    returns task status
	//    how many submissions from each server occurred, and when is the next submission

	// get task uuid
	taskIdString := c.Param("id")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task uuid"})
		return
	}

	// get task
	task, err := server.GetTask(taskId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	type sensorInfo struct {
		Ids             UUID  `json:"ids"`
		SubmittedTask   bool  `json:"submittedTask"`
		SubmittedCipher int32 `json:"submittedCiphers"`
	}

	response := struct {
		TaskId  UUID         `json:"taskId"`
		GroupId UUID         `json:"groupId"`
		Sensors []sensorInfo `json:"sensors"`
		SamplingParams
		BatchParams

		Result      int64 `json:"result"`
		ResultReady bool  `json:"resultReady"`

		Mgt string `json:"masterSecretKeyGenTime"`
		Dt  string `json:"decryptionTime"`
	}{
		TaskId:         task.Id,
		GroupId:        task.GroupId,
		Sensors:        make([]sensorInfo, len(task.Sensors)),
		SamplingParams: task.SamplingParams,
		BatchParams:    task.BatchParams,

		Mgt: fmt.Sprintf("%d ms", task.MasterSecKeyGenerationTime.Milliseconds()),
		Dt:  fmt.Sprintf("%d ms", task.DecryptionTime.Milliseconds()),
	}

	response.ResultReady = task.Result != nil
	if task.Result != nil {
		response.Result = task.Result.Int64()
	}

	for idx, sensor := range task.Sensors {
		response.Sensors[idx] = sensorInfo{
			Ids:             sensor.Id,
			SubmittedTask:   task.SubmittedTaskFlags[idx].Load(),
			SubmittedCipher: task.SubmittedCipherCnts[idx].Load(),
		}
	}

	c.JSON(http.StatusOK, response)
}

func (server *Server) submitCipherEndpoint(c *gin.Context) {

	// get task uuid
	taskIdString := c.Param("id")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task uuid"})
		return
	}

	// get task
	task, err := server.GetTask(taskId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	if task.FEDecryptionParams == nil {
		server.HttpLogger.Error("decryption key is not derived")
		c.JSON(http.StatusTooEarly, gin.H{"error": "try again later"})
		return
	}

	// todo assert that remote ip is the ip of the sensor

	// fixme do this in a separate goroutine

	// does not unmarshall cipher, as there's no way of knowing the type of the cipher
	request := new(SubmitCipherRequest)
	bytes, err := c.GetRawData()
	if err != nil {
		server.HttpLogger.Err(err)
		c.Status(http.StatusInternalServerError)
		return
	}

	err = request.UnmarshalJSON(task.GetSchemaName(), bytes)
	if err != nil {
		server.HttpLogger.Err(err)
		return
	}

	// assert that sensor works for the task
	sensorIdx, err := task.getSensorIdx(request.SensorId)
	if err != nil {
		server.HttpLogger.Err(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	switch task.GetSchemaName() {
	case SchemaFHIPE:

		cipher := request.Cipher.(*SingleFECipher)

		// generate schema
		schema := fullysec.NewFHIPEFromParams(task.FEParams.(*SingleFEParams).Params)

		start := time.Now()
		res, err := schema.Decrypt(cipher, task.FEDecryptionParams.(*SingleFEDecryptionParams).DecryptionKey)
		task.DecryptionTime = time.Since(start)
		if err != nil {
			fmt.Print(err)
			return
		}

		task.SubmittedCipherCnts[sensorIdx].Add(1)
		task.Result = res
		fmt.Printf("task %s Result %s\n", task.Id, res.String())

	case SchemaFHMultiIPE:

		cipher := request.Cipher.(*MultiFECipher)

		// generate schema
		decryptionParams := task.FEDecryptionParams.(*MultiFEDecryptionParams)
		u := decryptionParams.FHMultiIPEParallelDecryption
		key := decryptionParams.DecryptionKey

		start := time.Now()
		remainingBatches, err := u.ParallelDecryption(request.BatchIdx, *cipher, *key)
		fmt.Printf("sensor no %d, batch no %d, time %d ms", sensorIdx, request.BatchIdx, time.Since(start).Milliseconds())
		if err != nil {
			fmt.Print(err)
			return
		}

		task.SubmittedCipherCnts[sensorIdx].Add(1)

		if remainingBatches == 0 {
			start := time.Now()

			result, err := u.GetResult(false, decryptionParams.PubKey)
			task.DecryptionTime = time.Since(start)
			if err != nil {
				fmt.Printf("wtf")
				fmt.Print(err)
				return
			}
			task.Result = result
			task.logger.Info("task %s Result %s", task.Id, result.String())
			fmt.Printf("task %s Result %s\n", task.Id, result.String())
		}
	}

	c.Status(http.StatusAccepted)

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
		{"POST", "/task/:id/data", server.submitCipherEndpoint},
	}
}
