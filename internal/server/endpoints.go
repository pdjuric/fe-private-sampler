package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// addSensorEndpoint creates new Sensor (or fetches existing) and adds it to the specified Group
// @endpoint /groups/:id/sensors [POST]
func (server *Server) addSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	//region param parsing
	groupIdString := c.Param("id")

	var body RegisterSensorRequest
	if err := c.BindJSON(&body); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}
	//endregion

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	if !body.SensorId.Verify() {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("invalid uuid %s", body.SensorId)
	}

	server.AddSensorToGroup(body.SensorId, body.IP, group)

	return NoResponse, http.StatusNoContent, nil
}

func (server *Server) removeSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	//region param parsing
	groupIdString := c.Param("id")

	var body struct {
		SensorId string `json:"sensorId"`
	}

	if err := c.BindJSON(&body); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}
	//endregion

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	sensorId, err := NewUUIDFromString(body.SensorId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// todo move to server
	sensor, exists := server.sensors.Load(sensorId)
	if !exists {
		//todo return error
	}

	// ongoing tasks won't be affected!
	sensor.(*Sensor).RemoveFromGroup(group)

	// todo what should be the response
	return NoResponse, http.StatusNoContent, nil
}

func (server *Server) createGroupEndpoint(c *gin.Context) (ResponseType, int, any) {
	group := server.AddGroup()

	// return group uuid
	server.HttpLogger.Info("created group %s [%s]", group.Uuid, group)
	return JSONResponse, http.StatusCreated, gin.H{"id": group.Uuid}
}

// POST /group/:id
func (server *Server) getGroupDetailsEndpoint(c *gin.Context) (ResponseType, int, any) {
	//region param parsing
	groupIdString := c.Param("id")
	//endregion

	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return JSONResponse, http.StatusOK, group
}

// GET /group/:id/lock
func (server *Server) lockGroupEndpoint(c *gin.Context) (ResponseType, int, any) {
	groupIdString := c.Param("id")
	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group.Lock()

	return NoResponse, http.StatusNoContent, nil
}

func (server *Server) unlockGroupEndpoint(c *gin.Context) (ResponseType, int, any) {
	groupIdString := c.Param("id")
	groupId, err := NewUUIDFromString(groupIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group, err := server.GetGroup(groupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	group.Unlock()

	return NoResponse, http.StatusNoContent, nil
}

// addTaskEndpoint creates a new Task based on CreateTaskRequest and sends it to the TaskDaemon chan
// @endpoint /group/:id/task [POST]
func (server *Server) addTaskEndpoint(c *gin.Context) (ResponseType, int, any) {

	// todo assert that authority is set
	if !server.IsAuthoritySet() {
		return ErrorResponse, http.StatusBadRequest, "authority must be set before task creation"
	}

	// Parse the JSON data from the request body into the CreateTaskRequest struct
	var taskRequest CreateTaskRequest
	if err := c.BindJSON(&taskRequest); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	//region asserts
	errors := make([]error, 0)

	// there should be exactly SampleCount coefficients
	rate, exists := GetRate(taskRequest.RateId)
	if !exists {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("rate with id %s does not exist", taskRequest.RateId)
	}

	// submission frequency must be a divisor of sample count
	if taskRequest.Duration%(rate.BatchSize*rate.SamplingPeriod) != 0 {
		//todo edit message
		//errors = append(errors, fmt.Errorf("subscription dura...tion batch size must be a divisor of sample count (%d mod %d != 0)", taskRequest.SampleCount, taskRequest.BatchSize))
	}

	if taskRequest.Duration/(rate.SamplingPeriod*rate.BatchSize) == 0 {

	}

	// todo assert that maxvalue fits in int64
	// todo assert >=1 period
	// todo assert SampleCount > 0
	// todo assert t.BatchSize > 0
	// todo assert start is in the future

	if len(errors) > 0 {
		return ErrorResponse, http.StatusBadRequest, errors
	}

	//endregion

	// get GroupId
	group, err := server.GetGroup(taskRequest.GroupId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// create new Task

	task := server.NewTask(taskRequest)
	if err = task.SetSensors(group); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// send task to TaskDaemon
	server.AddTask(task)
	server.SendTaskToDaemon(task)
	server.HttpLogger.Info("task %s sent to task daemon", task.Id)

	return StringResponse, http.StatusAccepted, string(task.Id)
}

func (server *Server) addRateEndpoint(c *gin.Context) (ResponseType, int, any) {
	rate := NewRate()

	if err := c.BindJSON(&rate); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	SaveRate(rate)

	return StringResponse, http.StatusAccepted, string(rate.id)
}

func (server *Server) removeTaskEndpoint(c *gin.Context) (ResponseType, int, any) {
	return ErrorResponse, http.StatusInternalServerError, "not implemented"
}

func (server *Server) getTaskDetailsEndpoint(c *gin.Context) (ResponseType, int, any) {
	// todo
	//    returns task status
	//    how many submissions from each server occurred, and when is the next submission

	// get task uuid
	taskIdString := c.Param("id")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := server.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
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
	}{
		TaskId:         task.Id,
		GroupId:        task.GroupId,
		Sensors:        make([]sensorInfo, len(task.Sensors)),
		SamplingParams: task.SamplingParams,
		BatchParams:    task.BatchParams,
	}

	response.ResultReady = task.Result != nil
	if response.ResultReady {
		response.Result = task.Result.Int64()
	}

	for idx, sensor := range task.Sensors {
		response.Sensors[idx] = sensorInfo{
			Ids:             sensor.Id,
			SubmittedTask:   task.SubmittedTaskFlags[idx].Load(),
			SubmittedCipher: task.SubmittedCipherCnts[idx].Load(),
		}
	}

	return JSONResponse, http.StatusOK, response
}

func (server *Server) submitCipherEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := server.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	//// todo assert that remote ip is the ip of the sensor
	// asssert sensor id

	// get FECipher
	bytes, err := c.GetRawData()
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	feCipher, err := Decode(bytes)

	// won't return any errors, we'll need to check for errors
	go task.AddCipher(feCipher)

	return NoResponse, http.StatusAccepted, nil
}

func (server *Server) setAuthorityEndpoint(c *gin.Context) (ResponseType, int, any) {

	var ip IP
	if err := c.BindJSON(&ip); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	newAuthority := server.NewAuthority(ip)

	// assert that Sensor doesn't already have Server
	if server.Authority == nil {
		server.Authority = newAuthority
		msg := fmt.Sprintf("authority %s set successfully", ip)
		server.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else if server.Authority.IP.String() == newAuthority.IP.String() {
		msg := fmt.Sprintf("authority %s is already set", server.Authority.IP.String())
		server.HttpLogger.Info(msg)
		return StringResponse, http.StatusOK, msg

	} else {
		return ErrorResponse, http.StatusBadRequest, fmt.Errorf("could not set authority %s, as authority %s is already set", newAuthority.IP.String(), server.Authority.IP.String())
	}

}

func (server *Server) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"POST", "/group", server.createGroupEndpoint},
		{"GET", "/group/:id", server.getGroupDetailsEndpoint},

		//{"GET", "/group/:id/lock", server.lockGroupEndpoint},
		//{"GET", "/group/:id/unlock", server.unlockGroupEndpoint},

		{"POST", "/group/:id/sensor", server.addSensorEndpoint},
		{"DELETE", "/group/:id/sensor", server.removeSensorEndpoint},

		{"POST", "/task", server.addTaskEndpoint},
		{"DELETE", "/task/:id", server.removeTaskEndpoint},
		{"GET", "/task/:id", server.getTaskDetailsEndpoint},
		{"POST", "/task/:taskId/:sensorId", server.submitCipherEndpoint},

		{"POST", "/authority", server.setAuthorityEndpoint},
		{"POST", "/rate", server.addRateEndpoint},
	}
}
