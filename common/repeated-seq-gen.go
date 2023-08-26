package common

import (
	"math/rand"
	"sync"
)

type RepeatedSequenceGenerator struct {
	SampledSequence []int
	Mutex           sync.Mutex
}

func NewRepeatedSequenceGenerator() *RepeatedSequenceGenerator {
	return &RepeatedSequenceGenerator{
		SampledSequence: make([]int, 0),
	}
}

// ReadSample mocks a hardware sensor, and returns random sample in [0, maxValue]
func (s *RepeatedSequenceGenerator) ReadSample(maxValue int, idx *int) int {
	//todo ovo ne radi za paralelan pristup, mora eksterni kursor !!!
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	*idx += 1
	if *idx == len(s.SampledSequence)+1 {
		// generate new sample
		value := rand.Intn(maxValue)
		s.SampledSequence = append(s.SampledSequence, value)
		return value
	} else {
		// read existing sample
		return s.SampledSequence[*idx-1]
	}
}

// Reset does literally nothing
func (s *RepeatedSequenceGenerator) Reset(idx *int) {
	*idx = 0
}
