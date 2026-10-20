# Hospital de los Costeños con Narcolepsia

Object-Oriented Programming — Universidad EIA. Written in Go (1.21+), standard library only.

**Group members:** _<your name>_

## Run

```bash
go run .                 # runs the demonstration scenario (Section 6)
go test ./...            # runs the tests
go run ./cmd/web         # graphical interface (bonus) at http://localhost:8080
go run -race ./cmd/web   # same, with the race detector (concurrency bonus)
go test -race ./...      # tests with the race detector
```

## Layout

```
main.go              scenario + printing only (no business logic)
hospital/
  person.go          Person (id, name, age) — embedded in Patient, Doctor, Orderly
  patient.go         Patient, PatientState, NarcolepsyLevel
  doctor.go          Doctor (implements Attender)
  orderly.go         Orderly (second Attender implementation)
  attender.go        Attender interface
  room.go            Room, RoomState
  episode.go         EpisodeRecord
  hospital.go        Hospital + the four queries
  hospital_test.go   tests (external package: public API only)
web/                 presentation layer (bonus) — REST API + HTML/JS, standard library only
  server.go          net/http handlers, sync.Mutex around the hospital, error -> HTTP status
  snapshot.go        DTOs with JSON tags (the model has unexported fields, so the view copies them)
  simulation.go      "modo carnaval": one goroutine per patient, protected by the same mutex
  seed.go            builds the Section 6 cast through the public hospital API
  server_test.go     HTTP scenario, two goroutines racing for one bed, simulation toggle
  static/index.html  the page (vanilla HTML/CSS/JS, embedded in the binary with go:embed)
cmd/web/main.go      starts the web server
```

## Design decisions

- **Encapsulation:** every field is lowercase; state changes only through methods.
  Constructors (`NewPatient`, `NewRoom`, ...) validate data and return `error`.
- **Composition:** `Person` is embedded *by value* in `Patient`, `Doctor` and `Orderly`;
  `ID()` and `Name()` are promoted and satisfy part of `Attender` for free.
- **Polymorphism:** the hospital keeps `staff []Attender` (doctors + orderlies) and
  `dispatch()` picks the next one with `CanAttend()` in round-robin, never
  checking the concrete type.
- **Doctor.MyEpisodes() takes no hospital:** each doctor stores its own episodes.
- **No bed is not fatal:** `AssignRoom` returns the sentinel `ErrNoRoomAvailable`
  (check with `errors.Is`). `HandleSleepAttack` still attends and records the
  episode, returning the record plus that error; the patient stays in the hallway.
- **`SevereReport` returns a `map[*Patient]int`:** it is a patient -> count pairing.
  Maps are unordered, so `main` sorts the keys before printing.
- **Typed constants** (`iota`) with `String()` for patient state, narcolepsy level and room state.
- **GUI without touching the model (bonus):** the `hospital` package did not change
  at all. `web` only calls its public methods and converts the results to JSON.
  `main.go` (console scenario) is also untouched.
- **API:** `GET /api/state` (full snapshot, including queries 5.1, 5.2 and 5.4),
  `POST /api/patients/{id}/sleep-attack|wake-up|assign-room` (5.3 is `assign-room`),
  `POST /api/reset`, `POST /api/simulation`. Sentinel errors map to HTTP codes:
  `ErrPatientNotFound` -> 404, `ErrInvalidData` -> 400, the rest (no room, already
  asleep...) -> 409. A sleep attack without a bed is a 200 with a warning, because
  the episode *was* attended and recorded.
- **Concurrency:** `net/http` serves each request in its own goroutine and the
  simulation adds one goroutine per patient. The model has no locks, so the web
  layer serializes every access with one `sync.Mutex`. If two requests ask for the
  last bed at the same time, one gets it and the other gets `ErrNoRoomAvailable`
  (tested in `TestConcurrentAssignGetsOneBed`). Runs clean under `-race`.
