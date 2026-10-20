package hospital

import (
	"fmt"
	"sync/atomic"
	"time"
)

// episodeCounter genera los números consecutivos de los episodios. Es
// atómico para que sea seguro aunque en el futuro varias goroutines
// registren episodios a la vez (bonus de concurrencia).
var episodeCounter atomic.Int64

// EpisodeRecord es el registro inmutable de UN episodio de sueño súbito:
// cuándo ocurrió, quién se durmió, dónde, quién lo atendió y a qué cama
// fue llevado (nil si no hubo cama disponible).
//
// Se maneja siempre por VALOR y todos sus campos son privados: una vez
// creado, nadie fuera del paquete puede alterar el historial; solo lee
// mediante los métodos de consulta.
type EpisodeRecord struct {
	id       string    // identificador del episodio (EP-0001...)
	dateTime time.Time // fecha y hora del episodio
	patient  *Patient  // paciente que se durmió
	attender Attender  // quién lo atendió (Doctor u Orderly, vía interfaz)
	location string    // lugar donde cayó dormido
	room     *Room     // cama asignada; nil = "puede estar vacía"
}

// newEpisodeRecord construye un registro con la hora actual. Toma la cama
// que el paciente tenga EN ESTE MOMENTO, por eso el hospital asigna la
// habitación antes de despachar al personal. Es no exportada: solo las
// implementaciones de Attender dentro del paquete crean registros.
func newEpisodeRecord(p *Patient, a Attender, location string) EpisodeRecord {
	return EpisodeRecord{
		id:       fmt.Sprintf("EP-%04d", episodeCounter.Add(1)),
		dateTime: time.Now(),
		patient:  p,
		attender: a,
		location: location,
		room:     p.room,
	}
}

// ID devuelve el identificador del episodio.
func (e EpisodeRecord) ID() string { return e.id }

// DateTime devuelve la fecha y hora del episodio.
func (e EpisodeRecord) DateTime() time.Time { return e.dateTime }

// Patient devuelve el paciente del episodio.
func (e EpisodeRecord) Patient() *Patient { return e.patient }

// Attender devuelve quien atendió el episodio.
func (e EpisodeRecord) Attender() Attender { return e.attender }

// Location devuelve dónde se durmió el paciente.
func (e EpisodeRecord) Location() string { return e.location }

// Room devuelve la cama asignada, o nil si no hubo.
func (e EpisodeRecord) Room() *Room { return e.room }

// Summary devuelve una línea legible del episodio. No imprime: devuelve el
// texto para que main decida qué hacer con él.
func (e EpisodeRecord) Summary() string {
	if e.patient == nil {
		return "empty episode record"
	}
	destination := "no room (stays in the hallway)"
	if e.room != nil {
		destination = fmt.Sprintf("room %d", e.room.number)
	}
	return fmt.Sprintf("%s  %s  %-22s -> %s",
		e.dateTime.Format("2006-01-02 15:04"), e.patient.id, e.location, destination)
}
