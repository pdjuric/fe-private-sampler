package common

import (
	"encoding/json"
	"fmt"
	"github.com/fentec-project/bn256"
	. "github.com/fentec-project/gofe/data"
	"github.com/fentec-project/gofe/innerprod/fullysec"
	"github.com/google/uuid"
	"math/big"
	"sync"
)

//region data.Matrix

// MarshalJSON and UnmarshalJSON are written for MultiFEEncryptionParams due to problems with data.Matrix (de)serialization.
// data.Matrix is a matrix of big.Ints, which are not (de)serialized properly by default json.Marshal and json.Unmarshal.
// Therefore, we convert data.Matrix to [][]string and back.
// todo edit comment

func marshallMatrix(matrix *Matrix) [][][]byte {
	rows, cols := matrix.Rows(), matrix.Cols()
	matrixString := make([][][]byte, rows)

	var wg sync.WaitGroup

	for rowIdx, row := range *matrix {
		wg.Add(1)

		go func(i int, r Vector) {
			matrixString[i] = make([][]byte, cols)
			for j, value := range r {
				matrixString[i][j] = value.Bytes()
			}
			wg.Done()
		}(rowIdx, row)
	}

	wg.Wait()
	return matrixString
}

// unmarshallMatrix concurrently unmarshalls data.Matrix
func unmarshallMatrix(bytes [][][]byte) Matrix {

	rows, cols := len(bytes), len(bytes[0])
	vectors := make([]Vector, rows)

	var wg sync.WaitGroup

	for rowIdx, row := range bytes {
		wg.Add(1)

		go func(i int, r [][]byte) {
			defer wg.Done()
			vectors[i] = make([]*big.Int, cols)
			for j, valueString := range r {
				vectors[i][j] = big.NewInt(0).SetBytes(valueString)
			}
		}(rowIdx, row)

	}

	wg.Wait()
	return vectors
}

//endregion

// todo with and without *

//region MultiFEEncryptionParams

func (p MultiFEEncryptionParams) MarshalJSON() ([]byte, error) {
	secKeysString := make([][][][]byte, len(p.SecKeys))

	for idx, _ := range p.SecKeys {
		secKeysString[idx] = marshallMatrix(&p.SecKeys[idx])
	}

	kvMap := make(map[string]any)
	kvMap["secKeys"] = secKeysString
	kvMap["params"] = p.Params

	data, err := json.Marshal(kvMap)
	if err != nil {
		return nil, err
	}

	return data, nil

}

func (p *MultiFEEncryptionParams) UnmarshalJSON(data []byte) error {
	var T struct {
		SecKeys [][][][]byte               `json:"secKeys"`
		Params  *fullysec.FHMultiIPEParams `json:"params"`
	}

	err := json.Unmarshal(data, &T)
	if err != nil {
		fmt.Println(err)
		return err
	}

	secKeysCnt := len(T.SecKeys)
	p.SecKeys = make([]Matrix, secKeysCnt)
	for secKeyIdx, secKeyBytes := range T.SecKeys {
		p.SecKeys[secKeyIdx] = unmarshallMatrix(secKeyBytes)
	}

	p.Params = T.Params

	return nil
}

//endregion

//region SingleFEEncryptionParams

func (p SingleFEEncryptionParams) MarshalJSON() ([]byte, error) {
	secKey := make(map[string]any)
	secKey["B"] = marshallMatrix(&p.SecKey.B)
	secKey["BStar"] = marshallMatrix(&p.SecKey.BStar)
	secKey["G1"] = p.SecKey.G1
	secKey["G2"] = p.SecKey.G2

	kvMap := make(map[string]any)
	kvMap["secKey"] = secKey
	kvMap["params"] = p.Params

	data, err := json.Marshal(kvMap)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (p *SingleFEEncryptionParams) UnmarshalJSON(data []byte) error {

	var T struct {
		SecKey struct {
			G1    *bn256.G1  `json:"G1"`
			G2    *bn256.G2  `json:"G2"`
			B     [][][]byte `json:"B"`
			BStar [][][]byte `json:"BStar"`
		} `json:"secKey"`
		Params *fullysec.FHIPEParams `json:"params"`
	}

	err := json.Unmarshal(data, &T)
	if err != nil {
		fmt.Println(err)
		return err
	}

	p.Params = T.Params

	// Unmarshal secKey
	p.SecKey = new(fullysec.FHIPESecKey)
	p.SecKey.G1 = T.SecKey.G1
	p.SecKey.G2 = T.SecKey.G2
	p.SecKey.B = unmarshallMatrix(T.SecKey.B)
	p.SecKey.BStar = unmarshallMatrix(T.SecKey.BStar)

	return nil
}

//endregion

//region SubmitTaskRequest

func (r *SubmitTaskRequest) UnmarshalJSON(data []byte) error {

	type submitTaskRequestT struct {
		TaskId uuid.UUID `json:"id"`
		BatchParams
		SamplingParams

		Schema string
	}

	var taskRequest submitTaskRequestT
	err := json.Unmarshal(data, &taskRequest)
	if err != nil {
		return err
	}

	switch taskRequest.Schema {
	case SchemaFHIPE:
		tempParamsStruct := struct {
			Params *SingleFEEncryptionParams `json:"FEEncryptionParams"`
		}{&SingleFEEncryptionParams{}}

		err = json.Unmarshal(data, &tempParamsStruct)
		if err != nil {
			return err
		}

		r.FEEncryptionParams = tempParamsStruct.Params

	case SchemaFHMultiIPE:
		tempParamsStruct := struct {
			Params *MultiFEEncryptionParams `json:"FEEncryptionParams"`
		}{&MultiFEEncryptionParams{}}

		err = json.Unmarshal(data, &tempParamsStruct)
		if err != nil {
			return err
		}

		r.FEEncryptionParams = tempParamsStruct.Params

	default:
		return fmt.Errorf("unknown schema %s", taskRequest.Schema)
	}

	r.TaskId = taskRequest.TaskId
	r.BatchParams = taskRequest.BatchParams
	r.SamplingParams = taskRequest.SamplingParams
	r.Schema = taskRequest.Schema
	return nil
}

//endregion
