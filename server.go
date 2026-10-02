package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

const Puerto = 8765

var store *Store

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if v == nil {
		v = map[string]bool{"ok": true}
	}
	_ = json.NewEncoder(w).Encode(v)
}

func jsonErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(400)
	msg := err.Error()
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func leerBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func servirArchivo(w http.ResponseWriter, nombre, tipo string) {
	b, err := webFS.ReadFile("web/" + nombre)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

func rutasCentral(mux *http.ServeMux, datos string) {
	sub, _ := fs.Sub(webFS, "web")
	static := http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
	mux.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".woff2") {
			w.Header().Set("Cache-Control", "max-age=31536000")
		} else if strings.HasSuffix(r.URL.Path, ".mp3") {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		servirArchivo(w, "index.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("/tv", func(w http.ResponseWriter, r *http.Request) {
		servirArchivo(w, "tv.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		p := filepath.Join(datos, "logo.png")
		if _, err := os.Stat(p); err == nil {
			http.ServeFile(w, r, p)
			return
		}
		servirArchivo(w, "icono.png", "image/png")
	})
	mux.HandleFunc("/api/estado", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(store.Snapshot())
	})
	mux.HandleFunc("/api/eventos", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "sin streaming", 500)
			return
		}
		pc := r.URL.Query().Get("pc")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Connection", "keep-alive")
		c := store.Suscribir()
		defer store.Desuscribir(c)
		enviar := func() bool {
			if pc != "" {
				store.Visto(pc)
			}
			_, err := fmt.Fprintf(w, "retry: 2000\ndata: %s\n\n", store.Snapshot())
			fl.Flush()
			return err == nil
		}
		if !enviar() {
			return
		}
		tick := time.NewTicker(20 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-c:
				if !enviar() {
					return
				}
			case <-tick.C:
				if !enviar() {
					return
				}
			}
		}
	})
	mux.HandleFunc("/api/accion", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Accion, PC, ID string }
		if err := leerBody(r, &q); err != nil {
			jsonErr(w, err)
			return
		}
		if err := store.Accion(q.Accion, q.PC, q.ID); err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/cargar", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			jsonErr(w, UserErr{"No llegó el archivo."})
			return
		}
		f, h, err := r.FormFile("archivo")
		if err != nil {
			jsonErr(w, UserErr{"No llegó el archivo."})
			return
		}
		defer f.Close()
		b, _ := io.ReadAll(io.LimitReader(f, 20<<20))
		turnos, fecha, err := Importar(b)
		if err != nil {
			jsonErr(w, err)
			return
		}
		reg := r.FormValue("registro")
		if reg != "2" {
			reg = "1"
		}
		store.Cargar(turnos, fecha, h.Filename, reg)
		aviso := ""
		if fecha != "" && fecha != hoy() {
			aviso = "Ojo: el archivo tiene turnos del " + fecha + " y hoy es " + hoy() + "."
		}
		jsonOK(w, map[string]interface{}{"ok": true, "cantidad": len(turnos), "aviso": aviso})
	})
	mux.HandleFunc("/api/separar", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ ID string }
		if err := leerBody(r, &q); err != nil {
			jsonErr(w, err)
			return
		}
		if err := store.Separar(q.ID); err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/manual", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Nombre, Precarga string }
		if err := leerBody(r, &q); err != nil || strings.TrimSpace(q.Nombre) == "" {
			jsonErr(w, UserErr{"Falta el nombre."})
			return
		}
		store.AgregarManual(q.Nombre, q.Precarga)
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/pc", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			PC         string
			Rol        *string
			Box        *string
			MostrarBox *bool
		}
		if err := leerBody(r, &q); err != nil || q.PC == "" {
			jsonErr(w, UserErr{"Datos incompletos."})
			return
		}
		store.SetPC(q.PC, func(c *PCConf) {
			if q.Rol != nil {
				c.Rol = *q.Rol
			}
			if q.Box != nil {
				c.Box = *q.Box
			}
			if q.MostrarBox != nil {
				c.MostrarBox = *q.MostrarBox
			}
		})
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Sonido, Tema, Organismo, Oficina, TemaTV, Seccional, Canal *string
			Registros                                                  *[]string
			Ultimos, Ausentes, VolumenTV                               *int
		}
		if err := leerBody(r, &q); err != nil {
			jsonErr(w, err)
			return
		}
		store.SetConfig(func(c *Config) {
			if q.Sonido != nil {
				c.Sonido = *q.Sonido
			}
			if q.Tema != nil {
				c.Tema = *q.Tema
			}
			if q.Registros != nil && len(*q.Registros) <= 2 {
				var rs []string
				for i, n := range *q.Registros {
					n = strings.TrimSpace(n)
					if n == "" {
						n = "R" + strconv.Itoa(i+1)
					}
					if len(n) > 20 {
						n = n[:20]
					}
					rs = append(rs, n)
				}
				c.Registros = rs
			}
			if q.TemaTV != nil && temasTV[*q.TemaTV] {
				c.TemaTV = *q.TemaTV
			}
			if q.Organismo != nil {
				c.Organismo = *q.Organismo
			}
			if q.Oficina != nil {
				c.Oficina = *q.Oficina
			}
			if q.Canal != nil {
				cn := strings.ToLower(strings.TrimSpace(*q.Canal))
				if cn == "general" {
					cn = ""
				}
				if canalValido(cn) {
					c.Canal = cn
					setCanal(cn)
					go func() { buscarActualizacion(); store.avisar() }()
				}
			}
			if q.Seccional != nil {
				c.Seccional = strings.TrimSpace(*q.Seccional)
			}
			if q.Ultimos != nil && *q.Ultimos >= 1 && *q.Ultimos <= 8 {
				c.Ultimos = *q.Ultimos
			}
			if q.Ausentes != nil && *q.Ausentes >= 0 && *q.Ausentes <= 6 {
				c.Ausentes = *q.Ausentes
			}
			if q.VolumenTV != nil && *q.VolumenTV >= 0 && *q.VolumenTV <= 100 {
				c.VolumenTV = *q.VolumenTV
				caster.AjustarVolumen()
			}
		})
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/logo", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(datos, "logo.png")
		if r.Method == http.MethodDelete {
			_ = os.Remove(p)
			store.SetConfig(func(c *Config) { c.TieneLogo = false })
			jsonOK(w, nil)
			return
		}
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			jsonErr(w, UserErr{"No llegó la imagen."})
			return
		}
		f, _, err := r.FormFile("logo")
		if err != nil {
			jsonErr(w, UserErr{"No llegó la imagen."})
			return
		}
		defer f.Close()
		b, _ := io.ReadAll(io.LimitReader(f, 5<<20))
		if err := os.WriteFile(p, b, 0644); err != nil {
			jsonErr(w, err)
			return
		}
		store.SetConfig(func(c *Config) { c.TieneLogo = true })
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/buscar", func(w http.ResponseWriter, r *http.Request) {
		jsonOK(w, BuscarTVs(3*time.Second))
	})
	mux.HandleFunc("/api/tv/conectar", func(w http.ResponseWriter, r *http.Request) {
		var q TVConf
		if err := leerBody(r, &q); err != nil || q.Host == "" {
			jsonErr(w, UserErr{"Elegí un televisor."})
			return
		}
		store.SetConfig(func(c *Config) {
			rec := c.TV.Receptor
			c.TV = TVConf{ID: q.ID, Nombre: q.Nombre, Host: q.Host, Port: q.Port, Auto: true, Receptor: rec}
			if q.Receptor != "" {
				c.TV.Receptor = q.Receptor
			}
		})
		caster.Conectar()
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/desconectar", func(w http.ResponseWriter, r *http.Request) {
		store.SetConfig(func(c *Config) { c.TV.Auto = false })
		caster.Desconectar()
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/reconectar", func(w http.ResponseWriter, r *http.Request) {
		store.SetConfig(func(c *Config) { c.TV.Auto = true })
		caster.Conectar()
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/receptor", func(w http.ResponseWriter, r *http.Request) {
		var q struct{ Receptor string }
		_ = leerBody(r, &q)
		if q.Receptor != "dashcast" && q.Receptor != "urlcast" {
			jsonErr(w, UserErr{"Receptor inválido."})
			return
		}
		store.SetConfig(func(c *Config) { c.TV.Receptor = q.Receptor })
		caster.Conectar()
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/probar-sonido", func(w http.ResponseWriter, r *http.Request) {
		store.mu.Lock()
		pruebaSonido++
		tvSonido = nil
		store.mu.Unlock()
		store.avisar()
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/tv/reporte-sonido", func(w http.ResponseWriter, r *http.Request) {
		var q ReporteSonido
		if err := leerBody(r, &q); err != nil {
			jsonErr(w, err)
			return
		}
		if len(q.Detalle) > 300 {
			q.Detalle = q.Detalle[:300]
		}
		q.Hora = time.Now()
		logf("sonido en el TV: ok=%v prueba=%v %s", q.OK, q.Prueba, q.Detalle)
		store.mu.Lock()
		if q.Prueba || tvSonido == nil || !tvSonido.Prueba {
			tvSonido = &q
		}
		store.mu.Unlock()
		if q.Prueba {
			store.avisar()
		}
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/update/buscar", func(w http.ResponseWriter, r *http.Request) {
		buscarActualizacion()
		store.avisar()
		jsonOK(w, updInfo())
	})
	mux.HandleFunc("/api/update/aplicar", func(w http.ResponseWriter, r *http.Request) {
		if err := aplicarActualizacion(); err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/update/anterior", func(w http.ResponseWriter, r *http.Request) {
		if err := volverAnterior(); err != nil {
			jsonErr(w, err)
			return
		}
		jsonOK(w, nil)
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		h, _ := os.Hostname()
		updMu.Lock()
		cn := canal
		updMu.Unlock()
		jsonOK(w, map[string]interface{}{"app": "TeToca", "version": Version, "host": h, "ips": ipsLocales(), "canal": cn})
	})
}

func ipsLocales() []string {
	var out []string
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil {
				out = append(out, n.IP.String())
			}
		}
	}
	return out
}

// ipHacia devuelve la IP de esta PC que se usa para llegar a otro equipo (ej. el TV).
func ipHacia(host string) string {
	c, err := net.DialTimeout("udp", net.JoinHostPort(host, "9"), time.Second)
	if err != nil {
		ips := ipsLocales()
		if len(ips) > 0 {
			return ips[0]
		}
		return "127.0.0.1"
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}
