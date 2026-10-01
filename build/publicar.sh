#!/usr/bin/env bash
# Publica una versión nueva de TeToca en la rama "publicado".
#
# Uso:
#   ./build/publicar.sh 1.0.13 "Qué cambió"            -> para todos (canal general)
#   ./build/publicar.sh 1.0.13 "Qué cambió" prueba     -> solo para el canal "prueba"
#   ./build/publicar.sh 1.0.13 "Qué cambió" tandil     -> solo para el canal "tandil"
#
# Cada oficina elige su canal en Configuración → Actualizaciones. Una oficina en un
# canal recibe la versión de su canal o la general, la que sea más nueva.
# Para pasarle a todos lo que ya anduvo bien en un canal: ./build/promover.sh prueba
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?Falta la versión, ej: 1.0.13}"
NOTAS="${2:-}"
CANAL="${3:-general}"
if ! [[ "$CANAL" =~ ^[a-z0-9-]{1,30}$ ]]; then echo "Canal inválido: $CANAL" >&2; exit 1; fi
if [ "$CANAL" = "general" ]; then EXE="TeToca.exe"; JSON="ultima.json"; else EXE="TeToca-$CANAL.exe"; JSON="ultima-$CANAL.json"; fi
./build/compilar.sh "$VERSION"
SHA=$(cut -d' ' -f1 dist/TeToca.exe.sha256)
RAIZ=$(pwd)
TMP=$(mktemp -d)
git fetch -q origin publicado
git worktree add --detach "$TMP" origin/publicado >/dev/null
(
  cd "$TMP"
  # rama nueva sin historial, pero con los archivos que ya estaban (otros canales)
  git checkout -q --orphan publicado-nuevo
  cp "$RAIZ/dist/TeToca.exe" "$EXE"
  python3 - "$VERSION" "$NOTAS" "$SHA" "$EXE" "$JSON" <<'PY'
import json, sys
v, notas, sha, exe, js = sys.argv[1:6]
json.dump({"version": v, "notas": notas, "sha256": sha, "archivo": exe}, open(js, "w"), ensure_ascii=False, indent=1)
PY
  python3 "$RAIZ/build/readme_publicado.py" > README.md
  git add -A
  git commit -q -m "TeToca $VERSION ($CANAL)"
  git push -q -f origin HEAD:publicado
)
git worktree remove --force "$TMP"
git branch -D publicado-nuevo >/dev/null 2>&1 || true
echo "Publicada la versión $VERSION en el canal $CANAL"
