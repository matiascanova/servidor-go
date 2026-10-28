* **Servidor Web (SSR):** Generación de componentes HTML fuertemente tipados utilizando **Templ**.
* **API REST:** Endpoints JSON para la gestión CRUD completa de usuarios, materias y reseñas.
* **Acceso a Datos:** Consultas SQL parametrizadas compiladas a código Go nativo con **sqlc**.
* **Base de Datos:** PostgreSQL 17 orquestado con **Docker Compose**.
* **Patrón de Navegación Formulario:** Implementación del patrón **Post/Redirect/Get (PRG)** en `POST /votos` para prevenir reenvíos duplicados de formularios al actualizar la página.
* **Soporte de Edición (UPSERT):** Inserción/actualización automática de votos mediante `ON CONFLICT (idusuario, idmateria) DO UPDATE`.

---

##  Guía de Ejecución

### Opción 1: Ejecución Completa con Docker Compose (Recomendado)

1. **Levantar la aplicación y la base de datos:**
   ```bash
   docker compose up -d --build
   ```

2. **Acceder a la aplicación web:**
   Abre tu navegador e ingresa a:
   ```text
   http://localhost:8080/
   ```

3. **Detener y limpiar los contenedores:**
   ```bash
   docker compose down -v
   ```

---

##  Pruebas Automatizadas 

El repositorio incluye herramientas de testeo automatizado para garantizar el correcto funcionamiento del sistema:

### 1. Pruebas de Integración HTTP (`requests.sh`)

Este script valida de punta a punta la API REST y el comportamiento del servidor mediante peticiones HTTP reales enviadas con `curl`.

#### ¿Qué pasos ejecuta el script?
1. **Verificación de entorno:** Comprueba que el demonio de Docker esté activo.
2. **Limpieza previa:** Ejecuta `docker compose down -v` para garantizar un entorno completamente limpio y libre de datos residuales.
3. **Despliegue de infraestructura:** Levanta el contenedor de PostgreSQL y la API Go en segundo plano (`docker compose up -d --build`).
4. **Espera activa de base de datos:** Utiliza `pg_isready` en bucle hasta confirmar que PostgreSQL acepta conexiones.
5. **Espera activa de la API REST:** Realiza peticiones de sondeo a `http://localhost:8080/usuarios` hasta verificar que el servidor HTTP respondió.
6. **Ejecución de la Suite de 15 Pruebas HTTP:**
   * **Sección Usuarios (`/usuarios`):** Válida creación (`POST 201`), listado (`GET 200`), consulta por ID (`GET 200`) y actualización (`PUT 200`).
   * **Sección Materias (`/materias`):** Valida creación (`POST 201`), listado (`GET 200`), consulta por ID (`GET 200`) y modificación (`PUT 200`).
   * **Sección Votos/Reseñas (`/votos`):** Prueba la inserción de reseñas asociadas (`POST 201`), consulta de reseñas por clave compuesta (`GET 200`), actualización (`PUT 200`), borrado (`DELETE 204`) y verificación de borrado (`GET 404`).
   * **Limpieza de entidades:** Elimina los datos de prueba (`DELETE 204`).
7. **Limpieza automática al finalizar:** Ejecuta un *trap* que apaga los contenedores y elimina volúmenes al concluir el test.

Para ejecutar la suite de integración HTTP:
```bash
chmod +x requests.sh
./requests.sh
```

---

### 2. Pruebas Unitarias de Base de Datos (`test.sh` / `db_test.go`)

Valida la capa de datos aislada utilizando la suite idiomática *Table-Driven Tests* de Go contra la base de datos de pruebas:

```bash
chmod +x test.sh
./test.sh
```

* Valida la creación, lectura, actualización y borrado (`CRUD`) sobre `usuario`, `materia` y `voto`.
* Comprueba las restricciones de integridad referencial (claves foráneas) y dominios de puntuación (1 a 5).
