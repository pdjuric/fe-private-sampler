package sensor

import (
	. "fe/common"
	"sync/atomic"
	"time"
)

// StartTaskWorker starts taskWorker as a Runnable goroutine, for provided task,
// and it populates Task.stopFn with function that stops the taskWorker
func StartTaskWorker(task *Task) {
	taskWorkerHandle := NewRunnable("task worker", task.logger)
	task.stopFn = taskWorkerHandle.Stop
	go taskWorker(taskWorkerHandle, task)
	return
}

// taskWorker collects the sampled data, groups it into batches, encrypts batches and submits cyphers to the server.
func taskWorker(r *Runnable, task *Task) {
	r.Start()

	encryptionParamsFetched := make(chan bool, 1)
	// fetch encryption params
	go func() {
		for {
			if ok := task.FetchEncryptionParams(); ok {
				encryptionParamsFetched <- true
				return
			}
			time.Sleep(EncryptionParamsPollingInterval)
		}
	}()

	// start sampling
	stopSampler := StartSampler(&task.SamplingParams, &task.samplingChan, task.CloseSamplingChan, task.logger)

	// do not close these channels, close them through task
	samplingChan := task.samplingChan     // chan to wait on for new samples
	encryptionChan := task.encryptionChan // chan to wait on for encrypted batches

	// http request rate limiting and cancelling
	rateLimiter := make(chan bool, MaxParallelSubmissionsPerSensor)
	for i := 0; i < MaxParallelSubmissionsPerSensor; i++ {
		rateLimiter <- true
	}

	// http request cancelling in case the task daemon has been stopped
	cancelSubmission := make(chan bool, MaxParallelSubmissionsPerSensor)
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

		case idx, notEnd := <-encryptionChan:
			if !notEnd {
				// done, all batches collected
				encryptionChan = nil
				r.Logger.Info("all batches collected & encrypted")
				r.Done()
				continue
			}

			// do this in goroutine, as it could take up much time
			// in logger, it will still be displayed as TaskWorker
			// no need for Runnable as it will not be waiting on channels in a loop
			go func(batchIdx int) {
				// wait until encryption params are fetched
				<-encryptionParamsFetched
				encryptionParamsFetched <- true

				// encrypt the batch
				ok := task.EncryptBatch(batchIdx)
				if !ok {
					r.Logger.Info("could not encrypt batch no %d of task %s; aborting...", batchIdx, task.Id)
				}

				// either gets a token for submitting a cipher, or gets cancelled
				select {
				case <-rateLimiter:
					// send the cipher to the server
					task.SubmitCipher(batchIdx)
					if !ok {
						r.Logger.Info("could not submit cipher no %d of task %s; aborting...", batchIdx, task.Id)
					}

					// if the submission is not cancelled, return the token
					// if it is, do not return it, so that other goroutines don't start submitting (using this token)
					if !submissionCancelled.Load() {
						rateLimiter <- true
					}
				case <-cancelSubmission:
					cancelSubmission <- true
				}

			}(idx)

		case <-r.ExitChan:
			// CRITICAL -> setting encryptionChan to nil so as fewer batches as possible start encryption
			encryptionChan = nil

			stopSampler()              // if sampler isn't done, this will stop it, and make it close its chan
			task.CloseEncryptionChan() // already closed if task.AddSample() is called for all task.sampleCnt samples

			submissionCancelled.Store(true)
			for i := 0; i < MaxParallelSubmissionsPerSensor; i++ {
				cancelSubmission <- true
			}

			// task may be completed, stopped(cancelled), or failed
			// todo set task status
			task.cleanup()
			r.Close()
			return
		}

	}

}
