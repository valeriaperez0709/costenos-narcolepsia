package hospital

import "fmt"

// Orderly es el camillero del hospital. Es el SEGUNDO tipo que cumple la
// interfaz Attender: puede atender a un paciente dormido y trasladarlo a la
// cama, pero NO puede diagnosticar (por eso no tiene DiagnosePatient).
//
// Al igual que Doctor, incrusta Person y obtiene ID()/Name() gratis.
type Orderly struct {
	Person                   // embebido: id, name, age
	episodes []EpisodeRecord // traslados realizados
}

// Verificación en tiempo de compilación de que Orderly es un Attender.
var _ Attender = (*Orderly)(nil)

// NewOrderly valida los datos y crea un camillero.
func NewOrderly(id, name string, age int) (*Orderly, error) {
	person, err := newPerson(id, name, age)
	if err != nil {
		return nil, err
	}
	return &Orderly{Person: person}, nil
}

// Attend implementa Attender: el camillero atiende al paciente dormido
// (lo traslada a su cama si ya la tiene) y deja constancia del episodio.
func (o *Orderly) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if p == nil {
		return EpisodeRecord{}, fmt.Errorf("%w: nil patient", ErrInvalidData)
	}
	if p.state == Awake {
		return EpisodeRecord{}, fmt.Errorf("patient %s: %w", p.id, ErrNotAsleep)
	}
	record := newEpisodeRecord(p, o, location)
	o.episodes = append(o.episodes, record)
	return record, nil
}

// CanAttend implementa Attender: el camillero siempre está disponible.
func (o *Orderly) CanAttend() bool { return true }

// MyEpisodes devuelve una copia de los traslados realizados por el camillero.
func (o *Orderly) MyEpisodes() []EpisodeRecord {
	return append([]EpisodeRecord(nil), o.episodes...)
}
