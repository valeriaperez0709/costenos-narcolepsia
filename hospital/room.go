package hospital

import (
	"errors"
	"fmt"
)

// RoomState es el estado de una habitación (tipo propio + constantes).
type RoomState int

const (
	// Available: todavía hay cupo en la habitación.
	Available RoomState = iota
	// Occupied: la habitación alcanzó su capacidad.
	Occupied
)

// String permite imprimir el estado de forma legible.
func (s RoomState) String() string {
	switch s {
	case Available:
		return "Available"
	case Occupied:
		return "Occupied"
	default:
		return fmt.Sprintf("RoomState(%d)", int(s))
	}
}

// Errores propios de la habitación.
var (
	ErrRoomFull      = errors.New("room is full")
	ErrAlreadyInRoom = errors.New("patient is already in the room")
	ErrNotInRoom     = errors.New("patient is not in the room")
)

// Room representa una habitación del hospital con una capacidad fija.
//
// Su estado (Available/Occupied) NUNCA se asigna desde fuera: se recalcula
// internamente cada vez que alguien entra o sale, de modo que siempre es
// consistente con la lista de ocupantes.
type Room struct {
	number    int        // número visible de la habitación (101, 102...)
	capacity  int        // máximo de pacientes simultáneos
	state     RoomState  // derivado de occupants y capacity
	occupants []*Patient // pacientes que están dentro ahora mismo
}

// NewRoom crea una habitación vacía. Exige número y capacidad positivos.
func NewRoom(number, capacity int) (*Room, error) {
	if number <= 0 {
		return nil, fmt.Errorf("%w: room number must be positive", ErrInvalidData)
	}
	if capacity <= 0 {
		return nil, fmt.Errorf("%w: room capacity must be positive", ErrInvalidData)
	}
	return &Room{number: number, capacity: capacity, state: Available}, nil
}

// Number devuelve el número de la habitación.
func (r *Room) Number() int { return r.number }

// Capacity devuelve la capacidad máxima.
func (r *Room) Capacity() int { return r.capacity }

// State devuelve el estado actual de la habitación.
func (r *Room) State() RoomState { return r.state }

// Occupants devuelve una COPIA de la lista de ocupantes, para que quien
// llame no pueda modificar el slice interno de la habitación.
func (r *Room) Occupants() []*Patient {
	return append([]*Patient(nil), r.occupants...)
}

// IsAvailable indica si todavía cabe al menos un paciente más.
func (r *Room) IsAvailable() bool { return len(r.occupants) < r.capacity }

// Occupy mete al paciente en la habitación. Falla si está llena o si el
// paciente ya estaba dentro. Al terminar actualiza el estado de la sala.
func (r *Room) Occupy(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	if !r.IsAvailable() {
		return fmt.Errorf("room %d: %w", r.number, ErrRoomFull)
	}
	for _, o := range r.occupants {
		if o == p {
			return fmt.Errorf("room %d, patient %s: %w", r.number, p.id, ErrAlreadyInRoom)
		}
	}
	r.occupants = append(r.occupants, p)
	r.refreshState()
	return nil
}

// Release saca al paciente de la habitación. Falla si no estaba dentro.
func (r *Room) Release(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	for i, o := range r.occupants {
		if o == p {
			r.occupants = append(r.occupants[:i], r.occupants[i+1:]...)
			r.refreshState()
			return nil
		}
	}
	return fmt.Errorf("room %d, patient %s: %w", r.number, p.id, ErrNotInRoom)
}

// refreshState recalcula el estado a partir de los ocupantes actuales.
func (r *Room) refreshState() {
	if len(r.occupants) >= r.capacity {
		r.state = Occupied
	} else {
		r.state = Available
	}
}
