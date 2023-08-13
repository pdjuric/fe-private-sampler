package server

// StartTaskWorker starts a taskWorker goroutine for provided Task
func StartTaskWorker(task *Task) {
	go taskWorker(task)
}

// taskWorker submits Task to its sensors and derives the decryption key
func taskWorker(task *Task) {
	ok := task.SetFEParams()
	if !ok {
		return
	}

	// in parallel:
	// - derive functional encryption key
	// - send task to the server(s)

	go task.Submit()

	task.DeriveDecryptionKey()
}
