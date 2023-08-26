package server

import (
	"encoding/json"
	. "fe/common"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Task struct {
	Id        UUID
	Status    string
	Sensors   []*Sensor
	Authority *Authority

	// creation parameters
	CustomerId UUID
	SamplingParams

	Tariff *Tariff
	Rates  []int

	DecryptionParamsId UUID

	EncryptionEnabled bool
	feDecryptor       FEDecryptor

	Result *big.Int

	// status flags
	schemaParamsFetched     atomic.Bool
	submittedToSensors      []atomic.Bool
	ratesSubmittedCnt       atomic.Int32
	decryptionParamsFetched atomic.Bool
	ciphersReceived         atomic.Int32
	// total ciphers?

	decryptionParamsFetchedChan chan bool // when the key is derived, this channel will be closed

	logger *Logger
}

// NewTask creates a new Task from common.ServerTaskRequest
func (server *Server) NewTask(taskRequest ServerTaskRequest) *Task {
	id := NewUUID()
	tariff, _ := GetTariff(taskRequest.TariffId)
	task := &Task{
		Id:         id,
		Status:     "created",
		CustomerId: taskRequest.CustomerId,
		Authority:  server.Authority,

		SamplingParams: SamplingParams{
			Start:          taskRequest.Start,
			SamplingPeriod: tariff.SamplingPeriod,
			BatchParams: BatchParams{
				BatchSize: tariff.BatchSize,
				BatchCnt:  taskRequest.Duration / (tariff.SamplingPeriod * tariff.BatchSize), // this is number of batches per sensor !!
			},
			MaxSampleValue: tariff.MaxSampleValue,
		},
		EncryptionEnabled: taskRequest.EnableEncryption,

		decryptionParamsFetchedChan: make(chan bool, 1),
		Tariff:                      tariff,
		logger:                      GetLoggerForFile("", string(id)),
	}
	taskRequestJson, _ := json.MarshalIndent(taskRequest, "", "  ")
	task.logger.Info("Task params: %s", string(taskRequestJson))
	return task
}

func (t *Task) getSensorIdx(sensorId UUID) (int, error) {
	for idx, sensor := range t.Sensors {
		if sensor.Id == sensorId {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("sensor %s not found in task %s", sensorId, t.Id)
}

// SetSensors sets Sensors (from provided Customer) for Task, and calculates vectorLen and vectorCnt for the Task based on the number of  samplesPerSubmission
func (t *Task) SetSensors(g *Customer) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// check if there are any sensors at all
	if len(g.Sensors) == 0 {
		err := fmt.Errorf("no sensors in customer %s; tasks can be created for customers with at least one server", g.Uuid)
		t.logger.Err(err)
		return err
	}

	sensorCnt := len(g.Sensors)
	t.logger.Info("setting sensors for task %s", t.Id)
	t.submittedToSensors = make([]atomic.Bool, sensorCnt)
	t.Sensors = make([]*Sensor, sensorCnt)
	copy(t.Sensors, g.Sensors)

	t.Status = "sensors set"
	return nil
}

// SubmitToSensors sends the SensorTaskRequest to all Sensors in the Task's Customer (captured during SetSensors)
func (t *Task) SubmitToSensors() bool {
	// todo add parallel execution
	for idx, sensor := range t.Sensors {
		// assert that it isn't already sent

		t.logger.Info("submitting task to sensor %s", sensor.Id)

		statusCode, _, err := sensor.SubmitTask(t.Id, t.SamplingParams, t.Authority.IP)
		if err != nil {
			t.logger.Err(err)
			t.logger.Error("submission to sensor %s failed", sensor.Id)
			return false
		}

		if statusCode != http.StatusAccepted {
			t.logger.Error("submission to sensor %s failed", sensor.Id)
			return false
		}

		t.submittedToSensors[idx].Store(true)
		t.logger.Info("task submitted to sensor %s", sensor.Id)
		// todo check whether start time has already passed
	}
	return true
}

func (t *Task) GetFESchemaParams() bool {
	sensorIds := make([]UUID, len(t.Sensors))
	for idx, sensor := range t.Sensors {
		sensorIds[idx] = sensor.Id
	}

	t.logger.Info("submitting task to authority")
	err := t.Authority.SubmitTask(t.Id, sensorIds, t.BatchParams, t.Tariff.MaxTariffValue, t.MaxSampleValue, t.EncryptionEnabled)
	if err != nil {
		t.logger.Err(err)
		return false
	}

	for {
		time.Sleep(SchemaParamsPollingInterval)
		status, err := t.Authority.FetchSchemaParamsStatus(t.Id)
		*status = strings.Replace(*status, "\"", "", -1)
		switch *status {
		case StatusCreated:
			t.logger.Info("fe params not yet ready, polling again in %d ns", SchemaParamsPollingInterval.Nanoseconds())
			continue
		case StatusError, StatusInvalid:
			t.logger.Err(err)
			return false
		case StatusReady:
			t.schemaParamsFetched.Store(true)
			t.logger.Info("fe params ready")
			return true
		}
	}
}

func (t *Task) DeriveDecryptionKey() {
	var decryptionParamsId UUID
	decryptionParamsId, ok := t.SendRates()
	if !ok {
		return
	}

	for {
		time.Sleep(DecryptionParamsPollingInterval)
		status, err := t.Authority.FetchDecryptionParamsStatus(t.Id, decryptionParamsId)
		*status = strings.Replace(*status, "\"", "", -1)
		switch *status {
		case StatusCreated:
			t.logger.Info("fe decryption params not yet ready, polling again in %d ns", DecryptionParamsPollingInterval.Nanoseconds())
			continue
		case StatusError:
			t.logger.Err(err)
			return
		case StatusReady:
			t.logger.Info("fe decryption params ready")
			t.logger.Info("fetching fe decryption params")
			decryptionParams, err := t.Authority.FetchDecryptionParams(t.Id, decryptionParamsId)
			if err != nil {
				t.logger.Err(err)
				return
			}

			t.feDecryptor, err = NewFEDecryptor(decryptionParams, t.logger)
			if err != nil {
				t.logger.Err(err)
				return
			}

			t.decryptionParamsFetched.Store(true)
			close(t.decryptionParamsFetchedChan)
			t.logger.Info("fe decryption params fetched")
			return
		case StatusInvalid:
			t.logger.Info("rates invalid, regenerating")
			decryptionParamsId, ok = t.SendRates()
		}
	}
}

func (t *Task) SendRates() (UUID, bool) {
	rates, err := t.Tariff.GenerateRates(t.BatchCnt)
	if err != nil {
		t.logger.Err(err)
		return "", false
	}

	t.logger.Debug("generated rates: %v", rates)
	t.logger.Info("sending rates")
	decryptionParamsId, err := t.Authority.SendRates(t.Id, rates)
	if err != nil {
		t.logger.Err(err)
		t.logger.Error("sending rates failed")
		return "", false
	}
	// todo handle error

	t.logger.Info("rates sent successfully")
	t.ratesSubmittedCnt.Add(1)
	return decryptionParamsId, true
}

// potentially blocking method, should be done in goroutine
func (t *Task) AddCipher(feCipher FECipher) {
	_, opened := <-t.decryptionParamsFetchedChan

	if opened {
		t.logger.Err(fmt.Errorf("key derived channel nas NOT closed"))
		return
	}

	t.logger.Info("adding cipher")
	result, err := t.feDecryptor.AddCipher(feCipher)
	if err != nil {
		t.logger.Err(err)
	}

	if result != nil {
		t.Result = result
		t.logger.Debug("result: %d", result)
	}
}
