package hospital

// Attender es la interfaz de POLIMORFISMO del hospital: representa a
// cualquier miembro del personal capaz de atender a un paciente dormido.
//
// En Go las interfaces se satisfacen de forma implícita: basta con que un
// tipo tenga estos métodos para ser un Attender, sin declarar "implements".
// Hoy la cumplen *Doctor y *Orderly; mañana podría cumplirla un *Nurse sin
// tocar el código del hospital.
//
// ID() y Name() las obtienen gratis los tipos que incrustan Person
// (método promovido); Attend y CanAttend las escribe cada tipo a su manera.
//
// El Hospital guarda a todo su personal en un []Attender y despacha a quien
// esté disponible SIN saber si es médico o camillero (ver Hospital.dispatch).
type Attender interface {
	ID() string
	Name() string
	// Attend atiende al paciente dormido y devuelve el registro del episodio.
	Attend(p *Patient, location string) (EpisodeRecord, error)
	// CanAttend indica si la persona está disponible para una nueva emergencia.
	CanAttend() bool
}
