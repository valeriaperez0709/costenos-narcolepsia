package hospital

import (
	"errors"
	"fmt"
	"strings"
)

// maxPatientsPerDoctor es la carga máxima de pacientes a cargo de un médico.
// Cuando la alcanza, CanAttend() devuelve false y el hospital despacha a otro.
const maxPatientsPerDoctor = 5

// Errores propios del médico.
var (
	ErrAlreadyAssigned = errors.New("patient is already under another doctor's care")
	ErrDoctorOverload  = errors.New("doctor has reached the maximum number of patients")
)

// Doctor es un médico del hospital. Incrusta Person (composición) y cumple
// la interfaz Attender (polimorfismo).
//
// La relación "médico -> sus episodios" vive DENTRO de la entidad: el médico
// guarda su propio historial (episodes), por eso MyEpisodes() no necesita
// recibir el hospital como parámetro.
type Doctor struct {
	Person                    // embebido: id, name, age
	specialty string          // especialidad médica
	patients  []*Patient      // pacientes a su cargo
	episodes  []EpisodeRecord // episodios en los que intervino
}

// Verificación en tiempo de compilación: si Doctor deja de cumplir Attender,
// el proyecto no compila.
var _ Attender = (*Doctor)(nil)

// NewDoctor valida los datos y crea un médico sin pacientes ni episodios.
func NewDoctor(id, name string, age int, specialty string) (*Doctor, error) {
	person, err := newPerson(id, name, age)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(specialty) == "" {
		return nil, fmt.Errorf("%w: specialty is empty", ErrInvalidData)
	}
	return &Doctor{Person: person, specialty: specialty}, nil
}

// Specialty devuelve la especialidad del médico.
func (d *Doctor) Specialty() string { return d.specialty }

// PatientsUnderCare devuelve una copia de la lista de pacientes a cargo.
func (d *Doctor) PatientsUnderCare() []*Patient {
	return append([]*Patient(nil), d.patients...)
}

// DiagnosePatient evalúa al paciente y lo toma a su cargo. Falla si el
// paciente es nil, si ya lo atiende otro médico o si el médico está lleno.
func (d *Doctor) DiagnosePatient(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	if p.assignedDoctor != nil && p.assignedDoctor != d {
		return fmt.Errorf("patient %s, doctor %s: %w", p.id, p.assignedDoctor.name, ErrAlreadyAssigned)
	}
	return d.takeCare(p)
}

// AttendSleepEmergency atiende una emergencia de sueño: el paciente debe
// estar dormido. Si aún no tenía médico responsable, este lo toma a cargo.
func (d *Doctor) AttendSleepEmergency(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	if p.state == Awake {
		return fmt.Errorf("patient %s: %w", p.id, ErrNotAsleep)
	}
	if p.assignedDoctor == nil {
		return d.takeCare(p)
	}
	return nil
}

// Attend implementa Attender: atiende la emergencia y guarda el episodio en
// el historial propio del médico. Devuelve el registro para que el hospital
// lo archive en su historial general.
func (d *Doctor) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if err := d.AttendSleepEmergency(p); err != nil {
		return EpisodeRecord{}, err
	}
	record := newEpisodeRecord(p, d, location)
	d.episodes = append(d.episodes, record)
	return record, nil
}

// CanAttend implementa Attender: el médico está disponible mientras no
// supere su carga máxima de pacientes.
func (d *Doctor) CanAttend() bool { return len(d.patients) < maxPatientsPerDoctor }

// MyEpisodes devuelve todos los episodios en los que intervino el médico
// (consulta 5.2). Devuelve una copia para proteger el historial interno.
func (d *Doctor) MyEpisodes() []EpisodeRecord {
	return append([]EpisodeRecord(nil), d.episodes...)
}

// takeCare agrega al paciente a la lista del médico (sin duplicarlo) y, si
// el paciente no tenía responsable, lo fija como tal.
func (d *Doctor) takeCare(p *Patient) error {
	for _, existing := range d.patients {
		if existing == p {
			return nil // ya lo tenía a cargo: operación idempotente
		}
	}
	if len(d.patients) >= maxPatientsPerDoctor {
		return fmt.Errorf("doctor %s: %w", d.name, ErrDoctorOverload)
	}
	d.patients = append(d.patients, p)
	if p.assignedDoctor == nil {
		p.setAssignedDoctor(d)
	}
	return nil
}
