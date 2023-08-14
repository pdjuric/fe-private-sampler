package sensor

import (
	. "fe/internal/common"
	"sync"
	"sync/atomic"
)

type Task struct {
	Id       UUID   `json:"id"`
	SensorId UUID   `json:"sensorId"`
	stopFn   func() // stops TaskWorker execution when called

	BatchParams
	batches []Batch

	// sampling
	SamplingParams
	sampledBatchesCnt atomic.Int32 // atomic, if queried by another goroutine for task status
	samplingChan      chan int
	addingSampleMutex sync.Mutex // only in case of multiple goroutines calling AddSample

	// encryption
	schema string
	FEEncryptionParams
	encryptedBatchesCnt  atomic.Int32 // atomic, if queried by another goroutine for task status
	encryptionChan       chan int
	encryptionChanClosed atomic.Bool

	// submission
	server              *Server
	submittedBatchesCnt atomic.Int32

	logger *Logger
}

// NewTask creates a new Task from common.SubmitTaskRequest
func (s *Sensor) NewTask(taskRequest *SubmitTaskRequest) *Task {
	task := &Task{
		Id:       taskRequest.TaskId,
		SensorId: s.Id,

		BatchParams: taskRequest.BatchParams,
		batches:     make([]Batch, taskRequest.BatchCnt),

		SamplingParams: taskRequest.SamplingParams,
		samplingChan:   make(chan int, taskRequest.BatchSize*SensorSamplingChanSizeCoeff),

		schema:             taskRequest.Schema,
		FEEncryptionParams: taskRequest.FEEncryptionParams,
		encryptionChan:     make(chan int, taskRequest.BatchCnt*SensorEncryptionChanSizeCoeff),
		logger:             GetLoggerForFile("", string(taskRequest.TaskId)),

		server: s.Server,
	}

	for idx := 0; idx < taskRequest.BatchCnt; idx++ {
		task.batches[idx].InitBatch(idx, taskRequest.BatchSize)
	}

	task.logger.Info("task created")
	return task
}

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

	err := t.batches[batchIdx].Encrypt(t.schema, t.FEEncryptionParams)
	if err != nil {
		t.logger.Err(err)
		t.logger.Info("encryption of batch no %d failed", batchIdx)
		return false
	}
	t.encryptedBatchesCnt.Add(1)

	t.logger.Info("encryption of batch no %d successful", batchIdx)
	return true
}

func (t *Task) SubmitCipher(batchIdx int) bool {

	t.logger.Info("submitting cipher no %d", batchIdx)

	batch := &t.batches[batchIdx]
	err := t.server.SubmitCipher(t.Id, t.SensorId, batch.idx, batch.cipher)
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
