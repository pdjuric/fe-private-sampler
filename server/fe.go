package server

import (
	. "fe/common"
	"fmt"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"sync/atomic"
	"time"
)

//region FEDecryptor

type FEDecryptor interface {
	AddCipher(feCipher FECipher) (*big.Int, error)
	SetDecryptionParams(feParams FEDecryptionParams)
}

type SingleFEDecryptor struct {
	Schema *fullysec.FHIPE

	DecryptionParamsId      UUID
	DecryptionParams        *SingleFEDecryptionParams `json:"decryptionKey"`
	DecryptionParamsFetched atomic.Bool

	ResultReady    atomic.Bool
	DecryptionTime time.Duration
}

type MultiFEDecryptor struct {
	*fullysec.FHMultiIPEParallelDecryption

	DecryptionParamsId      UUID
	DecryptionParams        *MultiFEDecryptionParams `json:"decryptionKey"`
	DecryptionParamsFetched atomic.Bool

	ReceivedCipherFlags    []atomic.Bool
	PartialProcessingTimes []time.Duration
	ResultReady            atomic.Bool
	DecryptionTime         time.Duration
}

//endregion

func NewFEDecryptor(params FESchemaParams) (FEDecryptor, error) {
	switch params.(type) {

	case *SingleFESchemaParams:
		return &SingleFEDecryptor{
			Schema: fullysec.NewFHIPEFromParams(params.(*SingleFESchemaParams)),
		}, nil

	case *MultiFESchemaParams:
		schema := fullysec.NewFHMultiIPEFromParams(params.(*MultiFESchemaParams))
		vecCnt := params.(*MultiFESchemaParams).NumClients

		return &MultiFEDecryptor{
			FHMultiIPEParallelDecryption: schema.NewParallelDecryption(),
			ReceivedCipherFlags:          make([]atomic.Bool, vecCnt),
			PartialProcessingTimes:       make([]time.Duration, vecCnt),
		}, nil
	}

	return nil, nil

}

//region MultiFEParams

func (p *SingleFEDecryptor) AddCipher(feCipher FECipher) (*big.Int, error) {
	cipher := feCipher.(*SingleFECipher)

	start := time.Now()
	res, err := p.Schema.Decrypt(cipher, &p.DecryptionParams.DecryptionKey)
	//p.DecryptionTime = time.Since(start)
	fmt.Printf("cipher no 0, time %d ms", time.Since(start).Milliseconds())
	if err != nil {
		return nil, err
	}

	p.ResultReady.Store(true)
	return res, nil
}

func (p *SingleFEDecryptor) SetDecryptionParams(feParams FEDecryptionParams) {
	params := feParams.(*SingleFEDecryptionParams)
	p.DecryptionParams = params
	p.DecryptionParamsFetched.Store(true)
}

func (p *MultiFEDecryptor) AddCipher(feCipher FECipher) (*big.Int, error) {
	cipher := feCipher.(*MultiFECipher)

	start := time.Now()
	remainingBatches, err := p.ParallelDecryption(cipher.Idx, cipher.Payload, p.DecryptionParams.DecryptionKey)
	p.PartialProcessingTimes[cipher.Idx] = time.Since(start)
	fmt.Printf("cipher no %d, time %d ms", cipher.Idx, p.PartialProcessingTimes[cipher.Idx].Milliseconds())
	if err != nil {
		return nil, err
	}

	p.ReceivedCipherFlags[cipher.Idx].Store(true)

	if remainingBatches == 0 {
		start = time.Now()
		result, err := p.GetResult(true, p.DecryptionParams.PubKey)
		p.DecryptionTime = time.Since(start)
		if err != nil {
			return nil, err
		}
		// todo what to dop with the result?
		p.ResultReady.Store(true)
		return result, nil
		//task.logger.Info("task %s Result %s", task.Id, result.String())
		//fmt.Printf("task %s Result %s\n", task.Id, result.String())
	}

	return nil, nil
}

func (p *MultiFEDecryptor) SetDecryptionParams(feParams FEDecryptionParams) {
	params := feParams.(*MultiFEDecryptionParams)
	p.DecryptionParams = params
	p.DecryptionParamsFetched.Store(true)
}
