package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	db "tp2_entrega/db"
	"tp2_entrega/views"
)

// Server encapsula las dependencias de la aplicación (consultas sqlc y conexión BD)
type Server struct {
	queries *db.Queries
	dbConn  *sql.DB
}

// initDB inicializa el pool de conexiones a PostgreSQL y el cliente de sqlc
func initDB() (*sql.DB, *db.Queries, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/apirest?sslmode=disable"
	}

	dbConn, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("error al abrir el pool de conexiones: %w", err)
	}

	dbConn.SetMaxOpenConns(25)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(5 * time.Minute)

	if err := dbConn.Ping(); err != nil {
		dbConn.Close()
		return nil, nil, fmt.Errorf("error al verificar la respuesta de la base de datos (Ping): %w", err)
	}

	queries := db.New(dbConn)
	return dbConn, queries, nil
}

func main() {
	log.Println("Iniciando Servidor Web SSR con Templ y PostgreSQL (TP5)...")

	// 1. Inicialización de la base de datos
	dbConn, queries, err := initDB()
	if err != nil {
		log.Fatalf("Fallo crítico al inicializar la base de datos: %v", err)
	}
	defer dbConn.Close()

	server := &Server{
		queries: queries,
		dbConn:  dbConn,
	}

	// 2. Configuración del enrutador HTTP
	mux := http.NewServeMux()

	// Servir recursos estáticos (CSS, imágenes)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Rutas de la interfaz web (SSR con Templ)
	mux.HandleFunc("/", server.indexHandler)
	mux.HandleFunc("/materias/calificar", server.calificarHandler)
	mux.HandleFunc("/materias/resenas", server.resenasHandler)
	mux.HandleFunc("/votos", server.crearVotoHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Servidor escuchando en http://localhost:%s\n", port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
	}
}

// GET / - Página principal con el listado del plan de materias
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	materias, err := s.queries.ListarMaterias(r.Context())
	if err != nil {
		log.Printf("Error al consultar materias: %v", err)
		http.Error(w, "Error al recuperar el plan de materias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.IndexPage(materias).Render(r.Context(), w)
}

// GET /materias/calificar?id=X - Formulario para agregar una reseña a una materia específica
func (s *Server) calificarHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Identificador de materia inválido", http.StatusBadRequest)
		return
	}

	materia, err := s.queries.GetMateria(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "Materia no encontrada", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.EntityForm(materia).Render(r.Context(), w)
}

// GET /materias/resenas?id=X - Listado de reseñas registradas para una materia
func (s *Server) resenasHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Identificador de materia inválido", http.StatusBadRequest)
		return
	}

	materia, err := s.queries.GetMateria(r.Context(), int32(id))
	if err != nil {
		http.Error(w, "Materia no encontrada", http.StatusNotFound)
		return
	}

	votos, err := s.queries.ListarVotosPorMateria(r.Context(), int32(id))
	if err != nil {
		votos = []db.Voto{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.MateriaResenas(materia, votos).Render(r.Context(), w)
}

// POST /votos - Procesa la recepción de la reseña y aplica el patrón Post/Redirect/Get (PRG)
func (s *Server) crearVotoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error al procesar los datos del formulario", http.StatusBadRequest)
		return
	}

	idMateria, err := strconv.Atoi(r.FormValue("idmateria"))
	if err != nil {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
		return
	}

	puntuacionDif, _ := strconv.Atoi(r.FormValue("puntuacion_dif"))
	puntuacionRel, _ := strconv.Atoi(r.FormValue("puntuacion_rel"))
	puntuacionGusto, _ := strconv.Atoi(r.FormValue("puntuacion_gusto"))
	descripcion := r.FormValue("descripcion")

	// Guardar el voto en la base de datos (Usuario ID 1 por defecto)
	err = s.queries.InsertarVoto(r.Context(), db.InsertarVotoParams{
		Idusuario:       1,
		Idmateria:       int32(idMateria),
		PuntuacionDif:   sql.NullInt32{Int32: int32(puntuacionDif), Valid: puntuacionDif > 0},
		PuntuacionRel:   sql.NullInt32{Int32: int32(puntuacionRel), Valid: puntuacionRel > 0},
		PuntuacionGusto: sql.NullInt32{Int32: int32(puntuacionGusto), Valid: puntuacionGusto > 0},
		Descripcion:     sql.NullString{String: descripcion, Valid: descripcion != ""},
		FechaVoto:       time.Now(), // Se agrega el 7mo parámetro requerido por InsertarVoto
	})

	if err != nil {
		log.Printf("Error al insertar voto en la BD: %v", err)
		http.Error(w, "Error al guardar la reseña en la base de datos", http.StatusInternalServerError)
		return
	}

	// Aplicación estricta del patrón Post/Redirect/Get (PRG): Redirección 303 See Other
	http.Redirect(w, r, fmt.Sprintf("/materias/resenas?id=%d", idMateria), http.StatusSeeOther)
}