package server

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
	GetFEParams() any
	GetSchemaName() string
	GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error)
	GetDecryptionParams(y []int) (FEDecryptionParams, error)
}

//endregion

//region SingleFEParams

type SingleFEParams struct {
	Params *fullysec.FHIPEParams
	SecKey *fullysec.FHIPESecKey
}

func (feParams *SingleFEParams) GetFEParams() any {
	return feParams.Params
}

func (feParams *SingleFEParams) GetSchemaName() string {
	return SchemaFHIPE
}

func (feParams *SingleFEParams) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx != 0 {
		return nil, fmt.Errorf("requested EncryptionParams for sensorIdx %d for SingleFEParams", sensorIdx)
	}

	return &SingleFEEncryptionParams{
		SecKey: feParams.SecKey,
		Params: feParams.Params,
	}, nil
}

func (feParams *SingleFEParams) GetDecryptionParams(y []int) (FEDecryptionParams, error) {
	schema := fullysec.NewFHIPEFromParams(feParams.Params)

	// make []*big.Int from []int
	bigY := make([]*big.Int, len(y))
	for idx, val := range y {
		bigY[idx] = big.NewInt(int64(val))
	}

	fk, err := schema.DeriveKey(data.NewVector(bigY), feParams.SecKey)
	if err != nil {
		return nil, fmt.Errorf("error during key derivation")
	}

	return &SingleFEDecryptionParams{
		DecryptionKey: fk,
	}, nil
}

//endregion

//region MultiFEParams

type MultiFEParams struct {
	SensorCnt int
	// todo replace with batchparam?
	BatchesPerSensor int
	Params           *fullysec.FHMultiIPEParams
	SecKey           *fullysec.FHMultiIPESecKey
	PubKey           *bn256.GT
}

func (feParams *MultiFEParams) GetFEParams() any {
	return feParams.Params
}

func (feParams *MultiFEParams) GetSchemaName() string {
	return SchemaFHMultiIPE
}

func (feParams *MultiFEParams) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx < 0 || sensorIdx >= feParams.SensorCnt {
		return nil, fmt.Errorf("sensorIdx out of range (%d sensors, got %d )", feParams.SensorCnt, sensorIdx)
	}

	// every sensor submits batchCnt batches of samples, so it needs exactly BatchesPerSensor SecKeys
	encryptionParams := &MultiFEEncryptionParams{
		SecKeys: feParams.SecKey.BHat[sensorIdx*feParams.BatchesPerSensor : (sensorIdx+1)*feParams.BatchesPerSensor],
		Params:  *feParams.Params,
	}

	// overwrite Params.NumClients with BatchesPerSensor
	// this won't make a difference, as this param is not used in the encryption process !!!
	encryptionParams.Params.NumClients = feParams.BatchesPerSensor

	return encryptionParams, nil
}

func (feParams *MultiFEParams) GetDecryptionParams(y []int) (FEDecryptionParams, error) {
	// make data.Matrix from []int
	vecLen := feParams.Params.VecLen
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

	schema := fullysec.NewFHMultiIPEFromParams(feParams.Params)
	fk, err := schema.DeriveKey(matrix, feParams.SecKey)
	if err != nil {
		return nil, fmt.Errorf("error during key derivation: %s", err)
	}

	return &MultiFEDecryptionParams{
		FHMultiIPEParallelDecryption: schema.NewParallelDecryption(),
		DecryptionKey:                &fk,
		PubKey:                       feParams.PubKey,
	}, nil
}

//endregion

//region FEDecryptionParams

type FEDecryptionParams interface {
}

type SingleFEDecryptionParams struct {
	DecryptionKey SingleFEDecryptionKey `json:"decryptionKey"`
}

type MultiFEDecryptionParams struct {
	*fullysec.FHMultiIPEParallelDecryption
	DecryptionKey MultiFEDecryptionKey `json:"decryptionKey"`
	PubKey        *bn256.GT
}

//endregion

//region FEDecryptionKey

type FEDecryptionKey interface{}

type SingleFEDecryptionKey = *fullysec.FHIPEDerivedKey
type MultiFEDecryptionKey = *data.MatrixG2

//endregion
