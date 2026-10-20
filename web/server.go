// Package web es la capa de presentación del hospital: una API REST con la
// librería estándar (net/http) y una página HTML/JS servida desde el mismo
// binario.
//
// Regla de oro del bonus: el paquete hospital NO cambió ni una línea para
// agregar esta interfaz. Todo lo que es "de pantalla" (JSON, HTTP, textos de
// la bitácora, el modo carnaval) vive aquí.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"costenos-narcolepsia/hospital"
)

//go:embed static
var staticFiles embed.FS

// maxEvents limita la bitácora para que no crezca sin fin en modo carnaval.
const maxEvents = 60

// event es una línea de la bitácora que ve el usuario. A diferencia del
// historial del hospital (solo episodios), aquí también quedan los errores,
// como "no hay cama", para mostrarlos de forma legible.
type event struct {
	Time      time.Time `json:"time"`
	Kind      string    `json:"kind"` // attack, bed, wake, error, info
	Text      string    `json:"text"`
	Detail    string    `json:"detail,omitempty"` // texto original del error de Go
	PatientID string    `json:"patientId,omitempty"`
}

// Server guarda el hospital y atiende las peticiones HTTP.
//
// net/http atiende cada petición en su propia goroutine, y el modo carnaval
// lanza una goroutine por paciente. El paquete hospital no tiene locks, así
// que TODO acceso al hospital pasa por mu: una sola goroutine a la vez. Si
// dos peticiones piden la misma cama al mismo tiempo, la segunda espera a
// que termine la primera y recibe "no hay cama" en vez de una carrera.
type Server struct {
	mu      sync.Mutex
	h       *hospital.Hospital
	version int64         // sube con cada cambio; el navegador lo usa para saber si redibujar
	events  []event       // bitácora, la más reciente primero
	stopSim chan struct{} // nil = modo carnaval apagado

	mux *http.ServeMux
}

// NewServer crea el servidor con el elenco de la Sección 6 ya cargado.
func NewServer() (*Server, error) {
	h, err := newDemoHospital()
	if err != nil {
		return nil, err
	}
	s := &Server{h: h, mux: http.NewServeMux()}
	s.logEvent("info", "Hospital abierto: 2 médicos, 1 camillero, 3 camas y 5 pacientes", "", "")

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	s.mux.Handle("/", http.FileServer(http.FS(static)))
	s.mux.HandleFunc("/api/state", s.handleState)
	s.mux.HandleFunc("/api/reset", s.handleReset)
	s.mux.HandleFunc("/api/simulation", s.handleSimulation)
	s.mux.HandleFunc("/api/patients/", s.handlePatientAction)
	return s, nil
}

// ServeHTTP hace que *Server cumpla http.Handler (otra interfaz implícita).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// GET /api/state: foto completa del hospital, incluidas las consultas.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "use GET")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	writeJSON(w, http.StatusOK, s.snapshotLocked())
}

// POST /api/reset: vuelve al estado inicial del escenario.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	h, err := newDemoHospital()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopSimulationLocked()
	s.h = h
	s.events = nil
	s.logEvent("info", "Escenario reiniciado: todos despiertos en recepción", "", "")
	s.changedLocked()
	writeJSON(w, http.StatusOK, actionResult{OK: true, Message: "Escenario reiniciado", State: s.snapshotLocked()})
}

// POST /api/simulation {"running": true|false}: prende o apaga el modo carnaval.
func (s *Server) handleSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var body struct {
		Running bool `json:"running"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := "Modo carnaval apagado"
	if body.Running {
		s.startSimulationLocked()
		msg = "Modo carnaval prendido: cada paciente vive en su propia goroutine"
	} else {
		s.stopSimulationLocked()
	}
	s.logEvent("info", msg, "", "")
	s.changedLocked()
	writeJSON(w, http.StatusOK, actionResult{OK: true, Message: msg, State: s.snapshotLocked()})
}

// actionResult es la respuesta de toda acción: si salió bien, un mensaje
// para el usuario y la foto nueva (así el navegador no tiene que pedirla).
type actionResult struct {
	OK      bool     `json:"ok"`
	Message string   `json:"message"`
	Warning string   `json:"warning,omitempty"`
	State   snapshot `json:"state"`
}

// POST /api/patients/{id}/{sleep-attack|wake-up|assign-room}
//
// Go 1.21 no trae patrones con variables en ServeMux (llegaron en 1.22),
// así que la ruta se corta a mano con strings.Cut.
func (s *Server) handlePatientAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	id, action, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/api/patients/"), "/")
	if !ok || id == "" {
		writeError(w, http.StatusNotFound, "expected /api/patients/{id}/{action}")
		return
	}

	// El cuerpo solo lo usa sleep-attack; se lee antes de tomar el lock.
	var body struct {
		Location string `json:"location"`
	}
	if action == "sleep-attack" {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p := s.findPatientLocked(id)
	if p == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("patient %s: %v", id, hospital.ErrPatientNotFound))
		return
	}

	var (
		msg, warning string
		err          error
	)
	switch action {
	case "sleep-attack":
		msg, warning, err = s.sleepAttackLocked(p, body.Location)
	case "wake-up":
		msg, err = s.wakeUpLocked(p)
	case "assign-room":
		msg, err = s.assignRoomLocked(p)
	default:
		writeError(w, http.StatusNotFound, "unknown action "+action)
		return
	}
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, actionResult{OK: true, Message: msg, Warning: warning, State: s.snapshotLocked()})
}

// ---- Acciones. Todas asumen que mu está tomado (sufijo Locked). Las usan
// ---- tanto los handlers HTTP como las goroutines del modo carnaval.

// sleepAttackLocked delega en Hospital.HandleSleepAttack. Si no hubo cama,
// el episodio igual se registró: no es un fallo, es una advertencia.
func (s *Server) sleepAttackLocked(p *hospital.Patient, location string) (msg, warning string, err error) {
	record, err := s.h.HandleSleepAttack(p, location)
	if record.Patient() == nil { // no se registró nada: error de verdad
		s.logEvent("error", p.Name()+" no pudo registrar el episodio", errText(err), p.ID())
		s.changedLocked()
		return "", "", err
	}
	attender := record.Attender().Name()
	if errors.Is(err, hospital.ErrNoRoomAvailable) {
		msg = fmt.Sprintf("%s se durmió en %s y acudió %s", p.Name(), location, attender)
		warning = fmt.Sprintf("No hay cama libre: %s se queda en el pasillo", p.Name())
		s.logEvent("attack", msg, "", p.ID())
		s.logEvent("error", warning, err.Error(), p.ID())
	} else {
		msg = fmt.Sprintf("%s se durmió en %s, acudió %s y quedó en la %d",
			p.Name(), location, attender, record.Room().Number())
		s.logEvent("attack", msg, "", p.ID())
	}
	s.changedLocked()
	return msg, warning, nil
}

func (s *Server) wakeUpLocked(p *hospital.Patient) (string, error) {
	room := p.Room() // se lee antes: WakeUp la libera
	if err := p.WakeUp(); err != nil {
		return "", err
	}
	msg := p.Name() + " se despertó"
	if room != nil {
		msg += fmt.Sprintf(" y liberó la %d", room.Number())
	}
	s.logEvent("wake", msg, "", p.ID())
	s.changedLocked()
	return msg, nil
}

// assignRoomLocked es la consulta 5.3 tal cual: Hospital.AssignRoom.
func (s *Server) assignRoomLocked(p *hospital.Patient) (string, error) {
	room, err := s.h.AssignRoom(p)
	if err != nil {
		if errors.Is(err, hospital.ErrNoRoomAvailable) {
			s.logEvent("error", "Sin cama para "+p.Name()+": sigue en el pasillo", err.Error(), p.ID())
			s.changedLocked()
		}
		return "", err
	}
	msg := fmt.Sprintf("%s pasó del pasillo a la %d", p.Name(), room.Number())
	s.logEvent("bed", msg, "", p.ID())
	s.changedLocked()
	return msg, nil
}

// ---- Utilidades internas.

func (s *Server) findPatientLocked(id string) *hospital.Patient {
	for _, p := range s.h.Patients() {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

func (s *Server) snapshotLocked() snapshot {
	return buildSnapshot(s.h, s.version, s.stopSim != nil, append([]event(nil), s.events...))
}

func (s *Server) changedLocked() { s.version++ }

// logEvent agrega una línea al inicio de la bitácora (requiere mu tomado,
// salvo en NewServer, donde todavía nadie más ve el servidor).
func (s *Server) logEvent(kind, text, detail, patientID string) {
	e := event{Time: time.Now(), Kind: kind, Text: text, Detail: detail, PatientID: patientID}
	s.events = append([]event{e}, s.events...)
	if len(s.events) > maxEvents {
		s.events = s.events[:maxEvents]
	}
}

// statusFor traduce los errores centinela del hospital a códigos HTTP.
func statusFor(err error) int {
	switch {
	case errors.Is(err, hospital.ErrPatientNotFound):
		return http.StatusNotFound
	case errors.Is(err, hospital.ErrInvalidData):
		return http.StatusBadRequest
	default: // sin cama, ya dormido, ya despierto...: choca con el estado actual
		return http.StatusConflict
	}
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // si el cliente se fue, no hay a quién avisar
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}
