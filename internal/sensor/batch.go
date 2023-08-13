package sensor

import (
	. "fe/internal/common"
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

// AddSample adds a sample to the batch and returns true if the batch is full
func (b *Batch) AddSample(sample int) bool {
	newSampleIdx := b.receivedSamplesCnt.Add(1) - 1
	b.samples[newSampleIdx] = big.NewInt(int64(sample))

	return newSampleIdx == b.totalSamplesCnt-1
}

func (b *Batch) Encrypt(schemaName string, feEncryptionParams FEEncryptionParams) error {
	switch schemaName {
	case SchemaFHIPE:
		cipher, elapsedTime, err := SingleFEEncrypt(feEncryptionParams.(*SingleFEEncryptionParams), b.samples)
		b.cipher = cipher
		b.encryptionTime = elapsedTime
		return err

	case SchemaFHMultiIPE:
		cipher, elapsedTime, err := MultiFEEncrypt(b.idx, feEncryptionParams.(*MultiFEEncryptionParams), b.samples)
		b.cipher = cipher
		b.encryptionTime = elapsedTime
		return err

	default:
		// will never happen
		return nil
	}
}
