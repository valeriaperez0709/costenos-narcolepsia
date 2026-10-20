package web

import (
	"sort"
	"time"

	"costenos-narcolepsia/hospital"
)

// Este archivo traduce el modelo del hospital a structs "planos" con tags
// JSON (DTOs). Los structs del paquete hospital tienen todos sus campos en
// minúscula, así que encoding/json no puede verlos: la vista los lee con
// los métodos públicos y los copia aquí. Así el modelo no sabe nada de JSON.

// snapshot es la foto completa del hospital que recibe el navegador.
type snapshot struct {
	Version    int64         `json:"version"`
	Hospital   string        `json:"hospital"`
	Simulation bool          `json:"simulation"`
	Patients   []patientView `json:"patients"`
	Rooms      []roomView    `json:"rooms"`
	Staff      []staffView   `json:"staff"`
	History    []episodeView `json:"history"`
	Queries    queriesView   `json:"queries"`
	Events     []event       `json:"events"`
}

// queriesView agrupa el resultado de las consultas de la Sección 5.
// La 5.3 (AssignRoom) es una acción, no una lectura: su resultado se ve en
// las habitaciones y en la bitácora (events).
type queriesView struct {
	Hallway          []string         `json:"hallway"`          // 5.1 PatientsInHallway
	EpisodesByDoctor []doctorEpisodes `json:"episodesByDoctor"` // 5.2 MyEpisodes
	SevereReport     []severeRow      `json:"severeReport"`     // 5.4 SevereReport
}

type patientView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Level    string `json:"level"`
	State    string `json:"state"`
	Location string `json:"location"`
	Room     *int   `json:"room"`   // nil = sin cama
	Doctor   string `json:"doctor"` // "" = sin médico asignado
}

type roomView struct {
	Number    int      `json:"number"`
	Capacity  int      `json:"capacity"`
	State     string   `json:"state"`
	Occupants []string `json:"occupants"`
}

type staffView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Specialty string `json:"specialty,omitempty"`
	Available bool   `json:"available"`
	Episodes  int    `json:"episodes"`
}

type episodeView struct {
	ID           string    `json:"id"`
	Time         time.Time `json:"time"`
	PatientID    string    `json:"patientId"`
	PatientName  string    `json:"patientName"`
	AttenderName string    `json:"attenderName"`
	AttenderKind string    `json:"attenderKind"`
	Location     string    `json:"location"`
	Room         *int      `json:"room"`
	Summary      string    `json:"summary"`
}

type doctorEpisodes struct {
	DoctorID   string        `json:"doctorId"`
	DoctorName string        `json:"doctorName"`
	Episodes   []episodeView `json:"episodes"`
}

type severeRow struct {
	PatientID   string `json:"patientId"`
	PatientName string `json:"patientName"`
	Episodes    int    `json:"episodes"`
}

// episodeKeeper es una interfaz pequeña definida por QUIEN LA USA (la
// vista), no por el paquete hospital. Doctor y Orderly tienen MyEpisodes()
// aunque Attender no lo exige; con una aserción de tipo a esta interfaz se
// cuentan sus episodios sin preguntar por el tipo concreto.
type episodeKeeper interface {
	MyEpisodes() []hospital.EpisodeRecord
}

// buildSnapshot lee el hospital SOLO con métodos públicos. Debe llamarse
// con el mutex del servidor tomado.
func buildSnapshot(h *hospital.Hospital, version int64, simulating bool, events []event) snapshot {
	snap := snapshot{
		Version:    version,
		Hospital:   h.Name(),
		Simulation: simulating,
		Events:     events,
	}

	for _, p := range h.Patients() {
		snap.Patients = append(snap.Patients, newPatientView(p))
	}
	for _, r := range h.Rooms() {
		snap.Rooms = append(snap.Rooms, newRoomView(r))
	}
	for _, a := range h.Staff() { // []Attender: médicos y camilleros juntos
		snap.Staff = append(snap.Staff, newStaffView(a))
	}

	// Historial del más reciente al más antiguo (así lo lee una bitácora).
	history := h.History()
	for i := len(history) - 1; i >= 0; i-- {
		snap.History = append(snap.History, newEpisodeView(history[i]))
	}

	// 5.1 Pacientes dormidos en un pasillo.
	snap.Queries.Hallway = []string{}
	for _, p := range h.PatientsInHallway() {
		snap.Queries.Hallway = append(snap.Queries.Hallway, p.ID())
	}

	// 5.2 Historial por médico: cada médico responde por sí mismo.
	for _, d := range h.Doctors() {
		row := doctorEpisodes{DoctorID: d.ID(), DoctorName: d.Name(), Episodes: []episodeView{}}
		for _, e := range d.MyEpisodes() {
			row.Episodes = append(row.Episodes, newEpisodeView(e))
		}
		snap.Queries.EpisodesByDoctor = append(snap.Queries.EpisodesByDoctor, row)
	}

	// 5.4 Reporte de severidad. El map no tiene orden: se ordena por id
	// para que la tabla no "baile" en cada refresco.
	report := h.SevereReport()
	snap.Queries.SevereReport = []severeRow{}
	for p, count := range report {
		snap.Queries.SevereReport = append(snap.Queries.SevereReport,
			severeRow{PatientID: p.ID(), PatientName: p.Name(), Episodes: count})
	}
	sort.Slice(snap.Queries.SevereReport, func(i, j int) bool {
		return snap.Queries.SevereReport[i].PatientID < snap.Queries.SevereReport[j].PatientID
	})

	return snap
}

func newPatientView(p *hospital.Patient) patientView {
	v := patientView{
		ID:       p.ID(),
		Name:     p.Name(),
		Age:      p.Age(),
		Level:    p.Level().String(),
		State:    p.State().String(),
		Location: p.Location(),
		Room:     roomNumber(p.Room()),
	}
	if d := p.AssignedDoctor(); d != nil {
		v.Doctor = d.Name()
	}
	return v
}

func newRoomView(r *hospital.Room) roomView {
	v := roomView{Number: r.Number(), Capacity: r.Capacity(), State: r.State().String(), Occupants: []string{}}
	for _, p := range r.Occupants() {
		v.Occupants = append(v.Occupants, p.ID())
	}
	return v
}

func newStaffView(a hospital.Attender) staffView {
	v := staffView{ID: a.ID(), Name: a.Name(), Kind: kindOf(a), Available: a.CanAttend()}
	if d, ok := a.(*hospital.Doctor); ok {
		v.Specialty = d.Specialty()
	}
	if k, ok := a.(episodeKeeper); ok {
		v.Episodes = len(k.MyEpisodes())
	}
	return v
}

func newEpisodeView(e hospital.EpisodeRecord) episodeView {
	v := episodeView{
		ID:       e.ID(),
		Time:     e.DateTime(),
		Location: e.Location(),
		Room:     roomNumber(e.Room()),
		Summary:  e.Summary(),
	}
	if p := e.Patient(); p != nil {
		v.PatientID, v.PatientName = p.ID(), p.Name()
	}
	if a := e.Attender(); a != nil {
		v.AttenderName, v.AttenderKind = a.Name(), kindOf(a)
	}
	return v
}

// kindOf es el ÚNICO lugar donde la vista mira el tipo concreto de un
// Attender, y solo para elegir un ícono. El hospital nunca lo hace: despacha
// por la interfaz. Un tipo nuevo (por ejemplo *Nurse) cae en "staff".
func kindOf(a hospital.Attender) string {
	switch a.(type) {
	case *hospital.Doctor:
		return "doctor"
	case *hospital.Orderly:
		return "orderly"
	default:
		return "staff"
	}
}

// roomNumber convierte una habitación opcional en un número opcional
// (nil en JSON se ve como null: "sin cama").
func roomNumber(r *hospital.Room) *int {
	if r == nil {
		return nil
	}
	n := r.Number()
	return &n
}
