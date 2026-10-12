package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	db "tp2_entrega/db"
)

// Server encapsula las dependencias de la aplicación (consultas sqlc y conexión BD)
type Server struct {
	queries *db.Queries
	dbConn  *sql.DB
}

// UsuarioDTO representa la estructura para la transferencia de datos JSON de Usuario
type UsuarioDTO struct {
	ID       int32  `json:"idusuario"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Curso    bool   `json:"curso"`
}

type MateriaDTO struct {
	ID           int32  `json:"idmateria"`
	Nombre       string `json:"nombre"`
	Cuatrimestre int32  `json:"cuatrimestre"`
	Descripcion  string `json:"descripcion,omitempty"`
}

type VotoDTO struct {
	IDUsuario       int32     `json:"idusuario"`
	IDMateria       int32     `json:"idmateria"`
	PuntuacionDif   int32     `json:"puntuacion_dif"`
	PuntuacionRel   int32     `json:"puntuacion_rel"`
	PuntuacionGusto int32     `json:"puntuacion_gusto"`
	Descripcion     string    `json:"descripcion,omitempty"`
	FechaVoto       time.Time `json:"fecha_voto,omitempty"`
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

	// Configuración de parámetros del pool de conexiones
	dbConn.SetMaxOpenConns(25)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(5 * time.Minute)

	// Verificación de conectividad activa
	if err := dbConn.Ping(); err != nil {
		dbConn.Close()
		return nil, nil, fmt.Errorf("error al verificar la respuesta de la base de datos (Ping): %w", err)
	}

	queries := db.New(dbConn)
	return dbConn, queries, nil
}

func main() {
	log.Println("Iniciando servidor API REST (TP3)...")

	// 1. Inicialización de la base de datos
	dbConn, queries, err := initDB()
	if err != nil {
		log.Fatalf("Fallo crítico al inicializar la base de datos: %v", err)
	}
	defer dbConn.Close()

	// 2. Instanciación del servidor con inyección de dependencias
	server := &Server{
		queries: queries,
		dbConn:  dbConn,
	}

	log.Println("¡Base de datos PostgreSQL conectada y repositorio sqlc instanciado con éxito!")

	// 3. Configuración de rutas HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("/usuarios", server.usuariosHandler)
	mux.HandleFunc("/usuarios/", server.usuarioPorIDHandler)
	mux.HandleFunc("/materias", server.materiasHandler)
	mux.HandleFunc("/materias/", server.materiaPorIDHandler)
	mux.HandleFunc("/votos", server.votosHandler)
	mux.HandleFunc("/votos/", server.votoPorIDsHandler)

	// Obtener puerto desde variable de entorno
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
No Content

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Servidor escuchando peticiones en http://localhost:%s\n", port)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error fatal al iniciar el servidor HTTP: %v", err)
	}
}

// --- HANDLERS HTTP PARA LA ENTIDAD USUARIO ---

// usuariosHandler conmuta entre listar todos los usuarios (GET) y crear uno nuevo (POST)
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

// usuarioPorIDHandler gestiona las operaciones sobre un usuario específico por su ID (GET, PUT, DELETE)
func (s *Server) usuarioPorIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extraer el ID numérico desde la ruta de la URL (/usuarios/{id})
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

// GET /usuarios -> Lista todos los usuarios registrados
func (s *Server) listarUsuarios(w http.ResponseWriter, r *http.Request) {
	usuarios, err := s.queries.ListarUsuarios(r.Context())
	if err != nil {
		http.Error(w, "Error al consultar la lista de usuarios", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(usuarios)
}

// POST /usuarios -> Crea un nuevo usuario recibiendo un cuerpo JSON
func (s *Server) crearUsuario(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	// Validación de campos obligatorios
	if strings.TrimSpace(dto.Nombre) == "" || strings.TrimSpace(dto.Apellido) == "" {
		http.Error(w, "El nombre y el apellido son obligatorios", http.StatusBadRequest)
		return
	}

	u, err := s.queries.InsertarUsuario(r.Context(), db.InsertarUsuarioParams{
		Idusuario: dto.ID,
		Nombre:    dto.Nombre,
		Apellido:  dto.Apellido,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al insertar el usuario: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

// GET /usuarios/{id} -> Obtiene el usuario correspondiente al ID indicado
func (s *Server) obtenerUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	u, err := s.queries.GetUsuario(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Usuario no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al consultar el usuario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(u)
}

// PUT /usuarios/{id} -> Actualiza los datos de un usuario existente
func (s *Server) actualizarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	var dto UsuarioDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(dto.Nombre) == "" || strings.TrimSpace(dto.Apellido) == "" {
		http.Error(w, "El nombre y el apellido son obligatorios", http.StatusBadRequest)
		return
	}

	err := s.queries.ActualizarUsuario(r.Context(), db.ActualizarUsuarioParams{
		Idusuario: id,
		Nombre:    dto.Nombre,
		Apellido:  dto.Apellido,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Usuario no encontrado para actualizar", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al actualizar el usuario", http.StatusInternalServerError)
		return
	}

	// Recuperar el usuario actualizado para responder la entidad en la respuesta HTTP
	uActualizado, _ := s.queries.GetUsuario(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(uActualizado)
}

// DELETE /usuarios/{id} -> Elimina a un usuario por su ID
func (s *Server) eliminarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	err := s.queries.DeleteUsuario(r.Context(), id)
	if err != nil {
		http.Error(w, "Error al eliminar el usuario", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}


// ============================================================================
// --- HANDLERS HTTP PARA LA ENTIDAD MATERIA ---
// ============================================================================

func (s *Server) materiasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listarMaterias(w, r)
	case http.MethodPost:
		s.crearMateria(w, r)
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
		s.obtenerMateria(w, r, int32(id))
	case http.MethodPut:
		s.actualizarMateria(w, r, int32(id))
	case http.MethodDelete:
		s.eliminarMateria(w, r, int32(id))
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listarMaterias(w http.ResponseWriter, r *http.Request) {
	materias, err := s.queries.ListarMaterias(r.Context())
	if err != nil {
		http.Error(w, "Error al consultar la lista de materias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(materias)
}

func (s *Server) crearMateria(w http.ResponseWriter, r *http.Request) {
	var dto MateriaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(dto.Nombre) == "" || dto.Cuatrimestre <= 0 {
		http.Error(w, "El nombre y un cuatrimestre válido son obligatorios", http.StatusBadRequest)
		return
	}

	m, err := s.queries.InsertarMateria(r.Context(), db.InsertarMateriaParams{
		Idmateria:    dto.ID,
		Nombre:       dto.Nombre,
		Cuatrimestre: dto.Cuatrimestre,
		Descripcion:  sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al insertar la materia: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

func (s *Server) obtenerMateria(w http.ResponseWriter, r *http.Request, id int32) {
	m, err := s.queries.GetMateria(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Materia no encontrada", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al consultar la materia", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

func (s *Server) actualizarMateria(w http.ResponseWriter, r *http.Request, id int32) {
	var dto MateriaDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(dto.Nombre) == "" || dto.Cuatrimestre <= 0 {
		http.Error(w, "El nombre y un cuatrimestre válido son obligatorios", http.StatusBadRequest)
		return
	}

	err := s.queries.ActualizarMateria(r.Context(), db.ActualizarMateriaParams{
		Idmateria:    id,
		Nombre:       dto.Nombre,
		Cuatrimestre: dto.Cuatrimestre,
		Descripcion:  sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Materia no encontrada para actualizar", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al actualizar la materia", http.StatusInternalServerError)
		return
	}

	mActualizada, _ := s.queries.GetMateria(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(mActualizada)
}

func (s *Server) eliminarMateria(w http.ResponseWriter, r *http.Request, id int32) {
	err := s.queries.DeleteMateria(r.Context(), id)
	if err != nil {
		http.Error(w, "Error al eliminar la materia", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- HANDLERS HTTP PARA LA ENTIDAD VOTO ---
func (s *Server) votosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.crearVoto(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) votoPorIDsHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/votos/")
	partes := strings.Split(path, "/")

	if len(partes) < 2 {
		http.Error(w, "Formato de URL inválido. Se espera: /votos/{idusuario}/{idmateria}", http.StatusBadRequest)
		return
	}

	idUsuario, err1 := strconv.Atoi(partes[0])
	idMateria, err2 := strconv.Atoi(partes[1])

	if err1 != nil || err2 != nil || idUsuario <= 0 || idMateria <= 0 {
		http.Error(w, "IDs de usuario y materia deben ser enteros válidos", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.obtenerVoto(w, r, int32(idUsuario), int32(idMateria))
	case http.MethodPut:
		s.actualizarVoto(w, r, int32(idUsuario), int32(idMateria))
	case http.MethodDelete:
		s.eliminarVoto(w, r, int32(idUsuario), int32(idMateria))
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Server) crearVoto(w http.ResponseWriter, r *http.Request) {
	var dto VotoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	if dto.IDUsuario <= 0 || dto.IDMateria <= 0 {
		http.Error(w, "Los IDs de usuario y materia son obligatorios", http.StatusBadRequest)
		return
	}

	fecha := dto.FechaVoto
	if fecha.IsZero() {
		fecha = time.Now()
	}

	err := s.queries.InsertarVoto(r.Context(), db.InsertarVotoParams{
		Idusuario:       dto.IDUsuario,
		Idmateria:       dto.IDMateria,
		PuntuacionDif:   sql.NullInt32{Int32: dto.PuntuacionDif, Valid: dto.PuntuacionDif >= 1 && dto.PuntuacionDif <= 5},
		PuntuacionRel:   sql.NullInt32{Int32: dto.PuntuacionRel, Valid: dto.PuntuacionRel >= 1 && dto.PuntuacionRel <= 5},
		PuntuacionGusto: sql.NullInt32{Int32: dto.PuntuacionGusto, Valid: dto.PuntuacionGusto >= 1 && dto.PuntuacionGusto <= 5},
		Descripcion:     sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
		FechaVoto:       fecha,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al insertar el voto: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"mensaje": "Voto creado exitosamente"})
}

func (s *Server) obtenerVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	v, err := s.queries.GetVoto(r.Context(), db.GetVotoParams{
		Idusuario: idUsuario,
		Idmateria: idMateria,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Voto no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al consultar el voto", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) actualizarVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	var dto VotoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, "Cuerpo JSON inválido", http.StatusBadRequest)
		return
	}

	fecha := dto.FechaVoto
	if fecha.IsZero() {
		fecha = time.Now()
	}

	err := s.queries.ActualizarVoto(r.Context(), db.ActualizarVotoParams{
		Idusuario:       idUsuario,
		Idmateria:       idMateria,
		PuntuacionDif:   sql.NullInt32{Int32: dto.PuntuacionDif, Valid: dto.PuntuacionDif >= 1 && dto.PuntuacionDif <= 5},
		PuntuacionRel:   sql.NullInt32{Int32: dto.PuntuacionRel, Valid: dto.PuntuacionRel >= 1 && dto.PuntuacionRel <= 5},
		PuntuacionGusto: sql.NullInt32{Int32: dto.PuntuacionGusto, Valid: dto.PuntuacionGusto >= 1 && dto.PuntuacionGusto <= 5},
		Descripcion:     sql.NullString{String: dto.Descripcion, Valid: dto.Descripcion != ""},
		FechaVoto:       fecha,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Voto no encontrado para actualizar", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al actualizar el voto", http.StatusInternalServerError)
		return
	}

	vActualizado, _ := s.queries.GetVoto(r.Context(), db.GetVotoParams{
		Idusuario: idUsuario,
		Idmateria: idMateria,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(vActualizado)
}

func (s *Server) eliminarVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	err := s.queries.DeleteVoto(r.Context(), db.DeleteVotoParams{
		Idusuario: idUsuario,
		Idmateria: idMateria,
	})
	if err != nil {
		http.Error(w, "Error al eliminar el voto", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}