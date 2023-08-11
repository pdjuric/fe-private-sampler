package sensor

import (
	. "fe/internal/common"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"sync"
	"sync/atomic"
	"time"
)

type Batch struct {
	idx                int
	samples            []*big.Int
	receivedSamplesCnt int `default:"0"`

	isSubmitted bool `default:"false"`

	encryptionTime time.Duration
}

type sampledData struct {
	batches []Batch

	batchesSampledCnt   int
	mutex               sync.Mutex // needed only if AddSample is called from multiple goroutines, for batchesSampledCnt sync
	batchesSubmittedCnt int
}

func newSampledData(batchParams BatchParams) *sampledData {
	batches := make([]Batch, batchParams.BatchCnt)
	for idx := 0; idx < batchParams.BatchCnt; idx++ {
		batches[idx] = Batch{
			idx:     idx,
			samples: make([]*big.Int, batchParams.BatchSize),
		}
	}

	return &sampledData{
		batches:             batches,
		batchesSampledCnt:   0,
		mutex:               sync.Mutex{},
		batchesSubmittedCnt: 0,
	}
}

func (t *Task) AddSample(sample int) {
	t.mutex.Lock() // in case multiple goroutines call AddSample
	defer t.mutex.Unlock()
	currentBatchIdx := t.batchesSampledCnt

	currentBatch := &t.batches[currentBatchIdx]
	currentBatch.samples[currentBatch.receivedSamplesCnt] = big.NewInt(int64(sample))
	fmt.Println(sample)
	currentBatch.receivedSamplesCnt++
	if currentBatch.receivedSamplesCnt == t.BatchSize {
		t.batchesSampledCnt++
		t.encryptionChan <- currentBatch
		if t.batchesSampledCnt == t.BatchCnt {
			t.CloseEncryptionChan() // this is the signal that there won't be any more batches
		}
	}
}

type Task struct {
	Uuid   uuid.UUID `json:"id"`
	stopFn func()    // stops TaskWorker execution when called
	BatchParams

	// sampling
	SamplingParams
	samplingChan       chan int
	samplingChanClosed atomic.Bool
	*sampledData

	// encryption
	FEEncryptor
	encryptionChan       chan *Batch
	encryptionChanClosed atomic.Bool

	// submission
	server *Server

	logger *Logger
}

// NewTask creates a new Task from common.SubmitTaskRequest
func (s *Sensor) NewTask(taskRequest *SubmitTaskRequest) *Task {
	task := &Task{
		Uuid:        taskRequest.TaskId, // todo use: type Uuid string instead of uuid.Uuid -> easier parsing
		BatchParams: taskRequest.BatchParams,

		SamplingParams: taskRequest.SamplingParams,
		samplingChan:   make(chan int, taskRequest.BatchSize*2), // todo  ????
		sampledData:    newSampledData(taskRequest.BatchParams),

		FEEncryptor:    NewFEEncryptor(taskRequest.Schema, taskRequest.FEEncryptionParams),
		encryptionChan: make(chan *Batch, taskRequest.BatchCnt), // todo ???? (was /2 => rendez-vous occurred when BatchCnt = 1)
		logger:         GetLoggerForFile("", taskRequest.TaskId.String()),

		server: s.Server,
	}

	task.logger.Info("task created")

	return task
}

func (t *Task) CloseSamplingChan() bool {
	alreadyClosed := t.samplingChanClosed.Swap(true)
	if !alreadyClosed {
		close(t.samplingChan)
	}
	return !alreadyClosed
}

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
