package sensor

import (
	"encoding/json"
	. "fe/common"
	"sync"
	"sync/atomic"
)

type Task struct {
	Id       UUID   `json:"id"`
	SensorId UUID   `json:"sensorId"`
	stopFn   func() // stops TaskWorker execution when called

	batches []Batch

	// sampling
	SamplingParams
	sampledBatchesCnt atomic.Int32 // atomic, if queried by another goroutine for task status
	samplingChan      chan int
	addingSampleMutex sync.Mutex // only in case of multiple goroutines calling AddSample

	// encryption
	encryptor               FEEncryptor
	encryptionParamsFetched atomic.Bool
	encryptedBatchesCnt     atomic.Int32 // atomic, if queried by another goroutine for task status
	encryptionChan          chan int
	encryptionChanClosed    atomic.Bool

	// submission
	server              *Server
	authority           *Authority
	submittedBatchesCnt atomic.Int32

	logger *Logger
}

// NewTask creates a new Task from common.SensorTaskRequest
func (sensor *Sensor) NewTask(taskRequest *SensorTaskRequest) *Task {
	task := &Task{
		Id:       taskRequest.TaskId,
		SensorId: sensor.Id,

		batches: make([]Batch, taskRequest.BatchCnt),

		SamplingParams: taskRequest.SamplingParams,
		samplingChan:   make(chan int, taskRequest.BatchSize*SensorSamplingChanSizeCoeff),

		encryptionChan: make(chan int, taskRequest.BatchCnt*SensorEncryptionChanSizeCoeff),
		logger:         GetLoggerForFile("", string(taskRequest.TaskId)),

		server: sensor.Server,
		authority: &Authority{
			RemoteHttpServer: &RemoteHttpServer{
				IP:     taskRequest.AuthorityIP,
				Logger: GetLogger("authority", sensor.Logger),
			},
		},
	}

	for idx := 0; idx < taskRequest.BatchCnt; idx++ {
		task.batches[idx].InitBatch(idx, taskRequest.BatchSize)
	}

	task.logger.Info("task created")
	taskRequestJson, _ := json.MarshalIndent(taskRequest, "", "  ")
	task.logger.Info("Task params: %s", string(taskRequestJson))
	sensor.AddTask(task)
	return task
}

// AddSample adds a new sample to the next incomplete batch. If the batch is full, submits it for encryption.
func (t *Task) AddSample(sample int) {
	t.addingSampleMutex.Lock()
	defer t.addingSampleMutex.Unlock()

	currentBatchIdx := int(t.sampledBatchesCnt.Load())
	currentBatch := &t.batches[currentBatchIdx]
	currentBatchFull := currentBatch.AddSample(sample)

	if currentBatchFull {
		sampledBatchesCnt := int(t.sampledBatchesCnt.Add(1))
		t.encryptionChan <- currentBatchIdx
		if sampledBatchesCnt == t.BatchCnt {
			t.CloseEncryptionChan() // this is the signal that there won't be any more batches
		}
	}
}

func (t *Task) EncryptBatch(batchIdx int) bool {
	t.logger.Info("encrypting batch no %d", batchIdx)

	batch := &t.batches[batchIdx]
	cipher, elapsedTime, err := t.encryptor.Encrypt(batch)
	if err != nil {
		t.logger.Err(err)
		t.logger.Info("encryption of batch no %d failed", batchIdx)
		return false
	}
	batch.cipher = cipher
	batch.encryptionTime = elapsedTime
	t.encryptedBatchesCnt.Add(1)

	//t.logger.Info("encryption of batch no %d successful", batchIdx)
	return true
}

func (t *Task) SubmitCipher(batchIdx int) bool {

	t.logger.Info("submitting cipher no %d", batchIdx)

	batch := &t.batches[batchIdx]
	err := t.server.SubmitCipher(t.Id, t.SensorId, batch.cipher)
	if err != nil {
		t.logger.Err(err)
		t.logger.Info("submission of cipher no %d failed", batchIdx)
		return false
	}
	t.submittedBatchesCnt.Add(1)

	t.logger.Info("submission of cipher no %d successful", batchIdx)
	return true
}

// CloseSamplingChan closes the samplingChan
func (t *Task) CloseSamplingChan() {
	close(t.samplingChan)
}

// CloseEncryptionChan closes the encryptionChan; as it can be called from AddSample or from TaskWorker,
// it prevents closing the channel twice
func (t *Task) CloseEncryptionChan() bool {
	alreadyClosed := t.encryptionChanClosed.Swap(true)
	if !alreadyClosed {
		close(t.encryptionChan)
	}
	return !alreadyClosed
}

// cleanup must be called after Sampler and goroutine that does encryption and submitting are stopped !!
func (t *Task) cleanup() {
	// todo add cleanup
}

func (t *Task) GetSamples() [][]int32 {
	samples := make([][]int32, 0)
	for idx := 0; idx < t.BatchCnt; idx++ {
		samplesFromBatch := t.batches[idx].GetSamples()
		if samplesFromBatch == nil {
			break
		}
		samples = append(samples, samplesFromBatch)
	}
	return samples
}

func (t *Task) FetchEncryptionParams() bool {
	feEncryptionParams, err := t.authority.GetEncryptionParams(t.Id, t.SensorId)
	if err != nil {
		t.logger.Err(err)
		return false
	}

	t.encryptor = NewFEEncryptor(feEncryptionParams, t.logger)
	t.encryptionParamsFetched.Store(true)
	return true
}
