# TeToca — notas para Claude

Turnero para la sala de espera de un registro automotor (Azul, provincia de Buenos Aires).
El TV de la sala muestra nombre, precarga y "Diríjase a MESA DE ENTRADAS / CAJA".
Un solo `TeToca.exe` para todas las PC (central, mesas, caja); se actualiza solo desde GitHub.
El usuario es diseñador gráfico, no programador: hablarle en español argentino (voseo),
claro y sin jerga; él prueba en la oficina y cuenta qué pasó (a veces con fotos).

## Reglas que no se rompen

- **Nunca** subir datos reales del registro (nombres, DNI, dominios, archivos del día) al repo,
  a capturas ni a tests. Solo nombres inventados (ver `build/demo/`).
- **Windows 7 tiene que seguir andando**: compilar siempre con **Go 1.20.x**
  (`GOOS=windows GOARCH=386`). Nada de APIs de Go posteriores a 1.20.
- Las pantallas corren en Chrome 109 (Windows 7) y en el Chromecast (navegador viejo y lento):
  JS/CSS conservador, prefijos `-webkit-` en flex y animaciones, sin dependencias externas.
  Todo lo que usa la web va embebido en el exe (`//go:embed web`), incluidas las letras (`web/fonts/`).
- No usar PowerShell ni copias temporales que se borran solas desde el exe: Defender lo marcó
  como troyano (falso positivo Wacatac) en la 1.0.4. El acceso directo se crea por COM.
- El logo de la DNRPA no va incluido en el programa (cada oficina sube el suyo).
- Si el usuario dice "aún no hagas nada" / "busquemos ideas", proponer y esperar antes de tocar código.
- Siempre correr `go test ./...` antes de publicar.

## Estructura

| Archivo | Qué hay |
|---|---|
| `main.go` | Arranque, central / puesto, flags (`--sinventana`, `--permiso-red`, `--reinicio`…) |
| `state.go` | Config, turnos, acciones de mesa y caja, guardado en `datos/` |
| `registros.go` | Dos registros en la misma mesa: un archivo por registro, se juntan si coinciden nombre **y** horario; la lista se borra sola al otro día |
| `importer.go` | Lee el `.xls` (es HTML) / `.csv` del sistema de turnos |
| `server.go` | API y archivos estáticos |
| `cast.go` | Chromecast: búsqueda mDNS, protocolo Cast v2, receptor DashCast, volumen del Chromecast |
| `update.go` | Actualizaciones y canales (general / prueba / otros por oficina) |
| `instalar*.go` | Instalación en `C:\TeToca`, firewall, arranque automático, acceso directo |
| `web/tv.html` | Pantalla del TV (7 temas con variables CSS en `body.t-*`) |
| `web/index.html`, `web/app.js`, `web/app.css` | Pantallas de mesa, caja y configuración |
| `web/comun.js` | Sonidos de llamado, conexión en tiempo real (SSE) |
| `build/sonidos.py` | Genera `web/sonidos/*.mp3` (numpy + ffmpeg) |

## Probar y ver las pantallas

```bash
go test ./...
./build/demo.sh          # central de prueba con turnos inventados y llamados hechos
#   TV:   http://127.0.0.1:8765/tv
#   Mesa: http://127.0.0.1:8765/?pc=MESA-1
./build/demo.sh parar
```

Si TeToca de verdad está abierto en esta PC, ocupa el puerto 8765: cerrarlo antes de la demo.
Al cambiar el diseño del TV, revisar **todos los temas** (noche, celeste, albiceleste, sol, claro,
contraste, verde) a **1920×1080 y 1280×720** (el Chromecast suele mostrar a 720p), con nombres
cortos, nombres muy largos, llamado a caja y sala sin llamados.

## Diseño del TV (1.0.18)

- Nombre en Barlow Condensed, mayúsculas, apellido en un renglón y nombres en otro; se achica solo hasta entrar (`ajustar()`).
- El **dominio** del auto va en una **patente** como la chapa Mercosur (blanca, franja azul de borde a borde con "DOMINIO" chiquito a la izquierda, "REPÚBLICA ARGENTINA" al centro y la bandera a la derecha; 1.0.20), con espacios como en la chapa ("AB 123 CD"). La precarga NO va en el llamado actual: se ve en "Últimos llamados" y "No se presentaron" (pedido del usuario, 1.0.18). La patente es el sello visual, no repetir el recurso en otro lado.
- "Diríjase a …" en una franja de punta a punta: celeste = mesa, amarilla = caja (`--chipM` / `--chipC`).
- El punto verde de "Llamando ahora" tiene una onda que sale siempre hacia afuera y se desvanece; al usuario no le gusta que vuelva hacia el centro (1.0.19).
- Títulos en minúscula normal (nada de MAYÚSCULAS espaciadas). Al usuario le gusta lo celeste y blanco / argentino.
- Las pantallas de PC (mesa, caja, configuración) todavía tienen el diseño anterior. En "Atendiendo ahora" / "Cobrando ahora" el **nombre va más grande que la precarga** (pedido del usuario, 1.0.21); en "Llamar siguiente" también va primero el nombre.
- Mesa (1.0.22): al tocar a alguien de la lista se abre un cartel con Llamar / Volver / "Terminar sin llamar" (con confirmación; no pasa por el TV ni por caja: el usuario dice que en ese caso no paga). Desde Terminados se lo vuelve a pendiente. Acción `terminar_sin_llamar` en `state.go`.

## Compilar y publicar

Necesita Go 1.20.x, `goversioninfo` (github.com/josephspurrier/goversioninfo), git, bash y python3.

```bash
./build/compilar.sh 1.0.18                       # deja dist/TeToca.exe
./build/publicar.sh 1.0.18 "Qué cambió"          # para todos (rama publicado)
./build/publicar.sh 1.0.18 "Qué cambió" prueba   # solo oficinas en el canal prueba
./build/promover.sh prueba                       # pasa a todos lo del canal prueba
```

- Las notas de versión las lee el usuario en el cartel de "Hay una actualización": escribirlas para él.
- GitHub raw tarda ~5 minutos en mostrar la versión nueva.
- Se le sugirió poner su oficina en el canal **prueba** (no está confirmado). Última versión publicada: 1.0.22 (canal general).
- Mensajes de commit: `1.0.N: qué cambió` en español.

## PDFs (venta y guía para el equipo)

No están en este repo (es público). El usuario tiene la carpeta `TeToca-materiales` (zip) con
`venta_registros.py`, `guia_equipo.py`, `base.py`, letras y capturas con datos inventados; ver su `LEEME.md`.
Se generan con reportlab. Las capturas se sacan de `./build/demo.sh` (en el PDF de venta: seccional
"Tu Registro Seccional", registros "Reg. 1"/"Reg. 2"; nunca el nombre del registro del usuario ni la DNRPA).
En el PDF de venta, todo lo de cargar dos archivos tiene que aclarar que es **solo si en la oficina funcionan dos registros con la misma mesa de entradas**.
La guía para el equipo todavía tiene capturas del TV viejo (antes de la 1.0.17).

## Pendiente / a confirmar en la oficina

- Sonido del TV: en la 1.0.16 TeToca sube el volumen del Chromecast al 100% y los sonidos se rehicieron
  limpios. Falta que el usuario confirme que se escucha bien ("Probar sonido en el TV" en Configuración).
- Ideas ofrecidas y no pedidas todavía: sistema de licencias antes de vender a otros registros,
  rediseño de las pantallas de PC para que combinen con el TV.
