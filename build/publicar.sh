#!/usr/bin/env bash
# Publica una versión nueva de TeToca en la rama "publicado".
# Todas las PC centrales que tengan TeToca abierto van a ver el cartel
# "Hay una actualización disponible" en unas horas (o al tocar "Buscar actualizaciones").
#
# Uso: ./build/publicar.sh 1.0.1 "Qué cambió en esta versión"
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:?Falta la versión, ej: 1.0.1}"
NOTAS="${2:-}"
./build/compilar.sh "$VERSION"
SHA=$(cut -d' ' -f1 dist/TeToca.exe.sha256)
TMP=$(mktemp -d)
git worktree add --detach "$TMP" >/dev/null
(
  cd "$TMP"
  git checkout --orphan publicado-nuevo >/dev/null 2>&1
  git rm -rf . >/dev/null 2>&1 || true
  cp "$OLDPWD/dist/TeToca.exe" .
  python3 - "$VERSION" "$NOTAS" "$SHA" <<'EOF'
import json, sys
v, notas, sha = sys.argv[1:4]
json.dump({"version": v, "notas": notas, "sha256": sha, "archivo": "TeToca.exe"},
          open("ultima.json", "w"), ensure_ascii=False, indent=1)
EOF
  printf '# TeToca · versión publicada\n\nÚltima versión: **%s**\n\nDescargar: [TeToca.exe](https://github.com/imansillaaa-collab/TeToca/raw/publicado/TeToca.exe)\n\n%s\n' "$VERSION" "$NOTAS" > README.md
  git add TeToca.exe ultima.json README.md
  git commit -q -m "TeToca $VERSION"
  git push -f origin HEAD:publicado
)
git worktree remove --force "$TMP"
git branch -D publicado-nuevo >/dev/null 2>&1 || true
echo "Publicada la versión $VERSION"
