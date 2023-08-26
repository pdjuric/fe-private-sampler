package common

import (
	"github.com/fentec-project/bn256"
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"math/big"
)

type SamplingParams struct {
	Start          int `json:"start"` // timestamp when server resets for the first time and starts measuring
	SamplingPeriod int `json:"samplingPeriod"`
	BatchParams
	MaxSampleValue int `json:"maxSampleValue"`
}

type BatchParams struct {
	BatchSize int `json:"batchSize"`
	BatchCnt  int `json:"batchCnt"`
}

//region FE

//region FESchemaParams

type FESchemaParams any
type SingleFESchemaParams = fullysec.FHIPEParams
type MultiFESchemaParams = fullysec.FHMultiIPEParams

//endregion

//region Cipher

type FECipher any
type SingleFECipher = fullysec.FHIPECipher
type MultiFECipher struct {
	Idx     int
	Payload data.VectorG1
}

type DummyCipher struct {
	Idx     int
	Samples []*big.Int
}

//endregion

// region Encryption

type FEEncryptionParams any

type SingleFEEncryptionParams struct {
	SecKey       *fullysec.FHIPESecKey `json:"secKey"`
	SchemaParams *SingleFESchemaParams `json:"params"`
}

type MultiFEEncryptionParams struct {
	IdxOffset    int                  `json:"idxOffset"`
	SecKeys      []data.Matrix        `json:"secKeys"`
	SchemaParams *MultiFESchemaParams `json:"params"`
}

type DummyEncryptionParams struct {
	IdxOffset int `json:"idxOffset"`
}

//endregion

//region Decryption

type FEDecryptionParams any

type SingleFEDecryptionParams struct {
	SchemaParams  SingleFESchemaParams
	DecryptionKey fullysec.FHIPEDerivedKey
}

type MultiFEDecryptionParams struct {
	SchemaParams  MultiFESchemaParams
	PubKey        *bn256.GT
	DecryptionKey data.MatrixG2
}

type DummyDecryptionParams struct {
	BatchCnt int
	Rates    [][]*big.Int
}

//endregion

//endregion
