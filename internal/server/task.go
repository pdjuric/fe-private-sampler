package server

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"github.com/google/uuid"
	"math/big"
	"time"
)

type Task struct {
	Uuid    uuid.UUID `json:"id"`
	Status  string    `json:"status"`
	Sensors []*Sensor `json:"sensors" default:"nil"`

	// creation parameters
	Group uuid.UUID `json:"group"`
	SamplingParams
	BatchParams // derived

	// todo add flag for secret coeffs or not
	CoefficientsByPeriod []int `json:"coefficientsByPeriod"`
	FEParams

	// decryption
	decryptionKey FEDecryptionKey
	result        *big.Int

	// times
	SchemaGenerationTime       time.Duration `json:"-"`
	MasterSecKeyGenerationTime time.Duration `json:"-"`
}

// NewTaskFromTaskRequest creates a new Task from common.CreateTaskRequest
func NewTaskFromTaskRequest(taskRequest CreateTaskRequest) *Task {
	return &Task{
		Uuid:   uuid.New(),
		Status: "created",
		Group:  taskRequest.GroupId,
		BatchParams: BatchParams{
			BatchSize: taskRequest.BatchSize,
			BatchCnt:  taskRequest.SampleCount / taskRequest.BatchSize,
		},
		SamplingParams:       taskRequest.SamplingParams,
		CoefficientsByPeriod: taskRequest.CoefficientsByPeriod,
	}
}

// GetBounds calculates vector element bounds needed for FE schema generation
func (t *Task) GetBounds() (*big.Int, *big.Int) {
	boundX := big.NewInt(int64(t.MaxSampleValue))
	boundY := big.NewInt(int64(t.GetMaxCoefficientValue()))
	return boundX, boundY
}

// GetMaxCoefficientValue returns the maximum value of coefficients; used for FH(Multi)IPE scheme, for boundY
func (t *Task) GetMaxCoefficientValue() int {
	maxCoefficientValue := t.CoefficientsByPeriod[0]
	for _, coefficient := range t.CoefficientsByPeriod {
		if coefficient > maxCoefficientValue {
			maxCoefficientValue = coefficient
		}
	}
	return maxCoefficientValue + 1
}

// SetSensors sets Sensors (from provided Group) for Task, and calculates vectorLen and vectorCnt for the Task based on the number of  samplesPerSubmission
func (t *Task) SetSensors(g *Group) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// check if there are any sensors at all
	if len(g.Sensors) == 0 {
		err := fmt.Errorf("no sensors in group %s; tasks can be created for groups with at least one server", g.Uuid)
		logger.Err(err)
		return err
	}

	logger.Info("setting sensors for task %s", t.Uuid)
	t.Sensors = make([]*Sensor, len(g.Sensors))
	copy(t.Sensors, g.Sensors)

	if t.BatchSize == t.SampleCount {
		// opt 1: sending all samples at once, vectorLen = sampleCnt, vectors = sensorCnt

		logger.Info("sending all samples at once")
		t.BatchSize = t.SampleCount
		t.BatchCnt = len(t.Sensors)
	} else {
		// opt 2: sending samples in batches, vectorLen = SamplesPerSubmission, vectors = sensorCnt * submissionCnt

		logger.Info("sending samples in batches")
		//fixme zakomentarisano t.BatchSize = &t.SamplesPerSubmission
		t.BatchCnt = len(t.Sensors) * t.SampleCount / t.BatchSize
	}

	t.Status = "sensors set"
	return nil
}

func (t *Task) SetFEParams() error {
	// todo handle error
	if t.BatchCnt == 1 {
		_ = t.setSingleFEParams()
	} else {
		_ = t.setMultiFEParams()
	}
	return nil
}

// setSingleFEParams creates SingleFEParams for the Task - instantiates fullysec.FHIPE schema and generates master keys
func (t *Task) setSingleFEParams() error {
	var start, end time.Time

	feParams := new(SingleFEParams)
	t.FEParams = feParams

	boundX, boundY := t.GetBounds()
	vectorLen := t.BatchSize

	// generate FHIPE schema + measure time
	start = time.Now()
	schema, err := fullysec.NewFHIPE(vectorLen, boundX, boundY)
	end = time.Now()
	if err != nil {
		logger.Err(err)
		logger.Info("vector length: %d, max sample value: %d, max coefficient value: %d", vectorLen, t.MaxSampleValue, t.GetMaxCoefficientValue())
		return fmt.Errorf("failed to generate FHIPE schema")
	}

	t.SchemaGenerationTime = end.Sub(start)
	feParams.Params = schema.Params

	// generate master key + measure time
	start = time.Now()
	msk, err := schema.GenerateMasterKey()
	end = time.Now()
	if err != nil {
		logger.Error("error during master secret key generation: %s", err)
		return fmt.Errorf("error during master secret key generation")
	}

	t.MasterSecKeyGenerationTime = end.Sub(start)
	feParams.SecKey = msk

	return nil
}

// setMultiFEParams creates MultiFEParams for the Task, instantiates fullysec.FHMultiIPE schema and generates master keys
func (t *Task) setMultiFEParams() error {
	var start, end time.Time

	feParams := new(MultiFEParams)
	t.FEParams = feParams

	boundX, boundY := t.GetBounds()
	vectorLen := t.BatchSize
	vectorCnt := t.BatchCnt

	// generate FHIPE schema + measure time
	start = time.Now()
	schema := fullysec.NewFHMultiIPE(FHMultiIPESecLevel, vectorCnt, vectorLen, boundX, boundY)
	end = time.Now()

	t.SchemaGenerationTime = end.Sub(start)
	feParams.Params = schema.Params
	feParams.BatchCnt = t.SampleCount / t.BatchSize
	feParams.SensorCnt = len(t.Sensors)

	// generate master key + measure time
	start = time.Now()
	msk, mpk, err := schema.GenerateKeys()
	end = time.Now()
	if err != nil {
		logger.Error("error during generating master secret and public key: %s", err)
		return fmt.Errorf("error during generating master and public secret key")
	}

	t.MasterSecKeyGenerationTime = end.Sub(start)
	feParams.PubKey = mpk
	feParams.SecKey = msk

	return nil
}

// NewSubmitTaskRequest creates a new common.SubmitTaskRequest from Task, which will be submitted to the server
func NewSubmitTaskRequest(t *Task, sensorIdx int) (*SubmitTaskRequest, error) {
	feEncryptionParams, err := t.GetEncryptionParams(sensorIdx)
	if err != nil {
		// todo add task rollback
		return nil, err
	}

	return &SubmitTaskRequest{
		TaskId:             t.Uuid,
		BatchParams:        t.BatchParams,
		SamplingParams:     t.SamplingParams,
		Schema:             t.GetSchemaName(),
		FEEncryptionParams: feEncryptionParams,
	}, nil
}

// Submit sends the SubmitTaskRequest and FEEncryptionParams to all Sensors in the Task's Group (captured during SetSensors)
func (t *Task) Submit() (e error) {
	e = fmt.Errorf("the task could not be submitted")
	// todo add parallel execution
	for idx, sensor := range t.Sensors {

		submitTaskRequest, e := NewSubmitTaskRequest(t, idx)
		if e != nil {
			return e
		}

		statusCode, responseBody, e := sensor.SubmitTask(submitTaskRequest)
		if e != nil {
			return e
		}

		logger.Info("status code: %d, response body: %s", statusCode, responseBody)
		// todo check whether start time has already passed
	}
	return nil
}
