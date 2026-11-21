-- USUARIOS

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


-- AUTENTICACIÓN

-- Alta de una cuenta. El id lo genera la base (identity). Los casts a text hacen
-- que sqlc genere parámetros string comunes y no sql.NullString.
-- name: RegistrarUsuario :one
INSERT INTO usuario (nombre, apellido, email, password_hash)
VALUES (sqlc.arg(nombre), sqlc.arg(apellido), sqlc.arg(email)::text, sqlc.arg(password_hash)::text)
RETURNING idusuario;

-- name: GetUsuarioPorEmail :one
SELECT idusuario, password_hash FROM usuario WHERE email = sqlc.arg(email)::text;

-- name: CrearSesion :exec
INSERT INTO sesion (token_hash, idusuario, expira)
VALUES ($1, $2, $3);

-- Devuelve el usuario dueño de la sesión solo si no venció.
-- name: GetUsuarioPorSesion :one
SELECT u.idusuario, u.nombre, u.apellido
FROM sesion s
JOIN usuario u ON u.idusuario = s.idusuario
WHERE s.token_hash = $1 AND s.expira > now();

-- name: BorrarSesion :exec
DELETE FROM sesion WHERE token_hash = $1;

-- name: LimpiarSesionesExpiradas :exec
DELETE FROM sesion WHERE expira <= now();


-- MATERIAS

-- name: ListarMaterias :many
SELECT idmateria, nombre, cuatrimestre, descripcion FROM materia ORDER BY idmateria;

-- name: GetMateria :one
SELECT idmateria, nombre, cuatrimestre, descripcion FROM materia WHERE idmateria = $1;

-- name: InsertarMateria :one
INSERT INTO materia (idmateria, nombre, cuatrimestre, descripcion)
VALUES ($1, $2, $3, $4)
RETURNING idmateria, nombre, cuatrimestre, descripcion;

-- name: ActualizarMateria :one
UPDATE materia
SET nombre = $1,
    cuatrimestre = $2,
    descripcion = $3
WHERE idMateria = $4
RETURNING *;

-- name: DeleteMateria :exec
DELETE FROM materia WHERE idmateria = $1;


-- VOTOS / RESEÑAS

-- name: ListarVotosPorUsuario :many
SELECT v.idusuario, v.idmateria, v.puntuacion_dif, v.puntuacion_rel, v.puntuacion_gusto,
       v.descripcion, v.fecha_voto, m.nombre AS materia_nombre
FROM voto v
JOIN materia m ON m.idmateria = v.idmateria
WHERE v.idusuario = $1
ORDER BY v.fecha_voto DESC;

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
