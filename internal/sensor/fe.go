package sensor

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"time"
)

//region FEEncryptor

type FEEncryptor interface {
	EncryptBatch(b *Batch) (int, error)
	GetCipher(idx int) (any, error)
}

func NewFEEncryptor(schema string, params FEEncryptionParams) FEEncryptor {
	switch schema {
	case SchemaFHIPE:
		return &SingleFEEncryptor{
			SingleFEEncryptionParams: params.(*SingleFEEncryptionParams),
		}

	case SchemaFHMultiIPE:
		return &MultiFEEncryptor{
			MultiFEEncryptionParams: params.(*MultiFEEncryptionParams),
			ciphers:                 make([]MultiFECipher, 0),
		}
	default:
		// will never happen
		return nil
	}
}

//endregion

//region SingleFEEncryptor

type SingleFEEncryptor struct {
	*SingleFEEncryptionParams
	cipher SingleFECipher
}

func (e *SingleFEEncryptor) EncryptBatch(b *Batch) (int, error) {
	// generate schema
	schema := fullysec.NewFHIPEFromParams(e.Params)

	// encrypt + measure time
	start := time.Now()
	cipher, err := schema.Encrypt(b.samples, e.SecKey)
	end := time.Now() // todo elapsed := time.Since(start)
	b.encryptionTime = end.Sub(start)
	if err != nil {
		return -1, err
	}

	// store encrypted Batch
	e.cipher = cipher

	return 0, nil
}

func (e *SingleFEEncryptor) GetCipher(idx int) (any, error) {
	// assert that idx is 0
	if idx != 0 {
		return nil, fmt.Errorf("cipher at index %d does not exist in Single FE scheme", idx)
	}

	// assert that cipher is present (that batch is encrypted)
	if e.cipher == nil {
		return nil, fmt.Errorf("cipher at index 0 is not yet present - the batch is not (probably) yet encrypted")
	}

	return e.cipher, nil
}

//endregion

//region MultiFEEncryptor

type MultiFEEncryptor struct {
	*MultiFEEncryptionParams
	ciphers []MultiFECipher
}

// EncryptBatch returns index of the cipher, so the caller can find it later
func (e *MultiFEEncryptor) EncryptBatch(b *Batch) (int, error) {
	x := data.NewVector(b.samples)

	// generate schema
	schema := fullysec.NewFHMultiIPEFromParams(e.Params)

	// encrypt + measure time
	start := time.Now()
	cipher, err := schema.Encrypt(x, e.SecKeys[b.idx])
	end := time.Now()
	b.encryptionTime = end.Sub(start)
	if err != nil {
		return -1, err
	}

	// store encrypted Batch
	e.ciphers[b.idx] = &cipher

	return b.idx, nil
}

func (e *MultiFEEncryptor) GetCipher(idx int) (any, error) {
	// assert that idx is in range [0, batchCnt)
	if idx < 0 || idx >= len(e.ciphers) {
		return nil, fmt.Errorf("cipher at index %d does not exist in Multi FE scheme with %d batches", idx, len(e.ciphers))
	}

	// assert that cipher is present (that batch is encrypted)
	if e.ciphers[idx] == nil {
		return nil, fmt.Errorf("cipher at index %d is not yet present - the batch is not (probably) yet encrypted", idx)
	}

	return e.ciphers[idx], nil
}

//endregion
