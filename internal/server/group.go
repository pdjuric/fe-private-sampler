package server

import (
	"github.com/google/uuid"
	"sync"
)

type Group struct {
	Uuid     uuid.UUID    `json:"id"`
	Sensors  []*Sensor    `json:"sensors"`
	mutex    sync.RWMutex // lock to acquire when reading or changing the group
	isLocked bool         // fixme
}

func (s *Server) AddGroup() *Group {
	group := &Group{
		Uuid:     uuid.New(),
		Sensors:  make([]*Sensor, 0),
		mutex:    sync.RWMutex{},
		isLocked: false,
	}
	s.groups.Store(group.Uuid, group)
	return group
}

func (g *Group) Lock() {
	if !g.isLocked {
		g.mutex.Lock()
		g.isLocked = true
		logger.Info("group %s locked", g.Uuid)
	} else {
		logger.Info("group %s already locked", g.Uuid)
	}
}

func (g *Group) Unlock() {
	if g.isLocked {
		g.mutex.Unlock()
		g.isLocked = false
		logger.Info("group %s unlocked", g.Uuid)
	} else {
		logger.Info("group %s already unlocked", g.Uuid)
	}
}

func (g *Group) AddSensor(s *Sensor) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Sensors = append(g.Sensors, s)
	s.Groups = append(s.Groups, g)

	logger.Info("sensor %s added to group %s", s.Uuid, g.Uuid)
}
