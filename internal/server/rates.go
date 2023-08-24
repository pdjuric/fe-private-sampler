package server

import (
	. "fe/internal/common"
	"github.com/fentec-project/gofe/sample"
	"math/big"
	"sync"
)

type Rate struct {
	id             UUID
	Description    string `json:"description"`
	SamplingPeriod int    `json:"samplingPeriod"`
	BatchSize      int    `json:"batchSize"`
	MaxSampleValue int    `json:"maxSampleValue"`
	MaxRateValue   int    `json:"maxRateValue"`
}

var rateMap = sync.Map{}

func GetRate(rateId UUID) (*Rate, bool) {
	rate, exists := rateMap.Load(rateId)
	return rate.(*Rate), exists

}

func NewRate() *Rate {
	return &Rate{
		id: NewUUID(),
	}
}

func SaveRate(rate *Rate) {
	rateMap.Store(rate.id, rate)
}

func (r *Rate) GenerateCoefficients(vectorCnt int) ([]int, error) {
	sampler := sample.NewUniform(big.NewInt(int64(r.MaxRateValue)))
	coeffCnt := vectorCnt * r.BatchSize
	coefficients := make([]int, coeffCnt)

	for i := 0; i < coeffCnt; i++ {
		temp, err := sampler.Sample()
		if err != nil {
			return nil, err
		}
		coefficients[i] = int(temp.Int64())
	}
	return coefficients, nil
}
