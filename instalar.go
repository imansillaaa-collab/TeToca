package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Instalación en la PC: copia el programa a una carpeta fija (C:\TeToca), lo deja
// arrancando solo con Windows y, en la central, le da permiso en el firewall para
// todas las redes. Lo único que pide administrador es el permiso del firewall.

type EstadoInst struct {
	Soportado bool   `json:"soportado"`
	Carpeta   string `json:"carpeta"`   // donde se instala
	Actual    string `json:"actual"`    // donde está corriendo ahora
	Instalado bool   `json:"instalado"` // corre desde la carpeta de instalación
	Inicio    bool   `json:"inicio"`    // arranca con Windows
	Red       bool   `json:"red"`       // tiene la regla del firewall
	Central   bool   `json:"central"`
	Version   string `json:"version"`
}

func mismaRuta(a, b string) bool {
	a, _ = filepath.Abs(a)
	b, _ = filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func estadoInstalacion() EstadoInst {
	localMu.Lock()
	central := local.Modo == "central"
	localMu.Unlock()
	exe := rutaExe()
	dest := carpetaInstalacion()
	e := EstadoInst{Soportado: dest != "", Carpeta: dest, Actual: filepath.Dir(exe), Central: central, Version: Version}
	if !e.Soportado {
		return e
	}
	e.Instalado = mismaRuta(filepath.Dir(exe), dest)
	e.Inicio = inicioActivo(exe)
	e.Red = reglaRed()
	return e
}

func copiarArchivo(de, a string) error {
	in, err := os.Open(de)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := a + ".copiando"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(a)
	return os.Rename(tmp, a)
}

// instalar deja TeToca en la carpeta fija. Devuelve la ruta del exe instalado y,
// aparte, si no se pudo dar el permiso de red (por ejemplo, si tocaron "No").
func instalar(copiarDatos bool) (destExe string, errRed error, err error) {
	dest := carpetaInstalacion()
	if dest == "" {
		return "", nil, UserErr{"La instalación solo está disponible en Windows."}
	}
	exe := rutaExe()
	destExe = filepath.Join(dest, "TeToca.exe")
	if !mismaRuta(filepath.Dir(exe), dest) {
		if err := os.MkdirAll(filepath.Join(dest, "datos"), 0755); err != nil {
			return "", nil, UserErr{"No pude crear la carpeta " + dest + "."}
		}
		if err := copiarArchivo(exe, destExe); err != nil {
			logf("instalar: copiar exe: %v", err)
			return "", nil, UserErr{"No pude copiar TeToca a " + dest + ". Si ya hay un TeToca abierto desde esa carpeta, cerralo y probá de nuevo."}
		}
		if copiarDatos {
			for _, f := range []string{"turnos.json", "config.json", "logo.png"} {
				if _, err := os.Stat(filepath.Join(datos, f)); err == nil {
					_ = copiarArchivo(filepath.Join(datos, f), filepath.Join(dest, "datos", f))
				}
			}
		}
	} else {
		destExe = exe
	}
	_ = os.MkdirAll(filepath.Join(dest, "datos"), 0755)
	localMu.Lock()
	l := local
	localMu.Unlock()
	if err := escribirJSON(filepath.Join(dest, "datos", "esta-pc.json"), l); err != nil {
		return "", nil, err
	}
	if err := ponerInicio(destExe); err != nil {
		logf("instalar: inicio automático: %v", err)
	}
	if err := crearAccesoDirecto(destExe); err != nil {
		logf("instalar: acceso directo: %v", err)
	}
	if l.Modo == "central" && !reglaRed() {
		if err := elevar(destExe, "--permiso-red"); err != nil {
			logf("instalar: permiso de red: %v", err)
			errRed = err
		}
	}
	logf("instalado en %s (modo %s)", destExe, l.Modo)
	return destExe, errRed, nil
}

// desinstalar saca el arranque automático, el acceso directo y el permiso de red,
// y borra la lista y la configuración. El programa en sí no se puede borrar mientras
// está abierto: la carpeta la borra la persona después (se abre sola).
func desinstalar() (string, error) {
	if carpetaInstalacion() == "" {
		return "", UserErr{"La desinstalación solo está disponible en Windows."}
	}
	exe := rutaExe()
	_ = quitarInicio()
	quitarAccesoDirecto()
	if reglaRed() {
		if err := elevar(exe, "--quitar-red"); err != nil {
			logf("desinstalar: quitar permiso de red: %v", err)
		}
	}
	for _, f := range []string{"turnos.json", "config.json", "esta-pc.json", "logo.png"} {
		_ = os.Remove(filepath.Join(datos, f))
	}
	logf("desinstalado; queda borrar la carpeta %s", filepath.Dir(exe))
	return filepath.Dir(exe), nil
}

// Protección: solo aceptamos pedidos que vengan de las páginas de TeToca en esta
// misma PC, para que ninguna página web pueda instalar o desinstalar nada.
func pedidoLocal(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	if r.Method == http.MethodGet {
		return true
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return false
	}
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	o = strings.TrimPrefix(o, "http://")
	h, _, err := net.SplitHostPort(o)
	if err != nil {
		return false
	}
	return h == "127.0.0.1" || h == "localhost"
}

// manejarInstalacion atiende /local/instalacion. reiniciarDesde se llama cuando
// hay que volver a abrir TeToca desde la carpeta nueva.
func manejarInstalacion(w http.ResponseWriter, r *http.Request, reiniciarDesde func(exe string)) {
	if !pedidoLocal(r) {
		http.Error(w, "no permitido", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet {
		jsonOK(w, estadoInstalacion())
		return
	}
	var q struct{ Accion string }
	if err := leerBody(r, &q); err != nil {
		jsonErr(w, err)
		return
	}
	switch q.Accion {
	case "instalar":
		antes := rutaExe()
		destExe, errRed, err := instalar(true)
		if err != nil {
			jsonErr(w, err)
			return
		}
		mueve := !mismaRuta(destExe, antes)
		jsonOK(w, map[string]interface{}{"ok": true, "reinicia": mueve, "red": errRed == nil, "estado": estadoInstalacion()})
		if mueve {
			go reiniciarDesde(destExe)
		}
	case "desinstalar":
		dir, err := desinstalar()
		if err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, map[string]interface{}{"ok": true, "carpeta": dir})
		go func() {
			time.Sleep(1500 * time.Millisecond)
			abrirCarpeta(filepath.Dir(dir))
			logf("cerrado por desinstalación")
			os.Exit(0)
		}()
	default:
		jsonErr(w, errors.New("acción desconocida"))
	}
}
