package common

import (
	"bytes"
	"encoding/gob"
)

func GobInit() {
	gob.Register(&MultiFESchemaParams{})
	gob.Register(&SingleFESchemaParams{})
	gob.Register(&SingleFECipher{})
	gob.Register(&MultiFECipher{})
	gob.Register(&MultiFEDecryptionParams{})
	gob.Register(&SingleFEDecryptionParams{})
	gob.Register(&MultiFEEncryptionParams{})
	gob.Register(&SingleFEEncryptionParams{})
}

func Encode(data any) ([]byte, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)

	if err := enc.Encode(&data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Decode(bytess []byte) (any, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(bytess))
	var data any

	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
