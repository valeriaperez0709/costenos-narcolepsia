package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// post hace una petición POST contra el servidor y devuelve código y cuerpo.
func post(t *testing.T, s *Server, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: invalid JSON: %v", path, err)
	}
	return rec.Code, out
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// El escenario de la Sección 6 por HTTP: el cuarto paciente recibe una
// advertencia (no un fallo), pedir cama sin cupo da 409 y, al liberarse
// una, la asignación funciona.
func TestScenarioOverHTTP(t *testing.T) {
	s := newTestServer(t)
	for _, id := range []string{"P-001", "P-002", "P-003"} {
		if code, _ := post(t, s, "/api/patients/"+id+"/sleep-attack", `{"location":"cafetería"}`); code != http.StatusOK {
			t.Fatalf("%s sleep-attack: status %d", id, code)
		}
	}
	code, body := post(t, s, "/api/patients/P-004/sleep-attack", `{"location":"pasillo 2"}`)
	if code != http.StatusOK || body["warning"] == nil {
		t.Fatalf("4th attack: status %d, warning %v; want 200 with warning", code, body["warning"])
	}
	if code, _ := post(t, s, "/api/patients/P-004/assign-room", ""); code != http.StatusConflict {
		t.Fatalf("assign without free room: status %d, want 409", code)
	}
	if code, _ := post(t, s, "/api/patients/P-001/wake-up", ""); code != http.StatusOK {
		t.Fatalf("wake-up: status %d", code)
	}
	if code, _ := post(t, s, "/api/patients/P-004/assign-room", ""); code != http.StatusOK {
		t.Fatalf("assign after wake-up: status %d, want 200", code)
	}
}

// Dos peticiones piden la ÚNICA cama libre al mismo tiempo: gracias al
// mutex exactamente una gana (200) y la otra recibe "sin cama" (409).
// Correr con: go test -race ./web
func TestConcurrentAssignGetsOneBed(t *testing.T) {
	s := newTestServer(t)
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004", "P-005"} {
		post(t, s, "/api/patients/"+id+"/sleep-attack", `{"location":"pasillo"}`)
	}
	post(t, s, "/api/patients/P-001/wake-up", "") // libera una sola cama

	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, id := range []string{"P-004", "P-005"} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/patients/"+id+"/assign-room", nil)
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}(i, id)
	}
	wg.Wait()

	ok, conflict := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("codes = %v, want one 200 and one 409", codes)
	}
}

// Prender y apagar el modo carnaval mientras se lee el estado no debe
// producir carreras de datos (lo vigila -race).
func TestSimulationToggle(t *testing.T) {
	s := newTestServer(t)
	if code, _ := post(t, s, "/api/simulation", `{"running":true}`); code != http.StatusOK {
		t.Fatalf("start simulation: status %d", code)
	}
	for i := 0; i < 20; i++ {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/state", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("state: status %d", rec.Code)
		}
	}
	if code, _ := post(t, s, "/api/reset", ""); code != http.StatusOK {
		t.Fatalf("reset: status %d", code)
	}
}
