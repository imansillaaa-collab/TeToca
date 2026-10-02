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
go build -o $D/$EXE .
echo '{"modo":"central","pc":"CENTRAL"}' > $D/datos/esta-pc.json
echo '{"oficina":"Registro Automotor","seccional":"Azul 1 y 2","registros":["Azul 1","Azul 2"],"temaTV":"noche","sonido":"dingdong","pcs":{}}' > $D/datos/config.json
( cd $D && { ./$EXE --sinventana > salida.txt 2>&1 & echo $! > pid; } )
for i in $(seq 1 40); do curl -s -o /dev/null $B/api/estado && break; sleep 0.25; done
curl -s -o /dev/null -F archivo=@build/demo/registro1.xls -F registro=1 $B/api/cargar
curl -s -o /dev/null -F archivo=@build/demo/registro2.xls -F registro=2 $B/api/cargar
a() { curl -s -o /dev/null -H 'Content-Type: application/json' -d "{\"accion\":\"$1\",\"pc\":\"$2\"}" $B/api/accion; }
for i in 1 2 3 4; do a mesa_siguiente MESA-1; a pasar_caja MESA-1; done
a caja_siguiente CAJA; a mesa_siguiente MESA-1; a ausente MESA-1; a mesa_siguiente MESA-1
echo "Demo lista:"
echo "  TV:   $B/tv"
echo "  Mesa: $B/?pc=MESA-1"
echo "  Caja: $B/?pc=CAJA"
echo "Para cambiar el tema del TV: curl -H 'Content-Type: application/json' -d '{\"temaTV\":\"celeste\"}' $B/api/config"
echo "Para detenerla: ./build/demo.sh parar"
