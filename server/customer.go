package server

import (
	. "fe/common"
	"sync"
)

type Customer struct {
	Uuid    UUID         `json:"id"`
	Sensors []*Sensor    `json:"sensors"`
	mutex   sync.RWMutex // lock to acquire when reading or changing the group
	// isLocked bool
}

func (server *Server) AddCustomer() *Customer {
	customer := &Customer{
		Uuid:    NewUUID(),
		Sensors: make([]*Sensor, 0),
		mutex:   sync.RWMutex{},
	}
	server.customers.Store(customer.Uuid, customer)
	return customer
}

/*func (g *Customer) Lock() bool {
	if g.isLocked {
		return false
	}

	g.mutex.Lock()
	g.isLocked = true
	return true
}

func (g *Customer) Unlock() bool {
	if !g.isLocked {
		return false
	}

	g.mutex.Unlock()
	g.isLocked = false
	return true
}*/

func (g *Customer) AddSensor(s *Sensor) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.Sensors = append(g.Sensors, s)
	s.Customers = append(s.Customers, g)
}
