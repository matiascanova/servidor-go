package main

import (
	"fmt"
	"net/http"
)

func main() {
	
	htmlContent := `<!DOCTYPE html>
	<html>
	<head><title>Hola Mundo</title></head>
	<body><h1>¡Servidor Funcionando!</h1></body>
	</html>`

	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, htmlContent)
	})

	
	port := ":8080"
	fmt.Printf("Servidor escuchando en http://localhost%s\n", port)

	
	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}