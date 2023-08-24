package sensor

import (
	. "fe/internal/common"
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

	//EncryptedFlag     	atomic.Bool
	//EncryptionTime		time.Duration
}

type MultiFEEncryptor struct {
	Schema        *fullysec.FHMultiIPE
	IdxOffset     int
	EncryptionKey []data.Matrix

	//BatchEncryptedFlag     []atomic.Bool
	//EncryptionTimes		 []time.Duration
}

func NewFEEncryptor(feParams FEEncryptionParams) FEEncryptor {
	switch feParams.(type) {

	case *SingleFEEncryptionParams:
		params := feParams.(*SingleFEEncryptionParams)
		return &SingleFEEncryptor{
			Schema:        fullysec.NewFHIPEFromParams(params.SchemaParams),
			EncryptionKey: params.SecKey,
		}

	case *MultiFEEncryptionParams:
		params := feParams.(*MultiFEEncryptionParams)
		return &MultiFEEncryptor{
			Schema:        fullysec.NewFHMultiIPEFromParams(params.SchemaParams),
			IdxOffset:     params.IdxOffset,
			EncryptionKey: params.SecKeys,
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
	encryptionTime := time.Since(start)
	if err != nil {
		return nil, encryptionTime, err
	}

	return cipher, encryptionTime, nil
}

func (e *MultiFEEncryptor) Encrypt(batch *Batch) (FECipher, time.Duration, error) {
	// todo check batchIdx bound!

	// encrypt + measure time
	start := time.Now()
	cipher, err := e.Schema.Encrypt(batch.samples, e.EncryptionKey[batch.idx])
	encryptionTime := time.Since(start)
	if err != nil {
		return nil, encryptionTime, err
	}

	return &MultiFECipher{
		Idx:     batch.idx + e.IdxOffset,
		Payload: cipher,
	}, encryptionTime, nil
}
