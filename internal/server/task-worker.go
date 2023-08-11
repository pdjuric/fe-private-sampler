package server

import (
	"fmt"
)

// StartTaskWorker starts taskWorker as Runnable goroutine, for provided task,
// and it populates Task.stopFn with function that stops the taskWorker
func StartTaskWorker(task *Task) {
	//taskWorkerHandle := NewRunnable("task worker", task.logger)
	//task.stopFn = taskWorkerHandle.Stop
	go taskWorker(task)
	return
}

//todo decription

// taskWorker monitors Task's channels, and spawns a new goroutine for appropriate
func taskWorker(task *Task) {
	//logger := GetLogger("task worker", task)

	err := task.SetFEParams()
	if err != nil {
		fmt.Errorf(err.Error())
		return
	}

	// in parallel:
	// - derive functional encryption key
	// - send task to the server(s)

	go func() {
		err = task.Submit()
		if err != nil {
			return
		}
		// todo set status that the task is submitted

	}()

	key, err := task.FEParams.GetDecryptionKey(task.CoefficientsByPeriod)
	task.decryptionKey = key

}
