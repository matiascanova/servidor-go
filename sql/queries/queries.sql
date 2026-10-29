-- queries.sql - Consultas SQL parametrizadas para sqlc (Usuarios, Materias y Votos con UPSERT)

-- =============================================================================
-- USUARIOS
-- =============================================================================

-- name: ListarUsuarios :many
SELECT idusuario, nombre, apellido FROM usuario ORDER BY idusuario;

-- name: GetUsuario :one
SELECT idusuario, nombre, apellido FROM usuario WHERE idusuario = $1;

-- name: InsertarUsuario :one
INSERT INTO usuario (idusuario, nombre, apellido)
VALUES ($1, $2, $3)
RETURNING idusuario, nombre, apellido;

-- name: ActualizarUsuario :exec
UPDATE usuario
SET nombre = $2, apellido = $3
WHERE idusuario = $1;

-- name: DeleteUsuario :exec
DELETE FROM usuario WHERE idusuario = $1;


-- =============================================================================
-- MATERIAS
-- =============================================================================

-- name: ListarMaterias :many
SELECT idmateria, nombre, cuatrimestre, descripcion FROM materia ORDER BY idmateria;

-- name: GetMateria :one
SELECT idmateria, nombre, cuatrimestre, descripcion FROM materia WHERE idmateria = $1;

-- name: InsertarMateria :one
INSERT INTO materia (idmateria, nombre, cuatrimestre, descripcion)
VALUES ($1, $2, $3, $4)
RETURNING idmateria, nombre, cuatrimestre, descripcion;

-- name: DeleteMateria :exec
DELETE FROM materia WHERE idmateria = $1;


-- =============================================================================
-- VOTOS / RESEÑAS
-- =============================================================================

-- name: ListarVotosPorMateria :many
SELECT idusuario, idmateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion, fecha_voto
FROM voto
WHERE idmateria = $1
ORDER BY fecha_voto DESC;

-- name: GetVoto :one
SELECT idusuario, idmateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion, fecha_voto
FROM voto
WHERE idusuario = $1 AND idmateria = $2;

-- name: InsertarOActualizarVoto :one
INSERT INTO voto (idusuario, idmateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (idusuario, idmateria) 
DO UPDATE SET
    puntuacion_dif = EXCLUDED.puntuacion_dif,
    puntuacion_rel = EXCLUDED.puntuacion_rel,
    puntuacion_gusto = EXCLUDED.puntuacion_gusto,
    descripcion = EXCLUDED.descripcion,
    fecha_voto = CURRENT_TIMESTAMP
RETURNING idusuario, idmateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion, fecha_voto;

-- name: DeleteVoto :exec
DELETE FROM voto WHERE idusuario = $1 AND idmateria = $2;