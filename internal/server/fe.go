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
	GetDecryptionKey(y []int) (FEDecryptionKey, error)
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
		logger.Error("requested EncryptionParams for sensorIdx %d for SingleFEParams", sensorIdx)
		return nil, fmt.Errorf("sensorIdx out of range")
	}

	return &SingleFEEncryptionParams{
		SecKey: feParams.SecKey,
		Params: feParams.Params,
	}, nil
}

func (feParams *SingleFEParams) GetDecryptionKey(y []int) (FEDecryptionKey, error) {
	schema := fullysec.NewFHIPEFromParams(feParams.Params)

	// make []*big.Int from []int
	bigY := make([]*big.Int, len(y))
	for idx, val := range y {
		bigY[idx] = big.NewInt(int64(val))
	}

	fk, err := schema.DeriveKey(data.NewVector(bigY), feParams.SecKey)
	if err != nil {
		logger.Error("Error during key derivation")
	}

	return fk, nil

}

//endregion

//region MultiFEParams

type MultiFEParams struct {
	SensorCnt int
	// todo replace with batchparam?
	BatchCnt int
	Params   *fullysec.FHMultiIPEParams
	SecKey   *fullysec.FHMultiIPESecKey
	PubKey   *bn256.GT
}

func (feParams *MultiFEParams) GetFEParams() any {
	return feParams.Params
}

func (feParams *MultiFEParams) GetSchemaName() string {
	return SchemaFHMultiIPE
}

func (feParams *MultiFEParams) GetEncryptionParams(sensorIdx int) (FEEncryptionParams, error) {
	if sensorIdx < 0 || sensorIdx >= feParams.SensorCnt {
		err := fmt.Errorf("sensorIdx out of range (%d sensors, got %d )", feParams.SensorCnt, sensorIdx)
		logger.Err(err)
		return nil, err
	}

	// todo change NumClients or not ???????????????
	// every server submits batchCnt batches of samples
	return &MultiFEEncryptionParams{
		SecKeys: feParams.SecKey.BHat[sensorIdx*feParams.BatchCnt : (sensorIdx+1)*feParams.BatchCnt],
		Params:  feParams.Params,
	}, nil
}

func (feParams *MultiFEParams) GetDecryptionKey(y []int) (FEDecryptionKey, error) {

	schema := fullysec.NewFHMultiIPEFromParams(feParams.Params)

	// make data.Matrix from []int
	vectors := make([]data.Vector, feParams.BatchCnt*feParams.SensorCnt)
	for row := 0; row < len(y); row++ {
		bigY := make([]*big.Int, len(y))
		for _, val := range y {
			bigY = append(bigY, big.NewInt(int64(val)))
		}
		vectors = append(vectors, data.NewVector(bigY))
	}

	matrix, err := data.NewMatrix(vectors)
	if err != nil {
		logger.Error("error during matrix creation: %s", err)
	}

	fk, err := schema.DeriveKey(matrix, feParams.SecKey)
	if err != nil {
		logger.Error("error during key derivation: %s", err)
	}

	return &fk, nil

}

//endregion

//region FEDecryptionParams

// fixme not used for now, as only Cipher is sent during cipher submission
type FEDecryptionParams interface {
}

type SingleFEDecryptionParams struct {
	Key SingleFEDecryptionKey `json:"decryptionKey"`
}

type MultiFEDecryptionParams struct {
	Key MultiFEDecryptionKey `json:"decryptionKey"`
}

//endregion

//region FEDecryptionKey

type FEDecryptionKey interface{}

type SingleFEDecryptionKey = *fullysec.FHIPEDerivedKey
type MultiFEDecryptionKey = *data.MatrixG2

//endregion
