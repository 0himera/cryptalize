package infra

import "github.com/google/uuid"

type UUIDGenerator struct{}

func (UUIDGenerator) NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
