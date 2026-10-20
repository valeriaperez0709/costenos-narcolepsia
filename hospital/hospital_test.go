package hospital_test

import (
	"errors"
	"testing"

	"costenos-narcolepsia/hospital"
)

// fixture agrupa los objetos que usan casi todas las pruebas.
type fixture struct {
	h        *hospital.Hospital
	doctor   *hospital.Doctor
	orderly  *hospital.Orderly
	patients []*hospital.Patient // P-001 Mild, P-002 Severe, P-003 Severe
}

// newFixture arma un hospital con 1 médico, 1 camillero, 3 pacientes y el
// número de habitaciones (capacidad 1) indicado. Las pruebas están en el
// paquete externo hospital_test: solo usan la API pública, igual que main.
func newFixture(t *testing.T, rooms int) fixture {
	t.Helper()
	h := hospital.NewHospital("Test Hospital")

	doctor, err := hospital.NewDoctor("D-001", "Karen Ospina", 41, "Neurology")
	if err != nil {
		t.Fatal(err)
	}
	orderly, err := hospital.NewOrderly("O-001", "Camilo Barrios", 29)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.HireDoctor(doctor); err != nil {
		t.Fatal(err)
	}
	if err := h.HireOrderly(orderly); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < rooms; i++ {
		room, err := hospital.NewRoom(101+i, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.RegisterRoom(room); err != nil {
			t.Fatal(err)
		}
	}

	levels := []hospital.NarcolepsyLevel{hospital.Mild, hospital.Severe, hospital.Severe}
	ids := []string{"P-001", "P-002", "P-003"}
	var patients []*hospital.Patient
	for i, id := range ids {
		p, err := hospital.NewPatient(id, "Patient "+id, 40+i, levels[i])
		if err != nil {
			t.Fatal(err)
		}
		if err := h.AdmitPatient(p); err != nil {
			t.Fatal(err)
		}
		patients = append(patients, p)
	}
	return fixture{h: h, doctor: doctor, orderly: orderly, patients: patients}
}

// Caso feliz de AssignRoom: hay una habitación libre.
func TestAssignRoomWithFreeRoom(t *testing.T) {
	f := newFixture(t, 1)
	p := f.patients[0]
	if err := p.SufferSleepAttack("cafeteria"); err != nil {
		t.Fatal(err)
	}

	room, err := f.h.AssignRoom(p)
	if err != nil {
		t.Fatalf("expected a room, got error: %v", err)
	}
	if room.Number() != 101 {
		t.Errorf("room number = %d, want 101", room.Number())
	}
	if room.State() != hospital.Occupied || room.IsAvailable() {
		t.Errorf("room should be Occupied, got %s", room.State())
	}
	if p.State() != hospital.AsleepInBed || p.Room() != room {
		t.Errorf("patient should be AsleepInBed in room 101, got %s", p.State())
	}
}

// Caso de fallo de AssignRoom: no hay habitaciones libres.
func TestAssignRoomNoRoomAvailable(t *testing.T) {
	f := newFixture(t, 1)
	first, second := f.patients[0], f.patients[1]
	for _, p := range []*hospital.Patient{first, second} {
		if err := p.SufferSleepAttack("hallway"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.h.AssignRoom(first); err != nil {
		t.Fatal(err)
	}

	room, err := f.h.AssignRoom(second)
	if !errors.Is(err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("error = %v, want ErrNoRoomAvailable", err)
	}
	if room != nil {
		t.Error("no room should be returned (and none invented)")
	}
	if second.State() != hospital.AsleepInHallway {
		t.Errorf("patient should stay AsleepInHallway, got %s", second.State())
	}
}

// Consulta 5.1.
func TestPatientsInHallway(t *testing.T) {
	f := newFixture(t, 1)
	for _, p := range f.patients {
		if err := p.SufferSleepAttack("hallway"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.h.AssignRoom(f.patients[0]); err != nil {
		t.Fatal(err)
	}

	got := f.h.PatientsInHallway()
	if len(got) != 2 || got[0] != f.patients[1] || got[1] != f.patients[2] {
		t.Fatalf("PatientsInHallway() = %v patients, want P-002 and P-003", len(got))
	}
}

// Polimorfismo: el hospital despacha a Doctor y Orderly por la interfaz
// Attender. También cubre la consulta 5.2 (MyEpisodes).
func TestDispatchIsPolymorphicAndMyEpisodes(t *testing.T) {
	f := newFixture(t, 2)

	var records []hospital.EpisodeRecord
	for _, p := range f.patients {
		record, err := f.h.HandleSleepAttack(p, "somewhere")
		if err != nil && !errors.Is(err, hospital.ErrNoRoomAvailable) {
			t.Fatal(err)
		}
		records = append(records, record)
	}

	// Rotación sobre [doctor, orderly]: doctor, orderly, doctor.
	if _, ok := records[0].Attender().(*hospital.Doctor); !ok {
		t.Errorf("first attender should be a *Doctor, got %T", records[0].Attender())
	}
	if _, ok := records[1].Attender().(*hospital.Orderly); !ok {
		t.Errorf("second attender should be an *Orderly, got %T", records[1].Attender())
	}
	if got := len(f.doctor.MyEpisodes()); got != 2 {
		t.Errorf("doctor episodes = %d, want 2", got)
	}
	if got := len(f.orderly.MyEpisodes()); got != 1 {
		t.Errorf("orderly episodes = %d, want 1", got)
	}
	if got := len(f.h.History()); got != 3 {
		t.Errorf("history size = %d, want 3", got)
	}
}

// HandleSleepAttack sin cama: el episodio se atiende y registra igual.
func TestHandleSleepAttackWithoutRoomStillRecordsEpisode(t *testing.T) {
	f := newFixture(t, 0)
	record, err := f.h.HandleSleepAttack(f.patients[0], "pharmacy")
	if !errors.Is(err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("error = %v, want ErrNoRoomAvailable", err)
	}
	if record.Room() != nil || record.Attender() == nil {
		t.Error("record should have an attender and no room")
	}
	if len(f.h.History()) != 1 {
		t.Error("episode should be archived even without a room")
	}
}

// Consulta 5.4.
func TestSevereReport(t *testing.T) {
	f := newFixture(t, 3)
	for _, p := range f.patients {
		if _, err := f.h.HandleSleepAttack(p, "somewhere"); err != nil {
			t.Fatal(err)
		}
	}

	report := f.h.SevereReport()
	if len(report) != 2 {
		t.Fatalf("report has %d patients, want only the 2 severe ones", len(report))
	}
	if _, found := report[f.patients[0]]; found {
		t.Error("mild patient must not appear in the report")
	}
	if report[f.patients[1]] != 1 || report[f.patients[2]] != 1 {
		t.Errorf("episodes = %d and %d, want 1 and 1", report[f.patients[1]], report[f.patients[2]])
	}
}

// WakeUp libera la cama y permite reasignarla al paciente del pasillo.
func TestWakeUpReleasesRoomForNextPatient(t *testing.T) {
	f := newFixture(t, 1)
	first, second := f.patients[0], f.patients[1]

	if _, err := f.h.HandleSleepAttack(first, "cafeteria"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.h.HandleSleepAttack(second, "hallway"); !errors.Is(err, hospital.ErrNoRoomAvailable) {
		t.Fatalf("error = %v, want ErrNoRoomAvailable", err)
	}

	room := first.Room()
	if err := first.WakeUp(); err != nil {
		t.Fatal(err)
	}
	if first.State() != hospital.Awake || !room.IsAvailable() {
		t.Fatal("waking up should leave the patient Awake and the room free")
	}

	got, err := f.h.AssignRoom(second)
	if err != nil || got != room {
		t.Fatalf("second patient should take the freed room, got %v, %v", got, err)
	}
}

// Validaciones de Room y del constructor de Patient.
func TestRoomAndConstructorValidation(t *testing.T) {
	f := newFixture(t, 0)
	room, err := hospital.NewRoom(201, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := room.Occupy(f.patients[0]); err != nil {
		t.Fatal(err)
	}
	if err := room.Occupy(f.patients[1]); !errors.Is(err, hospital.ErrRoomFull) {
		t.Errorf("error = %v, want ErrRoomFull", err)
	}
	if err := room.Release(f.patients[1]); !errors.Is(err, hospital.ErrNotInRoom) {
		t.Errorf("error = %v, want ErrNotInRoom", err)
	}
	if _, err := hospital.NewPatient("", "No ID", 30, hospital.Mild); !errors.Is(err, hospital.ErrInvalidData) {
		t.Errorf("error = %v, want ErrInvalidData", err)
	}
}
