package db_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"
    _ "github.com/jackc/pgx/v5/stdlib"
	db "tp2_entrega/db"
)

var testQueries *db.Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/apirest?sslmode=disable"
	}

	var err error
	testDB, err = sql.Open("pgx", connStr)
	if err != nil {
		panic("Error al abrir conexión: " + err.Error())
	}

	if err = testDB.Ping(); err != nil {
		panic("Error al conectar a la BD: " + err.Error())
	}

	testQueries = db.New(testDB)

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

//TESTEO CRUD USUARIOS

func TestUsuariosTableDriven(t *testing.T) {
	ctx := context.Background()
	//DEFINICION CASOS
	casos := []struct {
		nombre string
		params db.InsertarUsuarioParams
	}{
		{
			nombre: "usuario N1",
			params: db.InsertarUsuarioParams{
				Idusuario: 1,
				Nombre:    "Juan",
				Apellido:  "Pérez",
			},
		},
		{
			nombre: "usuario N2",
			params: db.InsertarUsuarioParams{
				Idusuario: 2,
				Nombre:    "María",
				Apellido:  "Gómez",
			},
		},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			//CREATE
			usuario, err := testQueries.InsertarUsuario(ctx, tc.params)
			if err != nil {
				t.Fatalf("Error al insertar usuario: %v", err)
			}

			if usuario.Nombre != tc.params.Nombre {
				t.Errorf("Nombre obtenido = %s; se esperaba %s", usuario.Nombre, tc.params.Nombre)
			}
			//READ
			obtenido, err := testQueries.GetUsuario(ctx, tc.params.Idusuario)
			if err != nil {
				t.Fatalf("Error al recuperar usuario: %v", err)
			}
			if obtenido.Apellido != tc.params.Apellido {
				t.Errorf("Apellido obtenido = %s; se esperaba %s", obtenido.Apellido, tc.params.Apellido)
			}
		})
	}
	//UPDATE: DEFINICION CASOS
	casosActualizacion := []struct {
		nombre           string
		paramsUpdate     db.ActualizarUsuarioParams
		nombreEsperado   string
		apellidoEsperado string
	}{
		{
			nombre: "cambiar nombre",
			paramsUpdate: db.ActualizarUsuarioParams{
				Idusuario: 1,
				Nombre:    "Carlos Alberto",
				Apellido:  "López",
			},
			nombreEsperado:   "Carlos Alberto",
			apellidoEsperado: "López",
		},
		{
			nombre: "cambiar apellido",
			paramsUpdate: db.ActualizarUsuarioParams{
				Idusuario: 1,
				Nombre:    "Carlos Alberto",
				Apellido:  "Gómez",
			},
			nombreEsperado:   "Carlos Alberto",
			apellidoEsperado: "Gómez",
		},
	}
	//EJECUCION LOOP ACTUALIZACION
	for _, tc := range casosActualizacion {
		t.Run(tc.nombre, func(t *testing.T) {
			err := testQueries.ActualizarUsuario(ctx, tc.paramsUpdate)
			if err != nil {
				t.Fatalf("Error al actualizar usuario: %v", err)
			}
			actualizado, err := testQueries.GetUsuario(ctx, tc.paramsUpdate.Idusuario)
			if err != nil {
				t.Fatalf("Error al obtener usuario actualizado: %v", err)
			}
			if actualizado.Nombre != tc.nombreEsperado {
				t.Errorf("Nombre obtenido = %s; se esperaba %s", actualizado.Nombre, tc.nombreEsperado)
			}
			if actualizado.Apellido != tc.apellidoEsperado {
				t.Errorf("Apellido obtenido = %s; se esperaba %s", actualizado.Apellido, tc.apellidoEsperado)
			}
		})
	}
	//DELETE
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			_ = testQueries.DeleteUsuario(ctx, tc.params.Idusuario)
		})
	}

}

// TESTEO CRUD MATERIAS
func TestMateriasTableDriven(t *testing.T) {
	ctx := context.Background()

	casosMat := []struct {
		nombre string
		params db.InsertarMateriaParams
	}{
		{
			nombre: "materia cuatrimestre 1",
			params: db.InsertarMateriaParams{
				Idmateria:    101,
				Nombre:       "Programación Web",
				Cuatrimestre: 1,
				Descripcion:  sql.NullString{String: "Materia de Go y Docker", Valid: true},
			},
		},
	}

	for _, tc := range casosMat {
		t.Run(tc.nombre, func(t *testing.T) {
			// INSERT
			materia, err := testQueries.InsertarMateria(ctx, tc.params)
			if err != nil {
				t.Fatalf("Error al insertar materia: %v", err)
			}

			// READ
			obtenida, err := testQueries.GetMateria(ctx, materia.Idmateria)
			if err != nil {
				t.Fatalf("Error al recuperar materia: %v", err)
			}
			if obtenida.Nombre != tc.params.Nombre {
				t.Errorf("Materia obtenida = %s; se esperaba %s", obtenida.Nombre, tc.params.Nombre)
			}

			// DELETE
			_ = testQueries.DeleteMateria(ctx, materia.Idmateria)
		})
	}
}

// TESTEO CRUD VOTOS
func TestVotosTableDriven(t *testing.T) {
	ctx := context.Background()

	u, err := testQueries.InsertarUsuario(ctx, db.InsertarUsuarioParams{
		Idusuario: 99,
		Nombre:    "Votante",
		Apellido:  "Prueba",
	})
	if err != nil {
		t.Fatalf("Error al preparar usuario para votos: %v", err)
	}
	defer testQueries.DeleteUsuario(ctx, u.Idusuario)

	m, err := testQueries.InsertarMateria(ctx, db.InsertarMateriaParams{
		Idmateria:    999,
		Nombre:       "Materia Votable",
		Cuatrimestre: 1,
		Descripcion:  sql.NullString{String: "Materia de prueba para votos", Valid: true},
	})
	if err != nil {
		t.Fatalf("Error al preparar materia para votos: %v", err)
	}
	defer testQueries.DeleteMateria(ctx, m.Idmateria)

	fechaActual := time.Now().Round(time.Microsecond)

	// CREATE : casos
	casosInsercion := []struct {
		nombre string
		params db.InsertarVotoParams
	}{
		{
			nombre: "voto completo",
			params: db.InsertarVotoParams{
				Idusuario:       u.Idusuario,
				Idmateria:       m.Idmateria,
				PuntuacionDif:   sql.NullInt32{Int32: 3, Valid: true},
				PuntuacionRel:   sql.NullInt32{Int32: 4, Valid: true},
				PuntuacionGusto: sql.NullInt32{Int32: 5, Valid: true},
				Descripcion:     sql.NullString{String: "Muy buena materia", Valid: true},
				FechaVoto:       fechaActual,
			},
		},
	}

	for _, tc := range casosInsercion {
		t.Run(tc.nombre, func(t *testing.T) {
			// CREATE
			err := testQueries.InsertarVoto(ctx, tc.params)
			if err != nil {
				t.Fatalf("Error al insertar voto: %v", err)
			}

			// READ
			voto, err := testQueries.GetVoto(ctx, db.GetVotoParams{
				Idusuario: tc.params.Idusuario,
				Idmateria: tc.params.Idmateria,
			})
			if err != nil {
				t.Fatalf("Error al recuperar voto: %v", err)
			}

			if voto.PuntuacionGusto.Int32 != tc.params.PuntuacionGusto.Int32 {
				t.Errorf("Puntuación gusto obtenida = %d; se esperaba %d", voto.PuntuacionGusto.Int32, tc.params.PuntuacionGusto.Int32)
			}
		})
	}

	// 3. UPDATE
	casosActualizacion := []struct {
		nombre        string
		paramsUpdate  db.ActualizarVotoParams
		gustoEsperado int32
		descEsperada  string
	}{
		{
			nombre: "actualizar voto",
			paramsUpdate: db.ActualizarVotoParams{
				Idusuario:       u.Idusuario,
				Idmateria:       m.Idmateria,
				PuntuacionDif:   sql.NullInt32{Int32: 2, Valid: true},
				PuntuacionRel:   sql.NullInt32{Int32: 3, Valid: true},
				PuntuacionGusto: sql.NullInt32{Int32: 4, Valid: true},
				Descripcion:     sql.NullString{String: "Opinión actualizada", Valid: true},
				FechaVoto:       fechaActual,
			},
			gustoEsperado: 4,
			descEsperada:  "Opinión actualizada",
		},
	}

	for _, tc := range casosActualizacion {
		t.Run(tc.nombre, func(t *testing.T) {
			err := testQueries.ActualizarVoto(ctx, tc.paramsUpdate)
			if err != nil {
				t.Fatalf("Error al actualizar voto: %v", err)
			}

			votoActualizado, err := testQueries.GetVoto(ctx, db.GetVotoParams{
				Idusuario: tc.paramsUpdate.Idusuario,
				Idmateria: tc.paramsUpdate.Idmateria,
			})
			if err != nil {
				t.Fatalf("Error al obtener voto actualizado: %v", err)
			}

			if votoActualizado.PuntuacionGusto.Int32 != tc.gustoEsperado {
				t.Errorf("Puntuación gusto obtenida = %d; se esperaba %d", votoActualizado.PuntuacionGusto.Int32, tc.gustoEsperado)
			}
			if votoActualizado.Descripcion.String != tc.descEsperada {
				t.Errorf("Descripción obtenida = %s; se esperaba %s", votoActualizado.Descripcion.String, tc.descEsperada)
			}
		})
	}

	// DELETE
	t.Run("eliminar voto", func(t *testing.T) {
		err := testQueries.DeleteVoto(ctx, db.DeleteVotoParams{
			Idusuario: u.Idusuario,
			Idmateria: m.Idmateria,
		})
		if err != nil {
			t.Fatalf("Error al eliminar voto: %v", err)
		}
	})
}