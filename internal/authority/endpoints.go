package authority

import (
	. "fe/internal/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (authority *Authority) addTaskEndpoint(c *gin.Context) (ResponseType, int, any) {

	var taskRequest CreateAuthorityTaskRequest
	if err := c.BindJSON(&taskRequest); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	task := NewTaskFromTaskRequest(taskRequest)

	// send task to TaskDaemon
	authority.AddTask(task)

	ok := task.SetFEParams()
	if !ok {
		return ErrorResponse, http.StatusBadRequest, "failed to generate FE params"
	}

	data, err := Encode(task.FEParams.GetFESchemaParams())
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return DataResponse, http.StatusAccepted, data
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

func (authority *Authority) addCoefficientsForDecryptionEndpoint(c *gin.Context) (ResponseType, int, any) {

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

	coeffBytes, err := c.GetRawData()
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	coeffs, err := Decode(coeffBytes)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	fmt.Printf("coeffs: %v\n", coeffs)
	decryptionParamsId, err := task.AddNewDecryptionParams(coeffs.([]int))
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

func (authority *Authority) getUnapprovedCoeffsEndpoint(c *gin.Context) (ResponseType, int, any) {
	return ErrorResponse, http.StatusInternalServerError, "not implemented"
}

func (authority *Authority) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"GET", "/decryption/:taskId/:decryptionParamsId", authority.getDecryptionParamsEndpoint},
		{"GET", "/decryption-status/:taskId/:decryptionParamsId", authority.getDecryptionParamsStatusEndpoint},
		{"POST", "/decryption/:taskId", authority.addCoefficientsForDecryptionEndpoint},
		{"POST", "/task", authority.addTaskEndpoint},
		{"GET", "/approval", authority.getUnapprovedCoeffsEndpoint},
		//{"[POST]", "/approval/:id", authority.approveCoeffsEndpoint},
		{"GET", "/encryption/:taskId/:sensorId", authority.getEncryptionParamsEndpoint},
	}
}
