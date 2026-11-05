package main

import (
	"context"
	"database/sql"
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

// DTO para validación del formulario de votos
type VotoDTO struct {
	IDUsuario       int32   `json:"idusuario"`
	IDMateria       int32   `json:"idmateria"`
	PuntuacionDif   *int32  `json:"puntuacion_dif"`
	PuntuacionRel   *int32  `json:"puntuacion_rel"`
	PuntuacionGusto *int32  `json:"puntuacion_gusto"`
	Descripcion     *string `json:"descripcion"`
}

// CONVERSIONES PARA DB
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

// VALIDACIONES DE FORMULARIO
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

// HELPERS DE BD Y ERRORES AMIGABLES PARA EL USUARIO
func classifyDBError(err error) (int, string) {
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, "No se encontró el elemento solicitado."
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return http.StatusConflict, "Ya existe un registro con estos datos."
		case "23503":
			return http.StatusBadRequest, "El elemento seleccionado no es válido."
		case "23502", "23514", "22001", "22003", "22P02":
			return http.StatusBadRequest, "Los datos ingresados no son válidos o están incompletos."
		}
	}
	// Los errores técnicos internos no se muestran al usuario, solo se registran en logs
	log.Printf("Error interno de base de datos: %v", err)
	return http.StatusInternalServerError, "Ocurrió un error inesperado al procesar tu solicitud. Intentá de nuevo más tarde."
}

func parseID(raw string) (int32, bool) {
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n <= 0 {
		return 0, false
	}
	return int32(n), true
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
	log.Println("Iniciando servidor Web (vistas HTML)...")

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

// RUTAS (EXCLUSIVAMENTE HTML / FORMULARIOS)
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// ESTATICOS
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// VISTAS HTML (TEMPL)
	mux.HandleFunc("GET /{$}", s.homeHandler)
	mux.HandleFunc("GET /materias/calificar", withQueryID(s.calificarHandler))
	mux.HandleFunc("GET /materias/resenas", withQueryID(s.resenasHandler))

	// ACCIÓN DE FORMULARIO (POST) 
	mux.HandleFunc("POST /votos", s.crearVotoForm)
    // recordatorio mental, el borrar de form usa un post para borrar por lo que se saco
    //
	return mux
}

// --- HANDLERS DE VISTAS HTML ---

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

// --- PROCESAMIENTO DE FORMULARIOS ---

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

	http.Redirect(w, r, fmt.Sprintf("/materias/resenas?id=%d", idMateria), http.StatusSeeOther)
}
