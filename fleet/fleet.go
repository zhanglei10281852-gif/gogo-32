package fleet

import (
	internal "gridflex/internal/fleet"
	"gridflex/model"
)

type State = internal.State
type Fleet = internal.Fleet
type Priority = internal.Priority

func New(vehicles []model.Vehicle) (*Fleet, error) {
	return internal.New(vehicles)
}
