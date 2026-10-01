#!/usr/bin/env bash
# Compila TeToca.exe (Windows 7, 8, 10 y 11; 32 y 64 bits).
# Uso: ./build/compilar.sh 1.0.0
# Requiere Go 1.20.x (la última versión de Go que genera programas compatibles con Windows 7).
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?Pasá la versión, ej: ./build/compilar.sh 1.0.1}"
GO="${GO:-go}"
if ! "$GO" version | grep -q "go1.20"; then
  echo "Necesitás Go 1.20.x para que ande en Windows 7 (tenés: $($GO version))." >&2
  exit 1
fi
# Ícono, manifiesto y datos del programa (nombre, versión, descripción) que muestra
# Windows en Propiedades. Requiere goversioninfo (github.com/josephspurrier/goversioninfo).
GVI="${GVI:-goversioninfo}"
IFS=. read -r MA MI PA <<< "${VERSION}"
rm -f rsrc_windows_386.syso resource.syso
"$GVI" -icon web/icono.ico -manifest build/tetoca.manifest \
  -ver-major "$MA" -ver-minor "$MI" -ver-patch "${PA:-0}" -product-ver-major "$MA" -product-ver-minor "$MI" -product-ver-patch "${PA:-0}" \
  -file-version "$VERSION" -product-version "$VERSION" -o rsrc_windows_386.syso build/versioninfo.json
mkdir -p dist
GOOS=windows GOARCH=386 CGO_ENABLED=0 GOTOOLCHAIN=local "$GO" build -trimpath \
  -ldflags "-H windowsgui -s -w -X main.Version=${VERSION}" -o dist/TeToca.exe .
( cd dist && sha256sum TeToca.exe > TeToca.exe.sha256 )
echo "Listo: dist/TeToca.exe (versión ${VERSION})"
cat dist/TeToca.exe.sha256
