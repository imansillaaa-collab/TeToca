# Arma el README de la rama "publicado" a partir de los ultima*.json que haya.
import glob, json
print("# TeToca · versiones publicadas\n")
g = json.load(open("ultima.json")) if glob.glob("ultima.json") else None
if g:
    print(f"**Para todos:** versión {g['version']} — [descargar TeToca.exe](https://github.com/imansillaaa-collab/TeToca/raw/publicado/TeToca.exe)\n")
    if g.get("notas"):
        print(g["notas"] + "\n")
otros = sorted(f for f in glob.glob("ultima-*.json"))
if otros:
    print("## Canales\n")
    for f in otros:
        d = json.load(open(f))
        print(f"- **{f[7:-5]}**: versión {d['version']} ({d['archivo']})")
