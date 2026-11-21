package views

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	"tp2_entrega/db"
)

// HELPERS
func nullIntStr(n sql.NullInt32) string {
	if !n.Valid {
		return ""
	}
	return strconv.Itoa(int(n.Int32))
}

func scoreDif(v *db.Voto) string {
	if v == nil {
		return ""
	}
	return nullIntStr(v.PuntuacionDif)
}

func scoreRel(v *db.Voto) string {
	if v == nil {
		return ""
	}
	return nullIntStr(v.PuntuacionRel)
}

func scoreGusto(v *db.Voto) string {
	if v == nil {
		return ""
	}
	return nullIntStr(v.PuntuacionGusto)
}

func comentarioVoto(v *db.Voto) string {
	if v == nil || !v.Descripcion.Valid {
		return ""
	}
	return v.Descripcion.String
}

func scoreLabel(n sql.NullInt32) string {
	if !n.Valid {
		return "—"
	}
	return fmt.Sprintf("%d/5", n.Int32)
}

// loginURL arma el enlace al login para volver a la página de la materia después de
// ingresar (el invitado lo ve en lugar del formulario).
func loginURL(idMateria int32) string {
	return "/login?next=" + url.QueryEscape(fmt.Sprintf("/materias/resenas?id=%d", idMateria))
}

// SESIÓN
// El middleware de main (withUser) guarda el usuario logueado en el contexto de la
// petición. Los templates lo leen desde ahí (la variable ctx de templ), así que
// ninguna vista necesita recibirlo como parámetro.

type Usuario struct {
	ID       int32
	Nombre   string
	Apellido string
}

type ctxKey struct{}

func WithUsuario(ctx context.Context, u Usuario) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func UsuarioDe(ctx context.Context) (Usuario, bool) {
	u, ok := ctx.Value(ctxKey{}).(Usuario)
	return u, ok
}

func loggedIn(ctx context.Context) bool {
	_, ok := UsuarioDe(ctx)
	return ok
}

func nombreUsuario(ctx context.Context) string {
	u, _ := UsuarioDe(ctx)
	return u.Nombre
}

// RegistroForm conserva lo que el usuario escribió si el registro falla
// (la contraseña nunca se vuelve a mostrar).
type RegistroForm struct {
	Nombre   string
	Apellido string
	Email    string
}
