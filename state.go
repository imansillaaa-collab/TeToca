package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Estados posibles de un turno.
const (
	EstPendiente  = "pendiente"   // todavía no lo llamaron
	EstMesa       = "mesa"        // llamado a mesa de entradas, lo están atendiendo
	EstEsperaCaja = "espera_caja" // mesa lo pasó a caja, espera que lo llamen
	EstCaja       = "caja"        // llamado a caja, lo están cobrando
	EstTerminado  = "terminado"
	EstAusente    = "ausente"
	EstCancelado  = "cancelado"
)

type Turno struct {
	ID        string `json:"id"`
	Hora      string `json:"hora"`
	Precarga  string `json:"precarga"`
	Nombre    string `json:"nombre"`
	Tramite   string `json:"tramite"`
	Dominio   string `json:"dominio"`
	Orden     int    `json:"orden"`
	Estado    string `json:"estado"`
	AusenteEn string `json:"ausenteEn,omitempty"` // "mesa" o "caja"
	PC        string `json:"pc,omitempty"`        // PC que lo tiene tomado
	Manual    bool   `json:"manual,omitempty"`

	LlamadoMesa time.Time `json:"llamadoMesa,omitempty"`
	PasoCaja    time.Time `json:"pasoCaja,omitempty"`
	LlamadoCaja time.Time `json:"llamadoCaja,omitempty"`
	Termino     time.Time `json:"termino,omitempty"`
	AusenteA    time.Time `json:"ausenteA,omitempty"`
}

// Llamado es cada vez que un turno aparece en el TV.
type Llamado struct {
	Seq      int       `json:"seq"`
	TurnoID  string    `json:"turnoId"`
	Dest     string    `json:"dest"`     // "mesa" o "caja"
	Destino  string    `json:"destino"`  // texto para el TV, ej "MESA DE ENTRADAS · BOX 2"
	Etiqueta string    `json:"etiqueta"` // etiqueta corta, ej "MESA" o "BOX 2"
	PC       string    `json:"pc"`
	Hora     time.Time `json:"hora"`
}

type Estado struct {
	Fecha      string    `json:"fecha"`      // fecha de los turnos (dd/mm/aaaa)
	CargadoDia string    `json:"cargadoDia"` // día en que se cargó el archivo
	Archivo    string    `json:"archivo"`
	Turnos     []*Turno  `json:"turnos"`
	Llamados   []Llamado `json:"llamados"`
	Seq        int       `json:"seq"`
	Version    int64     `json:"version"`
}

type PCConf struct {
	Rol        string    `json:"rol"` // "mesa" o "caja"
	Box        string    `json:"box"`
	MostrarBox bool      `json:"mostrarBox"`
	Visto      time.Time `json:"visto"`
}

type TVConf struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Auto     bool   `json:"auto"`
	Receptor string `json:"receptor"` // "dashcast" o "urlcast"
}

type Config struct {
	Sonido    string             `json:"sonido"`
	Tema      string             `json:"tema"`
	Organismo string             `json:"organismo"` // texto grande al lado del logo, ej "DNRPA"
	Oficina   string             `json:"oficina"`   // ej "Registro Automotor"
	TieneLogo bool               `json:"tieneLogo"`
	PCs       map[string]*PCConf `json:"pcs"`
	TV        TVConf             `json:"tv"`
	Ultimos   int                `json:"ultimos"`
	Ausentes  int                `json:"ausentes"`
	TemaTV    string             `json:"temaTV"` // noche, celeste, albiceleste, sol, claro, contraste, verde
}

var temasTV = map[string]bool{"noche": true, "celeste": true, "albiceleste": true, "sol": true, "claro": true, "contraste": true, "verde": true}

type Store struct {
	mu     sync.Mutex
	dir    string
	E      *Estado
	C      *Config
	subs   map[chan struct{}]struct{}
	subsMu sync.Mutex
}

var errUsuario = errors.New("")

type UserErr struct{ Msg string }

func (e UserErr) Error() string { return e.Msg }

func hoy() string { return time.Now().Format("02/01/2006") }

func NewStore(dir string) *Store {
	s := &Store{dir: dir, subs: map[chan struct{}]struct{}{}}
	s.E = &Estado{}
	s.C = &Config{Sonido: "dingdong", Tema: "claro", Oficina: "Registro Automotor", PCs: map[string]*PCConf{},
		Ultimos: 5, Ausentes: 3, TV: TVConf{Auto: true, Receptor: "dashcast"}}
	leerJSON(filepath.Join(dir, "turnos.json"), s.E)
	leerJSON(filepath.Join(dir, "config.json"), s.C)
	if s.C.PCs == nil {
		s.C.PCs = map[string]*PCConf{}
	}
	if s.C.Ultimos == 0 {
		s.C.Ultimos = 5
	}
	if s.C.Ausentes == 0 {
		s.C.Ausentes = 3
	}
	if !temasTV[s.C.TemaTV] {
		s.C.TemaTV = "noche"
	}
	if s.C.TV.Receptor == "" {
		s.C.TV.Receptor = "dashcast"
	}
	_, err := os.Stat(filepath.Join(dir, "logo.png"))
	s.C.TieneLogo = err == nil
	return s
}

func leerJSON(p string, v interface{}) {
	b, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(b, v)
	}
}

func escribirJSON(p string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// guardar persiste y avisa a todas las pantallas. Llamar con mu tomado.
func (s *Store) guardar() {
	s.E.Version = time.Now().UnixNano()
	if err := escribirJSON(filepath.Join(s.dir, "turnos.json"), s.E); err != nil {
		logf("error guardando turnos: %v", err)
	}
	if err := escribirJSON(filepath.Join(s.dir, "config.json"), s.C); err != nil {
		logf("error guardando config: %v", err)
	}
	s.avisar()
}

func (s *Store) avisar() {
	s.subsMu.Lock()
	for c := range s.subs {
		select {
		case c <- struct{}{}:
		default:
		}
	}
	s.subsMu.Unlock()
}

func (s *Store) Suscribir() chan struct{} {
	c := make(chan struct{}, 1)
	s.subsMu.Lock()
	s.subs[c] = struct{}{}
	s.subsMu.Unlock()
	return c
}

func (s *Store) Desuscribir(c chan struct{}) {
	s.subsMu.Lock()
	delete(s.subs, c)
	s.subsMu.Unlock()
}

func (s *Store) get(id string) *Turno {
	for _, t := range s.E.Turnos {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// actualDe devuelve el turno que la PC tiene en atención (en mesa o en caja).
func (s *Store) actualDe(pc string) *Turno {
	for _, t := range s.E.Turnos {
		if (t.Estado == EstMesa || t.Estado == EstCaja) && t.PC == pc {
			return t
		}
	}
	return nil
}

func (s *Store) pcConf(pc string) *PCConf {
	c := s.C.PCs[pc]
	if c == nil {
		c = &PCConf{Rol: ""}
		s.C.PCs[pc] = c
	}
	return c
}

func (s *Store) registrarLlamado(t *Turno, dest, pc string) {
	s.E.Seq++
	pcc := s.pcConf(pc)
	destino, etiqueta := "MESA DE ENTRADAS", "MESA"
	if dest == "caja" {
		destino, etiqueta = "CAJA", "CAJA"
	}
	if pcc.MostrarBox && strings.TrimSpace(pcc.Box) != "" {
		b := strings.ToUpper(strings.TrimSpace(pcc.Box))
		destino += " · " + b
		etiqueta = b
	}
	s.E.Llamados = append(s.E.Llamados, Llamado{Seq: s.E.Seq, TurnoID: t.ID, Dest: dest, Destino: destino,
		Etiqueta: etiqueta, PC: pc, Hora: time.Now()})
	if len(s.E.Llamados) > 300 {
		s.E.Llamados = s.E.Llamados[len(s.E.Llamados)-300:]
	}
}

// Accion ejecuta una acción de mesa o caja. Devuelve un mensaje de error entendible si no se puede.
func (s *Store) Accion(accion, pc, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pc == "" {
		return UserErr{"Esta PC no tiene nombre asignado."}
	}
	actual := s.actualDe(pc)
	ocupado := func() error {
		if actual != nil {
			return UserErr{"Primero indicá qué pasó con " + actual.Nombre + "."}
		}
		return nil
	}
	now := time.Now()
	switch accion {
	case "mesa_siguiente":
		if err := ocupado(); err != nil {
			return err
		}
		for _, t := range s.E.Turnos {
			if t.Estado == EstPendiente {
				t.Estado, t.PC, t.LlamadoMesa = EstMesa, pc, now
				s.registrarLlamado(t, "mesa", pc)
				s.guardar()
				return nil
			}
		}
		return UserErr{"No quedan turnos pendientes."}
	case "mesa_llamar":
		if err := ocupado(); err != nil {
			return err
		}
		t := s.get(id)
		if t == nil {
			return UserErr{"No encontré ese turno."}
		}
		if !(t.Estado == EstPendiente || t.Estado == EstTerminado || (t.Estado == EstAusente && t.AusenteEn == "mesa")) {
			return UserErr{t.Nombre + " ya está siendo atendido o está en caja."}
		}
		t.Estado, t.PC, t.LlamadoMesa, t.AusenteEn = EstMesa, pc, now, ""
		s.registrarLlamado(t, "mesa", pc)
	case "caja_siguiente":
		if err := ocupado(); err != nil {
			return err
		}
		var sig *Turno
		for _, t := range s.E.Turnos {
			if t.Estado == EstEsperaCaja && (sig == nil || t.PasoCaja.Before(sig.PasoCaja)) {
				sig = t
			}
		}
		if sig == nil {
			return UserErr{"No hay nadie esperando en caja."}
		}
		sig.Estado, sig.PC, sig.LlamadoCaja = EstCaja, pc, now
		s.registrarLlamado(sig, "caja", pc)
	case "caja_llamar":
		if err := ocupado(); err != nil {
			return err
		}
		t := s.get(id)
		if t == nil {
			return UserErr{"No encontré ese turno."}
		}
		if !(t.Estado == EstEsperaCaja || (t.Estado == EstAusente && t.AusenteEn == "caja")) {
			return UserErr{t.Nombre + " no está en la fila de caja."}
		}
		t.Estado, t.PC, t.LlamadoCaja, t.AusenteEn = EstCaja, pc, now, ""
		s.registrarLlamado(t, "caja", pc)
	case "rellamar":
		if actual == nil {
			return UserErr{"No tenés a nadie llamado."}
		}
		s.registrarLlamado(actual, actual.Estado, pc)
	case "pasar_caja":
		if actual == nil || actual.Estado != EstMesa {
			return UserErr{"No tenés a nadie en mesa."}
		}
		actual.Estado, actual.PC, actual.PasoCaja = EstEsperaCaja, "", now
	case "terminar":
		if actual == nil {
			return UserErr{"No tenés a nadie llamado."}
		}
		actual.Estado, actual.PC, actual.Termino = EstTerminado, "", now
	case "ausente":
		if actual == nil {
			return UserErr{"No tenés a nadie llamado."}
		}
		actual.AusenteEn = actual.Estado // mesa o caja
		actual.Estado, actual.PC, actual.AusenteA = EstAusente, "", now
	case "deshacer_llamado":
		// Por si se llamó a alguien por error: vuelve a donde estaba.
		if actual == nil {
			return UserErr{"No tenés a nadie llamado."}
		}
		if actual.Estado == EstMesa {
			actual.Estado = EstPendiente
		} else {
			actual.Estado = EstEsperaCaja
		}
		actual.PC = ""
		// sacar sus llamados del TV
		out := s.E.Llamados[:0]
		for _, l := range s.E.Llamados {
			if !(l.TurnoID == actual.ID && l.Seq == s.lastSeqOf(actual.ID)) {
				out = append(out, l)
			}
		}
		s.E.Llamados = out
	case "volver_pendiente":
		t := s.get(id)
		if t == nil {
			return UserErr{"No encontré ese turno."}
		}
		t.Estado, t.PC, t.AusenteEn = EstPendiente, "", ""
	default:
		return UserErr{"Acción desconocida."}
	}
	s.guardar()
	return nil
}

func (s *Store) lastSeqOf(id string) int {
	m := 0
	for _, l := range s.E.Llamados {
		if l.TurnoID == id && l.Seq > m {
			m = l.Seq
		}
	}
	return m
}

func (s *Store) AgregarManual(nombre, precarga string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	t := &Turno{ID: "m" + now.Format("150405.000"), Hora: now.Format("15:04"), Precarga: strings.TrimSpace(precarga),
		Nombre: strings.ToUpper(strings.TrimSpace(nombre)), Tramite: "Agregado a mano", Estado: EstPendiente, Manual: true}
	s.E.Turnos = append(s.E.Turnos, t)
	ordenar(s.E.Turnos)
	if s.E.Fecha == "" {
		s.E.Fecha = hoy()
	}
	s.guardar()
}

func ordenar(ts []*Turno) {
	sort.SliceStable(ts, func(i, j int) bool {
		hi, hj := ts[i].Hora, ts[j].Hora
		if hi == "" {
			hi = "99"
		}
		if hj == "" {
			hj = "99"
		}
		if hi != hj {
			return hi < hj
		}
		return ts[i].Orden < ts[j].Orden
	})
	for i, t := range ts {
		t.Orden = i
	}
}

// Cargar reemplaza la lista del día. Si se vuelve a cargar el mismo día,
// conserva lo que ya pasó con cada persona.
func (s *Store) Cargar(nuevos []*Turno, fechaArchivo, archivo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := func(t *Turno) string { return t.Precarga + "|" + t.Nombre }
	if s.E.CargadoDia == hoy() && len(s.E.Turnos) > 0 {
		prev := map[string]*Turno{}
		for _, t := range s.E.Turnos {
			prev[key(t)] = t
		}
		ids := map[string]string{}
		vistos := map[string]bool{}
		for _, n := range nuevos {
			if p, ok := prev[key(n)]; ok {
				vistos[key(n)] = true
				ids[p.ID] = n.ID
				if n.Estado != EstCancelado {
					id := n.ID
					orden := n.Orden
					*n = *p
					n.ID, n.Orden = id, orden
				}
			}
		}
		for _, p := range s.E.Turnos {
			if p.Manual && !vistos[key(p)] {
				nuevos = append(nuevos, p)
				ids[p.ID] = p.ID
			}
		}
		ll := s.E.Llamados[:0]
		for _, l := range s.E.Llamados {
			if nid, ok := ids[l.TurnoID]; ok {
				l.TurnoID = nid
				ll = append(ll, l)
			}
		}
		s.E.Llamados = ll
	} else {
		s.E.Llamados = nil
	}
	ordenar(nuevos)
	s.E.Turnos = nuevos
	s.E.Fecha = fechaArchivo
	if s.E.Fecha == "" {
		s.E.Fecha = hoy()
	}
	s.E.CargadoDia = hoy()
	s.E.Archivo = archivo
	s.guardar()
}

func (s *Store) SetPC(pc string, f func(c *PCConf)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.pcConf(pc))
	s.guardar()
}

func (s *Store) SetConfig(f func(c *Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.C)
	s.guardar()
}

func (s *Store) Visto(pc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pcConf(pc).Visto = time.Now()
}

// Vista es lo que reciben las pantallas.
type Vista struct {
	Version string    `json:"version"`
	Estado  *Estado   `json:"estado"`
	Config  *Config   `json:"config"`
	TV      TVStatus  `json:"tv"`
	Update  UpdInfo   `json:"update"`
	Hoy     string    `json:"hoy"`
	Equipos int       `json:"equipos"`
	Ahora   time.Time `json:"ahora"`
	// Prueba de sonido: la PC pide, el TV suena y cuenta cómo le fue.
	PruebaSonido int            `json:"pruebaSonido"`
	TVSonido     *ReporteSonido `json:"tvSonido,omitempty"`
}

type ReporteSonido struct {
	Hora    time.Time `json:"hora"`
	OK      bool      `json:"ok"`
	Prueba  bool      `json:"prueba"`
	Detalle string    `json:"detalle"`
}

var (
	pruebaSonido int
	tvSonido     *ReporteSonido
)

func (s *Store) Snapshot() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	eq := 0
	for _, p := range s.C.PCs {
		if time.Since(p.Visto) < 90*time.Second {
			eq++
		}
	}
	v := Vista{Version: Version, Estado: s.E, Config: s.C, TV: tvStatus(), Update: updInfo(), Hoy: hoy(), Equipos: eq, Ahora: time.Now(),
		PruebaSonido: pruebaSonido, TVSonido: tvSonido}
	b, _ := json.Marshal(v)
	return b
}
