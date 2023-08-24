package server

// StartTaskWorker starts a taskWorker goroutine for provided Task
func StartTaskWorker(task *Task) {
	go taskWorker(task)
}

// taskWorker submits Task to its sensors and derives the decryption key
func taskWorker(task *Task) {

	// Generating FE params
	ok := task.GetFESchemaParams()
	if !ok {
		return
	}

	// in parallel:
	// - derive functional encryption key
	// - send task to the server(s)

	go task.SubmitToSensors()

	task.DeriveDecryptionKey()
}
