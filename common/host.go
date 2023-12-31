package common

import "fmt"

// Host has a HttpServer for accepting requests and a taskDaemon for executing tasks.
type Host[TaskT any] struct {
	// TaskDaemon
	taskChan          chan *TaskT // todo close
	taskDaemonStateFn func() RunnableState
	taskDaemonStopFn  func()

	// HttpServer
	*HttpServer

	Logger *Logger
}

// InitHost initializes a new host by creating log file, setting up Task Daemon and registering endpoints
// to the HttpServer.
func InitHost[TaskT any](logDir string, logFilename string, taskDaemonChanSize int, endpoints []Endpoint) *Host[TaskT] {
	err := InitLogger(logDir)
	if err != nil {
		fmt.Println("host not started")
		return nil
	}

	host := &Host[TaskT]{
		taskChan: make(chan *TaskT, taskDaemonChanSize),
		Logger:   GetLoggerForFile("", logFilename),
	}
	host.HttpServer = InitHttpServer(host.Logger, endpoints)

	return host
}

func (h *Host[TaskT]) StartTaskDaemon(startTaskWorkerFn func(*TaskT)) {
	taskDaemonHandle := NewRunnable("task daemon", h.Logger)
	h.taskDaemonStopFn = taskDaemonHandle.Stop
	h.taskDaemonStateFn = taskDaemonHandle.GetState
	go TaskDaemon(taskDaemonHandle, &h.taskChan, startTaskWorkerFn)
}

func (h *Host[TaskT]) StopTaskDaemon() {
	// todo
	// todo where to close taskChan?
}

func (h *Host[TaskT]) SendTaskToDaemon(t *TaskT) {
	h.taskChan <- t
}
