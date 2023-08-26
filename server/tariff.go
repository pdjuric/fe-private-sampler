package server

import (
	. "fe/common"
	"sync"
	"sync/atomic"
)

type Tariff struct {
	id             UUID
	Description    string `json:"description"`
	SamplingPeriod int    `json:"samplingPeriod"`
	BatchSize      int    `json:"batchSize"`
	MaxSampleValue int    `json:"maxSampleValue"`
	MaxTariffValue int    `json:"maxTariffValue"`
}

var tariffMap = sync.Map{}

func GetTariff(tariffId UUID) (*Tariff, bool) {
	tariff, exists := tariffMap.Load(tariffId)
	return tariff.(*Tariff), exists
}

func NewTariff() *Tariff {
	return &Tariff{
		id: NewUUID(),
	}
}

func SaveTariff(tariff *Tariff) {
	tariffMap.Store(tariff.id, tariff)
}

func (r *Tariff) GenerateRates(vectorCnt int) ([]int, error) {
	rateGen := NewRepeatedSequenceGenerator()
	if !ratesGenerator.CompareAndSwap(nil, rateGen) {
		rateGen = ratesGenerator.Load()
	}

	var idx int
	rateGen.Reset(&idx)

	ratesCnt := vectorCnt * r.BatchSize
	rates := make([]int, 0)

	for idx < ratesCnt {
		rates = append(rates, rateGen.ReadSample(r.MaxSampleValue, &idx))
	}
	return rates, nil
}

var ratesGenerator atomic.Pointer[RepeatedSequenceGenerator]
