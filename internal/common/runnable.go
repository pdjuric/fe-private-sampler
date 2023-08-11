package common

import (
	"sync"
)

/*
note: In order to execute a goroutine whole lifecycle is managed by Runnable, it has to be written by the following template:

func goroutine(r *Runnable, ...) {
	...
	r.Start()
	...

loop:
	for {
		select {

		...
		call r.Done() when the work is done
		...

		case <-r.ExitChan:
			...
			r.Close()
			break loop
		}
	}

}
*/

type RunnableState string

const (
	RunnableCreated   = "created"
	RunnableRunning   = "running"
	RunnableDone      = "done"
	RunnableStopped   = "stopped"
	RunnableCancelled = "cancelled"
	RunnableFailed    = "failed"
)

type Runnable struct {
	name  string
	state RunnableState
	mutex sync.Mutex

	ExitChan chan bool

	Logger *Logger
}

func (r *Runnable) GetState() RunnableState {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.state
}

// NewRunnable creates Runnable, and assigns provided httpLogger to it;
// if httpLogger is not provided, logged messages are discarded
func NewRunnable(name string, logger *Logger) *Runnable {
	if logger == nil {
		logger = GetDiscardLogger()
	} else {
		logger = GetLogger(name, logger)
	}

	return &Runnable{
		name:     name,
		ExitChan: make(chan bool, 1),
		state:    RunnableCreated,
		Logger:   logger,
	}
}

// Start updates Runnable's status when the execution starts; must be called from Runnable's goroutine when the work is started
func (r *Runnable) Start() (exiting bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.state == RunnableCreated {
		exiting = false
		r.state = RunnableRunning
		r.Logger.Info("starting...")
	} else {
		// cancelled
		exiting = true
	}
	return
}

// Stop stops the execution of Runnable, or no effect if it's already stopped; meant to be called by the creator of Runnable
func (r *Runnable) Stop() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.state == RunnableRunning {
		r.state = RunnableStopped
		r.ExitChan <- true
	} else if r.state == RunnableCreated {
		r.state = RunnableCancelled
	} else {
		// already stopped, cancelled or done
		r.Logger.Info("Stop() called, but "+r.name+" is already %s", r.state)
	}
}

// Done stops the execution of Runnable; must be called from Runnable's goroutine when the work is done
func (r *Runnable) Done() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.state == RunnableRunning {
		r.state = RunnableDone
		r.ExitChan <- true
	} else {
		// already stopped
		r.Logger.Info("done() called, but "+r.name+" is already %s", r.state)
	}
}

// Fail ; call only from Running state
// stops the execution of Runnable; must be called from Runnable's goroutine when the work is done
func (r *Runnable) Fail(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.state != RunnableRunning {
		return
	}

	r.state = RunnableFailed
	r.ExitChan <- true
	r.Logger.Info(r.name+"failed", r.state)
	r.Logger.Err(err)
}

// Close closes resources allocated by Runnable (only ExitChan - everything else must be closed manually)
func (r *Runnable) Close() {
	close(r.ExitChan)
	r.Logger.Info(r.name+" is %s at %d", r.state, Now().Unix())
}
