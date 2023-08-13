package sensor

import (
	. "fe/internal/common"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"time"
)

func SingleFEEncrypt(params *SingleFEEncryptionParams, samples []*big.Int) (*SingleFECipher, time.Duration, error) {
	// generate schema
	schema := fullysec.NewFHIPEFromParams(params.Params)

	// encrypt + measure time
	start := time.Now()
	cipher, err := schema.Encrypt(samples, params.SecKey)
	encryptionTime := time.Since(start)
	if err != nil {
		return nil, encryptionTime, err
	}

	return cipher, encryptionTime, nil
}

func MultiFEEncrypt(batchIdx int, params *MultiFEEncryptionParams, samples []*big.Int) (*MultiFECipher, time.Duration, error) {
	// generate schema
	schema := fullysec.NewFHMultiIPEFromParams(&params.Params)

	// encrypt + measure time
	start := time.Now()
	cipher, err := schema.Encrypt(samples, params.SecKeys[batchIdx])
	encryptionTime := time.Since(start)
	if err != nil {
		return nil, encryptionTime, err
	}

	return &cipher, encryptionTime, nil
}
