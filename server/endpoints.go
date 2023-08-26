package server

import (
	. "fe/common"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

//region SENSOR endpoints

// addSensorEndpoint creates new Sensor (or fetches existing) and adds it to the specified Customer.
//
// endpoint: [POST] /customers/:id/sensors
func (server *Server) addSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	//region param parsing
	customerIdString := c.Param("id")

	var body RegisterSensorRequest
	if err := c.BindJSON(&body); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}
	//endregion

	customerId, err := NewUUIDFromString(customerIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer, err := server.GetCustomer(customerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	if !body.SensorId.Verify() {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("invalid uuid %s", body.SensorId)
	}

	server.AddSensorToCustomer(body.SensorId, body.IP, customer)

	return NoResponse, http.StatusNoContent, nil
}

// endpoint: [DELETE] /group/:id/sensor
func (server *Server) removeSensorEndpoint(c *gin.Context) (ResponseType, int, any) {
	//region param parsing
	customerIdString := c.Param("id")

	var body struct {
		SensorId string `json:"sensorId"`
	}

	if err := c.BindJSON(&body); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}
	//endregion

	customerId, err := NewUUIDFromString(customerIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer, err := server.GetCustomer(customerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	sensorId, err := NewUUIDFromString(body.SensorId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	sensor, exists := server.sensors.Load(sensorId)
	if !exists {
		return ErrorResponse, http.StatusBadRequest, "Sensor with the provided id does not exist."
	}

	// ongoing tasks won't be affected!
	sensor.(*Sensor).RemoveFromCustomer(customer)

	return NoResponse, http.StatusNoContent, nil
}

// endpoint: [POST] /task/:taskId/:sensorId
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

//endregion

//region CUSTOMER endpoints

// endpoint: [POST] /customer
func (server *Server) createCustomerEndpoint(c *gin.Context) (ResponseType, int, any) {
	customer := server.AddCustomer()

	// return customer uuid
	server.HttpLogger.Info("created customer %s [%s]", customer.Uuid, customer)
	return JSONResponse, http.StatusCreated, gin.H{"id": customer.Uuid}
}

// endpoint: [GET] /customer/:id
func (server *Server) getCustomerDetailsEndpoint(c *gin.Context) (ResponseType, int, any) {
	customerIdString := c.Param("id")

	customerId, err := NewUUIDFromString(customerIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer, err := server.GetCustomer(customerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	return JSONResponse, http.StatusOK, customer
}

/*// endpoint: [GET] /customer/:id/lock
func (server *Server) lockCustomerEndpoint(c *gin.Context) (ResponseType, int, any) {
	customerIdString := c.Param("id")
	customerId, err := NewUUIDFromString(customerIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer, err := server.GetCustomer(customerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer.Lock()

	return NoResponse, http.StatusNoContent, nil
}

func (server *Server) unlockCustomerEndpoint(c *gin.Context) (ResponseType, int, any) {
	customerIdString := c.Param("id")
	customerId, err := NewUUIDFromString(customerIdString)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer, err := server.GetCustomer(customerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	customer.Unlock()

	return NoResponse, http.StatusNoContent, nil
}*/

//endregion

//region TASK endpoints

// addTaskEndpoint creates a new Task based on ServerTaskRequest and sends it to the TaskDaemon chan
//
// endpoint: [POST] /customer/:id/task
func (server *Server) addTaskEndpoint(c *gin.Context) (ResponseType, int, any) {

	if !server.IsAuthoritySet() {
		return ErrorResponse, http.StatusBadRequest, "authority must be set before task creation"
	}

	// Parse the JSON data from the request body into the ServerTaskRequest struct
	var taskRequest ServerTaskRequest
	if err := c.BindJSON(&taskRequest); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	//region asserts
	errors := make([]error, 0)

	// there should be exactly SampleCount rates
	tariff, exists := GetTariff(taskRequest.TariffId)
	if !exists {
		return ErrorResponse, http.StatusBadRequest, fmt.Sprintf("tariff with id %s does not exist", taskRequest.TariffId)
	}

	// submission frequency must be a divisor of sample count
	if taskRequest.Duration%(tariff.BatchSize*tariff.SamplingPeriod) != 0 || taskRequest.Duration/(tariff.SamplingPeriod*tariff.BatchSize) == 0 {
		return ErrorResponse, http.StatusBadRequest, "subscription duration must be a multiple of the time needed to generate one batch"
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

	// get CustomerId
	customer, err := server.GetCustomer(taskRequest.CustomerId)
	if err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// create new Task
	task := server.NewTask(taskRequest)
	if err = task.SetSensors(customer); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	// send task to TaskDaemon
	server.AddTask(task)
	server.SendTaskToDaemon(task)
	server.HttpLogger.Info("task %s sent to task daemon", task.Id)

	return StringResponse, http.StatusAccepted, string(task.Id)
}

// endpoint: [DELETE] /task/:id
func (server *Server) removeTaskEndpoint(c *gin.Context) (ResponseType, int, any) {
	return ErrorResponse, http.StatusInternalServerError, "not implemented"
}

// endpoint: [GET] /task/:id
func (server *Server) getTaskDetailsEndpoint(c *gin.Context) (ResponseType, int, any) {
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
		Id            UUID `json:"id"`
		SubmittedTask bool `json:"task_submitted"`
	}

	response := struct {
		TaskId     UUID         `json:"task_id"`
		CustomerId UUID         `json:"customer_id"`
		Sensors    []sensorInfo `json:"sensors"`
		SamplingParams

		DecryptorStats any   `json:"decryptor_stats"`
		Result         int64 `json:"result"`
	}{
		TaskId:         task.Id,
		CustomerId:     task.CustomerId,
		Sensors:        make([]sensorInfo, len(task.Sensors)),
		SamplingParams: task.SamplingParams,
	}

	if task.feDecryptor != nil {
		response.DecryptorStats = task.feDecryptor.GetStats()
	}

	if task.Result != nil {
		response.Result = task.Result.Int64()
	}

	for idx, sensor := range task.Sensors {
		response.Sensors[idx] = sensorInfo{
			Id:            sensor.Id,
			SubmittedTask: task.submittedToSensors[idx].Load(),
		}
	}

	return JSONResponse, http.StatusOK, response
}

//endregion

//region RATE endpoints

func (server *Server) addTariffEndpoint(c *gin.Context) (ResponseType, int, any) {
	tariff := NewTariff()

	if err := c.BindJSON(&tariff); err != nil {
		return ErrorResponse, http.StatusBadRequest, err
	}

	SaveTariff(tariff)

	return StringResponse, http.StatusAccepted, string(tariff.id)
}

//endregion

//region AUTHORITY endpoints

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

//endregion

func (server *Server) GetEndpoints() []Endpoint {
	return []Endpoint{
		{"POST", "/customer", server.createCustomerEndpoint},
		{"GET", "/customer/:id", server.getCustomerDetailsEndpoint},

		//{"GET", "/group/:id/lock", server.lockGroupEndpoint},
		//{"GET", "/group/:id/unlock", server.unlockGroupEndpoint},

		{"POST", "/group/:id/sensor", server.addSensorEndpoint},
		{"DELETE", "/group/:id/sensor", server.removeSensorEndpoint},

		{"POST", "/task", server.addTaskEndpoint},
		{"DELETE", "/task/:id", server.removeTaskEndpoint},
		{"GET", "/task/:id", server.getTaskDetailsEndpoint},
		{"POST", "/task/:taskId/:sensorId", server.submitCipherEndpoint},

		{"POST", "/authority", server.setAuthorityEndpoint},
		{"POST", "/tariff", server.addTariffEndpoint},
	}
}
