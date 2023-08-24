package authority

import (
	. "fe/internal/common"
	"fmt"
	"github.com/fentec-project/bn256"
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
)

//region FEParams

type FEParams interface {
	GetFESchemaParams() FESchemaParams
	GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error)
	GetDecryptionParams(y []int) (FEDecryptionParams, error)
}

//endregion

//region SingleFEParams

type SingleFEParams struct {
	SchemaParams *SingleFESchemaParams
	SecKey       *fullysec.FHIPESecKey
}

func (feParams *SingleFEParams) GetFESchemaParams() FESchemaParams {
	return feParams.SchemaParams
}

func (feParams *SingleFEParams) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx != 0 {
		return nil, fmt.Errorf("requested EncryptionParams for sensorIdx %d for SingleFEParams", sensorIdx)
	}

	return &SingleFEEncryptionParams{
		SecKey:       feParams.SecKey,
		SchemaParams: feParams.SchemaParams,
	}, nil
}

func (feParams *SingleFEParams) GetDecryptionParams(y []int) (FEDecryptionParams, error) {
	schema := fullysec.NewFHIPEFromParams(feParams.SchemaParams)

	bigY := make([]*big.Int, len(y))
	for idx, val := range y {
		bigY[idx] = big.NewInt(int64(val))
	}

	fk, err := schema.DeriveKey(data.NewVector(bigY), feParams.SecKey)
	if err != nil {
		return nil, fmt.Errorf("error during key derivation")
	}

	return &SingleFEDecryptionParams{
		DecryptionKey: *fk,
	}, nil
}

//endregion

//region MultiFEParams

type MultiFEParams struct {
	SensorCnt int
	// todo replace with batchparam?
	BatchesPerSensor int
	SchemaParams     *MultiFESchemaParams
	SecKey           *fullysec.FHMultiIPESecKey
	PubKey           *bn256.GT
}

func (feParams *MultiFEParams) GetFESchemaParams() FESchemaParams {
	return feParams.SchemaParams
}

func (feParams *MultiFEParams) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx < 0 || sensorIdx >= feParams.SensorCnt {
		return nil, fmt.Errorf("sensorIdx out of range (%d sensors, got %d )", feParams.SensorCnt, sensorIdx)
	}

	// every sensor submits batchCnt batches of samples, so it needs exactly BatchesPerSensor SecKeys
	encryptionParams := &MultiFEEncryptionParams{
		IdxOffset:    sensorIdx * feParams.BatchesPerSensor,
		SecKeys:      feParams.SecKey.BHat[sensorIdx*feParams.BatchesPerSensor : (sensorIdx+1)*feParams.BatchesPerSensor],
		SchemaParams: feParams.SchemaParams,
	}

	// overwrite SchemaParams.NumClients with BatchesPerSensor
	// this won't make a difference, as this param is not used in the encryption process !!!
	encryptionParams.SchemaParams.NumClients = feParams.BatchesPerSensor

	return encryptionParams, nil
}

func (feParams *MultiFEParams) GetDecryptionParams(y []int) (FEDecryptionParams, error) {
	// make data.Matrix from []int
	vecLen := feParams.SchemaParams.VecLen
	cnt := feParams.BatchesPerSensor * feParams.SensorCnt * vecLen
	matrix := make([]data.Vector, cnt)
	i := -1
	for j, val := range y {
		if j%vecLen == 0 {
			i++
			matrix[i] = make([]*big.Int, vecLen)
		}
		matrix[i][j%vecLen] = big.NewInt(int64(val))
	}

	schema := fullysec.NewFHMultiIPEFromParams(feParams.SchemaParams)
	fk, err := schema.DeriveKey(matrix, feParams.SecKey)
	if err != nil {
		return nil, fmt.Errorf("error during key derivation: %s", err)
	}

	return &MultiFEDecryptionParams{
		DecryptionKey: fk,
		PubKey:        feParams.PubKey,
	}, nil
}

//endregion
