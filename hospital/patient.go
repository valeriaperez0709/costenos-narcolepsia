package hospital

import (
	"errors"
	"fmt"
	"strings"
)

// PatientState es el estado de un paciente. Se declara como tipo propio con
// constantes (iota) para que el compilador detecte errores de digitación que
// con strings sueltos pasarían desapercibidos.
type PatientState int

const (
	// Awake: el paciente está despierto.
	Awake PatientState = iota
	// AsleepInHallway: se durmió y sigue tirado en un pasillo (sin cama).
	AsleepInHallway
	// AsleepInBed: se durmió y ya tiene una cama asignada.
	AsleepInBed
)

// String permite imprimir el estado de forma legible (implementa fmt.Stringer).
func (s PatientState) String() string {
	switch s {
	case Awake:
		return "Awake"
	case AsleepInHallway:
		return "AsleepInHallway"
	case AsleepInBed:
		return "AsleepInBed"
	default:
		return fmt.Sprintf("PatientState(%d)", int(s))
	}
}

// NarcolepsyLevel es la severidad de la narcolepsia del paciente.
type NarcolepsyLevel int

const (
	Mild NarcolepsyLevel = iota
	Moderate
	Severe
)

// String permite imprimir el nivel de forma legible.
func (l NarcolepsyLevel) String() string {
	switch l {
	case Mild:
		return "Mild"
	case Moderate:
		return "Moderate"
	case Severe:
		return "Severe"
	default:
		return fmt.Sprintf("NarcolepsyLevel(%d)", int(l))
	}
}

// Errores propios del paciente. Son "sentinel errors": quien llama puede
// distinguirlos con errors.Is sin comparar textos.
var (
	ErrAlreadyAsleep = errors.New("patient is already asleep")
	ErrAlreadyAwake  = errors.New("patient is already awake")
	ErrNotAsleep     = errors.New("patient is not asleep")
)

// Patient representa a un paciente costeño con narcolepsia.
//
// Incrusta Person por valor (composición), por lo que hereda ID(), Name() y
// Age() sin duplicar campos. Todo el estado es privado: solo cambia a través
// de los métodos (SufferSleepAttack, WakeUp) o de operaciones internas del
// paquete (moveToBed, setAssignedDoctor), lo que garantiza que el estado, la
// habitación y la ubicación siempre sean coherentes entre sí.
type Patient struct {
	Person                          // embebido: reutiliza los métodos de Person
	level           NarcolepsyLevel // severidad de la enfermedad
	state           PatientState    // Awake / AsleepInHallway / AsleepInBed
	currentLocation string          // dónde está (o dónde se durmió) el paciente
	assignedDoctor  *Doctor         // médico responsable; nil si no tiene
	room            *Room           // cama ocupada; nil si no tiene
}

// NewPatient valida los datos y crea un paciente despierto en recepción.
func NewPatient(id, name string, age int, level NarcolepsyLevel) (*Patient, error) {
	person, err := newPerson(id, name, age)
	if err != nil {
		return nil, err
	}
	if level < Mild || level > Severe {
		return nil, fmt.Errorf("%w: unknown narcolepsy level %d", ErrInvalidData, int(level))
	}
	return &Patient{
		Person:          person,
		level:           level,
		state:           Awake,
		currentLocation: "reception",
	}, nil
}

// Level devuelve la severidad de la narcolepsia.
func (p *Patient) Level() NarcolepsyLevel { return p.level }

// State devuelve el estado actual del paciente.
func (p *Patient) State() PatientState { return p.state }

// Location devuelve la ubicación actual (o donde se durmió) del paciente.
func (p *Patient) Location() string { return p.currentLocation }

// AssignedDoctor devuelve el médico responsable, o nil si aún no tiene.
func (p *Patient) AssignedDoctor() *Doctor { return p.assignedDoctor }

// Room devuelve la habitación que ocupa, o nil si no tiene cama.
func (p *Patient) Room() *Room { return p.room }

// SufferSleepAttack simula un ataque de sueño súbito en la ubicación dada.
// El paciente queda dormido en el pasillo (AsleepInHallway) hasta que el
// hospital le asigne una cama. Falla si ya estaba dormido.
func (p *Patient) SufferSleepAttack(location string) error {
	if p.state != Awake {
		return fmt.Errorf("patient %s: %w", p.id, ErrAlreadyAsleep)
	}
	if strings.TrimSpace(location) == "" {
		return fmt.Errorf("%w: location is empty", ErrInvalidData)
	}
	p.state = AsleepInHallway
	p.currentLocation = location
	return nil
}

// WakeUp despierta al paciente. Si estaba en una cama, la libera para que
// pueda ser reutilizada. Primero libera la habitación y solo si eso salió
// bien cambia el estado, para no dejar el objeto a medias si algo falla.
func (p *Patient) WakeUp() error {
	if p.state == Awake {
		return fmt.Errorf("patient %s: %w", p.id, ErrAlreadyAwake)
	}
	if p.room != nil {
		if err := p.room.Release(p); err != nil {
			return err
		}
		p.room = nil
	}
	p.state = Awake
	return nil
}

// moveToBed deja al paciente dormido en la cama indicada. Es no exportada:
// solo Hospital.AssignRoom debe llamarla, después de ocupar la habitación.
func (p *Patient) moveToBed(r *Room) {
	p.room = r
	p.state = AsleepInBed
}

// setAssignedDoctor registra el médico responsable (uso interno del paquete).
func (p *Patient) setAssignedDoctor(d *Doctor) { p.assignedDoctor = d }
