package sensor

import (
	. "fe/common"
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"time"
)

type FEEncryptor interface {
	Encrypt(batch *Batch) (FECipher, time.Duration, error)
}

type SingleFEEncryptor struct {
	Schema        *fullysec.FHIPE
	EncryptionKey *fullysec.FHIPESecKey

	logger *Logger
}

type MultiFEEncryptor struct {
	Schema        *fullysec.FHMultiIPE
	IdxOffset     int
	EncryptionKey []data.Matrix

	logger *Logger
}

type DummyEncryptor struct {
	IdxOffset int
	logger    *Logger
}

func NewFEEncryptor(feParams FEEncryptionParams, logger *Logger) FEEncryptor {
	switch feParams.(type) {

	case *SingleFEEncryptionParams:
		params := feParams.(*SingleFEEncryptionParams)
		return &SingleFEEncryptor{
			Schema:        fullysec.NewFHIPEFromParams(params.SchemaParams),
			EncryptionKey: params.SecKey,
			logger:        GetLogger("fe encryptor", logger),
		}

	case *MultiFEEncryptionParams:
		params := feParams.(*MultiFEEncryptionParams)
		return &MultiFEEncryptor{
			Schema:        fullysec.NewFHMultiIPEFromParams(params.SchemaParams),
			IdxOffset:     params.IdxOffset,
			EncryptionKey: params.SecKeys,
			logger:        GetLogger("fe encryptor", logger),
		}

	case *DummyEncryptionParams:
		params := feParams.(*DummyEncryptionParams)
		return &DummyEncryptor{
			IdxOffset: params.IdxOffset,
			logger:    GetLogger("fe encryptor", logger),
		}

	default:
		//todo !!!
		return nil
	}
}

func (e *SingleFEEncryptor) Encrypt(batch *Batch) (FECipher, time.Duration, error) {
	// batchIdx is ignored

	// encrypt + measure time
	start := time.Now()
	cipher, err := e.Schema.Encrypt(batch.samples, e.EncryptionKey)
	elapsed := time.Since(start)
	e.logger.Info("batch no %d encryption time: %d ns", batch.idx, elapsed.Nanoseconds())
	if err != nil {
		return nil, elapsed, err
	}

	return cipher, elapsed, nil
}

func (e *MultiFEEncryptor) Encrypt(batch *Batch) (FECipher, time.Duration, error) {
	// todo check batchIdx bound!

	// encrypt + measure time
	start := time.Now()
	cipher, err := e.Schema.Encrypt(batch.samples, e.EncryptionKey[batch.idx])
	elapsed := time.Since(start)
	e.logger.Info("batch no %d encryption time: %d ns", batch.idx, elapsed.Nanoseconds())
	if err != nil {
		return nil, elapsed, err
	}

	return &MultiFECipher{
		Idx:     batch.idx + e.IdxOffset,
		Payload: cipher,
	}, elapsed, nil
}

func (e *DummyEncryptor) Encrypt(batch *Batch) (FECipher, time.Duration, error) {

	// encrypt + measure time
	start := time.Now()
	cipher := DummyCipher{
		Idx:     batch.idx + e.IdxOffset,
		Samples: batch.samples,
	}
	elapsed := time.Since(start)
	e.logger.Info("batch no %d encryption time: %d ns", batch.idx, elapsed.Nanoseconds())
	return cipher, elapsed, nil

}
