#!/usr/bin/env bash
# Pasa a todos (canal general) la versión que está publicada en un canal.
# Uso: ./build/promover.sh prueba
set -euo pipefail
cd "$(dirname "$0")/.."
CANAL="${1:?Falta el canal, ej: prueba}"
RAIZ=$(pwd)
TMP=$(mktemp -d)
git fetch -q origin publicado
git worktree add --detach "$TMP" origin/publicado >/dev/null
(
  cd "$TMP"
  [ -f "ultima-$CANAL.json" ] || { echo "No hay nada publicado en el canal $CANAL" >&2; exit 1; }
  git checkout -q --orphan publicado-nuevo
  cp "TeToca-$CANAL.exe" TeToca.exe
  python3 - "$CANAL" <<'PY'
import json, sys
c = sys.argv[1]
d = json.load(open(f"ultima-{c}.json"))
d["archivo"] = "TeToca.exe"
json.dump(d, open("ultima.json", "w"), ensure_ascii=False, indent=1)
print("Versión", d["version"], "pasada a todos")
PY
  python3 "$RAIZ/build/readme_publicado.py" > README.md
  git add -A
  git commit -q -m "Promovida desde $CANAL"
  git push -q -f origin HEAD:publicado
)
git worktree remove --force "$TMP"
git branch -D publicado-nuevo >/dev/null 2>&1 || true
