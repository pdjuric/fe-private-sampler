package common

// StartTaskDaemon starts taskDaemon as Runnable goroutine, and returns function that stops the taskDaemon, and another that queries its state
//
// The caller is responsible for closing taskChan
// todo is this true ??
func StartTaskDaemon[TaskT any](taskChan *chan *TaskT, startTaskWorkerFn func(task *TaskT), logger *Logger) (stopFn func(), stateFn func() RunnableState) {
	taskDaemonHandle := NewRunnable("task daemon", logger)
	stopFn = taskDaemonHandle.Stop
	stateFn = taskDaemonHandle.GetState
	go TaskDaemon(taskDaemonHandle, taskChan, startTaskWorkerFn)
	return
}

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
