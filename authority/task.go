package authority

import (
	. "fe/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	Id        UUID   `json:"id"`
	Status    string `json:"status"`
	SensorIds []UUID `json:"sensorIds" default:"nil"`

	// creation parameters
	BatchParams

	MaxSampleValue int `json:"maxSampleValue"`
	MaxCoeffValue  int `json:"maxCoeffValue"`

	FEParams

	decryptionParams       sync.Map
	decryptionParamsStatus sync.Map

	// times
	// todo
	MasterSecKeyGenerationTime time.Duration `json:"-"`
	//DecryptionTime             time.Duration `json:"-"`

	// status
	SubmittedTaskFlags  []atomic.Bool
	SubmittedCipherCnts []atomic.Int32
	KeyDerivedStatus    atomic.Bool

	logger *Logger
}

// NewTaskFromTaskRequest creates a new Task from common.CreateAuthorityTaskRequest
func NewTaskFromTaskRequest(taskRequest CreateAuthorityTaskRequest) *Task {
	return &Task{
		Id:        taskRequest.Id,
		Status:    "created",
		SensorIds: taskRequest.SensorIds,

		BatchParams: taskRequest.BatchParams,

		MaxSampleValue: taskRequest.MaxSampleValue,
		MaxCoeffValue:  taskRequest.MaxRateValue,

		logger: GetLoggerForFile("", string(taskRequest.Id)),
	}
}

// getBounds calculates vector element bounds needed for FE schema generation
func (t *Task) getBounds() (*big.Int, *big.Int) {
	boundX := big.NewInt(int64(t.MaxSampleValue))
	boundY := big.NewInt(int64(t.MaxCoeffValue))
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
	if t.BatchCnt == 1 {
		return t.setSingleFEParams()
	} else {
		return t.setMultiFEParams()
	}
}

// setSingleFEParams creates SingleFEParams for the Task - instantiates fullysec.FHIPE schema and generates master keys
func (t *Task) setSingleFEParams() bool {
	feParams := new(SingleFEParams)
	t.FEParams = feParams

	boundX, boundY := t.getBounds()
	vectorLen := t.BatchSize

	// generate FHIPE schema
	schema, err := fullysec.NewFHIPE(vectorLen, boundX, boundY)
	if err != nil {
		t.logger.Err(err)
		t.logger.Info("vector length: %d, max coefficient value: %d", vectorLen, t.MaxCoeffValue)
		// fmt.Errorf("failed to generate FHIPE schema")
		return false
	}

	feParams.SchemaParams = schema.Params

	// generate master key + measure time
	start := time.Now()
	feParams.SecKey, err = schema.GenerateMasterKey()
	t.MasterSecKeyGenerationTime = time.Since(start)
	if err != nil {
		t.logger.Error("error during master secret key generation: %s", err)
		return false
	}

	return true
}

// setMultiFEParams creates MultiFEParams for the Task, instantiates fullysec.FHMultiIPE schema and generates master keys
func (t *Task) setMultiFEParams() bool {
	feParams := new(MultiFEParams)
	t.FEParams = feParams

	boundX, boundY := t.getBounds()
	vectorLen := t.BatchSize
	vectorCnt := t.BatchCnt * len(t.SensorIds)

	// generate FHIPE schema
	schema := fullysec.NewFHMultiIPE(FHMultiIPESecLevel, vectorCnt, vectorLen, boundX, boundY)
	feParams.SchemaParams = schema.Params
	feParams.BatchesPerSensor = /*t.SampleCount /*/ t.BatchCnt
	feParams.SensorCnt = len(t.SensorIds)

	// generate master key + measure time
	start := time.Now()
	msk, mpk, err := schema.GenerateKeys()
	t.MasterSecKeyGenerationTime = time.Since(start)
	if err != nil {
		t.logger.Error("error during generating master secret and public key: %s", err)
		// fmt.Errorf("error during generating master and public secret key")
		return false
	}

	feParams.PubKey = mpk
	feParams.SecKey = msk

	return true
}

// Submit sends the SubmitSensorTaskRequest to all Sensors in the Task's Group (captured during SetSensors)
func (t *Task) GetEncryptionParams(SensorId UUID) (FEEncryptionParams, error) {
	sensorIdx, err := t.getSensorIdx(SensorId)
	// todo check err

	feEncryptionParams, err := t.FEParams.GetEncryptionParams(sensorIdx)
	if err != nil {
		t.logger.Err(err)
		return nil, err
	}

	return feEncryptionParams, nil
}

// Submit sends the SubmitSensorTaskRequest to all Sensors in the Task's Group (captured during SetSensors)
func (t *Task) AddNewDecryptionParams(CoefficientsByPeriod []int) (UUID, error) {
	// check coefficient count
	if len(CoefficientsByPeriod) != t.BatchCnt*t.BatchSize {
		return "", fmt.Errorf("invalid coefficient count")
	}

	// check bounds in coefficients
	decryptionParamsId := NewUUID()
	t.decryptionParamsStatus.Store(decryptionParamsId, StatusCreated)

	go func(task *Task) {
		task.logger.Info("deriving decryption key")
		decryptionParams, err := task.FEParams.GetDecryptionParams(CoefficientsByPeriod)
		if err != nil {
			task.logger.Err(err)
			task.logger.Info("error during deriving decryption key")
			t.decryptionParamsStatus.Store(decryptionParamsId, StatusError)
		}

		// todo status invalid

		// todo what if error occurred during key derivation?
		task.decryptionParamsStatus.Store(decryptionParamsId, StatusReady)
		task.decryptionParams.Store(decryptionParamsId, decryptionParams)
	}(t)

	return decryptionParamsId, nil
}

// Submit sends the SubmitSensorTaskRequest to all Sensors in the Task's Group (captured during SetSensors)
func (t *Task) GetDecryptionParams(decryptionParamsId UUID) (FEDecryptionParams, error) {

	decryptionParams, ok := t.decryptionParams.Load(decryptionParamsId)
	if !ok {
		return nil, fmt.Errorf("decryption params not found")
	}
	return decryptionParams.(FEDecryptionParams), nil
}

func (t *Task) GetDecryptionParamsStatus(decryptionParamsId UUID) string {
	decryptionParamsStatus, ok := t.decryptionParamsStatus.Load(decryptionParamsId)
	if !ok {
		return StatusNotFound
	} else {
		return decryptionParamsStatus.(string)
	}
}
