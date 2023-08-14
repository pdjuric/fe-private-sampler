package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"net/http"
	"sync/atomic"
	"time"
)

type Task struct {
	Id      UUID      `json:"id"`
	Status  string    `json:"status"`
	Sensors []*Sensor `json:"sensors" default:"nil"`

	// creation parameters
	GroupId UUID `json:"group"`
	SamplingParams
	BatchParams

	// todo add flag for secret coeffs or not
	FEParams
	FEDecryptionParams
	CoefficientsByPeriod []int `json:"coefficientsByPeriod"`

	Result *big.Int

	// times
	MasterSecKeyGenerationTime time.Duration `json:"-"`
	DecryptionTime             time.Duration `json:"-"`

	// status
	SubmittedTaskFlags  []atomic.Bool
	SubmittedCipherCnts []atomic.Int32
	KeyDerivedStatus    atomic.Bool

	logger *Logger
}

// NewTaskFromTaskRequest creates a new Task from common.CreateTaskRequest
func NewTaskFromTaskRequest(taskRequest CreateTaskRequest) *Task {
	id := NewUUID()
	return &Task{
		Id:      id,
		Status:  "created",
		GroupId: taskRequest.GroupId,

		SamplingParams: taskRequest.SamplingParams,
		BatchParams: BatchParams{
			BatchSize: taskRequest.BatchSize,
			BatchCnt:  taskRequest.SampleCount / taskRequest.BatchSize,
		},

		CoefficientsByPeriod: taskRequest.CoefficientsByPeriod,

		SubmittedTaskFlags: make([]atomic.Bool, taskRequest.SampleCount),

		logger: GetLoggerForFile("", string(id)),
	}
}

// getBounds calculates vector element bounds needed for FE schema generation
func (t *Task) getBounds() (*big.Int, *big.Int) {
	boundX := big.NewInt(int64(t.MaxSampleValue))
	boundY := big.NewInt(int64(t.getMaxCoefficientValue()))
	return boundX, boundY
}

// getMaxCoefficientValue returns the maximum value of coefficients; used for FH(Multi)IPE scheme, for boundY
func (t *Task) getMaxCoefficientValue() int {
	maxCoefficientValue := t.CoefficientsByPeriod[0]
	for _, coefficient := range t.CoefficientsByPeriod {
		if coefficient > maxCoefficientValue {
			maxCoefficientValue = coefficient
		}
	}
	return maxCoefficientValue + 1
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

	t.BatchCnt = sensorCnt * t.SampleCount / t.BatchSize
	t.Status = "sensors set"
	return nil
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
		t.logger.Info("vector length: %d, max sample value: %d, max coefficient value: %d", vectorLen, t.MaxSampleValue, t.getMaxCoefficientValue())
		// fmt.Errorf("failed to generate FHIPE schema")
		return false
	}

	feParams.Params = schema.Params

	// generate master key + measure time
	start := time.Now()
	feParams.SecKey, err = schema.GenerateMasterKey()
	t.MasterSecKeyGenerationTime = time.Since(start)
	if err != nil {
		t.logger.Error("error during master secret key generation: %s", err)
		// fmt.Errorf("error during master secret key generation")
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
	vectorCnt := t.BatchCnt

	// generate FHIPE schema
	schema := fullysec.NewFHMultiIPE(FHMultiIPESecLevel, vectorCnt, vectorLen, boundX, boundY)
	feParams.Params = schema.Params
	feParams.BatchesPerSensor = t.SampleCount / t.BatchSize
	feParams.SensorCnt = len(t.Sensors)

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

// Submit sends the SubmitTaskRequest to all Sensors in the Task's Group (captured during SetSensors)
func (t *Task) Submit() bool {
	// todo add parallel execution
	for idx, sensor := range t.Sensors {
		// assert that it isn't already sent

		t.logger.Info("submitting task to sensor %s", sensor.Id)
		feEncryptionParams, err := t.GetEncryptionParams(idx)
		if err != nil {
			t.logger.Err(err) // todo add task rollback ????
			t.logger.Error("submission to sensor %s failed", sensor.Id)
			return false
		}

		statusCode, _, err := sensor.SubmitTask(t.Id, t.BatchParams, t.SamplingParams, t.GetSchemaName(), feEncryptionParams)
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

func (t *Task) DeriveDecryptionKey() bool {
	t.logger.Info("deriving decryption key")
	decryptionParams, err := t.GetDecryptionParams(t.CoefficientsByPeriod)
	if err != nil {
		t.logger.Err(err)
		t.logger.Info("error during deriving decryption key")
		return false
	}

	t.KeyDerivedStatus.Store(true)
	t.logger.Info("decryption key derived successfully")
	t.FEDecryptionParams = decryptionParams
	return true
}
