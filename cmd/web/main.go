// Command web levanta la interfaz gráfica del hospital (bonus de la
// Sección 7): una API REST hecha con net/http y una página HTML/JS que la
// consume. El paquete hospital no se modificó para agregarla.
//
//	go run ./cmd/web              # abre http://localhost:8080
//	go run -race ./cmd/web        # igual, vigilando carreras de datos
package main

import (
	"flag"
	"log"
	"net/http"

	"costenos-narcolepsia/web"
)

func main() {
	addr := flag.String("addr", ":8080", "dirección donde escucha el servidor")
	flag.Parse()

	srv, err := web.NewServer()
	if err != nil {
		log.Fatalf("could not build the demo hospital: %v", err)
	}
	log.Printf("Hospital de los Costeños abierto en http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv))
}
