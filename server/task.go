package server

import (
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
	GroupId UUID
	SamplingParams

	Rate *Rate

	DecryptionParamsId   UUID
	CoefficientsByPeriod []int
	feDecryptor          FEDecryptor

	Result *big.Int

	// status
	SubmittedTaskFlags  []atomic.Bool
	SubmittedCipherCnts []atomic.Int32
	CoeffsSubmittedCnt  atomic.Int32
	KeyDerivedStatus    atomic.Bool
	keyDerivedChan      chan bool // when the key is derived, this channel will be closed
	logger              *Logger
}

// NewTask creates a new Task from common.CreateTaskRequest
func (server *Server) NewTask(taskRequest CreateTaskRequest) *Task {
	id := NewUUID()
	rate, _ := GetRate(taskRequest.RateId)
	return &Task{
		Id:        id,
		Status:    "created",
		GroupId:   taskRequest.GroupId,
		Authority: server.Authority,

		SamplingParams: SamplingParams{
			Start:          taskRequest.Start,
			SamplingPeriod: rate.SamplingPeriod,
			BatchParams: BatchParams{
				BatchSize: rate.BatchSize,
				BatchCnt:  taskRequest.Duration / (rate.SamplingPeriod * rate.BatchSize), // this is number of batches per sensor !!
			},
			MaxSampleValue: rate.MaxSampleValue,
		},

		keyDerivedChan: make(chan bool, 1),
		Rate:           rate,
		logger:         GetLoggerForFile("", string(id)),
	}
}

func (t *Task) getSensorIdx(sensorId UUID) (int, error) {
	for idx, sensor := range t.Sensors {
		if sensor.Id == sensorId {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("sensor %s not found in task %s", sensorId, t.Id)
}

// SetSensors sets Sensors (from provided Group) for Task, and calculates vectorLen and vectorCnt for the Task based on the number of  samplesPerSubmission
func (t *Task) SetSensors(g *Group) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// check if there are any sensors at all
	if len(g.Sensors) == 0 {
		err := fmt.Errorf("no sensors in group %s; tasks can be created for groups with at least one server", g.Uuid)
		t.logger.Err(err)
		return err
	}

	sensorCnt := len(g.Sensors)
	t.logger.Info("setting sensors for task %s", t.Id)
	t.SubmittedTaskFlags = make([]atomic.Bool, sensorCnt)
	t.SubmittedCipherCnts = make([]atomic.Int32, sensorCnt)
	t.Sensors = make([]*Sensor, sensorCnt)
	copy(t.Sensors, g.Sensors)

	t.Status = "sensors set"
	return nil
}

// Submit sends the SubmitSensorTaskRequest to all Sensors in the Task's Group (captured during SetSensors)
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

		t.SubmittedTaskFlags[idx].Store(true)
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

	feSchemaParams, err := t.Authority.GenerateFESchemaParams(t.Id, sensorIds, t.BatchParams, t.Rate.MaxRateValue, t.MaxSampleValue)
	if err != nil {
		t.logger.Err(err)
		return false
	}

	t.feDecryptor, err = NewFEDecryptor(feSchemaParams)
	return true
}

func (t *Task) DeriveDecryptionKey() {
	t.SendCoeffs()
	for {
		time.Sleep(DecryptionParamsPollingInterval)
		status, err := t.Authority.FetchDecryptionParamsStatus(t.Id, t.DecryptionParamsId)
		*status = strings.Replace(*status, "\"", "", -1)
		switch *status {
		case StatusCreated:
			continue
		case StatusError:
			t.logger.Err(err)
			return
		case StatusReady:
			decryptionParams, err := t.Authority.FetchDecryptionParams(t.Id, t.DecryptionParamsId)
			if err != nil {
				t.logger.Err(err)
				return
			}

			t.feDecryptor.SetDecryptionParams(decryptionParams)

			t.KeyDerivedStatus.Store(true)
			close(t.keyDerivedChan)
			return
		case StatusInvalid:
			t.SendCoeffs()
		}
	}
}

func (t *Task) SendCoeffs() bool {
	coeffs, err := t.Rate.GenerateCoefficients(len(t.Sensors) * t.BatchCnt)
	if err != nil {
		t.logger.Err(err)
		return false
	}

	decryptionParamsId, err := t.Authority.SendCoefficients(t.Id, coeffs)
	if err != nil {
		t.logger.Err(err)
		return false
	}
	// todo handle error

	t.DecryptionParamsId = decryptionParamsId
	t.CoeffsSubmittedCnt.Add(1)
	return true
}

// potentially blocking method, should be done in goroutine
func (t *Task) AddCipher(feCipher FECipher) {
	_, opened := <-t.keyDerivedChan

	if opened {
		t.logger.Err(fmt.Errorf("key derived channel nas NOT closed"))
		return
	}

	result, err := t.feDecryptor.AddCipher(feCipher)
	if err != nil {
		t.logger.Err(err)
	}

	if result != nil {
		t.Result = result
		fmt.Printf(result.String())
	}
}
