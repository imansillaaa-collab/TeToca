# TeToca

Turnero para salas de espera: la gente ve en el televisor su nombre y a dónde tiene que ir,
y los empleados llaman desde sus computadoras. Pensado para registros automotores
(lee el archivo de turnos del día del sistema de la DNRPA), pero sirve para cualquier oficina
con sala de espera.

**Descargar la última versión:** [TeToca.exe](https://github.com/imansillaaa-collab/TeToca/raw/publicado/TeToca.exe)

## Qué hace

- **Una PC central** guarda la lista del día, atiende a las demás PC y transmite al televisor.
- **Puestos de mesa de entradas y de caja**: el mismo `TeToca.exe`; encuentran solos a la central en la red.
- **Televisor**: se conecta por Chromecast directamente desde el programa (sin Chrome ni "transmitir pestaña"),
  usando el receptor gratuito DashCast (o URL Cast Receiver como alternativa). Si se corta, se reconecta solo.
  También se puede abrir `http://IP-DE-LA-CENTRAL:8765/tv` en cualquier navegador o Smart TV.
- **Recorrido del turno**: pendiente → mesa → (caja) → terminado, con ausentes que se pueden volver a llamar.
- **Carga del día**: el `.xls` (en realidad HTML) o el `.csv` del sistema de turnos. Corrige los acentos rotos del CSV.
- **Actualizaciones**: la central busca versiones nuevas en la rama `publicado` de este repo y avisa con un cartel.
  Guarda la versión anterior para volver atrás si hace falta.
- **Todo local**: los datos de las personas nunca salen de la red de la oficina.
- **Compatible con Windows 7, 8, 10 y 11** (32 y 64 bits). Usa Chrome (109 en Windows 7) o Edge para mostrar las pantallas.

## Estructura

| Archivo | Qué hay |
|---|---|
| `main.go` | Arranque, modo central / puesto, página local de primera vez |
| `state.go` | Turnos, acciones de mesa y caja, guardado en disco |
| `importer.go` | Lectura del .xls / .csv del sistema de turnos |
| `server.go` | Servidor web y API que usan las pantallas |
| `discovery.go` | Cómo los puestos encuentran a la central (UDP 8766) |
| `cast.go` | Búsqueda de Chromecast (mDNS) y protocolo Cast v2 |
| `update.go` | Actualizaciones desde GitHub |
| `windows.go` | Abrir Chrome/Edge en modo ventana, ícono en la bandeja, evitar suspensión |
| `web/` | Pantallas: `index.html` + `app.js` (mesa, caja, configuración), `tv.html`, `local.html` |

Puertos: 8765 (TCP, la central), 8766 (UDP, descubrimiento), 8767 (TCP local, solo 127.0.0.1).

## Compilar y publicar

Hace falta **Go 1.20.x** (la última versión de Go que genera programas para Windows 7) y
[`rsrc`](https://github.com/akavel/rsrc) para el ícono.

```bash
go test ./...
./build/compilar.sh 1.0.1                       # deja dist/TeToca.exe
./build/publicar.sh 1.0.1 "Qué cambió"          # lo sube a la rama publicado
```
