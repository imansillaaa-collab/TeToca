package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Version se completa al compilar (-ldflags "-X main.Version=1.0.0").
var Version = "0.0.0-dev"

const PuertoLocal = 8767

type Local struct {
	Modo        string `json:"modo"` // "central" o "puesto"
	PC          string `json:"pc"`
	CentralIP   string `json:"centralIP"`
	CentralHost string `json:"centralHost"`
}

var (
	carpeta   string // donde está el exe
	datos     string // carpeta de datos
	local     Local
	localMu   sync.Mutex
	logger    *log.Logger
	escucha   net.Listener
	escuchaMu sync.Mutex
)

func logf(f string, a ...interface{}) {
	if logger != nil {
		logger.Printf(f, a...)
	}
}

func rutaLocal() string { return filepath.Join(datos, "esta-pc.json") }

func guardarLocal() {
	localMu.Lock()
	defer localMu.Unlock()
	_ = escribirJSON(rutaLocal(), local)
}

func prepararCarpetas() {
	carpeta = filepath.Dir(rutaExe())
	datos = filepath.Join(carpeta, "datos")
	if err := os.MkdirAll(datos, 0755); err != nil || !escribible(datos) {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.TempDir()
		}
		datos = filepath.Join(base, "TeToca")
		_ = os.MkdirAll(datos, 0755)
	}
	lp := filepath.Join(datos, "tetoca.log")
	if st, err := os.Stat(lp); err == nil && st.Size() > 2<<20 {
		_ = os.Rename(lp, lp+".viejo")
	}
	f, err := os.OpenFile(lp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		logger = log.New(f, "", log.LstdFlags)
	} else {
		logger = log.New(io.Discard, "", 0)
	}
	// limpiar restos de actualizaciones
	_ = os.Remove(filepath.Join(carpeta, "TeToca.nuevo.exe"))
	_ = os.Remove(filepath.Join(carpeta, "TeToca.cambio.exe"))
}

func escribible(dir string) bool {
	p := filepath.Join(dir, ".prueba")
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		return false
	}
	_ = os.Remove(p)
	return true
}

func escuchar(addr string, reintentar bool) (net.Listener, error) {
	var l net.Listener
	var err error
	for i := 0; i < 20; i++ {
		l, err = net.Listen("tcp4", addr)
		if err == nil || !reintentar {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	return l, err
}

func liberarPuerto() {
	escuchaMu.Lock()
	defer escuchaMu.Unlock()
	if escucha != nil {
		_ = escucha.Close()
	}
}

func main() {
	reinicio := len(os.Args) > 1 && os.Args[1] == "--reinicio"
	prepararCarpetas()
	leerJSON(rutaLocal(), &local)
	if local.PC == "" {
		h, _ := os.Hostname()
		if h == "" {
			h = "PC"
		}
		local.PC = h
	}
	logf("TeToca %s iniciando (modo=%q, pc=%s)", Version, local.Modo, local.PC)
	if local.Modo == "central" {
		modoCentral(reinicio)
	} else {
		modoPuesto(reinicio)
	}
}

// ---------------------------------------------------------------------------
// PC central: guarda la lista, sirve a las demás PC y transmite al TV.

func modoCentral(reinicio bool) {
	l, err := escuchar(fmt.Sprintf(":%d", Puerto), reinicio)
	if err != nil {
		// Probablemente ya está abierto: solo mostramos la ventana.
		logf("puerto ocupado (%v), abro la ventana y salgo", err)
		abrirVentana(fmt.Sprintf("http://127.0.0.1:%d/?pc=%s", Puerto, urlq(local.PC)))
		return
	}
	escucha = l
	store = NewStore(datos)
	prevenirSuspension()
	mux := http.NewServeMux()
	rutasCentral(mux, datos)
	go func() { _ = http.Serve(l, mux) }()
	go responderDescubrimiento()
	go caster.Loop()
	go chequeoPeriodico()
	url := fmt.Sprintf("http://127.0.0.1:%d/?pc=%s", Puerto, urlq(local.PC))
	if !reinicio {
		go func() { time.Sleep(300 * time.Millisecond); abrirVentana(url) }()
	}
	bandeja("TeToca · PC central", []itemBandeja{
		{"Abrir TeToca", func() { abrirVentana(url) }},
		{"Abrir pantalla de sala en el navegador", func() { abrirNavegador(fmt.Sprintf("http://127.0.0.1:%d/tv", Puerto)) }},
		{"Abrir carpeta de datos", func() { abrirCarpeta(datos) }},
	})
}

// ---------------------------------------------------------------------------
// Puesto (o primera vez): una pequeña página local que busca la central
// y lleva a la pantalla de mesa o caja.

func modoPuesto(reinicio bool) {
	l, err := escuchar(fmt.Sprintf("127.0.0.1:%d", PuertoLocal), reinicio)
	url := fmt.Sprintf("http://127.0.0.1:%d/", PuertoLocal)
	if err != nil {
		abrirVentana(url)
		return
	}
	escucha = l
	mux := http.NewServeMux()
	rutasLocal(mux)
	go func() { _ = http.Serve(l, mux) }()
	go func() { time.Sleep(300 * time.Millisecond); abrirVentana(url) }()
	titulo := "TeToca"
	if local.Modo == "puesto" {
		titulo = "TeToca · Puesto"
	}
	bandeja(titulo, []itemBandeja{
		{"Abrir TeToca", func() { abrirVentana(url) }},
	})
}

func urlq(s string) string {
	r := strings.NewReplacer(" ", "%20", "&", "%26", "#", "%23", "?", "%3F")
	return r.Replace(s)
}

func probarCentral(ip string) bool {
	c := http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := c.Get(fmt.Sprintf("http://%s:%d/api/info", ip, Puerto))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var i struct{ App string }
	_ = json.NewDecoder(resp.Body).Decode(&i)
	return i.App == "TeToca"
}

func rutasLocal(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			b, err := webFS.ReadFile("web/" + strings.TrimPrefix(r.URL.Path, "/static/"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if strings.HasSuffix(r.URL.Path, ".woff2") {
				w.Header().Set("Content-Type", "font/woff2")
			} else if strings.HasSuffix(r.URL.Path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			} else if strings.HasSuffix(r.URL.Path, ".png") {
				w.Header().Set("Content-Type", "image/png")
			} else if strings.HasSuffix(r.URL.Path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}
			_, _ = w.Write(b)
			return
		}
		servirArchivo(w, "local.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("/local/estado", func(w http.ResponseWriter, r *http.Request) {
		localMu.Lock()
		defer localMu.Unlock()
		jsonOK(w, map[string]interface{}{"local": local, "version": Version, "ips": ipsLocales()})
	})
	mux.HandleFunc("/local/buscar", func(w http.ResponseWriter, r *http.Request) {
		jsonOK(w, BuscarCentrales(2*time.Second))
	})
	mux.HandleFunc("/local/conectar", func(w http.ResponseWriter, r *http.Request) {
		// Busca la central (por si cambió de dirección) y devuelve a dónde ir.
		localMu.Lock()
		l := local
		localMu.Unlock()
		ip := ""
		for _, c := range BuscarCentrales(1500 * time.Millisecond) {
			if l.CentralHost == "" || strings.EqualFold(c.Host, l.CentralHost) {
				ip = c.IP
				if c.Host != "" {
					l.CentralHost = c.Host
				}
				break
			}
		}
		if ip == "" && l.CentralIP != "" && probarCentral(l.CentralIP) {
			ip = l.CentralIP
		}
		if ip == "" {
			jsonErr(w, UserErr{"No encuentro la PC central."})
			return
		}
		if ip != l.CentralIP {
			localMu.Lock()
			local.CentralIP, local.CentralHost = ip, l.CentralHost
			localMu.Unlock()
			guardarLocal()
		}
		jsonOK(w, map[string]string{"url": fmt.Sprintf("http://%s:%d/?pc=%s&puesto=1", ip, Puerto, urlq(l.PC))})
	})
	mux.HandleFunc("/local/elegir", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Modo, IP, Host, PC string }
		if err := leerBody(r, &q); err != nil {
			jsonErr(w, err)
			return
		}
		if q.Modo == "puesto" {
			if q.IP == "" || !probarCentral(q.IP) {
				jsonErr(w, UserErr{"No pude conectarme a esa dirección. Revisá que la PC central esté prendida con TeToca abierto."})
				return
			}
		}
		localMu.Lock()
		local.Modo = q.Modo
		if q.PC != "" {
			local.PC = q.PC
		}
		if q.Modo == "puesto" {
			local.CentralIP, local.CentralHost = q.IP, q.Host
		}
		localMu.Unlock()
		guardarLocal()
		jsonOK(w, nil)
		if q.Modo == "central" {
			// arrancar de nuevo, ahora como central
			go func() { _ = reiniciarComoCentral() }()
		}
	})
}

func reiniciarComoCentral() error {
	time.Sleep(500 * time.Millisecond)
	liberarPuerto()
	return reiniciar(rutaExe())
}
