package web

import "costenos-narcolepsia/hospital"

// newDemoHospital arma el mismo elenco del escenario de la Sección 6
// (2 médicos, 1 camillero, 3 habitaciones de capacidad 1 y 5 pacientes),
// usando únicamente la API pública del paquete hospital.
//
// Los datos son fijos y válidos, así que un error aquí sería un bug del
// propio seed: por eso se devuelve y el servidor no arranca.
func newDemoHospital() (*hospital.Hospital, error) {
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")

	karen, err := hospital.NewDoctor("D-001", "Karen Ospina", 41, "Neurology")
	if err != nil {
		return nil, err
	}
	andres, err := hospital.NewDoctor("D-002", "Andrés Mejía", 37, "Sleep medicine")
	if err != nil {
		return nil, err
	}
	camilo, err := hospital.NewOrderly("O-001", "Camilo Barrios", 29)
	if err != nil {
		return nil, err
	}
	for _, d := range []*hospital.Doctor{karen, andres} {
		if err := h.HireDoctor(d); err != nil {
			return nil, err
		}
	}
	if err := h.HireOrderly(camilo); err != nil {
		return nil, err
	}

	for _, number := range []int{101, 102, 103} {
		room, err := hospital.NewRoom(number, 1)
		if err != nil {
			return nil, err
		}
		if err := h.RegisterRoom(room); err != nil {
			return nil, err
		}
	}

	patients := []struct {
		id, name string
		age      int
		level    hospital.NarcolepsyLevel
	}{
		{"P-001", "Luz Marina Cassiani", 34, hospital.Mild},
		{"P-002", "Yeimy Padilla", 52, hospital.Severe},
		{"P-003", "Jhon Jairo Ortega", 45, hospital.Moderate},
		{"P-004", "Wilfrido Berrio", 61, hospital.Severe},
		{"P-005", "Dayana Mercado", 28, hospital.Mild},
	}
	for _, data := range patients {
		p, err := hospital.NewPatient(data.id, data.name, data.age, data.level)
		if err != nil {
			return nil, err
		}
		if err := h.AdmitPatient(p); err != nil {
			return nil, err
		}
	}
	return h, nil
}
