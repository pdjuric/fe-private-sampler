package common

// TaskDaemon accepts Tasks from taskChan and spawns taskWorker for that task
func TaskDaemon[TaskT any](r *Runnable, taskChan *chan *TaskT, startTaskWorkerFn func(task *TaskT)) {
	r.Start()

	for {
		select {
		case task, notEnd := <-*taskChan:
			if !notEnd {
				r.Done()
			} else {
				startTaskWorkerFn(task)
			}

		case <-r.ExitChan:
			r.Close()
			return
		}
	}

}
