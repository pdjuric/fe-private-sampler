package authority

import (
	"encoding/json"
	. "fe/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	Id     UUID   `json:"id"`
	Status string `json:"status"`

	SensorIds           []UUID `json:"sensorIds" default:"nil"`
	SensorFetchedParams []atomic.Bool

	// creation parameters
	BatchParams

	MaxSampleValue int
	MaxRateValue   int

	EnableEncryption bool
	FEParamGenerator
	schemaParamsStatus         atomic.Value
	MasterSecKeyGenerationTime time.Duration

	decryptionParams       sync.Map
	decryptionParamsStatus sync.Map

	logger *Logger
}

// NewTask creates a new Task from common.AuthorityTaskRequest
func NewTask(taskRequest AuthorityTaskRequest) *Task {
	task := &Task{
		Id:                  taskRequest.Id,
		Status:              "created",
		SensorIds:           taskRequest.SensorIds,
		SensorFetchedParams: make([]atomic.Bool, len(taskRequest.SensorIds)),

		BatchParams: taskRequest.BatchParams,

		MaxSampleValue: taskRequest.MaxSampleValue,
		MaxRateValue:   taskRequest.MaxTariffValue,

		EnableEncryption: taskRequest.EnableEncryption,

		logger: GetLoggerForFile("", string(taskRequest.Id)),
	}
	taskRequestJson, _ := json.MarshalIndent(taskRequest, "", "  ")
	task.logger.Info("Task params: %s", string(taskRequestJson))
	return task
}

// getBounds calculates vector element bounds needed for FE schema generation
func (t *Task) getBounds() (*big.Int, *big.Int) {
	boundX := big.NewInt(int64(t.MaxSampleValue))
	boundY := big.NewInt(int64(t.MaxRateValue))
	return boundX, boundY
}

func (t *Task) getSensorIdx(sensorId UUID) (int, error) {
	for idx, id := range t.SensorIds {
		if id == sensorId {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("sensor %s not found in task %s", sensorId, t.Id)
}

func (t *Task) SetFEParams() bool {
	var ok bool
	t.schemaParamsStatus.Store(StatusCreated)

	if !t.EnableEncryption {
		ok = t.setDummyParams()
	} else if t.BatchCnt == 1 {
		ok = t.setSingleFEParams()
	} else {
		ok = t.setMultiFEParams()
	}

	if ok {
		t.schemaParamsStatus.Store(StatusReady)
	} else {
		t.schemaParamsStatus.Store(StatusError)
	}

	return ok
}

// setSingleFEParams creates SingleFEParamGenerator for the Task - instantiates fullysec.FHIPE schema and generates master keys
func (t *Task) setSingleFEParams() bool {
	t.logger.Info("using SingleFE")
	feParams := new(SingleFEParamGenerator)
	t.FEParamGenerator = feParams
	feParams.logger = GetLogger("fe param generator", t.logger)
	t.logger = GetLogger("fe param generator", t.logger)

	boundX, boundY := t.getBounds()
	vectorLen := t.BatchSize

	// generate FHIPE schema
	t.logger.Info("generating FE Scheme")
	schema, err := fullysec.NewFHIPE(vectorLen, boundX, boundY)
	if err != nil {
		t.logger.Err(err)
		t.logger.Debug("vector length: %d, max rate value: %d", vectorLen, t.MaxRateValue)
		t.logger.Error("generating FE Scheme failed")
		return false
	}

	t.logger.Info("setting FE Scheme Params")
	feParams.SchemaParams = schema.Params

	// generate master key + measure time
	t.logger.Info("generating FE Master Key")
	start := time.Now()
	feParams.SecKey, err = schema.GenerateMasterKey()
	t.MasterSecKeyGenerationTime = time.Since(start)
	t.logger.Info("elapsed: %d ns", t.MasterSecKeyGenerationTime.Nanoseconds())
	if err != nil {
		t.logger.Err(err)
		t.logger.Error("FE Master key generation failed")
		return false
	}

	t.logger.Info("FE Master key generation succeeded")
	return true
}

// setMultiFEParams creates MultiFEParamGenerator for the Task, instantiates fullysec.FHMultiIPE schema and generates master keys
func (t *Task) setMultiFEParams() bool {
	t.logger.Info("using MultiFE")
	feParams := new(MultiFEParamGenerator)
	feParams.logger = GetLogger("fe param generator", t.logger)
	t.FEParamGenerator = feParams
	t.logger = GetLogger("fe param generator", t.logger)

	boundX, boundY := t.getBounds()
	vectorLen := t.BatchSize
	vectorCnt := t.BatchCnt * len(t.SensorIds)

	// generate FHIPE schema
	t.logger.Info("generating FE Scheme")
	schema := fullysec.NewFHMultiIPE(FHMultiIPESecLevel, vectorCnt, vectorLen, boundX, boundY)
	t.logger.Info("setting FE Scheme Params")
	feParams.SchemaParams = schema.Params
	feParams.BatchesPerSensor = t.BatchCnt
	feParams.SensorCnt = len(t.SensorIds)

	// generate master key + measure time
	t.logger.Info("generating FE Master Key")
	start := time.Now()
	msk, mpk, err := schema.GenerateKeys()
	t.MasterSecKeyGenerationTime = time.Since(start)
	t.logger.Info("elapsed: %d ns", t.MasterSecKeyGenerationTime.Nanoseconds())
	if err != nil {
		t.logger.Err(err)
		t.logger.Error("FE Master key generation failed")
		return false
	}

	feParams.PubKey = mpk
	feParams.SecKey = msk
	t.logger.Info("FE Master key generation succeeded")
	return true
}

// setDummyParams creates DummyGenerator for the Task, with no encryption of samples
func (t *Task) setDummyParams() bool {
	t.logger.Info("encryption turned off")
	t.FEParamGenerator = &DummyGenerator{
		BatchCnt:  t.BatchCnt * len(t.SensorIds),
		BatchSize: t.BatchSize,
	}
	return true
}

// Submit sends the SensorTaskRequest to all Sensors in the Task's Customer (captured during SetSensors)
func (t *Task) GetEncryptionParams(SensorId UUID) (FEEncryptionParams, error) {
	sensorIdx, err := t.getSensorIdx(SensorId)
	t.logger.Info("sensor no %d fetched encryption params", sensorIdx)

	feEncryptionParams, err := t.FEParamGenerator.GetEncryptionParams(sensorIdx)
	if err != nil {
		t.logger.Err(err)
		t.logger.Error("fetching encryption params failed for sensor no %d", sensorIdx)
		return nil, err
	}

	t.SensorFetchedParams[sensorIdx].Store(true)

	return feEncryptionParams, nil
}

// AddNewDecryptionParams generates new decryption params for the provided rates.
// Returns a UUID of the decryption params.
func (t *Task) AddNewDecryptionParams(rates []int) (UUID, error) {
	// check rates count
	if len(rates) != t.BatchCnt*t.BatchSize {
		return "", fmt.Errorf("invalid rates count")
	}

	// check bounds in rates
	decryptionParamsId := NewUUID()
	t.decryptionParamsStatus.Store(decryptionParamsId, StatusCreated)

	go func(task *Task) {
		decryptionParams := task.FEParamGenerator.GetDecryptionParams(rates)
		if decryptionParams == nil {
			t.decryptionParamsStatus.Store(decryptionParamsId, StatusError)
			return
		}
		task.logger.Info("decryption key derived successfully")

		task.decryptionParamsStatus.Store(decryptionParamsId, StatusReady)
		task.decryptionParams.Store(decryptionParamsId, decryptionParams)
	}(t)

	return decryptionParamsId, nil
}

// GetDecryptionParams returns FEDecryptionParams with the provided decryptionParamsId.
func (t *Task) GetDecryptionParams(decryptionParamsId UUID) (FEDecryptionParams, error) {
	t.logger.Info("server fetched decryption params")
	decryptionParams, ok := t.decryptionParams.Load(decryptionParamsId)
	if !ok {
		return nil, fmt.Errorf("decryption params not found")
	}
	return decryptionParams.(FEDecryptionParams), nil
}

// GetDecryptionParamsStatus returns the status for the decryption params with decryptionParamsId.
// Indicates whether the parameters are successfully generated, or the generating is still in progress (or it failed).
func (t *Task) GetDecryptionParamsStatus(decryptionParamsId UUID) string {
	decryptionParamsStatus, ok := t.decryptionParamsStatus.Load(decryptionParamsId)
	if !ok {
		return StatusNotFound
	} else {
		return decryptionParamsStatus.(string)
	}
}

func (t *Task) GetSchemaParamsStatus() string {
	return t.schemaParamsStatus.Load().(string)
}
