CREATE TABLE usuario (
	idUsuario int NOT NULL,
	nombre varchar(50) NOT NULL,
	apellido varchar(50) NOT NULL,
	CONSTRAINT usuario_pk PRIMARY KEY (idUsuario)
);

CREATE TABLE voto (
    idUsuario int NOT NULL,
    idMateria int NOT NULL,
    puntuacion_dif int NULL,
    puntuacion_rel int NULL,
    puntuacion_gusto int NULL,
    descripcion varchar(140) NULL,
    fecha_voto date NOT NULL,
    CONSTRAINT voto_pk PRIMARY KEY (idUsuario,idMateria),
    CONSTRAINT chk_al_menos_un_voto CHECK (
            puntuacion_dif IS NOT NULL OR 
            puntuacion_rel IS NOT NULL OR 
            puntuacion_gusto IS NOT NULL
        ),
    CONSTRAINT chk_rango_votos CHECK (
        (puntuacion_dif BETWEEN 1 AND 5 OR puntuacion_dif IS NULL) AND
        (puntuacion_rel BETWEEN 1 AND 5 OR puntuacion_rel IS NULL) AND
        (puntuacion_gusto BETWEEN 1 AND 5 OR puntuacion_gusto IS NULL)
    )
);

CREATE TABLE materia (
	idMateria int NOT NULL, 
	nombre varchar(50) NOT NULL,
	cuatrimestre int NOT NULL,
	descripcion varchar(50) NULL,
	CONSTRAINT materia_pk PRIMARY KEY (idMateria)
);

ALTER TABLE voto
ADD CONSTRAINT fk_usuario FOREIGN KEY (idUsuario) REFERENCES usuario (idUsuario)
ON UPDATE CASCADE
ON DELETE CASCADE;

ALTER TABLE voto
ADD CONSTRAINT fk_materia FOREIGN KEY (idMateria) REFERENCES materia(idMateria)
ON UPDATE CASCADE
ON DELETE CASCADE;

