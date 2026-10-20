// Command main construye el escenario de demostración del Hospital de los
// Costeños con Narcolepsia e imprime los resultados.
//
// Este archivo NO contiene lógica de negocio: solo crea objetos, llama a los
// métodos del paquete hospital e imprime lo que estos devuelven.
package main

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"costenos-narcolepsia/hospital"
)

// must devuelve el valor si no hubo error y, si lo hubo, termina el programa.
// Solo se usa al armar el escenario, donde un error sería un bug del script.
func must[T any](value T, err error) T {
	if err != nil {
		log.Fatalf("scenario setup failed: %v", err)
	}
	return value
}

// check termina el programa si err no es nil (para llamadas sin valor).
func check(err error) {
	if err != nil {
		log.Fatalf("scenario setup failed: %v", err)
	}
}

func main() {
	// ---- Paso 1: hospital, 2 médicos, 1 camillero, 3 habitaciones, 5 pacientes
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")

	karen := must(hospital.NewDoctor("D-001", "Karen Ospina", 41, "Neurology"))
	andres := must(hospital.NewDoctor("D-002", "Andrés Mejía", 37, "Sleep medicine"))
	camilo := must(hospital.NewOrderly("O-001", "Camilo Barrios", 29))
	check(h.HireDoctor(karen))
	check(h.HireDoctor(andres))
	check(h.HireOrderly(camilo))

	for _, number := range []int{101, 102, 103} {
		check(h.RegisterRoom(must(hospital.NewRoom(number, 1))))
	}

	luz := must(hospital.NewPatient("P-001", "Luz Marina Cassiani", 34, hospital.Mild))
	yeimy := must(hospital.NewPatient("P-002", "Yeimy Padilla", 52, hospital.Severe))
	jhon := must(hospital.NewPatient("P-003", "Jhon Jairo Ortega", 45, hospital.Moderate))
	wilfrido := must(hospital.NewPatient("P-004", "Wilfrido Berrio", 61, hospital.Severe))
	dayana := must(hospital.NewPatient("P-005", "Dayana Mercado", 28, hospital.Mild))
	for _, p := range []*hospital.Patient{luz, yeimy, jhon, wilfrido, dayana} {
		check(h.AdmitPatient(p))
	}

	fmt.Printf("%s: %d staff members, %d rooms, %d patients admitted\n\n",
		h.Name(), len(h.Staff()), len(h.Rooms()), len(h.Patients()))

	// ---- Paso 2: cuatro ataques de sueño; los tres primeros consiguen cama
	attacks := []struct {
		patient  *hospital.Patient
		location string
	}{
		{luz, "cafeteria"},
		{yeimy, "waiting room"},
		{jhon, "radiology queue"},
		{wilfrido, "hallway 2, second floor"},
	}

	var failedAssign error // el error de "sin cama", para mostrarlo luego
	fmt.Println("== Sleep attacks ==")
	for _, a := range attacks {
		record, err := h.HandleSleepAttack(a.patient, a.location)
		switch {
		case err == nil:
			fmt.Printf("  %s (attended by %s)\n", record.Summary(), record.Attender().Name())
		case errors.Is(err, hospital.ErrNoRoomAvailable):
			// No es fatal: el episodio se atendió, pero el paciente sigue en el pasillo.
			failedAssign = err
			fmt.Printf("  %s (attended by %s)\n", record.Summary(), record.Attender().Name())
			fmt.Printf("  ! %v\n", err)
		default:
			log.Fatalf("unexpected error: %v", err)
		}
	}
	fmt.Println()

	// ---- Paso 3: un paciente despierta, libera su cama y el del pasillo la toma
	check(luz.WakeUp())
	fmt.Printf("== %s woke up and released their room ==\n\n", luz.Name())
	assignedRoom, err := h.AssignRoom(wilfrido)
	check(err)

	// ---- Paso 4: resultado de las cuatro consultas de la Sección 5

	// Consulta 5.1: pacientes dormidos en pasillo.
	fmt.Println("== Patients asleep in hallway ==")
	hallway := h.PatientsInHallway()
	if len(hallway) == 0 {
		fmt.Println("  (nobody: every sleeping patient has a bed)")
	}
	for _, p := range hallway {
		fmt.Printf("  %s  %-22s (%s)\n", p.ID(), p.Name(), p.Location())
	}
	fmt.Println()

	// Consulta 5.2: episodios por médico.
	for _, d := range h.Doctors() {
		fmt.Printf("== Episodes attended by Dr. %s ==\n", d.Name())
		episodes := d.MyEpisodes()
		if len(episodes) == 0 {
			fmt.Println("  (none)")
		}
		for _, e := range episodes {
			fmt.Printf("  %s\n", e.Summary())
		}
		fmt.Println()
	}

	// Consulta 5.3: asignación de cama (primer intento fallido y reintento).
	fmt.Printf("== Assigning room to %s ==\n", wilfrido.ID())
	fmt.Printf("  first attempt: %v\n", failedAssign)
	fmt.Printf("  after %s woke up: assigned to room %d (state: %s)\n\n",
		luz.ID(), assignedRoom.Number(), wilfrido.State())

	// Consulta 5.4: reporte de severidad. Los maps no tienen orden, así que
	// se ordenan las claves por id antes de imprimir.
	fmt.Println("== Severity report (Severe patients) ==")
	report := h.SevereReport()
	severe := make([]*hospital.Patient, 0, len(report))
	for p := range report {
		severe = append(severe, p)
	}
	sort.Slice(severe, func(i, j int) bool { return severe[i].ID() < severe[j].ID() })
	for _, p := range severe {
		fmt.Printf("  %s  %-22s %d episodes today\n", p.ID(), p.Name(), report[p])
	}
}
