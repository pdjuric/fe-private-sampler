package authority

import (
	. "fe/common"
	"fmt"
	"sync"
)

type Authority struct {
	tasks sync.Map

	*Host[Task]
}

func InitAuthority() *Authority {
	authority := &Authority{}
	authority.Host = InitHost[Task](AuthorityLogDir, AuthorityLogFilename, SensorTaskChanSize, authority.GetEndpoints())
	return authority
}

func (authority *Authority) AddTask(task *Task) {
	authority.tasks.Store(task.Id, task)
}

func (authority *Authority) GetTask(taskId UUID) (*Task, error) {
	taskAny, exists := authority.tasks.Load(taskId)
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskId)
	}

	return taskAny.(*Task), nil
}
