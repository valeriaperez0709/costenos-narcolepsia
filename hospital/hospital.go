// Package hospital contiene el modelo de dominio del "Hospital de los
// Costeños con Narcolepsia": pacientes, personal, habitaciones y el
// historial de episodios de sueño súbito.
//
// El paquete NO imprime nada ni conoce ninguna interfaz gráfica: solo
// guarda estado y devuelve datos. Quien quiera mostrarlos (main, una GUI,
// una API REST) lo hace desde afuera sin tocar este paquete.
package hospital

import (
	"errors"
	"fmt"
	"time"
)

// Errores del hospital. Se exportan para que los llamadores puedan usar
// errors.Is y reaccionar (por ejemplo, "sin cama" no es un error fatal).
var (
	ErrInvalidData         = errors.New("invalid data")
	ErrDuplicate           = errors.New("already registered")
	ErrPatientNotFound     = errors.New("patient is not admitted to this hospital")
	ErrNotInHallway        = errors.New("patient is not asleep in a hallway")
	ErrNoRoomAvailable     = errors.New("no room available: patient stays in the hallway")
	ErrNoAttenderAvailable = errors.New("no staff member available to attend the emergency")
)

// Hospital es la entidad central: coordina pacientes, personal y camas, y
// guarda el historial de todos los episodios.
//
// Todos los campos son privados. El personal completo vive en staff, un
// []Attender (médicos y camilleros mezclados), que es donde se aprovecha el
// polimorfismo; doctors guarda además la lista específica de médicos.
type Hospital struct {
	name      string          // nombre del hospital
	doctors   []*Doctor       // solo los médicos
	staff     []Attender      // TODO el personal que puede atender (polimorfismo)
	patients  []*Patient      // pacientes admitidos
	rooms     []*Room         // habitaciones registradas, en orden
	history   []EpisodeRecord // historial de todos los episodios
	nextStaff int             // índice por donde sigue el despacho rotativo
}

// NewHospital crea un hospital vacío con el nombre dado.
func NewHospital(name string) *Hospital {
	return &Hospital{name: name}
}

// Name devuelve el nombre del hospital.
func (h *Hospital) Name() string { return h.name }

// Doctors devuelve una copia de la lista de médicos.
func (h *Hospital) Doctors() []*Doctor { return append([]*Doctor(nil), h.doctors...) }

// Patients devuelve una copia de la lista de pacientes admitidos.
func (h *Hospital) Patients() []*Patient { return append([]*Patient(nil), h.patients...) }

// Rooms devuelve una copia de la lista de habitaciones.
func (h *Hospital) Rooms() []*Room { return append([]*Room(nil), h.rooms...) }

// Staff devuelve una copia de la lista de personal (médicos y camilleros).
func (h *Hospital) Staff() []Attender { return append([]Attender(nil), h.staff...) }

// History devuelve una copia del historial completo de episodios.
func (h *Hospital) History() []EpisodeRecord {
	return append([]EpisodeRecord(nil), h.history...)
}

// AdmitPatient registra a un paciente. Falla si es nil o si ya hay otro
// paciente con el mismo id.
func (h *Hospital) AdmitPatient(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	for _, existing := range h.patients {
		if existing.id == p.id {
			return fmt.Errorf("patient %s: %w", p.id, ErrDuplicate)
		}
	}
	h.patients = append(h.patients, p)
	return nil
}

// HireDoctor contrata a un médico: lo agrega a la lista de médicos y al
// personal general (staff). Falla si es nil o si el id ya existe.
func (h *Hospital) HireDoctor(d *Doctor) error {
	if d == nil {
		return fmt.Errorf("%w: nil doctor", ErrInvalidData)
	}
	if err := h.addStaff(d); err != nil {
		return err
	}
	h.doctors = append(h.doctors, d)
	return nil
}

// HireOrderly contrata a un camillero (solo entra al staff general).
func (h *Hospital) HireOrderly(o *Orderly) error {
	if o == nil {
		return fmt.Errorf("%w: nil orderly", ErrInvalidData)
	}
	return h.addStaff(o)
}

// RegisterRoom agrega una habitación. Falla si es nil o si el número ya existe.
func (h *Hospital) RegisterRoom(r *Room) error {
	if r == nil {
		return fmt.Errorf("%w: nil room", ErrInvalidData)
	}
	for _, existing := range h.rooms {
		if existing.number == r.number {
			return fmt.Errorf("room %d: %w", r.number, ErrDuplicate)
		}
	}
	h.rooms = append(h.rooms, r)
	return nil
}

// RegisterEpisode archiva un episodio en el historial general. Valida que
// el registro tenga paciente y personal, y que el paciente esté admitido.
func (h *Hospital) RegisterEpisode(record EpisodeRecord) error {
	if record.attender == nil {
		return fmt.Errorf("%w: episode without attender", ErrInvalidData)
	}
	if err := h.requireAdmitted(record.patient); err != nil {
		return err
	}
	h.history = append(h.history, record)
	return nil
}

// AssignRoom busca la PRIMERA habitación disponible y se la da al paciente
// (consulta 5.3): la habitación pasa a Occupied (si se llena) y el paciente
// a AsleepInBed.
//
// Si no hay ninguna libre devuelve un error (ErrNoRoomAvailable) y el
// paciente sigue en AsleepInHallway. Nunca hace panic ni inventa una sala.
func (h *Hospital) AssignRoom(p *Patient) (*Room, error) {
	if err := h.requireAdmitted(p); err != nil {
		return nil, err
	}
	if p.state != AsleepInHallway {
		return nil, fmt.Errorf("patient %s is %s: %w", p.id, p.state, ErrNotInHallway)
	}
	for _, room := range h.rooms {
		if !room.IsAvailable() {
			continue
		}
		if err := room.Occupy(p); err != nil {
			return nil, err
		}
		p.moveToBed(room)
		return room, nil
	}
	return nil, fmt.Errorf("patient %s: %w", p.id, ErrNoRoomAvailable)
}

// HandleSleepAttack orquesta un ataque de sueño completo:
//
//  1. El paciente cae dormido en la ubicación (queda en el pasillo).
//  2. Se intenta asignarle cama.
//  3. Se despacha al primer miembro del personal disponible (sin saber si
//     es médico o camillero: eso es el polimorfismo).
//  4. Se archiva el episodio en el historial.
//
// OJO con el contrato: si no hay cama, el episodio SÍ se atiende y se
// registra; en ese caso se devuelve el registro válido junto con un error
// que envuelve ErrNoRoomAvailable (se detecta con errors.Is) y el paciente
// queda en el pasillo.
func (h *Hospital) HandleSleepAttack(p *Patient, location string) (EpisodeRecord, error) {
	if err := h.requireAdmitted(p); err != nil {
		return EpisodeRecord{}, err
	}
	if err := p.SufferSleepAttack(location); err != nil {
		return EpisodeRecord{}, err
	}

	_, roomErr := h.AssignRoom(p) // se guarda: no impide atender al paciente

	attender, err := h.dispatch()
	if err != nil {
		return EpisodeRecord{}, errors.Join(err, roomErr)
	}
	record, err := attender.Attend(p, location)
	if err != nil {
		return EpisodeRecord{}, errors.Join(err, roomErr)
	}
	if err := h.RegisterEpisode(record); err != nil {
		return EpisodeRecord{}, errors.Join(err, roomErr)
	}
	return record, roomErr
}

// PatientsInHallway devuelve los pacientes dormidos en un pasillo
// (consulta 5.1), para poder despachar un camillero. Solo devuelve datos.
func (h *Hospital) PatientsInHallway() []*Patient {
	var result []*Patient
	for _, p := range h.patients {
		if p.state == AsleepInHallway {
			result = append(result, p)
		}
	}
	return result
}

// SevereReport cruza dos fuentes (consulta 5.4): los pacientes con nivel
// Severe (lista de pacientes) y cuántos episodios tuvo cada uno HOY
// (historial). Devuelve el emparejamiento paciente -> número de episodios.
//
// Se usa un map porque la relación es "clave -> valor"; incluye también a
// los severos con 0 episodios. Los maps de Go no tienen orden: quien
// imprima debe ordenar las claves (main lo hace por id).
func (h *Hospital) SevereReport() map[*Patient]int {
	report := make(map[*Patient]int)
	for _, p := range h.patients {
		if p.level == Severe {
			report[p] = 0
		}
	}
	now := time.Now()
	for _, record := range h.history {
		if _, isSevere := report[record.patient]; isSevere && sameDay(record.dateTime, now) {
			report[record.patient]++
		}
	}
	return report
}

// dispatch elige al siguiente miembro del personal disponible, rotando
// (round-robin) para repartir la carga. Solo conoce la interfaz Attender:
// no le importa si quien responde es un *Doctor o un *Orderly.
func (h *Hospital) dispatch() (Attender, error) {
	n := len(h.staff)
	for i := 0; i < n; i++ {
		idx := (h.nextStaff + i) % n
		if h.staff[idx].CanAttend() {
			h.nextStaff = (idx + 1) % n
			return h.staff[idx], nil
		}
	}
	return nil, ErrNoAttenderAvailable
}

// addStaff agrega a alguien al personal general, rechazando ids repetidos.
func (h *Hospital) addStaff(a Attender) error {
	for _, existing := range h.staff {
		if existing.ID() == a.ID() {
			return fmt.Errorf("staff %s: %w", a.ID(), ErrDuplicate)
		}
	}
	h.staff = append(h.staff, a)
	return nil
}

// requireAdmitted verifica que el paciente exista y esté admitido aquí.
func (h *Hospital) requireAdmitted(p *Patient) error {
	if p == nil {
		return fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	for _, existing := range h.patients {
		if existing == p {
			return nil
		}
	}
	return fmt.Errorf("patient %s: %w", p.id, ErrPatientNotFound)
}

// sameDay indica si dos instantes caen en el mismo día calendario.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
