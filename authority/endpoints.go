package authority

import (
	. "fe/common"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (authority *Authority) addTaskEndpoint(c *gin.Context) (ResponseType, int, any) {

	var taskRequest AuthorityTaskRequest
	if err := c.BindJSON(&taskRequest); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// create a task from TaskRequest
	task := NewTask(taskRequest)

	// send task to TaskDaemon
	authority.AddTask(task)
	authority.SendTaskToDaemon(task)

	return NoResponse, http.StatusAccepted, nil
}

func (authority *Authority) getTaskDetailsEndpoint(c *gin.Context) (ResponseType, int, any) {
	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// send task to TaskDaemon
	authority.AddTask(task)
	authority.SendTaskToDaemon(task)

	return NoResponse, http.StatusAccepted, nil
}

func (authority *Authority) getSchemaParamsStatusEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	status := task.GetSchemaParamsStatus()

	return StringResponse, http.StatusOK, status
}

func (authority *Authority) getEncryptionParamsEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	sensorIdString := c.Param("sensorId")
	sensorId, err := NewUUIDFromString(sensorIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid sensor uuid"
	}

	// todo check that enc params are ready
	feEncryptionParams, err := task.GetEncryptionParams(sensorId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	data, err := Encode(feEncryptionParams)
	if err != nil {
		return ErrorResponse, http.StatusInternalServerError, err
	}

	return DataResponse, http.StatusOK, data
}

func (authority *Authority) addRatesEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	ratesBytes, err := c.GetRawData()
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	rates, err := Decode(ratesBytes)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	decryptionParamsId, err := task.AddNewDecryptionParams(rates.([]int))
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return StringResponse, http.StatusAccepted, string(decryptionParamsId)
}

func (authority *Authority) getDecryptionParamsEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// get decryptionKeyId
	decryptionParamsIdString := c.Param("decryptionParamsId")
	decryptionParamsId, err := NewUUIDFromString(decryptionParamsIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid decryption params id"
	}

	decryptionParams, err := task.GetDecryptionParams(decryptionParamsId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	data, err := Encode(decryptionParams)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return DataResponse, http.StatusOK, data
}

func (authority *Authority) getDecryptionParamsStatusEndpoint(c *gin.Context) (ResponseType, int, any) {

	// get task uuid
	taskIdString := c.Param("taskId")
	taskId, err := NewUUIDFromString(taskIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid task uuid"
	}

	// get task
	task, err := authority.GetTask(taskId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// get decryptionKeyId
	decryptionParamsIdString := c.Param("decryptionParamsId")
	decryptionParamsId, err := NewUUIDFromString(decryptionParamsIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, "invalid decryption params uuid"
	}

	status := task.GetDecryptionParamsStatus(decryptionParamsId)

	return JSONResponse, http.StatusOK, status
}

func (authority *Authority) getUnapprovedRatesEndpoint(c *gin.Context) (ResponseType, int, any) {
	return ErrorResponse, http.StatusInternalServerError, "not implemented"
}

func (authority *Authority) GetEndpoints() []Endpoint {
	return []Endpoint{

		{"POST", "/task", authority.addTaskEndpoint},
		//{"GET", "/task/:taskId", authority.getTaskDetailsEndpoint},

		{"POST", "/rates/:taskId", authority.addRatesEndpoint},

		{"GET", "/rates", authority.getUnapprovedRatesEndpoint},
		//{"POST", "/rates-approval/:taskId", authority.getApproveRatesEndpoint},

		{"GET", "/schema-status/:taskId", authority.getSchemaParamsStatusEndpoint},
		{"GET", "/decryption-status/:taskId/:decryptionParamsId", authority.getDecryptionParamsStatusEndpoint},

		{"GET", "/encryption/:taskId/:sensorId", authority.getEncryptionParamsEndpoint},
		{"GET", "/decryption/:taskId/:decryptionParamsId", authority.getDecryptionParamsEndpoint},
	}
}
