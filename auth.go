package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgconn"

	db "tp2_entrega/db"
	"tp2_entrega/views"
)

const (
	sessionCookie = "session"
	sessionTTL    = 7 * 24 * time.Hour

	// PBKDF2-HMAC-SHA256 (crypto/pbkdf2 está en la biblioteca estándar desde Go 1.24).
	// 600.000 iteraciones es la recomendación de OWASP para esta variante.
	pbkdf2Iter   = 600_000
	pbkdf2KeyLen = 32
	saltLen      = 16

	minPasswordLen = 8
	maxPasswordLen = 128
)

// ---------------------------------------------------------------------------
// CONTRASEÑAS
// ---------------------------------------------------------------------------

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iter, pbkdf2KeyLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func verifyPassword(stored, password string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 || iter > 10_000_000 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}

// dummyHash se usa en el login cuando el email no existe, para gastar el mismo
// tiempo de CPU que con un usuario real y no revelar qué cuentas están registradas.
var dummyHash, _ = hashPassword("contraseña-descartable")

// ---------------------------------------------------------------------------
// SESIONES
// ---------------------------------------------------------------------------

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Server) newSessionCookie(value string, expira time.Time, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    value,
		Path:     "/",
		Expires:  expira,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, idUsuario int32) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	expira := time.Now().Add(sessionTTL)
	if err := s.queries.CrearSesion(r.Context(), db.CrearSesionParams{
		TokenHash: hashToken(token),
		Idusuario: idUsuario,
		Expira:    expira,
	}); err != nil {
		return err
	}
	http.SetCookie(w, s.newSessionCookie(token, expira, int(sessionTTL.Seconds())))
	return nil
}

func (s *Server) endSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.queries.BorrarSesion(r.Context(), hashToken(c.Value)); err != nil {
			log.Printf("BorrarSesion: %v", err)
		}
	}
	http.SetCookie(w, s.newSessionCookie("", time.Unix(0, 0), -1))
}

// Borra de a poco las sesiones vencidas para que la tabla no crezca sin límite.
func (s *Server) limpiarSesionesPeriodicamente() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for ; ; <-t.C {
		if err := s.queries.LimpiarSesionesExpiradas(context.Background()); err != nil {
			log.Printf("LimpiarSesionesExpiradas: %v", err)
		}
	}
}


// withUser lee la cookie de sesión y, si es válida, deja el usuario en el contexto
// de la petición (las vistas lo usan para mostrar "Hola, X / Salir"). Si no hay
// sesión o venció, la petición sigue como invitado.
func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
				u, err := s.queries.GetUsuarioPorSesion(r.Context(), hashToken(c.Value))
				switch {
				case err == nil:
					ctx := views.WithUsuario(r.Context(), views.Usuario{
						ID:       u.Idusuario,
						Nombre:   u.Nombre,
						Apellido: u.Apellido,
					})
					r = r.WithContext(ctx)
				case errors.Is(err, sql.ErrNoRows):
					// sesión inexistente o vencida: se trata como invitado
				default:
					log.Printf("GetUsuarioPorSesion: %v", err)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireUser protege un handler: sin sesión no se ejecuta.
//   - Petición de HTMX: 401 + header HX-Redirect, así el navegador va a /login
//     (un 302 haría que HTMX metiera la página de login dentro del target).
//   - GET normal: redirección a /login?next=<página pedida>.
//   - Cualquier otro método: 401.
func (s *Server) requireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := views.UsuarioDe(r.Context()); ok {
			next(w, r)
			return
		}
		switch {
		case r.Header.Get("HX-Request") == "true":
			w.Header().Set("HX-Redirect", "/login")
			http.Error(w, "Tenés que iniciar sesión.", http.StatusUnauthorized)
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		default:
			http.Error(w, "Tenés que iniciar sesión.", http.StatusUnauthorized)
		}
	}
}

// userID devuelve el id del usuario logueado. Solo se usa en handlers envueltos
// por requireUser, donde siempre hay sesión. El id sale de la sesión y nunca del
// formulario o de la URL: por eso nadie puede tocar reseñas ajenas.
func userID(r *http.Request) int32 {
	u, _ := views.UsuarioDe(r.Context())
	return u.ID
}

// ---------------------------------------------------------------------------
// UTILIDADES
// ---------------------------------------------------------------------------

func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("render: %v", err)
	}
}

// safeNext evita open redirects: solo se acepta un camino local ("/algo"), nunca
// "//otro-sitio.com" ni una URL absoluta.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") ||
		strings.ContainsAny(next, "\\\r\n") {
		return "/"
	}
	return next
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isEmailTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "usuario_email_uk"
}

func validarRegistro(f views.RegistroForm, pass, pass2 string) error {
	if f.Nombre == "" || f.Apellido == "" {
		return errors.New("Nombre y apellido son obligatorios.")
	}
	if utf8.RuneCountInString(f.Nombre) > 50 || utf8.RuneCountInString(f.Apellido) > 50 {
		return errors.New("Nombre y apellido admiten hasta 50 caracteres.")
	}
	addr, err := mail.ParseAddress(f.Email)
	if err != nil || addr.Address != f.Email || len(f.Email) > 120 {
		return errors.New("El email no es válido.")
	}
	if n := utf8.RuneCountInString(pass); n < minPasswordLen || n > maxPasswordLen {
		return fmt.Errorf("La contraseña debe tener entre %d y %d caracteres.", minPasswordLen, maxPasswordLen)
	}
	if pass != pass2 {
		return errors.New("Las contraseñas no coinciden.")
	}
	return nil
}

// ---------------------------------------------------------------------------
// HANDLERS
// ---------------------------------------------------------------------------

func (s *Server) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	if _, ok := views.UsuarioDe(r.Context()); ok {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, views.LoginPage("", next, ""))
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	next := safeNext(r.FormValue("next"))
	const credencialesInvalidas = "Email o contraseña incorrectos."

	u, err := s.queries.GetUsuarioPorEmail(r.Context(), email)
	hash, found := dummyHash, false
	switch {
	case err == nil && u.PasswordHash.Valid:
		hash, found = u.PasswordHash.String, true
	case err == nil, errors.Is(err, sql.ErrNoRows):
		// cuenta sin contraseña (usuario del seed) o email inexistente
	default:
		log.Printf("GetUsuarioPorEmail: %v", err)
		http.Error(w, "Ocurrió un error inesperado. Intentá de nuevo más tarde.", http.StatusInternalServerError)
		return
	}

	// Se verifica siempre, exista o no la cuenta, y el mensaje de error es el mismo
	// en ambos casos: así no se revela qué emails están registrados.
	ok := verifyPassword(hash, password)
	if !found || !ok {
		render(w, r, http.StatusUnauthorized, views.LoginPage(email, next, credencialesInvalidas))
		return
	}

	if err := s.startSession(w, r, u.Idusuario); err != nil {
		log.Printf("startSession: %v", err)
		http.Error(w, "Ocurrió un error inesperado. Intentá de nuevo más tarde.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) registroPageHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := views.UsuarioDe(r.Context()); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, views.RegistroPage(views.RegistroForm{}, ""))
}

func (s *Server) registroHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulario inválido", http.StatusBadRequest)
		return
	}
	f := views.RegistroForm{
		Nombre:   strings.TrimSpace(r.FormValue("nombre")),
		Apellido: strings.TrimSpace(r.FormValue("apellido")),
		Email:    normalizeEmail(r.FormValue("email")),
	}
	pass, pass2 := r.FormValue("password"), r.FormValue("password2")

	if err := validarRegistro(f, pass, pass2); err != nil {
		render(w, r, http.StatusBadRequest, views.RegistroPage(f, err.Error()))
		return
	}

	hash, err := hashPassword(pass)
	if err != nil {
		log.Printf("hashPassword: %v", err)
		http.Error(w, "Ocurrió un error inesperado. Intentá de nuevo más tarde.", http.StatusInternalServerError)
		return
	}

	id, err := s.queries.RegistrarUsuario(r.Context(), db.RegistrarUsuarioParams{
		Nombre:       f.Nombre,
		Apellido:     f.Apellido,
		Email:        f.Email,
		PasswordHash: hash,
	})
	if err != nil {
		if isEmailTaken(err) {
			render(w, r, http.StatusConflict, views.RegistroPage(f, "Ya existe una cuenta con ese email."))
			return
		}
		status, msg := classifyDBError(err)
		render(w, r, status, views.RegistroPage(f, msg))
		return
	}

	// Tras registrarse queda logueado.
	if err := s.startSession(w, r, id); err != nil {
		log.Printf("startSession: %v", err)
		http.Error(w, "Cuenta creada, pero no se pudo iniciar sesión. Ingresá manualmente.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	s.endSession(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
