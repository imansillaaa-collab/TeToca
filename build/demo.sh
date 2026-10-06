#!/usr/bin/env bash
# Levanta una central de prueba con turnos inventados y algunos llamados hechos,
# para ver y retocar las pantallas sin tocar la oficina.
#   ./build/demo.sh          → http://127.0.0.1:8765/tv  (TV)  y  /?pc=MESA-1  (mesa)
#   ./build/demo.sh parar
# Los nombres de build/demo/*.xls son inventados. Nunca usar archivos reales del registro.
set -euo pipefail
cd "$(dirname "$0")/.."
D=.demo
B=http://127.0.0.1:8765
EXE=tetoca; case "$(go env GOOS)" in windows) EXE=tetoca.exe;; esac
parar() { [ -f $D/pid ] && kill "$(cat $D/pid)" 2>/dev/null || true; rm -f $D/pid; }
parar
[ "${1:-}" = "parar" ] && { echo "Demo detenida."; exit 0; }
rm -rf $D/datos; mkdir -p $D/datos
# versión alta para que la demo no muestre el cartel de "Hay una actualización"
go build -ldflags "-X main.Version=99.0.0" -o $D/$EXE .
echo '{"modo":"central","pc":"CENTRAL"}' > $D/datos/esta-pc.json
echo '{"oficina":"Registro Automotor","seccional":"Azul 1 y 2","registros":["Azul 1","Azul 2"],"temaTV":"noche","sonido":"dingdong","pcs":{}}' > $D/datos/config.json
( cd $D && { ./$EXE --sinventana > salida.txt 2>&1 & echo $! > pid; } )
for i in $(seq 1 40); do curl -s -o /dev/null $B/api/estado && break; sleep 0.25; done
# los archivos de ejemplo se cargan con la fecha de hoy, para que no salga el aviso de "turnos viejos"
HOY=$(date +%d/%m/%Y)
for n in 1 2; do
  sed -E "s#[0-9]{2}/[0-9]{2}/20[0-9]{2}#$HOY#g" build/demo/registro$n.xls > $D/registro$n.xls
  curl -s -o /dev/null -F archivo=@$D/registro$n.xls -F registro=$n $B/api/cargar
done
p() { curl -s -o /dev/null -H 'Content-Type: application/json' -d "{\"pc\":\"$1\",\"rol\":\"$2\"}" $B/api/pc; }
p MESA-1 mesa; p CAJA caja
a() { curl -s -o /dev/null -H 'Content-Type: application/json' -d "{\"accion\":\"$1\",\"pc\":\"$2\"}" $B/api/accion; }
for i in 1 2 3 4; do a mesa_siguiente MESA-1; a pasar_caja MESA-1; done
a caja_siguiente CAJA; a mesa_siguiente MESA-1; a ausente MESA-1; a mesa_siguiente MESA-1
echo "Demo lista:"
echo "  TV:   $B/tv"
echo "  Mesa: $B/?pc=MESA-1"
echo "  Caja: $B/?pc=CAJA"
echo "Para cambiar el tema del TV: curl -H 'Content-Type: application/json' -d '{\"temaTV\":\"celeste\"}' $B/api/config"
echo "Para detenerla: ./build/demo.sh parar"
