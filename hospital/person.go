package hospital

import (
	"fmt"
	"strings"
)

// Person agrupa los datos de identidad que comparten pacientes, médicos y
// camilleros (id, nombre y edad).
//
// Es la pieza de COMPOSICIÓN (embedding) del modelo: en lugar de copiar los
// tres campos en cada struct, Patient, Doctor y Orderly incrustan Person por
// VALOR. Gracias a eso, los métodos ID(), Name() y Age() quedan "promovidos":
// se pueden llamar directamente sobre un *Patient o un *Doctor, y además
// son justamente los métodos que la interfaz Attender exige para ID/Name.
//
// Los campos van en minúscula (no exportados): desde fuera del paquete solo
// se pueden leer mediante los métodos, nunca modificar.
type Person struct {
	id   string
	name string
	age  int
}

// newPerson valida los datos y construye una Person. Es no exportada porque
// solo los constructores públicos (NewPatient, NewDoctor, NewOrderly) deben
// usarla.
func newPerson(id, name string, age int) (Person, error) {
	if strings.TrimSpace(id) == "" {
		return Person{}, fmt.Errorf("%w: id is empty", ErrInvalidData)
	}
	if strings.TrimSpace(name) == "" {
		return Person{}, fmt.Errorf("%w: name is empty", ErrInvalidData)
	}
	if age < 0 || age > 130 {
		return Person{}, fmt.Errorf("%w: age %d out of range", ErrInvalidData, age)
	}
	return Person{id: id, name: name, age: age}, nil
}

// ID devuelve el identificador de la persona. Receptor por valor: solo lee.
func (p Person) ID() string { return p.id }

// Name devuelve el nombre de la persona.
func (p Person) Name() string { return p.name }

// Age devuelve la edad de la persona.
func (p Person) Age() int { return p.age }
