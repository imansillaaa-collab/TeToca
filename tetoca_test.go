package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func certPrueba(t *testing.T) tls.Certificate {
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "tv"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k}
}

// tvFalso imita un Chromecast: responde estado, abre la app y registra lo que le mandan.
func tvFalso(t *testing.T, recibido chan castMsg) (string, int) {
	l, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certPrueba(t)}})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		cc := &castConn{c: nil}
		_ = cc
		leer := func() (castMsg, error) {
			var h [4]byte
			if _, err := io.ReadFull(c, h[:]); err != nil {
				return castMsg{}, err
			}
			b := make([]byte, binary.BigEndian.Uint32(h[:]))
			if _, err := io.ReadFull(c, b); err != nil {
				return castMsg{}, err
			}
			return decodeMsg(b)
		}
		enviar := func(src, ns string, v interface{}) {
			p, _ := json.Marshal(v)
			b := encodeMsg(castMsg{Src: src, Dst: "sender-tetoca", NS: ns, Payload: string(p)})
			var h [4]byte
			binary.BigEndian.PutUint32(h[:], uint32(len(b)))
			_, _ = c.Write(h[:])
			_, _ = c.Write(b)
		}
		abierta := false
		nivel := 0.3
		estado := func() {
			apps := []map[string]interface{}{{"appId": "E8C28D3C", "displayName": "Backdrop", "isIdleScreen": true, "transportId": "bd"}}
			if abierta {
				apps = []map[string]interface{}{{"appId": "84912283", "displayName": "DashCast", "transportId": "web-7"}}
			}
			enviar("receiver-0", nsReceptor, map[string]interface{}{"type": "RECEIVER_STATUS", "status": map[string]interface{}{"applications": apps,
				"volume": map[string]interface{}{"controlType": "attenuation", "level": nivel, "muted": false}}})
		}
		for {
			m, err := leer()
			if err != nil {
				return
			}
			recibido <- m
			switch {
			case strings.Contains(m.Payload, "GET_STATUS"):
				estado()
			case strings.Contains(m.Payload, "SET_VOLUME") && strings.Contains(m.Payload, `"level"`):
				var q struct{ Volume struct{ Level float64 } }
				_ = json.Unmarshal([]byte(m.Payload), &q)
				nivel = q.Volume.Level
				estado()
			case strings.Contains(m.Payload, "LAUNCH"):
				abierta = true
				estado()
			case strings.Contains(m.Payload, `"PING"`):
				enviar("receiver-0", nsLatido, map[string]string{"type": "PONG"})
			}
		}
	}()
	a := l.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port
}

func TestTransmitirAlTV(t *testing.T) {
	dir := t.TempDir()
	store = NewStore(dir)
	recibido := make(chan castMsg, 50)
	host, port := tvFalso(t, recibido)
	store.C.TV = TVConf{ID: "x", Nombre: "TV Sala", Host: host, Port: port, Auto: true, Receptor: "dashcast"}
	go caster.Loop()
	fin := time.After(8 * time.Second)
	var lanzo, url, vol bool
	for !(lanzo && url && vol) {
		select {
		case m := <-recibido:
			if strings.Contains(m.Payload, `"LAUNCH"`) && strings.Contains(m.Payload, "84912283") {
				lanzo = true
			}
			if m.NS == "urn:x-cast:com.madmod.dashcast" && m.Dst == "web-7" && strings.Contains(m.Payload, "/tv") && strings.Contains(m.Payload, `"force":true`) {
				url = true
			}
			if strings.Contains(m.Payload, "SET_VOLUME") && strings.Contains(m.Payload, `"level":1`) {
				vol = true
			}
		case <-fin:
			t.Fatalf("no se completó la transmisión (lanzó=%v, url=%v, volumen=%v)", lanzo, url, vol)
		}
	}
	time.Sleep(200 * time.Millisecond)
	if st := tvStatus(); st.Estado != "conectado" {
		t.Fatalf("estado esperado conectado, fue %q (%s)", st.Estado, st.Mensaje)
	}
	if st := tvStatus(); st.Volumen != 100 || st.Control != "attenuation" {
		t.Fatalf("el Chromecast debería quedar al 100%%: %+v", st)
	}
	// con un volumen fijo elegido en Configuración, se respeta ese
	store.mu.Lock()
	store.C.VolumenTV = 60
	store.mu.Unlock()
	caster.AjustarVolumen()
	fin = time.After(5 * time.Second)
	for {
		select {
		case m := <-recibido:
			if strings.Contains(m.Payload, "SET_VOLUME") && strings.Contains(m.Payload, `"level":0.6`) {
				return
			}
		case <-fin:
			t.Fatal("no se ajustó el volumen al 60%")
		}
	}
}

func TestVolumenDeseado(t *testing.T) {
	if volumenDeseado(0, "attenuation") != 1 || volumenDeseado(0, "master") != -1 || volumenDeseado(0, "fixed") != -1 || volumenDeseado(40, "master") != 0.4 {
		t.Fatal("volumenDeseado mal")
	}
}

func TestProtobufIdaYVuelta(t *testing.T) {
	m := castMsg{Src: "a", Dst: "b", NS: "urn:x", Payload: `{"type":"PING","ñ":"á"}`}
	d, err := decodeMsg(encodeMsg(m))
	if err != nil || d != m {
		t.Fatalf("%v %+v", err, d)
	}
}

const xlsPrueba = `<html><head><meta http-equiv="Content-Type"content="text/html; charset=utf-8"/></head><body><table border = 1><thead><tr><b><th colspan=4>Solicitante</th><th>Telefono de Contacto</th><th>Tipo Usuario</th><th>Fecha de Turno</th><th>Horario Turno</th><th>Dominio</th><th>Nro. Precarga</th><th>Estado Turno</th><th>Fecha de Cancelacion/ Desistido</th><th>Motivo Cancelacion</th><th>SITE Pago</th><th colspan=5>Tramites</th></b></tr>
<tr><td colspan=4><font>PEREZ, MARIA LAURA</Font></td><td>1</td><td>NO</td><td>28/09/2026</td><td>09:20:00</td><td>AB123CD</td><td>90000002</td><td>O</td><td>-</td><td>-</td><td>NO</td><td colspan=5>083000 - TRANSFERENCIA NACIONAL</br></td></tr>
<tr><td colspan=4>SA, AGRO PRUEBA</td><td>1</td><td>NO</td><td>28/09/2026</td><td>08:00:00</td><td></td><td>90000001</td><td>O</td><td>-</td><td>-</td><td>NO</td><td colspan=5>999999 - CONSULTAS</td></tr>
<tr><td colspan=4>GOMEZ, IVÁN</td><td>1</td><td>NO</td><td>28/09/2026</td><td>10:00:00</td><td>XX1</td><td>90000003</td><td>C</td><td>27/09/2026</td><td>x</td><td>NO</td><td colspan=5>022001 - CONSULTA DE LEGAJO</td></tr>
</table></body></html>`

func TestImportarXLS(t *testing.T) {
	ts, fecha, err := Importar([]byte(xlsPrueba))
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 3 || fecha != "28/09/2026" {
		t.Fatalf("%d turnos, fecha %s", len(ts), fecha)
	}
	ordenar(ts)
	if ts[0].Precarga != "90000001" || ts[1].Nombre != "PEREZ, MARIA LAURA" || ts[1].Tramite != "TRANSFERENCIA NACIONAL" {
		t.Fatalf("orden o datos mal: %+v %+v", ts[0], ts[1])
	}
	if ts[2].Estado != EstCancelado {
		t.Fatal("el cancelado no quedó cancelado")
	}
}

func TestImportarCSVConAcentosRotos(t *testing.T) {
	csv := "Solicitante;;;;Telefono;Tipo;Fecha de Turno;Horario Turno;Dominio;Nro. Precarga;Estado Turno;Fecha de Cancelacion/ Desistido;Motivo;SITE;Tramites;;;;\r\n" +
		"RODRжGUEZ, RAщL;;;;1;NO;28/09/2026;08:00:00;AA1;91;O;-;-;NO;999998 - RETIRO DE DOCUMENTACIрN;;;;\r\n" +
		"GAVIЅA, IVЕN;;;;1;NO;28/09/2026;08:20:00;AA2;92;O;-;-;NO;x;;;;\r\n"
	ts, _, err := Importar([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if ts[0].Nombre != "RODRÍGUEZ, RAÚL" || ts[1].Nombre != "GAVIÑA, IVÁN" || ts[0].Tramite != "RETIRO DE DOCUMENTACIÓN" {
		t.Fatalf("acentos: %q %q %q", ts[0].Nombre, ts[1].Nombre, ts[0].Tramite)
	}
}

func TestFlujoMesaCaja(t *testing.T) {
	store = NewStore(t.TempDir())
	ts, _, _ := Importar([]byte(xlsPrueba))
	store.Cargar(ts, "28/09/2026", "x.xls", "1")
	if err := store.Accion("mesa_siguiente", "m1", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.Accion("mesa_siguiente", "m1", ""); err == nil {
		t.Fatal("dejó llamar dos veces sin resolver")
	}
	if err := store.Accion("mesa_siguiente", "m2", ""); err != nil {
		t.Fatal(err)
	}
	if a, b := store.actualDe("m1"), store.actualDe("m2"); a == nil || b == nil || a.ID == b.ID {
		t.Fatal("dos PC agarraron el mismo turno")
	}
	_ = store.Accion("pasar_caja", "m1", "")
	_ = store.Accion("ausente", "m2", "")
	if err := store.Accion("caja_siguiente", "c1", ""); err != nil {
		t.Fatal(err)
	}
	if store.actualDe("c1") == nil || store.actualDe("c1").Estado != EstCaja {
		t.Fatal("caja no tomó al turno")
	}
	_ = store.Accion("terminar", "c1", "")
	if err := store.Accion("caja_siguiente", "c1", ""); err == nil {
		t.Fatal("caja llamó a alguien que no existe")
	}
	// recargar el mismo día conserva estados
	ts2, _, _ := Importar([]byte(xlsPrueba))
	store.E.CargadoDia = hoy()
	store.Cargar(ts2, "28/09/2026", "x.xls", "1")
	cuenta := map[string]int{}
	for _, t := range store.E.Turnos {
		cuenta[t.Estado]++
	}
	if cuenta[EstTerminado] != 1 || cuenta[EstAusente] != 1 {
		t.Fatalf("se perdió el estado al recargar: %v", cuenta)
	}
	if _, err := os.Stat(store.dir + "/turnos.json"); err != nil {
		t.Fatal("no guardó en disco")
	}
}

func TestVersiones(t *testing.T) {
	if !versionMayor("v1.2.0", "1.1.9") || versionMayor("1.0.0", "1.0.0") || !versionMayor("1.10.0", "1.9.0") {
		t.Fatal("comparación de versiones mal")
	}
}

func TestActualizacion(t *testing.T) {
	exe := []byte("programa nuevo")
	sum := sha256.Sum256(exe)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ultima.json":
			_, _ = w.Write([]byte(`{"version":"1.0.1","notas":"Arreglos","sha256":"` + hex.EncodeToString(sum[:]) + `"}`))
		case "/TeToca.exe":
			_, _ = w.Write(exe)
		}
	}))
	defer srv.Close()
	basePublicado = srv.URL + "/"
	Version = "1.0.0"
	buscarActualizacion()
	u := updInfo()
	if !u.Disponible || u.Version != "1.0.1" || u.Notas != "Arreglos" {
		t.Fatalf("no detectó la versión nueva: %+v", u)
	}
	b, err := descargarVerificado(urlExe, urlHash)
	if err != nil || string(b) != "programa nuevo" {
		t.Fatal(err)
	}
	if _, err := descargarVerificado(urlExe, strings.Repeat("0", 64)); err == nil {
		t.Fatal("aceptó una descarga que no coincide con el control")
	}
	Version = "1.0.1"
	upd = UpdInfo{}
	buscarActualizacion()
	if updInfo().Disponible {
		t.Fatal("ofreció actualizar a la misma versión")
	}
}

func TestPedidoLocal(t *testing.T) {
	mk := func(remote, metodo, tipo, origen string) *http.Request {
		r := httptest.NewRequest(metodo, "/local/instalacion", strings.NewReader("{}"))
		r.RemoteAddr = remote
		if tipo != "" {
			r.Header.Set("Content-Type", tipo)
		}
		if origen != "" {
			r.Header.Set("Origin", origen)
		}
		return r
	}
	casos := []struct {
		r  *http.Request
		ok bool
	}{
		{mk("127.0.0.1:5000", "GET", "", ""), true},
		{mk("192.168.0.5:5000", "GET", "", ""), false},
		{mk("127.0.0.1:5000", "POST", "application/json", "http://127.0.0.1:8767"), true},
		{mk("127.0.0.1:5000", "POST", "text/plain", ""), false},
		{mk("127.0.0.1:5000", "POST", "application/json", "http://malo.com"), false},
		{mk("127.0.0.1:5000", "POST", "application/json", "http://192.168.0.10:8765"), false},
	}
	for i, c := range casos {
		if pedidoLocal(c.r) != c.ok {
			t.Errorf("caso %d: esperaba %v", i, c.ok)
		}
	}
}

func TestDosRegistros(t *testing.T) {
	store = NewStore(t.TempDir())
	mk := func(h, pre, nom, tr string) *Turno {
		return &Turno{Hora: h, Precarga: pre, Nombre: nom, Tramite: tr, Estado: EstPendiente}
	}
	r1 := []*Turno{mk("09:00", "100", "GÓMEZ, JUAN PABLO", "Transferencia"), mk("09:20", "101", "SOSA, ANA", "Consulta"),
		mk("09:00", "102", "GOMEZ, JUAN PABLO", "Inscripción"), mk("11:00", "103", "GOMEZ, JUAN PABLO", "Otro horario")}
	r2 := []*Turno{mk("09:00", "200", "Gomez Juan Pablo", "Legajo"), mk("09:10", "201", "RÍOS, VALENTINA", "Retiro")}
	store.Cargar(r1, "", "a.xls", "1")
	if len(store.E.Turnos) != 3 {
		t.Fatalf("mismo nombre y horario no se juntó (u otro horario sí): %d filas", len(store.E.Turnos))
	}
	if err := store.Accion("mesa_siguiente", "m1", ""); err != nil {
		t.Fatal(err)
	}
	store.Cargar(r2, "", "b.xls", "2")
	if len(store.E.Turnos) != 4 {
		t.Fatalf("esperaba 4 filas, hay %d", len(store.E.Turnos))
	}
	g := store.E.Turnos[0]
	if normNombre(g.Nombre) != normNombre("JUAN PABLO GOMEZ") || len(g.Tramites) != 3 || g.Hora != "09:00" || len(g.Registros) != 2 {
		t.Fatalf("no juntó al gestor: %+v", g)
	}
	if g.Estado != EstMesa || g.PC != "m1" {
		t.Fatalf("se perdió que estaba en mesa: %s %s", g.Estado, g.PC)
	}
	if len(store.E.Llamados) != 1 || store.E.Llamados[0].TurnoID != g.ID {
		t.Fatal("el llamado quedó apuntando a otro lado")
	}
	if err := store.Separar(g.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.E.Turnos) != 6 {
		t.Fatalf("separar: esperaba 6 filas, hay %d", len(store.E.Turnos))
	}
	if a := store.actualDe("m1"); a == nil || a.Precarga != "100" {
		t.Fatal("al separar, la atención tenía que seguir con el primer trámite")
	}
	// al otro día la lista vieja se borra sola
	store.E.CargadoDia = "01/01/2000"
	store.limpiarSiOtroDia()
	if len(store.E.Turnos) != 0 || len(store.E.Fuentes) != 0 {
		t.Fatal("no borró la lista del día anterior")
	}
}

func TestCanales(t *testing.T) {
	pub := map[string]string{
		"/ultima.json":        `{"version":"2.0.0","sha256":"` + strings.Repeat("a", 64) + `","archivo":"TeToca.exe"}`,
		"/ultima-prueba.json": `{"version":"2.1.0","sha256":"` + strings.Repeat("b", 64) + `","archivo":"TeToca-prueba.exe"}`,
		"/ultima-viejo.json":  `{"version":"1.9.0","sha256":"` + strings.Repeat("c", 64) + `","archivo":"TeToca-viejo.exe"}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := pub[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(b))
	}))
	defer srv.Close()
	antes, vAntes := basePublicado, Version
	defer func() { basePublicado, Version = antes, vAntes; setCanal("") }()
	basePublicado, Version = srv.URL+"/", "1.0.0"
	for _, c := range []struct{ canal, version, archivo string }{
		{"", "2.0.0", "TeToca.exe"}, {"prueba", "2.1.0", "TeToca-prueba.exe"},
		{"viejo", "2.0.0", "TeToca.exe"}, {"noexiste", "2.0.0", "TeToca.exe"},
	} {
		updMu.Lock()
		upd = UpdInfo{}
		updMu.Unlock()
		setCanal(c.canal)
		buscarActualizacion()
		if upd.Version != c.version || !strings.HasSuffix(urlExe, "/"+c.archivo) {
			t.Fatalf("canal %q: esperaba %s (%s), vino %s (%s)", c.canal, c.version, c.archivo, upd.Version, urlExe)
		}
	}
	if canalValido("Tandil!") || !canalValido("tandil-2") {
		t.Fatal("validación de canal")
	}
}
