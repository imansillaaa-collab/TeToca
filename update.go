package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const Repo = "imansillaaa-collab/TeToca"

// Certificados raíz incluidos, para que las descargas anden aunque
// Windows 7 no tenga sus certificados actualizados.
//
//go:embed certs/roots.pem
var raicesPEM []byte

type UpdInfo struct {
	Disponible  bool   `json:"disponible"`
	Version     string `json:"version"`
	Notas       string `json:"notas"`
	Estado      string `json:"estado"` // "", "descargando", "reiniciando", "error"
	Mensaje     string `json:"mensaje"`
	HayAnterior bool   `json:"hayAnterior"`
	Actual      string `json:"actual"`
}

var (
	updMu   sync.Mutex
	upd     UpdInfo
	urlExe  string
	urlHash string
)

func clienteHTTP() *http.Client {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raicesPEM) {
		pool = nil
	}
	return &http.Client{Timeout: 5 * time.Minute, Transport: &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}}
}

func updInfo() UpdInfo {
	updMu.Lock()
	defer updMu.Unlock()
	u := upd
	u.Actual = Version
	_, err := os.Stat(rutaAnterior())
	u.HayAnterior = err == nil
	return u
}

func rutaExe() string {
	p, err := os.Executable()
	if err != nil {
		return "TeToca.exe"
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func rutaAnterior() string { return filepath.Join(filepath.Dir(rutaExe()), "TeToca.anterior.exe") }

func versionMayor(a, b string) bool {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// Las versiones publicadas viven en la rama "publicado" del repo:
//
//	ultima.json  -> {"version":"1.0.1","notas":"...","sha256":"..."}
//	TeToca.exe   -> el programa
var basePublicado = "https://raw.githubusercontent.com/" + Repo + "/publicado/"

func urlPublicado(archivo string) string { return basePublicado + archivo }

func buscarActualizacion() {
	req, _ := http.NewRequest("GET", urlPublicado("ultima.json")+"?t="+strconv.FormatInt(time.Now().Unix(), 10), nil)
	req.Header.Set("User-Agent", "TeToca/"+Version)
	req.Header.Set("Cache-Control", "no-cache")
	c := clienteHTTP()
	c.Timeout = 20 * time.Second
	resp, err := c.Do(req)
	if err != nil {
		logf("buscar actualización: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logf("buscar actualización: HTTP %d", resp.StatusCode)
		return
	}
	var ult struct {
		Version string `json:"version"`
		Notas   string `json:"notas"`
		SHA256  string `json:"sha256"`
		Archivo string `json:"archivo"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&ult); err != nil {
		return
	}
	if ult.Archivo == "" {
		ult.Archivo = "TeToca.exe"
	}
	updMu.Lock()
	defer updMu.Unlock()
	if ult.SHA256 != "" && versionMayor(ult.Version, Version) {
		upd.Disponible, upd.Version, upd.Notas = true, strings.TrimPrefix(ult.Version, "v"), strings.TrimSpace(ult.Notas)
		urlExe, urlHash = urlPublicado(ult.Archivo), strings.ToLower(strings.TrimSpace(ult.SHA256))
	} else if upd.Estado == "" {
		upd.Disponible = false
	}
}

func descargar(url string, max int64) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "TeToca/"+Version)
	resp, err := clienteHTTP().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, max))
}

func setUpd(estado, msg string) {
	updMu.Lock()
	upd.Estado, upd.Mensaje = estado, msg
	updMu.Unlock()
	if store != nil {
		store.avisar()
	}
}

func aplicarActualizacion() error {
	updMu.Lock()
	ue, uh, disp := urlExe, urlHash, upd.Disponible
	updMu.Unlock()
	if !disp {
		return UserErr{"No hay ninguna actualización pendiente."}
	}
	setUpd("descargando", "Descargando la versión nueva…")
	go func() {
		err := func() error {
			b, err := descargarVerificado(ue, uh)
			if err != nil {
				return err
			}
			return reemplazarYReiniciar(b)
		}()
		if err != nil {
			logf("actualización: %v", err)
			setUpd("error", "No se pudo actualizar: "+err.Error()+". Se sigue usando la versión actual.")
		}
	}()
	return nil
}

func descargarVerificado(url, esperado string) ([]byte, error) {
	if len(esperado) != 64 {
		return nil, errors.New("falta el código de control de la versión")
	}
	b, err := descargar(url, 200<<20)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != esperado {
		return nil, errors.New("la descarga llegó dañada (no coincide el control)")
	}
	return b, nil
}

// reemplazarYReiniciar deja la versión actual como "anterior", pone la nueva y reinicia.
func reemplazarYReiniciar(nuevo []byte) error {
	actual := rutaExe()
	dir := filepath.Dir(actual)
	tmp := filepath.Join(dir, "TeToca.nuevo.exe")
	if err := os.WriteFile(tmp, nuevo, 0755); err != nil {
		return err
	}
	ant := rutaAnterior()
	_ = os.Remove(ant)
	if err := os.Rename(actual, ant); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, actual); err != nil {
		_ = os.Rename(ant, actual)
		return err
	}
	setUpd("reiniciando", "Reiniciando con la versión nueva…")
	return reiniciar(actual)
}

func volverAnterior() error {
	ant := rutaAnterior()
	if _, err := os.Stat(ant); err != nil {
		return UserErr{"No hay una versión anterior guardada."}
	}
	actual := rutaExe()
	dir := filepath.Dir(actual)
	tmp := filepath.Join(dir, "TeToca.cambio.exe")
	_ = os.Remove(tmp)
	if err := os.Rename(actual, tmp); err != nil {
		return err
	}
	if err := os.Rename(ant, actual); err != nil {
		_ = os.Rename(tmp, actual)
		return err
	}
	_ = os.Rename(tmp, ant)
	setUpd("reiniciando", "Volviendo a la versión anterior…")
	go func() { _ = reiniciar(actual) }()
	return nil
}

func reiniciar(exe string) error {
	time.Sleep(800 * time.Millisecond)
	liberarPuerto()
	cmd := exec.Command(exe, "--reinicio")
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	logf("reiniciando: %s", exe)
	os.Exit(0)
	return nil
}

func chequeoPeriodico() {
	time.Sleep(15 * time.Second)
	for {
		buscarActualizacion()
		if store != nil {
			store.avisar()
		}
		time.Sleep(3 * time.Hour)
	}
}
