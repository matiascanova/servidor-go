# Foro de Alumnos – Exactas UNICEN

**cómo ejecutar la aplicación y sus funciones**
## 1. Requisitos previos

### Obligatorios

**Docker** /
**Make** /
**curl**

>  **Docker tiene que estar iniciado antes de ejecutar cualquier comando.**

## 2. Ejecutar la aplicación

1. **Iniciár Docker** y confirmá con `docker info`.

2. **ir a carpeta de proyecto** :

3. **Levantar la base de datos y la API:**

   ```bash
   make up
   ```
4. **Abrí el navegador** en <http://localhost:8080>.

5. **Detener todo (borra datos cargados):**

   ```bash
   make down
   ```

---

## 3. Ejecutar las pruebas

```bash
make test-full
```


