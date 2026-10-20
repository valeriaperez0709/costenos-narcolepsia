package web

import (
	"fmt"
	"math/rand"
	"time"

	"costenos-narcolepsia/hospital"
)

// Modo carnaval (bonus de concurrencia): cada paciente vive en su propia
// goroutine y, a intervalos aleatorios, se duerme, pide cama o se despierta.
// Todas las goroutines comparten el hospital, así que cada paso toma s.mu.
// Probar con:  go run -race ./cmd/web

// carnivalSpots son los lugares donde un costeño puede caer rendido.
var carnivalSpots = []string{
	"la cafetería",
	"la sala de espera",
	"la cola de radiología",
	"el pasillo 2, segundo piso",
	"la terraza, bailando champeta",
	"la sala de TV, viendo béisbol",
	"la farmacia",
	"el ascensor",
	"el parqueadero",
	"el almuerzo, después del sancocho",
}

// startSimulationLocked lanza una goroutine por paciente (requiere mu).
func (s *Server) startSimulationLocked() {
	if s.stopSim != nil {
		return // ya estaba prendido
	}
	stop := make(chan struct{})
	s.stopSim = stop
	for _, p := range s.h.Patients() {
		go s.livePatient(p, stop)
	}
}

// stopSimulationLocked avisa a todas las goroutines que terminen cerrando
// el canal: un close le llega a TODOS los que esperan en él (requiere mu).
func (s *Server) stopSimulationLocked() {
	if s.stopSim == nil {
		return
	}
	close(s.stopSim)
	s.stopSim = nil
}

// livePatient es la vida de un paciente: esperar un rato y dar un paso.
func (s *Server) livePatient(p *hospital.Patient, stop <-chan struct{}) {
	for {
		s.mu.Lock()
		wait := delayFor(p.State())
		s.mu.Unlock()

		select {
		case <-stop:
			return
		case <-time.After(wait):
		}

		s.mu.Lock()
		// Mientras esperábamos el lock pudieron apagar la simulación o
		// reiniciar el hospital: se revisa otra vez antes de tocar nada.
		select {
		case <-stop:
			s.mu.Unlock()
			return
		default:
		}
		s.stepLocked(p)
		s.mu.Unlock()
	}
}

// stepLocked decide el siguiente paso según el estado del paciente. Solo
// usa métodos públicos del hospital; los errores esperables (sin cama) ya
// quedan en la bitácora, así que aquí se ignoran.
func (s *Server) stepLocked(p *hospital.Patient) {
	switch p.State() {
	case hospital.Awake:
		spot := carnivalSpots[rand.Intn(len(carnivalSpots))]
		_, _, _ = s.sleepAttackLocked(p, spot)
	case hospital.AsleepInHallway:
		// Reintenta en silencio; solo se anota si consigue cama.
		if room, err := s.h.AssignRoom(p); err == nil {
			s.logEvent("bed", fmt.Sprintf("%s consiguió cama en la %d", p.Name(), room.Number()), "", p.ID())
			s.changedLocked()
		}
	case hospital.AsleepInBed:
		_, _ = s.wakeUpLocked(p)
	}
}

// delayFor: despierto aguanta poco, en cama duerme un rato largo.
func delayFor(state hospital.PatientState) time.Duration {
	switch state {
	case hospital.Awake:
		return randomBetween(3*time.Second, 9*time.Second)
	case hospital.AsleepInHallway:
		return randomBetween(1500*time.Millisecond, 3*time.Second)
	default:
		return randomBetween(5*time.Second, 11*time.Second)
	}
}

// randomBetween usa las funciones globales de math/rand, que son seguras
// para usar desde varias goroutines a la vez.
func randomBetween(min, max time.Duration) time.Duration {
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
