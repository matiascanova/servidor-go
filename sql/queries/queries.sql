-- name: InsertarUsuario :one
INSERT INTO usuario (idUsuario, nombre, apellido)
VALUES ($1, $2, $3)
RETURNING idUsuario, nombre, apellido;

-- name: GetUsuario :one
SELECT idUsuario, nombre, apellido
FROM usuario
WHERE idUsuario = $1;

-- name: ListarUsuarios :many
SELECT idUsuario, nombre, apellido
FROM usuario
ORDER BY apellido, nombre;

-- name: ActualizarUsuario :exec
UPDATE usuario
SET nombre = $2, 
    apellido = $3
WHERE idUsuario = $1;

-- name: DeleteUsuario :exec
DELETE FROM usuario 
WHERE idUsuario = $1;

-- name: InsertarVoto :exec
INSERT INTO voto (idUsuario, idMateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion,fecha_voto)
VALUES ($1, $2, $3, $4, $5, $6,$7);

-- name: GetVoto :one
SELECT idUsuario, idMateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion,fecha_voto
FROM voto
WHERE idUsuario = $1 AND idMateria = $2;

-- name: ListarVotosPorMateria :many
SELECT idUsuario, idMateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion,fecha_voto
FROM voto
WHERE idMateria = $1;

-- name: ListarVotosPorUsuario :many
SELECT idUsuario, idMateria, puntuacion_dif, puntuacion_rel, puntuacion_gusto, descripcion,fecha_voto
FROM voto
WHERE idUsuario = $1;

-- name: ActualizarVoto :exec
UPDATE voto
SET puntuacion_dif = $3, 
    puntuacion_rel = $4, 
    puntuacion_gusto = $5, 
    descripcion = $6,
    fecha_voto= $7
WHERE idUsuario = $1 AND idMateria = $2;

-- name: DeleteVoto :exec
DELETE FROM voto
WHERE idUsuario = $1 AND idMateria = $2;


-- name: InsertarMateria :one
INSERT INTO materia (idMateria, nombre, cuatrimestre, descripcion)
VALUES ($1, $2, $3, $4)
RETURNING idMateria, nombre, cuatrimestre, descripcion;

-- name: GetMateria :one
SELECT idMateria, nombre, cuatrimestre, descripcion
FROM materia
WHERE idMateria = $1;

-- name: ListarMaterias :many
SELECT idMateria, nombre, cuatrimestre, descripcion
FROM materia
ORDER BY cuatrimestre, nombre;

-- name: ActualizarMateria :exec
UPDATE materia
SET nombre = $2, cuatrimestre = $3, descripcion = $4
WHERE idMateria = $1;

-- name: DeleteMateria :exec
DELETE FROM materia 
WHERE idMateria = $1;

