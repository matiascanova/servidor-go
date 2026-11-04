package main

import (
	"context"
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
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	db "tp2_entrega/db"
	"tp2_entrega/views"
)

// CONFIGURACION
const usuarioDemoID int32 = 1

type Server struct {
	queries *db.Queries
	dbConn  *sql.DB
}

// DTOs
type UsuarioDTO struct {
	ID       int32  `json:"idusuario"`
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

type MateriaDTO struct {
	ID           int32   `json:"idmateria"`
	Nombre       string  `json:"nombre"`
	Cuatrimestre int32   `json:"cuatrimestre"`
	Descripcion  *string `json:"descripcion"`
}

type VotoDTO struct {
	IDUsuario       int32   `json:"idusuario"`
	IDMateria       int32   `json:"idmateria"`
	PuntuacionDif   *int32  `json:"puntuacion_dif"`
	PuntuacionRel   *int32  `json:"puntuacion_rel"`
	PuntuacionGusto *int32  `json:"puntuacion_gusto"`
	Descripcion     *string `json:"descripcion"`
}

// CONVERSIONES
func intPtr(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	v := n.Int32
	return &v
}

func strPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}

func nullInt32(p *int32) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *p, Valid: true}
}

func nullString(p *string) sql.NullString {
	if p == nil || strings.TrimSpace(*p) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(*p), Valid: true}
}

func toUsuarioDTO(u db.Usuario) UsuarioDTO {
	return UsuarioDTO{ID: u.Idusuario, Nombre: u.Nombre, Apellido: u.Apellido}
}

func toMateriaDTO(m db.Materia) MateriaDTO {
	return MateriaDTO{
		ID:           m.Idmateria,
		Nombre:       m.Nombre,
		Cuatrimestre: m.Cuatrimestre,
		Descripcion:  strPtr(m.Descripcion),
	}
}

func toVotoDTO(v db.Voto) VotoDTO {
	return VotoDTO{
		IDUsuario:       v.Idusuario,
		IDMateria:       v.Idmateria,
		PuntuacionDif:   intPtr(v.PuntuacionDif),
		PuntuacionRel:   intPtr(v.PuntuacionRel),
		PuntuacionGusto: intPtr(v.PuntuacionGusto),
		Descripcion:     strPtr(v.Descripcion),
	}
}

// VALIDACIONES
func (d UsuarioDTO) validate() error {
	if strings.TrimSpace(d.Nombre) == "" || strings.TrimSpace(d.Apellido) == "" {
		return errors.New("nombre y apellido son obligatorios")
	}
	if utf8.RuneCountInString(d.Nombre) > 50 || utf8.RuneCountInString(d.Apellido) > 50 {
		return errors.New("nombre y apellido admiten hasta 50 caracteres")
	}
	return nil
}

func (d MateriaDTO) validate() error {
	if strings.TrimSpace(d.Nombre) == "" {
		return errors.New("el nombre es obligatorio")
	}
	if utf8.RuneCountInString(d.Nombre) > 50 {
		return errors.New("el nombre admite hasta 50 caracteres")
	}
	if d.Cuatrimestre < 1 {
		return errors.New("el cuatrimestre debe ser mayor a 0")
	}
	if d.Descripcion != nil && utf8.RuneCountInString(*d.Descripcion) > 50 {
		return errors.New("la descripción admite hasta 50 caracteres")
	}
	return nil
}

func (d VotoDTO) validate() error {
	if d.IDUsuario <= 0 || d.IDMateria <= 0 {
		return errors.New("idusuario e idmateria son obligatorios")
	}
	if d.PuntuacionDif == nil && d.PuntuacionRel == nil && d.PuntuacionGusto == nil {
		return errors.New("se requiere al menos una puntuación")
	}
	for _, p := range []*int32{d.PuntuacionDif, d.PuntuacionRel, d.PuntuacionGusto} {
		if p != nil && (*p < 1 || *p > 5) {
			return errors.New("las puntuaciones deben estar entre 1 y 5")
		}
	}
	if d.Descripcion != nil && utf8.RuneCountInString(*d.Descripcion) > 140 {
		return errors.New("la descripción admite hasta 140 caracteres")
	}
	return nil
}

// HELPERS HTTP
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error al escribir JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

func classifyDBError(err error) (int, string) {
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, "recurso no encontrado"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return http.StatusConflict, "el recurso ya existe"
		case "23503":
			return http.StatusBadRequest, "referencia a un recurso inexistente"
		case "23502", "23514", "22001", "22003", "22P02":
			return http.StatusBadRequest, "datos inválidos"
		}
	}
	log.Printf("error de base de datos: %v", err)
	return http.StatusInternalServerError, "error interno del servidor"
}

func dbError(w http.ResponseWriter, err error, notFoundMsg string) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, notFoundMsg)
		return
	}
	status, msg := classifyDBError(err)
	writeError(w, status, msg)
}

func parseID(raw string) (int32, bool) {
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n <= 0 {
		return 0, false
	}
	return int32(n), true
}

func withID(msg string, h func(w http.ResponseWriter, r *http.Request, id int32)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(r.PathValue("id"))
		if !ok {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		h(w, r, id)
	}
}

func withVotoIDs(h func(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idUsuario, ok1 := parseID(r.PathValue("idusuario"))
		idMateria, ok2 := parseID(r.PathValue("idmateria"))
		if !ok1 || !ok2 {
			writeError(w, http.StatusBadRequest, "IDs inválidos")
			return
		}
		h(w, r, idUsuario, idMateria)
	}
}

func withQueryID(h func(w http.ResponseWriter, r *http.Request, id int32)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(r.URL.Query().Get("id"))
		if !ok {
			http.Error(w, "ID de materia inválido", http.StatusBadRequest)
			return
		}
		h(w, r, id)
	}
}

// INICIALIZACION
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

	const intentos = 15
	for i := 1; i <= intentos; i++ {
		if err = dbConn.Ping(); err == nil {
			return dbConn, db.New(dbConn), nil
		}
		log.Printf("BD no disponible (intento %d/%d): %v", i, intentos, err)
		time.Sleep(2 * time.Second)
	}
	dbConn.Close()
	return nil, nil, fmt.Errorf("error al conectar con la BD: %w", err)
}

func main() {
	log.Println("Iniciando servidor Web y API REST...")

	dbConn, queries, err := initDB()
	if err != nil {
		log.Fatalf("Fallo crítico al inicializar la base de datos: %v", err)
	}
	defer dbConn.Close()

	server := &Server{queries: queries, dbConn: dbConn}

	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      server.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Servidor corriendo en http://localhost%s\n", httpServer.Addr)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
	}
}

// RUTAS
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// ESTATICOS
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// HTML
	mux.HandleFunc("GET /{$}", s.homeHandler)
	mux.HandleFunc("GET /materias/calificar", withQueryID(s.calificarHandler))
	mux.HandleFunc("GET /materias/resenas", withQueryID(s.resenasHandler))

	// USUARIOS
	mux.HandleFunc("GET /usuarios", s.listarUsuarios)
	mux.HandleFunc("POST /usuarios", s.crearUsuario)
	mux.HandleFunc("GET /usuarios/{id}", withID("ID de usuario inválido", s.obtenerUsuario))
	mux.HandleFunc("PUT /usuarios/{id}", withID("ID de usuario inválido", s.actualizarUsuario))
	mux.HandleFunc("DELETE /usuarios/{id}", withID("ID de usuario inválido", s.eliminarUsuario))

	// MATERIAS
	mux.HandleFunc("GET /materias", s.listarMaterias)
	mux.HandleFunc("POST /materias", s.crearMateria)
	mux.HandleFunc("GET /materias/{id}", withID("ID de materia inválido", s.obtenerMateria))
	mux.HandleFunc("PUT /materias/{id}", withID("ID de materia inválido", s.actualizarMateria))
	mux.HandleFunc("DELETE /materias/{id}", withID("ID de materia inválido", s.eliminarMateria))

	// VOTOS
	mux.HandleFunc("POST /votos", s.crearVotoHandler)
	mux.HandleFunc("POST /votos/eliminar", s.eliminarVotoForm)
	mux.HandleFunc("GET /votos", s.listarVotos)
	mux.HandleFunc("GET /votos/{idusuario}/{idmateria}", withVotoIDs(s.obtenerVoto))
	mux.HandleFunc("PUT /votos/{idusuario}/{idmateria}", withVotoIDs(s.actualizarVoto))
	mux.HandleFunc("DELETE /votos/{idusuario}/{idmateria}", withVotoIDs(s.eliminarVoto))

	return mux
}

// HTML
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	materias, err := s.queries.ListarMaterias(r.Context())
	if err != nil {
		log.Printf("ListarMaterias: %v", err)
		http.Error(w, "Error al consultar materias", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.IndexPage(materias).Render(r.Context(), w); err != nil {
		log.Printf("render IndexPage: %v", err)
	}
}

func (s *Server) calificarHandler(w http.ResponseWriter, r *http.Request, idMateria int32) {
	materia, err := s.queries.GetMateria(r.Context(), idMateria)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Materia no encontrada", http.StatusNotFound)
		} else {
			log.Printf("GetMateria: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
		}
		return
	}

	var votoPtr *db.Voto
	voto, err := s.queries.GetVoto(r.Context(), db.GetVotoParams{
		Idusuario: usuarioDemoID,
		Idmateria: idMateria,
	})
	switch {
	case err == nil:
		votoPtr = &voto
	case errors.Is(err, sql.ErrNoRows):
	default:
		log.Printf("GetVoto: %v", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.EntityForm(materia, votoPtr).Render(r.Context(), w); err != nil {
		log.Printf("render EntityForm: %v", err)
	}
}

func (s *Server) resenasHandler(w http.ResponseWriter, r *http.Request, id int32) {
	materia, err := s.queries.GetMateria(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Materia no encontrada", http.StatusNotFound)
		} else {
			log.Printf("GetMateria: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
		}
		return
	}

	votos, err := s.queries.ListarVotosPorMateria(r.Context(), id)
	if err != nil {
		log.Printf("ListarVotosPorMateria: %v", err)
		http.Error(w, "Error al consultar reseñas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := views.MateriaResenas(materia, votos).Render(r.Context(), w); err != nil {
		log.Printf("render MateriaResenas: %v", err)
	}
}

// VOTOS
func (s *Server) guardarVoto(ctx context.Context, d VotoDTO) error {
	_, err := s.queries.InsertarOActualizarVoto(ctx, db.InsertarOActualizarVotoParams{
		Idusuario:       d.IDUsuario,
		Idmateria:       d.IDMateria,
		PuntuacionDif:   nullInt32(d.PuntuacionDif),
		PuntuacionRel:   nullInt32(d.PuntuacionRel),
		PuntuacionGusto: nullInt32(d.PuntuacionGusto),
		Descripcion:     nullString(d.Descripcion),
	})
	return err
}

func (s *Server) crearVotoHandler(w http.ResponseWriter, r *http.Request) {
	s.crearVotoForm(w, r)
}

func formInt(r *http.Request, key string) (int32, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(r.FormValue(key)), 10, 32)
	return int32(n), err
}

func (s *Server) crearVotoForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}

	idMateria, err1 := formInt(r, "idmateria")
	dif, err2 := formInt(r, "puntuacion_dif")
	rel, err3 := formInt(r, "puntuacion_rel")
	gusto, err4 := formInt(r, "puntuacion_gusto")
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		http.Error(w, "Formulario inválido: valores numéricos incorrectos", http.StatusBadRequest)
		return
	}

	dto := VotoDTO{
		IDUsuario:       usuarioDemoID,
		IDMateria:       idMateria,
		PuntuacionDif:   &dif,
		PuntuacionRel:   &rel,
		PuntuacionGusto: &gusto,
	}
	if desc := strings.TrimSpace(r.FormValue("descripcion")); desc != "" {
		dto.Descripcion = &desc
	}

	if err := dto.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.guardarVoto(r.Context(), dto); err != nil {
		status, msg := classifyDBError(err)
		http.Error(w, msg, status)
		return
	}
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) eliminarVotoForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}
	idMateria, err := formInt(r, "idmateria")
	if err != nil || idMateria <= 0 {
		http.Error(w, "ID de materia inválido", http.StatusBadRequest)
		return
	}

	if err := s.queries.DeleteVoto(r.Context(), db.DeleteVotoParams{
		Idusuario: usuarioDemoID,
		Idmateria: idMateria,
	}); err != nil {
		status, msg := classifyDBError(err)
		http.Error(w, msg, status)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) listarVotos(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.URL.Query().Get("idmateria"))
	if !ok {
		writeError(w, http.StatusBadRequest, "idmateria es obligatorio y debe ser positivo")
		return
	}
	votos, err := s.queries.ListarVotosPorMateria(r.Context(), id)
	if err != nil {
		dbError(w, err, "")
		return
	}

	out := make([]VotoDTO, 0, len(votos))
	for _, v := range votos {
		out = append(out, toVotoDTO(v))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) obtenerVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	v, err := s.queries.GetVoto(r.Context(), db.GetVotoParams{Idusuario: idUsuario, Idmateria: idMateria})
	if err != nil {
		dbError(w, err, "Voto no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, toVotoDTO(v))
}

func (s *Server) actualizarVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	var dto VotoDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	dto.IDUsuario, dto.IDMateria = idUsuario, idMateria
	if err := dto.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := db.GetVotoParams{Idusuario: idUsuario, Idmateria: idMateria}
	if _, err := s.queries.GetVoto(r.Context(), params); err != nil {
		dbError(w, err, "Voto no encontrado")
		return
	}
	if err := s.guardarVoto(r.Context(), dto); err != nil {
		dbError(w, err, "")
		return
	}
	v, err := s.queries.GetVoto(r.Context(), params)
	if err != nil {
		dbError(w, err, "Voto no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, toVotoDTO(v))
}

func (s *Server) eliminarVoto(w http.ResponseWriter, r *http.Request, idUsuario, idMateria int32) {
	if err := s.queries.DeleteVoto(r.Context(), db.DeleteVotoParams{Idusuario: idUsuario, Idmateria: idMateria}); err != nil {
		dbError(w, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MATERIAS
func (s *Server) listarMaterias(w http.ResponseWriter, r *http.Request) {
	materias, err := s.queries.ListarMaterias(r.Context())
	if err != nil {
		dbError(w, err, "")
		return
	}
	out := make([]MateriaDTO, 0, len(materias))
	for _, m := range materias {
		out = append(out, toMateriaDTO(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) crearMateria(w http.ResponseWriter, r *http.Request) {
	var dto MateriaDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if dto.ID <= 0 {
		writeError(w, http.StatusBadRequest, "idmateria debe ser positivo")
		return
	}
	if err := dto.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.queries.InsertarMateria(r.Context(), db.InsertarMateriaParams{
		Idmateria:    dto.ID,
		Nombre:       strings.TrimSpace(dto.Nombre),
		Cuatrimestre: dto.Cuatrimestre,
		Descripcion:  nullString(dto.Descripcion),
	}); err != nil {
		dbError(w, err, "")
		return
	}
	m, err := s.queries.GetMateria(r.Context(), dto.ID)
	if err != nil {
		dbError(w, err, "Materia no encontrada")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/materias/%d", dto.ID))
	writeJSON(w, http.StatusCreated, toMateriaDTO(m))
}

func (s *Server) obtenerMateria(w http.ResponseWriter, r *http.Request, id int32) {
	m, err := s.queries.GetMateria(r.Context(), id)
	if err != nil {
		dbError(w, err, "Materia no encontrada")
		return
	}
	writeJSON(w, http.StatusOK, toMateriaDTO(m))
}

func (s *Server) actualizarMateria(w http.ResponseWriter, r *http.Request, id int32) {
	var dto MateriaDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	dto.ID = id
	if err := dto.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	m, err := s.queries.ActualizarMateria(r.Context(), db.ActualizarMateriaParams{
		Idmateria:    id,
		Nombre:       strings.TrimSpace(dto.Nombre),
		Cuatrimestre: dto.Cuatrimestre,
		Descripcion:  nullString(dto.Descripcion),
	})
	if err != nil {
		dbError(w, err, "Materia no encontrada")
		return
	}
	writeJSON(w, http.StatusOK, toMateriaDTO(m))
}

func (s *Server) eliminarMateria(w http.ResponseWriter, r *http.Request, id int32) {
	if err := s.queries.DeleteMateria(r.Context(), id); err != nil {
		dbError(w, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// USUARIOS
func (s *Server) listarUsuarios(w http.ResponseWriter, r *http.Request) {
	usuarios, err := s.queries.ListarUsuarios(r.Context())
	if err != nil {
		dbError(w, err, "")
		return
	}
	out := make([]UsuarioDTO, 0, len(usuarios))
	for _, u := range usuarios {
		out = append(out, toUsuarioDTO(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) crearUsuario(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if dto.ID <= 0 {
		writeError(w, http.StatusBadRequest, "idusuario debe ser positivo")
		return
	}
	if err := dto.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.queries.InsertarUsuario(r.Context(), db.InsertarUsuarioParams{
		Idusuario: dto.ID,
		Nombre:    strings.TrimSpace(dto.Nombre),
		Apellido:  strings.TrimSpace(dto.Apellido),
	}); err != nil {
		dbError(w, err, "")
		return
	}
	u, err := s.queries.GetUsuario(r.Context(), dto.ID)
	if err != nil {
		dbError(w, err, "Usuario no encontrado")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/usuarios/%d", dto.ID))
	writeJSON(w, http.StatusCreated, toUsuarioDTO(u))
}

func (s *Server) obtenerUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	u, err := s.queries.GetUsuario(r.Context(), id)
	if err != nil {
		dbError(w, err, "Usuario no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, toUsuarioDTO(u))
}

func (s *Server) actualizarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	var dto UsuarioDTO
	if !decodeJSON(w, r, &dto) {
		return
	}
	if err := dto.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := s.queries.GetUsuario(r.Context(), id); err != nil {
		dbError(w, err, "Usuario no encontrado")
		return
	}
	if err := s.queries.ActualizarUsuario(r.Context(), db.ActualizarUsuarioParams{
		Idusuario: id,
		Nombre:    strings.TrimSpace(dto.Nombre),
		Apellido:  strings.TrimSpace(dto.Apellido),
	}); err != nil {
		dbError(w, err, "")
		return
	}
	u, err := s.queries.GetUsuario(r.Context(), id)
	if err != nil {
		dbError(w, err, "Usuario no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, toUsuarioDTO(u))
}

func (s *Server) eliminarUsuario(w http.ResponseWriter, r *http.Request, id int32) {
	if err := s.queries.DeleteUsuario(r.Context(), id); err != nil {
		dbError(w, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
