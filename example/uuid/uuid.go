package uuid

import (
	"log"

	_uuid "github.com/gofrs/uuid/v5"
)

var Nil = _uuid.Nil

func NewV4() _uuid.UUID {
	id, err := _uuid.NewV4()
	if err != nil {
		log.Panicln(err)
	}
	return id
}

func FromString(id string) (_uuid.UUID, error) {
	return _uuid.FromString(id)
}

func FromBytes(input []byte) (_uuid.UUID, error) {
	return _uuid.FromBytes(input)
}
