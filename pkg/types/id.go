package types

import (
	"github.com/google/uuid"
)

type ID = uuid.UUID

func NewID() ID {
	return uuid.New()
}

func ParseID(s string) (ID, error) {
	return uuid.Parse(s)
}

func MustParseID(s string) ID {
	return uuid.MustParse(s)
}
