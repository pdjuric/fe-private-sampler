package common

import (
	"github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
)

type SamplingParams struct {
	Start          int `json:"start"` // timestamp when server resets for the first time and starts measuring
	SamplingPeriod int `json:"samplingPeriod"`
	SampleCount    int `json:"sampleCount"` // must be MeasuringCount % SubmissionPeriod == 0
	MaxSampleValue int `json:"maxSampleValue"`
}

type BatchParams struct {
	BatchSize int `json:"batchSize"`
	BatchCnt  int `json:"batchCnt"`
}

//region FE

// region Encryption

type FEEncryptionParams interface {
}

type SingleFEEncryptionParams struct {
	SecKey *fullysec.FHIPESecKey `json:"secKey"`
	Params *fullysec.FHIPEParams `json:"params"`
}

type MultiFEEncryptionParams struct {
	SecKeys []data.Matrix             `json:"secKeys"`
	Params  fullysec.FHMultiIPEParams `json:"params"`
}

//endregion

//region Cipher

type FECipher = any
type SingleFECipher = fullysec.FHIPECipher
type MultiFECipher = data.VectorG1

//endregion

//endregion
