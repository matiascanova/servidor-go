package views

import (
	"database/sql"
	"fmt"
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

func formTitle(m db.Materia, v *db.Voto) string {
	if v != nil {
		return "Editar reseña de " + m.Nombre
	}
	return "Calificar " + m.Nombre
}

func scoreLabel(n sql.NullInt32) string {
	if !n.Valid {
		return "—"
	}
	return fmt.Sprintf("%d/5", n.Int32)
}
