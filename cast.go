package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// ---------------------------------------------------------------------------
// Búsqueda de televisores (Chromecast) en la red, por mDNS.

type TVEncontrado struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Modelo string `json:"modelo"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

func BuscarTVs(espera time.Duration) []TVEncontrado {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil
	}
	defer conn.Close()
	nombre := dnsmessage.MustNewName("_googlecast._tcp.local.")
	msg := dnsmessage.Message{Questions: []dnsmessage.Question{{Name: nombre, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}}}
	q, _ := msg.Pack()
	dst := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	for i := 0; i < 3; i++ {
		_, _ = conn.WriteTo(q, dst)
		time.Sleep(150 * time.Millisecond)
	}
	type inst struct {
		target string
		port   int
		txt    map[string]string
		ip     string
	}
	insts := map[string]*inst{}
	hosts := map[string]string{}
	get := func(n string) *inst {
		if insts[n] == nil {
			insts[n] = &inst{txt: map[string]string{}}
		}
		return insts[n]
	}
	fin := time.Now().Add(espera)
	buf := make([]byte, 9000)
	for time.Now().Before(fin) {
		_ = conn.SetReadDeadline(fin)
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			break
		}
		var p dnsmessage.Parser
		if _, err := p.Start(buf[:n]); err != nil {
			continue
		}
		_ = p.SkipAllQuestions()
		var rrs []dnsmessage.Resource
		for {
			r, err := p.Answer()
			if err != nil {
				break
			}
			rrs = append(rrs, r)
		}
		_ = p.SkipAllAuthorities()
		for {
			r, err := p.Additional()
			if err != nil {
				break
			}
			rrs = append(rrs, r)
		}
		for _, r := range rrs {
			switch b := r.Body.(type) {
			case *dnsmessage.PTRResource:
				if strings.Contains(r.Header.Name.String(), "_googlecast") {
					i := get(b.PTR.String())
					if i.ip == "" {
						i.ip = from.(*net.UDPAddr).IP.String()
					}
				}
			case *dnsmessage.SRVResource:
				i := get(r.Header.Name.String())
				i.target, i.port = b.Target.String(), int(b.Port)
				if i.ip == "" {
					i.ip = from.(*net.UDPAddr).IP.String()
				}
			case *dnsmessage.TXTResource:
				i := get(r.Header.Name.String())
				for _, t := range b.TXT {
					if k := strings.Index(t, "="); k > 0 {
						i.txt[t[:k]] = t[k+1:]
					}
				}
			case *dnsmessage.AResource:
				hosts[r.Header.Name.String()] = net.IP(b.A[:]).String()
			}
		}
	}
	var out []TVEncontrado
	vistos := map[string]bool{}
	for name, i := range insts {
		ip := hosts[i.target]
		if ip == "" {
			ip = i.ip
		}
		if ip == "" {
			continue
		}
		port := i.port
		if port == 0 {
			port = 8009
		}
		id := i.txt["id"]
		if id == "" {
			id = name
		}
		if vistos[id] {
			continue
		}
		vistos[id] = true
		fn := i.txt["fn"]
		if fn == "" {
			fn = strings.SplitN(name, ".", 2)[0]
		}
		out = append(out, TVEncontrado{ID: id, Nombre: fn, Modelo: i.txt["md"], Host: ip, Port: port})
	}
	return out
}

// ---------------------------------------------------------------------------
// Protocolo Cast v2 (el mismo que usa Chrome para transmitir).

const (
	nsConexion = "urn:x-cast:com.google.cast.tp.connection"
	nsLatido   = "urn:x-cast:com.google.cast.tp.heartbeat"
	nsReceptor = "urn:x-cast:com.google.cast.receiver"
)

// Apps de receptor públicas y gratuitas que muestran una página web.
var receptores = map[string]struct{ AppID, NS string }{
	"dashcast": {"84912283", "urn:x-cast:com.madmod.dashcast"},
	"urlcast":  {"5CB45E5A", "urn:x-cast:com.url.cast"},
}

type castMsg struct{ Src, Dst, NS, Payload string }

func pbVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func pbString(b []byte, field int, s string) []byte {
	b = pbVarint(b, uint64(field<<3|2))
	b = pbVarint(b, uint64(len(s)))
	return append(b, s...)
}

func encodeMsg(m castMsg) []byte {
	var b []byte
	b = pbVarint(b, 1<<3|0)
	b = pbVarint(b, 0) // CASTV2_1_0
	b = pbString(b, 2, m.Src)
	b = pbString(b, 3, m.Dst)
	b = pbString(b, 4, m.NS)
	b = pbVarint(b, 5<<3|0)
	b = pbVarint(b, 0) // STRING
	b = pbString(b, 6, m.Payload)
	return b
}

func decodeMsg(b []byte) (castMsg, error) {
	var m castMsg
	i := 0
	readVar := func() (uint64, error) {
		var v uint64
		var s uint
		for {
			if i >= len(b) {
				return 0, io.ErrUnexpectedEOF
			}
			c := b[i]
			i++
			v |= uint64(c&0x7f) << s
			if c < 0x80 {
				return v, nil
			}
			s += 7
		}
	}
	for i < len(b) {
		k, err := readVar()
		if err != nil {
			return m, err
		}
		field, wt := int(k>>3), int(k&7)
		switch wt {
		case 0:
			if _, err := readVar(); err != nil {
				return m, err
			}
		case 2:
			l, err := readVar()
			if err != nil || i+int(l) > len(b) {
				return m, io.ErrUnexpectedEOF
			}
			s := string(b[i : i+int(l)])
			i += int(l)
			switch field {
			case 2:
				m.Src = s
			case 3:
				m.Dst = s
			case 4:
				m.NS = s
			case 6:
				m.Payload = s
			}
		default:
			return m, fmt.Errorf("tipo %d no soportado", wt)
		}
	}
	return m, nil
}

type castConn struct {
	c    *tls.Conn
	r    *bufio.Reader
	wmu  sync.Mutex
	reqs int
}

func dialCast(host string, port int) (*castConn, error) {
	d := &net.Dialer{Timeout: 5 * time.Second}
	c, err := tls.DialWithDialer(d, "tcp", net.JoinHostPort(host, itoa(port)), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return nil, err
	}
	return &castConn{c: c, r: bufio.NewReader(c)}, nil
}

func (cc *castConn) send(dst, ns string, payload interface{}) error {
	p, _ := json.Marshal(payload)
	b := encodeMsg(castMsg{Src: "sender-tetoca", Dst: dst, NS: ns, Payload: string(p)})
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	cc.wmu.Lock()
	defer cc.wmu.Unlock()
	_ = cc.c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := cc.c.Write(hdr[:]); err != nil {
		return err
	}
	_, err := cc.c.Write(b)
	return err
}

func (cc *castConn) read() (castMsg, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(cc.r, hdr[:]); err != nil {
		return castMsg{}, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > 1<<20 {
		return castMsg{}, errors.New("mensaje muy grande")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(cc.r, b); err != nil {
		return castMsg{}, err
	}
	return decodeMsg(b)
}

// ---------------------------------------------------------------------------
// Caster: mantiene el TV mostrando la pantalla de sala y lo reconecta solo.

type TVStatus struct {
	Estado  string `json:"estado"` // sin_tv, desconectado, conectando, conectado, ocupado, error
	Nombre  string `json:"nombre"`
	Mensaje string `json:"mensaje"`
	URL     string `json:"url"`
	// Volumen del Chromecast tal como lo informa (0-100, -1 si no se sabe) y si
	// controla el volumen del propio TV ("master") o solo la salida ("attenuation").
	Volumen    int    `json:"volumen"`
	Control    string `json:"control"`
	Silenciado bool   `json:"silenciado"`
}

type Caster struct {
	mu      sync.Mutex
	st      TVStatus
	forzar  chan bool
	cortar  chan struct{}
	activo  *castConn
	volumen chan struct{}
}

var caster = &Caster{forzar: make(chan bool, 1), cortar: make(chan struct{}, 1), volumen: make(chan struct{}, 1)}

// AjustarVolumen pide revisar el volumen del Chromecast ahora (cambió la configuración).
func (c *Caster) AjustarVolumen() {
	select {
	case c.volumen <- struct{}{}:
	default:
	}
}

// volumenDeseado: a qué nivel (0-1) hay que dejar el Chromecast, o -1 para no tocarlo.
func volumenDeseado(config int, control string) float64 {
	if config > 0 {
		return float64(config) / 100
	}
	if control == "attenuation" {
		// Chromecast enchufado al TV: su volumen solo achica la salida. Al 100% suena
		// todo lo que puede y el volumen se maneja con el control remoto del TV.
		return 1
	}
	return -1
}

func (c *Caster) setVolumen(v int, control string, mute bool) {
	c.mu.Lock()
	cambio := c.st.Volumen != v || c.st.Control != control || c.st.Silenciado != mute
	c.st.Volumen, c.st.Control, c.st.Silenciado = v, control, mute
	c.mu.Unlock()
	if cambio && store != nil {
		store.avisar()
	}
}

func tvStatus() TVStatus {
	caster.mu.Lock()
	defer caster.mu.Unlock()
	return caster.st
}

func (c *Caster) set(estado, msg string) {
	c.mu.Lock()
	cambio := c.st.Estado != estado || c.st.Mensaje != msg
	c.st.Estado, c.st.Mensaje = estado, msg
	c.mu.Unlock()
	if cambio && store != nil {
		store.avisar()
	}
}

// Conectar pide (re)conectar ahora y lanzar la pantalla aunque el TV esté mostrando otra cosa.
func (c *Caster) Conectar() {
	select {
	case c.forzar <- true:
	default:
	}
}

func (c *Caster) Desconectar() {
	c.mu.Lock()
	if c.activo != nil {
		_ = c.activo.c.Close()
	}
	c.mu.Unlock()
	select {
	case c.cortar <- struct{}{}:
	default:
	}
}

func (c *Caster) Loop() {
	forzar := false
	for {
		store.mu.Lock()
		tv := store.C.TV
		store.mu.Unlock()
		c.mu.Lock()
		c.st.Nombre = tv.Nombre
		c.mu.Unlock()
		if tv.Host == "" {
			c.set("sin_tv", "Todavía no se eligió un televisor.")
		} else if !tv.Auto {
			c.set("desconectado", "Transmisión apagada.")
		} else {
			c.set("conectando", "Conectando con "+tv.Nombre+"…")
			err := c.sesion(tv, forzar)
			if err != nil {
				logf("tv: %v", err)
				c.set("error", "No se pudo conectar con el TV. Reintentando…")
			}
		}
		forzar = false
		select {
		case forzar = <-c.forzar:
		case <-c.cortar:
		case <-time.After(15 * time.Second):
		}
	}
}

type recvStatus struct {
	Type   string `json:"type"`
	Status struct {
		Volume *struct {
			ControlType string   `json:"controlType"`
			Level       *float64 `json:"level"`
			Muted       bool     `json:"muted"`
		} `json:"volume"`
		Applications []struct {
			AppID        string `json:"appId"`
			DisplayName  string `json:"displayName"`
			TransportID  string `json:"transportId"`
			IsIdleScreen bool   `json:"isIdleScreen"`
		} `json:"applications"`
	} `json:"status"`
}

// sesion mantiene una conexión abierta con el TV hasta que se corte.
func (c *Caster) sesion(tv TVConf, forzar bool) error {
	port := tv.Port
	if port == 0 {
		port = 8009
	}
	c.setVolumen(-1, "", false)
	cc, err := dialCast(tv.Host, port)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.activo = cc
	c.mu.Unlock()
	defer func() {
		_ = cc.c.Close()
		c.mu.Lock()
		c.activo = nil
		c.mu.Unlock()
	}()
	rec, ok := receptores[tv.Receptor]
	if !ok {
		rec = receptores["dashcast"]
	}
	url := fmt.Sprintf("http://%s:%d/tv", ipHacia(tv.Host), Puerto)
	c.mu.Lock()
	c.st.URL = url
	c.mu.Unlock()

	if err := cc.send("receiver-0", nsConexion, map[string]string{"type": "CONNECT"}); err != nil {
		return err
	}
	req := 1
	pedirEstado := func() error {
		req++
		return cc.send("receiver-0", nsReceptor, map[string]interface{}{"type": "GET_STATUS", "requestId": req})
	}
	lanzar := func() error {
		req++
		c.set("conectando", "Abriendo la pantalla de sala en "+tv.Nombre+"…")
		return cc.send("receiver-0", nsReceptor, map[string]interface{}{"type": "LAUNCH", "appId": rec.AppID, "requestId": req})
	}
	if err := pedirEstado(); err != nil {
		return err
	}
	msgs := make(chan castMsg, 16)
	errs := make(chan error, 1)
	go func() {
		for {
			m, err := cc.read()
			if err != nil {
				errs <- err
				return
			}
			msgs <- m
		}
	}()
	latido := time.NewTicker(5 * time.Second)
	defer latido.Stop()
	control := time.NewTicker(30 * time.Second)
	defer control.Stop()
	ultimo := time.Now()
	lanzado := false
	cargadoEn := ""
	var ultimoLanzamiento, ultimoVolumen time.Time
	for {
		select {
		case err := <-errs:
			return err
		case <-c.cortar:
			return nil
		case f := <-c.forzar:
			if f {
				forzar = true
				lanzado = false
				cargadoEn = ""
				ultimoLanzamiento = time.Time{}
				_ = pedirEstado()
			}
		case <-latido.C:
			if time.Since(ultimo) > 20*time.Second {
				return errors.New("el TV dejó de responder")
			}
			if err := cc.send("receiver-0", nsLatido, map[string]string{"type": "PING"}); err != nil {
				return err
			}
		case <-control.C:
			if err := pedirEstado(); err != nil {
				return err
			}
		case <-c.volumen:
			ultimoVolumen = time.Time{}
			if err := pedirEstado(); err != nil {
				return err
			}
		case m := <-msgs:
			ultimo = time.Now()
			if m.NS == nsLatido && strings.Contains(m.Payload, `"PING"`) {
				_ = cc.send(m.Src, nsLatido, map[string]string{"type": "PONG"})
				continue
			}
			if m.NS != nsReceptor {
				continue
			}
			var st recvStatus
			if json.Unmarshal([]byte(m.Payload), &st) != nil || st.Type != "RECEIVER_STATUS" {
				if strings.Contains(m.Payload, "LAUNCH_ERROR") {
					c.set("error", "El TV no pudo abrir la pantalla de sala. Probá cambiar el receptor en Configuración.")
				}
				continue
			}
			if v := st.Status.Volume; v != nil && v.Level != nil {
				c.setVolumen(int(*v.Level*100+0.5), v.ControlType, v.Muted)
				store.mu.Lock()
				conf := store.C.VolumenTV
				store.mu.Unlock()
				quiero := volumenDeseado(conf, v.ControlType)
				if quiero >= 0 && time.Since(ultimoVolumen) > 3*time.Second && (v.Muted || *v.Level < quiero-0.01 || *v.Level > quiero+0.01) {
					ultimoVolumen = time.Now()
					logf("tv: volumen del Chromecast %d%% (%s%s) → %d%%", int(*v.Level*100+0.5), v.ControlType, map[bool]string{true: ", silenciado", false: ""}[v.Muted], int(quiero*100+0.5))
					req++
					_ = cc.send("receiver-0", nsReceptor, map[string]interface{}{"type": "SET_VOLUME", "volume": map[string]interface{}{"level": quiero}, "requestId": req})
					if v.Muted {
						req++
						_ = cc.send("receiver-0", nsReceptor, map[string]interface{}{"type": "SET_VOLUME", "volume": map[string]interface{}{"muted": false}, "requestId": req})
					}
				}
			}
			var nuestra, otra string
			var transporte string
			for _, a := range st.Status.Applications {
				if a.AppID == rec.AppID {
					nuestra, transporte = a.AppID, a.TransportID
				} else if !a.IsIdleScreen && a.AppID != "E8C28D3C" {
					otra = a.DisplayName
				}
			}
			if nuestra != "" {
				if cargadoEn != transporte {
					_ = cc.send(transporte, nsConexion, map[string]string{"type": "CONNECT"})
					var payload interface{}
					if tv.Receptor == "urlcast" {
						payload = map[string]string{"type": "loc", "url": url}
					} else {
						payload = map[string]interface{}{"url": url, "force": true, "reload": false, "reload_time": 0}
					}
					if err := cc.send(transporte, rec.NS, payload); err != nil {
						return err
					}
					cargadoEn = transporte
				}
				lanzado = true
				forzar = false
				c.set("conectado", "Transmitiendo la pantalla de sala.")
				continue
			}
			cargadoEn = ""
			if otra != "" && !forzar {
				c.set("ocupado", "El TV está mostrando otra cosa ("+otra+"). Tocá Transmitir para volver a la sala.")
				continue
			}
			if (!lanzado || forzar || otra == "") && time.Since(ultimoLanzamiento) > 25*time.Second {
				lanzado = true
				forzar = false
				ultimoLanzamiento = time.Now()
				if err := lanzar(); err != nil {
					return err
				}
			}
		}
	}
}
