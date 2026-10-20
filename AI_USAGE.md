# AI_USAGE.md
## Herramientas usadas y para qué

- **Claude (Anthropic):** Lo utilicé como herramienta de apoyo para mejorar el proyecto. Primero desarrollé un código base por mi cuenta y después compartí mi implementación inicial junto con el enunciado completo del proyecto en PDF. Le pedí revisar y mejorar el backend en Go de acuerdo con los requisitos, criterios de evaluación y estructura solicitada. Claude me ayudó a corregir errores, mejorar la organización del código y revisar componentes como el modelo, la interfaz `Attender`, las consultas, `main.go`, las pruebas y el README. También me explicó las decisiones de diseño realizadas.

- **Mi parte:** Desarrollé la primera versión del código a partir de los requisitos iniciales del proyecto. Posteriormente, revisé las sugerencias de Claude y utilicé sus recomendaciones para mejorar la implementación. Ejecuté el proyecto con `go run .` y las pruebas con `go test ./...`. También revisé el formato con `gofmt` y realicé comprobaciones con `go vet`. Finalmente, estudié el código para comprender su funcionamiento y poder explicarlo y modificarlo durante la sustentación.

## Qué aprendí

- Cómo Go permite utilizar la composición mediante estructuras embebidas, como `Person`, para reutilizar características sin recurrir a la herencia tradicional.
- Cómo funcionan las interfaces en Go y cómo una estructura puede implementarlas de manera implícita, como ocurre con `Attender`.
- Por qué los campos con nombres en minúscula no son accesibles directamente desde otros paquetes y cómo se puede controlar el acceso mediante métodos.
- Por qué una situación esperada, como no disponer de una cama, debe manejarse mediante errores y no mediante `panic`.
- La importancia de revisar el código generado o sugerido por herramientas de inteligencia artificial, ejecutar pruebas y comprobar que la implementación cumpla los requisitos del proyecto.
