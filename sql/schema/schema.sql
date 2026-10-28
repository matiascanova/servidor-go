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

INSERT INTO usuario (idusuario, nombre, apellido) VALUES
  (1, 'Francisco', 'Turraspe'),
  (2, 'Matias', 'Canoa')
ON CONFLICT (idusuario) DO NOTHING;

-- 2. Insertar Materias extraídas de la imagen del plan de estudios (organizadas por cuatrimestre)
INSERT INTO materia (idmateria, nombre, cuatrimestre, descripcion) VALUES
  -- 1er Cuatrimestre
  (1, 'Introducción a la Programación I', 1, 'Materia del 1er cuatrimestre'),
  (2, 'Introducción al Álgebra', 1, 'Materia del 1er cuatrimestre'),
  (3, 'Introducción a los Sistemas Informáticos', 1, 'Materia del 1er cuatrimestre'),
  (4, 'Inglés', 1, 'Materia del 1er cuatrimestre'),

  -- 2do Cuatrimestre
  (5, 'Diseño Lógico', 2, 'Materia del 2do cuatrimestre'),
  (6, 'Introducción a la Programación II', 2, 'Materia del 2do cuatrimestre'),
  (7, 'Lenguajes Formales y Autómatas', 2, 'Materia del 2do cuatrimestre'),
  (8, 'Cálculo I', 2, 'Materia del 2do cuatrimestre'),

  -- 3er Cuatrimestre
  (9, 'Arquitectura de Computadoras I', 3, 'Materia del 3er cuatrimestre'),
  (10, 'Análisis y Diseño de Algoritmos I', 3, 'Materia del 3er cuatrimestre'),
  (11, 'Programación Orientada a Objetos', 3, 'Materia del 3er cuatrimestre'),
  (12, 'Álgebra Lineal', 3, 'Materia del 3er cuatrimestre'),

  -- 4to Cuatrimestre
  (13, 'Redes de Computadoras I', 4, 'Materia del 4to cuatrimestre'),
  (14, 'Cálculo II', 4, 'Materia del 4to cuatrimestre'),
  (15, 'Análisis y Diseño de Algoritmos II', 4, 'Materia del 4to cuatrimestre'),
  (16, 'Metodologías de Desarrollo de Software', 4, 'Materia del 4to cuatrimestre'),

  -- 5to Cuatrimestre
  (17, 'Física I', 5, 'Materia del 5to cuatrimestre'),
  (18, 'Lenguajes y Paradigmas', 5, 'Materia del 5to cuatrimestre'),
  (19, 'Sistemas Operativos', 5, 'Materia del 5to cuatrimestre'),
  (20, 'Base de Datos I', 5, 'Materia del 5to cuatrimestre'),

  -- 6to Cuatrimestre
  (21, 'Base de Datos II', 6, 'Materia del 6to cuatrimestre'),
  (22, 'Probabilidad y Estadística', 6, 'Materia del 6to cuatrimestre'),
  (23, 'Programación WEB', 6, 'Materia del 6to cuatrimestre'),
  (24, 'Ingeniería de Software I', 6, 'Materia del 6to cuatrimestre'),

  -- 7mo Cuatrimestre
  (25, 'Organización y Gestión Empresarial', 7, 'Materia del 7mo cuatrimestre'),
  (26, 'Teoría de la Información', 7, 'Materia del 7mo cuatrimestre'),
  (27, 'Redes de Computadoras II', 7, 'Materia del 7mo cuatrimestre'),
  (28, 'Física II', 7, 'Materia del 7mo cuatrimestre'),

  -- 8vo Cuatrimestre
  (29, 'Calidad de Software', 8, 'Materia del 8vo cuatrimestre'),
  (30, 'Arquitectura de Computadoras II', 8, 'Materia del 8vo cuatrimestre'),
  (31, 'Fundamentos de la Ciencia de Datos', 8, 'Materia del 8vo cuatrimestre'),
  (32, 'Compiladores e Intérpretes', 8, 'Materia del 8vo cuatrimestre'),

  -- 9no Cuatrimestre
  (33, 'Ingeniería de Software II', 9, 'Materia del 9no cuatrimestre'),
  (34, 'Ciberseguridad', 9, 'Materia del 9no cuatrimestre'),
  (35, 'Inteligencia Artificial', 9, 'Materia del 9no cuatrimestre'),

  -- 10mo Cuatrimestre
  (36, 'Formulación y Evaluación de Proyectos TICs', 10, 'Materia del 10mo cuatrimestre'),
  (37, 'Ética y Legislación de la Práctica Profesional', 10, 'Materia del 10mo cuatrimestre')
ON CONFLICT (idmateria) DO NOTHING;
