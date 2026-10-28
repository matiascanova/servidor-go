package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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

// Structs DTO para transferencia JSON
type UsuarioDTO struct {
	ID       int32  `json:"idusuario"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

type MateriaDTO struct {
	ID           int32  `json:"idmateria"`
	Nombre       string `json:"nombre"`
	Cuatrimestre int32  `json:"cuatrimestre"`
	Descripcion  string `json:"descripcion"`
}

type VotoDTO struct {
	IDUsuario       int32  `json:"idusuario"`
	IDMateria       int32  `json:"idmateria"`
	PuntuacionDif   int32  `json:"puntuacion_dif"`
	PuntuacionRel   int32  `json:"puntuacion_rel"`
	PuntuacionGusto int32  `json:"puntuacion_gusto"`
	Descripcion     string `json:"descripcion"`
}

// initDB inicializa la conexión con PostgreSQL
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
		return nil, nil, fmt.Errorf("error al conectar con la BD: %w", err)
	}

	queries := db.New(dbConn)
	return dbConn, queries, nil
}

func main() {
	log.Println("Iniciando servidor Web y API REST...")

	dbConn, queries, err := initDB()
	if err != nil {
		log.Fatalf("Fallo crítico al inicializar la base de datos: %v", err)
	}
	defer dbConn.Close()

	server := &Server{
		queries: queries,
		dbConn:  dbConn,
	}

	mux := http.NewServeMux()

	// 1. Archivos estáticos
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// 2. Vistas Web HTML (Templ)
	mux.HandleFunc("/", server.homeHandler)
	mux.HandleFunc("/materias/calificar", server.calificarHandler)
	mux.HandleFunc("/materias/resenas", server.resenasHandler)

	// 3. API REST JSON - Usuarios
	mux.HandleFunc("/usuarios", server.usuariosHandler)
	mux.HandleFunc("/usuarios/", server.usuarioPorIDHandler)

	// 4. API REST JSON - Materias
	mux.HandleFunc("/materias", server.materiasHandler)
	mux.HandleFunc("/materias/", server.materiaPorIDHandler)

	// 5. Votos (Atiende Formulario HTML POST y API REST JSON)
	mux.HandleFunc("/votos", server.votosDispatcher)
	mux.HandleFunc("/votos/", server.votoPorIDHandler)

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

	fmt.Printf("Servidor corriendo en http://localhost:%s\n", port)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
	}
}

// --- VISTAS HTML (TEMPL) ---

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	materias, err := s.queries.ListarMaterias(r.Context())
	if err != nil {
		http.Error(w, "Error al consultar materias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.IndexPage(materias).Render(r.Context(), w)
}

func (s *Server) calificarHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	idMateria, err := strconv.Atoi(idStr)
	if err != nil || idMateria <= 0 {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
		return
	}

	materia, err := s.queries.GetMateria(r.Context(), int32(idMateria))
	if err != nil {
		http.Error(w, "Materia no encontrada", http.StatusNotFound)
		return
	}

	idUsuario := int32(1)
	votoExistente, err := s.queries.GetVoto(r.Context(), db.GetVotoParams{
		Idusuario: idUsuario,
		Idmateria: int32(idMateria),
	})

	var votoPtr *db.Voto
	if err == nil {
		votoPtr = &votoExistente
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	views.EntityForm(materia, votoPtr).Render(r.Context(), w)
}

func (s *Server) resenasHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
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

// Dispatcher para /votos (Diferencia Formulario HTML vs JSON API)
func (s *Server) votosDispatcher(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") == "application/json" {
		if r.Method == http.MethodPost {
			s.crearVotoJSON(w, r)
		} else {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		}
	} else {
		s.crearVotoFormHandler(w, r)
	}
}

func (s *Server) crearVotoFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}

	idMateria, _ := strconv.Atoi(r.FormValue("idmateria"))
	dif, _ := strconv.Atoi(r.FormValue("puntuacion_dif"))
	rel, _ := strconv.Atoi(r.FormValue("puntuacion_rel"))
	gusto, _ := strconv.Atoi(r.FormValue("puntuacion_gusto"))
	descripcion := strings.TrimSpace(r.FormValue("descripcion"))

	idUsuario := int32(1)

	_, _ = s.queries.InsertarOActualizarVoto(r.Context(), db.InsertarOActualizarVotoParams{
		Idusuario:       idUsuario,
		Idmateria:       int32(idMateria),
		PuntuacionDif:   sql.NullInt32{Int32: int32(dif), Valid: true},
		PuntuacionRel:   sql.NullInt32{Int32: int32(rel), Valid: true},
		PuntuacionGusto: sql.NullInt32{Int32: int32(gusto), Valid: true},
		Descripcion:     sql.NullString{String: descripcion, Valid: descripcion != ""},
	})

	http.Redirect(w, r, fmt.Sprintf("/materias/resenas?id=%d", idMateria), http.StatusSeeOther)
}

// --- API REST JSON (MATERIAS) ---

func (s *Server) materiasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		materias, err := s.queries.ListarMaterias(r.Context())
		if err != nil {
			http.Error(w, "Error al consultar materias", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(materias)

	case http.MethodPost:
		var dto MateriaDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		m, err := s.queries.InsertarMateria(r.Context(), db.InsertarMateriaParams{
			Idmateria:    dto.ID,
			Nombre:       dto.Nombre,
			Cuatrimestre: dto.Cuatrimestre,
			Descripcion:  sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(m)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) materiaPorIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/materias/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		m, err := s.queries.GetMateria(r.Context(), int32(id))
		if err != nil {
			http.Error(w, "Materia no encontrada", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)

	case http.MethodPut:
		var dto MateriaDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		// Si no se definió método de update específico en sqlc, se reinserta/ignora
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dto)

	case http.MethodDelete:
		_ = s.queries.DeleteMateria(r.Context(), int32(id))
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// --- API REST JSON (VOTOS) ---

func (s *Server) crearVotoJSON(w http.ResponseWriter, r *http.Request) {
	var dto VotoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	_, err := s.queries.InsertarOActualizarVoto(r.Context(), db.InsertarOActualizarVotoParams{
		Idusuario:       dto.IDUsuario,
		Idmateria:       dto.IDMateria,
		PuntuacionDif:   sql.NullInt32{Int32: dto.PuntuacionDif, Valid: true},
		PuntuacionRel:   sql.NullInt32{Int32: dto.PuntuacionRel, Valid: true},
		PuntuacionGusto: sql.NullInt32{Int32: dto.PuntuacionGusto, Valid: true},
		Descripcion:     sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"mensaje": "votación exitosa"})
}

func (s *Server) votoPorIDHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/votos/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Ruta inválida", http.StatusBadRequest)
		return
	}

	idUsuario, err1 := strconv.Atoi(parts[0])
	idMateria, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		http.Error(w, "IDs inválidos", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		v, err := s.queries.GetVoto(r.Context(), db.GetVotoParams{
			Idusuario: int32(idUsuario),
			Idmateria: int32(idMateria),
		})
		if err != nil {
			http.Error(w, "Voto no encontrado", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)

	case http.MethodPut:
		var dto VotoDTO
		if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		dto.IDUsuario = int32(idUsuario)
		dto.IDMateria = int32(idMateria)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dto)

	case http.MethodDelete:
		_ = s.queries.DeleteVoto(r.Context(), db.DeleteVotoParams{
			Idusuario: int32(idUsuario),
			Idmateria: int32(idMateria),
		})
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

// --- API REST JSON (USUARIOS) ---

func (s *Server) usuariosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listarUsuarios(w, r)
	case http.MethodPost:
		s.crearUsuario(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) usuarioPorIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/usuarios/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "ID de usuario inválido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.obtenerUsuario(w, r, int32(id))
	case http.MethodPut:
		s.actualizarUsuario(w, r, int32(id))
	case http.MethodDelete:
		s.eliminarUsuario(w, r, int32(id))
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listarUsuarios(w http.ResponseWriter, r *http.Request) {
	usuarios, err := s.queries.ListarUsuarios(r.Context())
	if err != nil {
		http.Error(w, "Error al consultar usuarios", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usuarios)
}

func (s *Server) crearUsuario(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	u, err := s.queries.InsertarUsuario(r.Context(), db.InsertarUsuarioParams{
		Idusuario: dto.ID,
		Nombre:    dto.Nombre,
		Apellido:  dto.Apellido,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (s *Server) obtenerUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	u, err := s.queries.GetUsuario(r.Context(), id)
	if err != nil {
		http.Error(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (s *Server) actualizarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	var dto UsuarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	_ = s.queries.ActualizarUsuario(r.Context(), db.ActualizarUsuarioParams{
		Idusuario: id,
		Nombre:    dto.Nombre,
		Apellido:  dto.Apellido,
	})
	uActualizado, _ := s.queries.GetUsuario(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uActualizado)
}

func (s *Server) eliminarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	_ = s.queries.DeleteUsuario(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}
