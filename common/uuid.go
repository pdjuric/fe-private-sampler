package common

import (
	"fmt"
	_uuid "github.com/google/uuid"
)

type UUID string

func (uuid UUID) Verify() bool {
	_, err := _uuid.Parse(string(uuid))
	return err == nil
}

func (uuid UUID) IsNil() bool {
	return string(uuid) == ""
}

func NewUUID() UUID {
	return UUID(_uuid.New().String())
}

func NewUUIDFromString(uuidString string) (UUID, error) {
	uuid := UUID(uuidString)
	if !uuid.Verify() {
		return UUID(""), fmt.Errorf("invalid UUID: %s", uuidString)
	} else {
		return uuid, nil
	}
}
