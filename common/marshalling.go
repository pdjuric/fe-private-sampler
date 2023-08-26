package common

import (
	"bytes"
	"encoding/gob"
)

// GobInit registers all the structs that can be marshalled.
func GobInit() {
	gob.Register(&MultiFESchemaParams{})
	gob.Register(&SingleFESchemaParams{})

	gob.Register(&SingleFECipher{})
	gob.Register(&MultiFECipher{})
	gob.Register(&DummyCipher{})

	gob.Register(&MultiFEDecryptionParams{})
	gob.Register(&SingleFEDecryptionParams{})
	gob.Register(&DummyDecryptionParams{})

	gob.Register(&MultiFEEncryptionParams{})
	gob.Register(&SingleFEEncryptionParams{})
	gob.Register(&DummyEncryptionParams{})
}

func Encode(data any) ([]byte, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)

	if err := enc.Encode(&data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Decode(bytesToDecode []byte) (any, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(bytesToDecode))
	var data any

	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
