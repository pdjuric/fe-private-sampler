package authority

// StartTaskWorker starts a taskWorker goroutine for provided Task
func StartTaskWorker(task *Task) {
	go taskWorker(task)
}

// taskWorker submits Task to its sensors and derives the decryption key
func taskWorker(task *Task) {

	// Generating FE params
	_ = task.SetFEParams()

}
