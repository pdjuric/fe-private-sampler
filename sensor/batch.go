package sensor

import (
	. "fe/common"
	"math/big"
	"sync/atomic"
	"time"
)

type Batch struct {
	idx int

	samples            []*big.Int
	receivedSamplesCnt atomic.Int32
	totalSamplesCnt    int32

	cipher         FECipher
	encryptionTime time.Duration
	isEncrypted    atomic.Bool

	isSubmitted atomic.Bool
}

func (b *Batch) InitBatch(idx int, samplesCnt int) {
	b.idx = idx
	b.samples = make([]*big.Int, samplesCnt)
	b.totalSamplesCnt = int32(samplesCnt)
}

// AddSample adds a sample to the non-full batch and returns true if the batch is full. Adding to the full batch
// will throw an error.
func (b *Batch) AddSample(sample int) bool {
	newSampleIdx := b.receivedSamplesCnt.Add(1) - 1
	b.samples[newSampleIdx] = big.NewInt(int64(sample))

	return newSampleIdx == b.totalSamplesCnt-1
}

func (b *Batch) GetSamples() []int32 {
	sampleCnt := b.receivedSamplesCnt.Load()
	if sampleCnt == 0 {
		return nil
	}

	samples := make([]int32, sampleCnt)
	var idx int32
	for idx = 0; idx < sampleCnt; idx++ {
		samples[idx] = int32(b.samples[idx].Int64())
	}

	return samples
}
