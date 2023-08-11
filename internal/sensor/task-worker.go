package sensor

import (
	. "fe/internal/common"
	"sync/atomic"
)

// StartTaskWorker starts taskWorker as Runnable goroutine, for provided task,
// and it populates Task.stopFn with function that stops the taskWorker
func StartTaskWorker(task *Task) {
	taskWorkerHandle := NewRunnable("task worker", task.logger)
	task.stopFn = taskWorkerHandle.Stop
	go taskWorker(taskWorkerHandle, task)
	return
}

//todo decription

// taskWorker monitors Task's channels, and spawns a new goroutine for appropriate
func taskWorker(r *Runnable, task *Task) {
	r.Start()

	//start Sampler
	stopSampler := StartSampler(&task.SamplingParams, &task.samplingChan, task.CloseSamplingChan, task.logger)

	// do not close these channel, close them through task
	samplingChan := task.samplingChan
	encryptionChan := task.encryptionChan

	// http request rate limiting, and cancelling
	rateLimiter := make(chan bool, SensorMaxParallelSubmitBatches)
	for i := 0; i < SensorMaxParallelSubmitBatches; i++ {
		rateLimiter <- true
	}

	// http request cancelling in case of stopping task daemon
	cancelSubmission := make(chan bool, SensorMaxParallelSubmitBatches)
	var submissionCancelled atomic.Bool
	submissionCancelled.Store(false)

	for {
		select {
		case sample, notEnd := <-samplingChan:

			if !notEnd {
				// done, all samples received
				samplingChan = nil
				r.Logger.Info("all samples received")
				continue
			}

			// if the batch is ready, this will put it into BatchesForSubmissionChan
			// no need to do this in goroutine, TaskWorker will be able to keep up with all the incoming samples
			task.AddSample(sample)

		case batch, notEnd := <-encryptionChan:
			if !notEnd {
				// done, all batches collected
				encryptionChan = nil
				r.Logger.Info("all batches collected & encrypted")
				r.Done()
				continue
			}

			// do this in goroutine, as it could take up much time
			// in logger, it will still be displayed as TaskWorker
			// no ned for Runnable as it will not be waiting on channels in a loop
			// do not use batch in the anonymous function, as it can be replaced ????
			// todo is this true?
			go func(b *Batch) {
				// encrypt the batch, and send it to server
				r.Logger.Info("encrypting batch no %d", b.idx)
				batchIdx, err := task.EncryptBatch(b)
				if err != nil {
					r.Fail(err) //todo replace
					return
				}

				cipher, err := task.GetCipher(batchIdx)
				if err != nil {
					r.Fail(err) //todo replace
					return
				}

				// either gets a token for submitting a cipher, or gets cancelled
				select {
				case <-rateLimiter:
					task.server.SubmitBatch(task.Uuid, cipher)

					// if the submission is not cancelled, return the token
					// if it is, do not return it, so that other goroutines don't start submitting (using this token)
					if !submissionCancelled.Load() {
						rateLimiter <- true
					}
				case <-cancelSubmission:
					// todo cancel submission
					cancelSubmission <- true
				}

			}(batch)

		case <-r.ExitChan:
			//todo drain chans ?

			// CRITICAL -> setting encryptionChan to nil so as fewer batches as possible start encryption
			encryptionChan = nil

			stopSampler()              // if sampler isn't done, this will stop it, and make it close its chan
			task.CloseEncryptionChan() // already closed if task.AddSample() is called for all task.sampleCnt samples

			// fixme no way to stop goroutines that are already encrypting and submitting batches
			cancelling := false
			if cancelling {
				submissionCancelled.Store(true)
				for i := 0; i < SensorMaxParallelSubmitBatches; i++ {
					cancelSubmission <- true
				}
			}

			// task may be completed, stopped(cancelled), or failed
			// todo handle these cases, set status
			task.cleanup()
			r.Close()
			return
		}

		// todo if stopped, avoid entering cases

	}

}
