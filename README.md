>-Se asume que el usuario ya tiene isntalado
-Docker y Docker Compose
-Go 
-sqlc

>-Pasos

1- Una vez clonado el repositorio se le debe dar permiso de ejecucion al archivo test.sh con el comando:
->chmod +x test.sh

2-ejecutar el comando de testeo desde la consola con el comando:
->./test.sh
que hace test.sh:
a- borrar cualquier contenedor que haya quedado en docker con docker compose down -v
b- ejecutar sqlc generate para generar el paquete db
c- levantar el servicio de Postgre con docker compose up -d
d- ejecutar el archivo de testeo de go
e- eliminar los contenedores creados por la prueba con docker compose down -v

3-En la consola se va a poder observar los resultados del testeo. Para tener mas informacion sobre que se testea ir a la documentacion 

nota extra: Docker genero un error al tratar de ejecutarlo en su version 18 pero en otra computadora pudo ejecutar sin problemas con la version 18 por ende es probable que el problema se origine del hardware, por eso la version se retrocedio a la version 17.Los archivos go.mod y go.sum son necesarios para la ejecucion del programa, no borrar.
Nota 2: La documentación se subió en ese formato para mantener la imagen.
