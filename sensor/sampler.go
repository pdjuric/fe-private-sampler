package sensor

import (
	. "fe/common"
	"math/rand"
	"time"
)

// StartSampler starts sampler as Runnable goroutine, with samplingDetails, and returns function that stops the sampler
//
// The caller is responsible for closing sampleChan
func StartSampler(samplingDetails *SamplingParams, sampleChan *chan int, closeChannelFn func(), logger *Logger) (stopFn func()) {
	samplerHandle := NewRunnable("sampler", logger)
	stopFn = samplerHandle.Stop
	go sampler(samplerHandle, samplingDetails, sampleChan, closeChannelFn)
	return
}

// sampler reads the sensor with readSampleFromSensor and writes samples to sampleChan;
// it reads sampling details (start, period, sampleCount, maxSampleValue) from samplingDetails;
// it first resets the sensor at *start* time, then it samples the sensor every *period* seconds, *sampleCount* times
func sampler(r *Runnable, samplingDetails *SamplingParams, sampleChan *chan int, closeChannelFn func()) {

	start := time.Unix(int64(samplingDetails.Start), 0)
	period := time.Duration(samplingDetails.SamplingPeriod) * time.Second
	sampleCount := samplingDetails.BatchCnt * samplingDetails.BatchSize
	maxSampleValue := samplingDetails.MaxSampleValue

	r.Start()

	// wait for Start time
	timeToSleep := start.Sub(Now())
	r.Logger.Info("waiting for reset time %d (sleeping %ds)", start.Unix(), int(timeToSleep.Seconds()))
	time.Sleep(timeToSleep)
	// todo if late?

	r.Logger.Info("resetting sampler at %d", Now().Unix())
	resetSensor()

	for {
		select {
		case <-time.After(period):
			// if more than one case is possible, case choice is random
			// when the work is done, sampler is no longer in RunnableRunning, but timer will fire anyway,
			// and sampler will push new samples instead of exiting
			// -> check if sampler's still running
			if r.GetState() != RunnableRunning {
				continue
			}

			// if server reading takes too long, time will be off -> time.After should be used in a separate goroutine
			// it's ok here
			*sampleChan <- readSampleFromSensor(maxSampleValue)
			r.Logger.Info("sampled at %d", Now().Unix())

			sampleCount--
			if sampleCount == 0 {
				r.Done()
			}

		case <-r.ExitChan:
			r.Close()
			closeChannelFn()
			return
		}
	}

}

// readSampleFromSensor mocks a hardware sensor, and returns random sample in [0, maxValue]
func readSampleFromSensor(maxValue int) int {
	return rand.Intn(maxValue)
}

// resetSensor does literally nothing
func resetSensor() {
}
