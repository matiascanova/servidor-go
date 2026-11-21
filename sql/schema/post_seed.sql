-- Se ejecuta después de seed.sql (ver docker-compose.yml).
-- El seed inserta usuarios con id explícito, y eso no avanza la secuencia de la
-- columna identity. Sin este ajuste, el primer registro desde la app chocaría con
-- el id 1 del usuario demo. Acá se deja la secuencia en (máximo id + 1).
SELECT setval(
	pg_get_serial_sequence('usuario', 'idusuario'),
	(SELECT COALESCE(MAX(idusuario), 0) + 1 FROM usuario),
	false
);
