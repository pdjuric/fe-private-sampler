package server

import (
	. "fe/common"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"sync/atomic"
	"time"
)

//region FEDecryptor

type FEDecryptor interface {
	AddCipher(feCipher FECipher) (*big.Int, error)
	GetStats() any
}

type SingleFEDecryptor struct {
	*fullysec.FHIPE
	*SingleFEDecryptionParams

	Result         *big.Int
	ResultReady    atomic.Bool
	DecryptionTime *time.Duration

	logger *Logger
}

type MultiFEDecryptor struct {
	*fullysec.FHMultiIPEParallelDecryption
	*MultiFEDecryptionParams

	ReceivedCiphers        []atomic.Bool
	PartialProcessingTimes []*time.Duration

	Result              *big.Int
	ResultReady         atomic.Bool
	DecryptionTime      *time.Duration
	TotalDecryptionTime int64 //in nanoseconds

	logger *Logger
}

type DummyDecryptor struct {
	*DummyDecryptionParams

	RemainingCiphers int64
	DecryptionTime   atomic.Int64

	Result      *big.Int
	ResultReady atomic.Bool

	logger *Logger
}

//endregion

func NewFEDecryptor(params FEDecryptionParams, logger *Logger) (FEDecryptor, error) {
	switch params.(type) {

	case *SingleFEDecryptionParams:
		feParams := params.(*SingleFEDecryptionParams)
		return &SingleFEDecryptor{
			SingleFEDecryptionParams: feParams,
			FHIPE:                    fullysec.NewFHIPEFromParams(&feParams.SchemaParams),
			logger:                   GetLogger("fe decryptor", logger),
		}, nil

	case *MultiFEDecryptionParams:
		feParams := params.(*MultiFEDecryptionParams)
		schema := fullysec.NewFHMultiIPEFromParams(&feParams.SchemaParams)
		vecCnt := feParams.SchemaParams.NumClients

		return &MultiFEDecryptor{
			FHMultiIPEParallelDecryption: schema.NewParallelDecryption(),
			MultiFEDecryptionParams:      feParams,
			ReceivedCiphers:              make([]atomic.Bool, vecCnt),
			PartialProcessingTimes:       make([]*time.Duration, vecCnt),
			logger:                       GetLogger("fe decryptor", logger),
		}, nil

	case *DummyDecryptionParams:
		feParams := params.(*DummyDecryptionParams)

		return &DummyDecryptor{
			DummyDecryptionParams: feParams,
			Result:                big.NewInt(0),
			RemainingCiphers:      int64(feParams.BatchCnt),
			logger:                GetLogger("dummy decryptor", logger),
		}, nil

	}

	return nil, nil

}

//region SingleFEDecryptor

func (p *SingleFEDecryptor) AddCipher(feCipher FECipher) (*big.Int, error) {
	cipher := feCipher.(*SingleFECipher)

	start := time.Now()
	res, err := p.Decrypt(cipher, &p.DecryptionKey)
	elapsed := time.Since(start)
	p.DecryptionTime = &elapsed
	p.logger.Info("cipher no 0: decryption time: %d ns", time.Since(start).Nanoseconds())
	if err != nil {
		return nil, err
	}

	p.ResultReady.Store(true)
	return res, nil
}

func (p *SingleFEDecryptor) GetStats() any {
	stats := struct {
		Finished        bool   `json:"finished"`
		DecryptionTime  *int64 `json:"decryption_time"`
		TotalCiphers    int    `json:"total_ciphers"`
		ReceivedCiphers int    `json:"received_ciphers"`
	}{}

	stats.Finished = p.ResultReady.Load()
	stats.TotalCiphers = 1
	stats.DecryptionTime = GetIntPtrFromDuration(p.DecryptionTime)
	if stats.Finished {
		stats.ReceivedCiphers = 1
	} else {
		stats.ReceivedCiphers = 0
	}
	return stats
}

//endregion

//region MultiFEDecryptor

func (p *MultiFEDecryptor) AddCipher(feCipher FECipher) (*big.Int, error) {
	cipher := feCipher.(*MultiFECipher)

	start := time.Now()
	remainingBatches, err := p.ParallelDecryption(cipher.Idx, cipher.Payload, p.DecryptionKey)
	elapsed := time.Since(start)
	p.PartialProcessingTimes[cipher.Idx] = &elapsed
	p.logger.Info("cipher no %d: partial processing time: %d ns", cipher.Idx, elapsed.Nanoseconds())
	if err != nil {
		return nil, err
	}

	p.ReceivedCiphers[cipher.Idx].Store(true)

	if remainingBatches == 0 {
		start = time.Now()
		result, err := p.GetResult(false, p.PubKey)
		elapsed = time.Since(start)
		p.DecryptionTime = &elapsed
		p.logger.Info("decryption time: %d ns", p.DecryptionTime.Nanoseconds())
		if err != nil {
			return nil, err
		}

		sum := elapsed.Nanoseconds()
		for _, ppt := range p.PartialProcessingTimes {
			sum += ppt.Nanoseconds()
		}
		p.logger.Info("total decryption time: %d ns", sum)
		p.TotalDecryptionTime = sum

		p.Result = result
		p.ResultReady.Store(true)
		return result, nil
	}

	return nil, nil
}

func (p *MultiFEDecryptor) GetStats() any {
	stats := struct {
		Finished               bool           `json:"finished"`
		DecryptionTime         *int64         `json:"decryption_time"`
		TotalDecryptionTime    *int64         `json:"total_decryption_time"`
		TotalCiphers           int            `json:"total_ciphers"`
		ReceivedCiphers        int            `json:"received_ciphers"`
		PartialProcessingTimes map[int]*int64 `json:"partial_processing_times"`
	}{}

	stats.Finished = p.ResultReady.Load()
	if stats.Finished {
		stats.TotalDecryptionTime = &p.TotalDecryptionTime
	}
	stats.TotalCiphers = p.SchemaParams.NumClients
	stats.DecryptionTime = GetIntPtrFromDuration(p.DecryptionTime)

	stats.PartialProcessingTimes = make(map[int]*int64)
	for i := 0; i < stats.TotalCiphers; i++ {
		if p.ReceivedCiphers[i].Load() {
			stats.PartialProcessingTimes[i] = GetIntPtrFromDuration(p.PartialProcessingTimes[i])
		}
	}

	return stats
}

//endregion

//region DummyDecryptor

func (p *DummyDecryptor) AddCipher(feCipher FECipher) (*big.Int, error) {
	cipher := feCipher.(*DummyCipher)

	start := time.Now()
	samplesCnt := len(cipher.Samples)
	sum := big.NewInt(0)
	for i := 0; i < samplesCnt; i++ {
		product := big.NewInt(0).Add(cipher.Samples[i], p.DummyDecryptionParams.Rates[cipher.Idx][i])
		sum = sum.Add(sum, product)
	}
	p.Result.Add(p.Result, sum)
	elapsed := time.Since(start)
	p.DecryptionTime.Add(elapsed.Nanoseconds())
	p.logger.Info("cipher processing time: %d ns", elapsed.Nanoseconds())

	if atomic.AddInt64(&p.RemainingCiphers, -1) == 0 {
		p.logger.Info("total decryption time: %d ns", p.DecryptionTime.Load())
		return p.Result, nil
	} else {
		return nil, nil
	}
}

func (p *DummyDecryptor) GetStats() any {
	stats := struct {
		Finished       bool   `json:"finished"`
		DecryptionTime *int64 `json:"decryption_time"`
	}{}

	stats.Finished = p.ResultReady.Load()
	decryptionTime := p.DecryptionTime.Load()
	stats.DecryptionTime = &decryptionTime
	return stats
}

//endregion
