package server

import (
	. "fe/internal/common"
	"sync"
)

type Group struct {
	Uuid     UUID         `json:"id"`
	Sensors  []*Sensor    `json:"sensors"`
	mutex    sync.RWMutex // lock to acquire when reading or changing the group
	isLocked bool         // fixme
}

func (server *Server) AddGroup() *Group {
	group := &Group{
		Uuid:     NewUUID(),
		Sensors:  make([]*Sensor, 0),
		mutex:    sync.RWMutex{},
		isLocked: false,
	}
	server.groups.Store(group.Uuid, group)
	return group
}

func (g *Group) Lock() bool {
	if g.isLocked {
		return false
	}

	g.mutex.Lock()
	g.isLocked = true
	return true
}

func (g *Group) Unlock() bool {
	if !g.isLocked {
		return false
	}

	g.mutex.Unlock()
	g.isLocked = false
	return true
}

func (g *Group) AddSensor(s *Sensor) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Sensors = append(g.Sensors, s)
	s.Groups = append(s.Groups, g)
}
