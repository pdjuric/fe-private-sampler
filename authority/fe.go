package authority

import (
	. "fe/common"
	"fmt"
	"github.com/fentec-project/bn256"
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
	"time"
)

//region FEParamGenerator

type FEParamGenerator interface {
	GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error)
	GetDecryptionParams(y []int) FEDecryptionParams
}

//endregion

//region SingleFEParamGenerator

type SingleFEParamGenerator struct {
	SchemaParams *SingleFESchemaParams
	SecKey       *fullysec.FHIPESecKey

	DecryptionKeyTime time.Duration

	logger *Logger
}

func (g *SingleFEParamGenerator) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx != 0 {
		return nil, fmt.Errorf("requested EncryptionParams for sensorIdx %d for SingleFEParamGenerator", sensorIdx)
	}

	return &SingleFEEncryptionParams{
		SecKey:       g.SecKey,
		SchemaParams: g.SchemaParams,
	}, nil
}

func (g *SingleFEParamGenerator) GetDecryptionParams(y []int) FEDecryptionParams {
	schema := fullysec.NewFHIPEFromParams(g.SchemaParams)

	bigY := make([]*big.Int, len(y))
	for idx, val := range y {
		bigY[idx] = big.NewInt(int64(val))
	}

	g.logger.Info("deriving decryption key")
	start := time.Now()
	fk, err := schema.DeriveKey(data.NewVector(bigY), g.SecKey)
	elapsed := time.Since(start)
	g.DecryptionKeyTime = elapsed
	g.logger.Info("elapsed: %d ns", elapsed.Nanoseconds())
	if err != nil {
		g.logger.Err(err)
		g.logger.Error("deriving decryption key failed")
		return nil
	}

	return &SingleFEDecryptionParams{
		SchemaParams:  *g.SchemaParams,
		DecryptionKey: *fk,
	}
}

//endregion

//region MultiFEParamGenerator

type MultiFEParamGenerator struct {
	SensorCnt        int
	BatchesPerSensor int
	SchemaParams     *MultiFESchemaParams
	SecKey           *fullysec.FHMultiIPESecKey
	PubKey           *bn256.GT

	DecryptionKeyTime time.Duration

	logger *Logger
}

func (g *MultiFEParamGenerator) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx < 0 || sensorIdx >= g.SensorCnt {
		return nil, fmt.Errorf("sensorIdx out of range (%d sensors, got %d )", g.SensorCnt, sensorIdx)
	}

	// every sensor submits batchCnt batches of samples, so it needs exactly BatchesPerSensor SecKeys
	encryptionParams := &MultiFEEncryptionParams{
		IdxOffset:    sensorIdx * g.BatchesPerSensor,
		SecKeys:      g.SecKey.BHat[sensorIdx*g.BatchesPerSensor : (sensorIdx+1)*g.BatchesPerSensor],
		SchemaParams: g.SchemaParams,
	}

	// overwrite SchemaParams.NumClients with BatchesPerSensor
	// this won't make a difference, as this param is not used in the encryption process !!!
	encryptionParams.SchemaParams.NumClients = g.BatchesPerSensor

	return encryptionParams, nil
}

func (g *MultiFEParamGenerator) GetDecryptionParams(y []int) FEDecryptionParams {
	matrix, err := NewMatrix(g.BatchesPerSensor*g.SensorCnt, y, g.SensorCnt)
	// todo handle err

	schema := fullysec.NewFHMultiIPEFromParams(g.SchemaParams)
	g.logger.Info("deriving decryption key")
	start := time.Now()
	fk, err := schema.DeriveKey(matrix, g.SecKey)
	elapsed := time.Since(start)
	g.DecryptionKeyTime = elapsed
	g.logger.Info("elapsed: %d ns", elapsed.Nanoseconds())
	if err != nil {
		g.logger.Err(err)
		g.logger.Error("deriving decryption key failed")
		return nil
	}

	return &MultiFEDecryptionParams{
		SchemaParams:  *g.SchemaParams,
		DecryptionKey: fk,
		PubKey:        g.PubKey,
	}
}

//endregion

// region DummyGenerator

type DummyGenerator struct {
	BatchCnt  int
	BatchSize int

	logger *Logger
}

func (g *DummyGenerator) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx != 0 {
		return nil, fmt.Errorf("requested EncryptionParams for sensorIdx %d for SingleFEParamGenerator", sensorIdx)
	}

	return &DummyEncryptionParams{}, nil
}

func (g *DummyGenerator) GetDecryptionParams(y []int) FEDecryptionParams {
	matrix := make([][]*big.Int, g.BatchCnt)
	for i := 0; i < g.BatchCnt; i++ {
		matrix[i] = make([]*big.Int, g.BatchSize)
		for j := 0; j < g.BatchSize; j++ {
			matrix[i][j] = big.NewInt(int64(y[i*g.BatchCnt+j]))
		}
	}

	return DummyDecryptionParams{BatchCnt: g.BatchCnt, Rates: matrix}
}

//endregion
