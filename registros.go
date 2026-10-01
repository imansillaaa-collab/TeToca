package main

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"time"
	"unicode"
)

var sinTildes = strings.NewReplacer("Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
	"À", "A", "È", "E", "Ì", "I", "Ò", "O", "Ù", "U", "Â", "A", "Ê", "E", "Ô", "O", "Ç", "C")

// Un trámite de una persona. Con dos registros (o varios turnos de la misma
// persona) se juntan en una sola fila para llamarla una sola vez.
type Tramite struct {
	Registro string `json:"registro"`
	Hora     string `json:"hora"`
	Precarga string `json:"precarga"`
	Tramite  string `json:"tramite"`
	Dominio  string `json:"dominio"`
}

// Fuente: un archivo cargado hoy para un registro.
type Fuente struct {
	Archivo  string    `json:"archivo"`
	Cargado  time.Time `json:"cargado"`
	Cantidad int       `json:"cantidad"`
	Turnos   []*Turno  `json:"turnos"` // tal como vinieron en el archivo
}

// normNombre deja el nombre comparable: sin tildes, sin comas ni puntos y con las
// palabras ordenadas ("PÉREZ, JUAN" = "Perez Juan" = "JUAN PEREZ").
func normNombre(s string) string {
	var b strings.Builder
	for _, r := range sinTildes.Replace(strings.ToUpper(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	ps := strings.Fields(b.String())
	sort.Strings(ps)
	return strings.Join(ps, " ")
}

func idDe(clave string) string {
	h := sha1.Sum([]byte(clave))
	return "t" + hex.EncodeToString(h[:6])
}

func (s *Store) limpiarSiOtroDia() {
	if s.E.CargadoDia != "" && s.E.CargadoDia != hoy() {
		logf("cambió el día (%s → %s): se borra la lista vieja", s.E.CargadoDia, hoy())
		seq := s.E.Seq
		s.E = &Estado{Seq: seq}
		s.guardar()
	}
}

// VigilarDia borra la lista vieja apenas cambia el día, aunque nadie cargue nada.
func (s *Store) VigilarDia() {
	for {
		time.Sleep(time.Minute)
		s.mu.Lock()
		s.limpiarSiOtroDia()
		s.mu.Unlock()
	}
}

// Cargar guarda el archivo de un registro ("1" o "2") y rearma la lista del día.
// Si se vuelve a cargar el mismo día, conserva lo que ya pasó con cada persona.
func (s *Store) Cargar(nuevos []*Turno, fechaArchivo, archivo, registro string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if registro == "" {
		registro = "1"
	}
	if s.E.CargadoDia != hoy() {
		seq := s.E.Seq
		s.E = &Estado{Seq: seq}
	}
	if s.E.Fuentes == nil {
		s.E.Fuentes = map[string]*Fuente{}
	}
	s.E.Fuentes[registro] = &Fuente{Archivo: archivo, Cargado: time.Now(), Cantidad: len(nuevos), Turnos: nuevos}
	s.rearmar(nil)
	s.E.Fecha = fechaArchivo
	if s.E.Fecha == "" {
		s.E.Fecha = hoy()
	}
	s.E.CargadoDia = hoy()
	s.E.Archivo = archivo
	s.guardar()
}

// Separar divide una fila que juntó a dos personas distintas con el mismo nombre.
func (s *Store) Separar(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.get(id)
	if t == nil {
		return UserErr{"No encontré ese turno."}
	}
	if len(t.Tramites) < 2 {
		return UserErr{"Ese turno tiene un solo trámite."}
	}
	if s.E.Separados == nil {
		s.E.Separados = map[string]bool{}
	}
	nk := normNombre(t.Nombre)
	s.E.Separados[nk+"@"+t.Hora] = true
	// lo que estaba pasando con la fila (por ejemplo, en mesa) sigue con su primer trámite
	pri := t.Tramites[0]
	s.rearmar(map[string]*Turno{claveSeparada(pri.Registro, pri.Precarga, nk): t})
	s.guardar()
	return nil
}

func claveSeparada(reg, pre, nk string) string { return "p:" + reg + "|" + pre + "|" + nk }

// rearmar arma la lista del día juntando los archivos de todos los registros.
// La misma persona (mismo nombre) queda en una sola fila con todos sus trámites.
func (s *Store) rearmar(traspaso map[string]*Turno) {
	type grupo struct {
		clave string
		items []*Turno
		regs  []string
	}
	var orden []string
	grupos := map[string]*grupo{}
	var cancelados []*Turno
	regs := make([]string, 0, len(s.E.Fuentes))
	for r := range s.E.Fuentes {
		regs = append(regs, r)
	}
	sort.Strings(regs)
	for _, r := range regs {
		for _, t := range s.E.Fuentes[r].Turnos {
			nk := normNombre(t.Nombre)
			if t.Estado == EstCancelado {
				c := *t
				c.Clave = "c:" + r + "|" + t.Precarga + "|" + nk
				c.ID, c.Registros = idDe(c.Clave), []string{r}
				c.Tramites = nil
				cancelados = append(cancelados, &c)
				continue
			}
			// se juntan solo si coinciden nombre Y horario: un gestor con turnos en
			// horarios distintos sigue apareciendo una vez por cada horario
			gk := nk + "@" + t.Hora
			clave := "n:" + gk
			if s.E.Separados[gk] || nk == "" {
				clave = claveSeparada(r, t.Precarga, nk)
			}
			g := grupos[clave]
			if g == nil {
				g = &grupo{clave: clave}
				grupos[clave] = g
				orden = append(orden, clave)
			}
			g.items = append(g.items, t)
			g.regs = append(g.regs, r)
		}
	}

	// estado anterior, por clave (las listas viejas no tenían clave: se calcula)
	prev := map[string]*Turno{}
	for _, t := range s.E.Turnos {
		if t.Manual {
			continue
		}
		k := t.Clave
		if k == "" {
			k = "n:" + normNombre(t.Nombre) + "@" + t.Hora
		}
		if p, ok := prev[k]; !ok || (p.Estado == EstPendiente && t.Estado != EstPendiente) {
			prev[k] = t
		}
	}
	ids := map[string]string{}
	usados := map[*Turno]bool{}
	var lista []*Turno
	for _, clave := range orden {
		g := grupos[clave]
		var trs []Tramite
		vistos := map[string]bool{}
		for i, it := range g.items {
			trs = append(trs, Tramite{Registro: g.regs[i], Hora: it.Hora, Precarga: it.Precarga, Tramite: it.Tramite, Dominio: it.Dominio})
			vistos[g.regs[i]] = true
		}
		sort.SliceStable(trs, func(i, j int) bool { return horaOrden(trs[i].Hora) < horaOrden(trs[j].Hora) })
		var rs []string
		for r := range vistos {
			rs = append(rs, r)
		}
		sort.Strings(rs)
		pri := trs[0]
		n := &Turno{ID: idDe(clave), Clave: clave, Hora: pri.Hora, Precarga: pri.Precarga, Nombre: g.items[0].Nombre,
			Tramite: pri.Tramite, Dominio: pri.Dominio, Orden: g.items[0].Orden, Estado: EstPendiente, Registros: rs}
		if len(trs) > 1 {
			n.Tramites = trs
		} else {
			n.Tramites = nil
		}
		p := traspaso[clave]
		if p == nil {
			p = prev[clave]
		}
		if p != nil && !usados[p] {
			usados[p] = true
			copiarEstado(n, p)
			ids[p.ID] = n.ID
		}
		lista = append(lista, n)
	}
	// nadie que esté siendo atendido puede desaparecer por recargar un archivo
	for _, p := range s.E.Turnos {
		if p.Manual {
			lista = append(lista, p)
			ids[p.ID] = p.ID
			continue
		}
		if !usados[p] && (p.Estado == EstMesa || p.Estado == EstCaja || p.Estado == EstEsperaCaja) {
			lista = append(lista, p)
			ids[p.ID] = p.ID
		}
	}
	lista = append(lista, cancelados...)
	ordenar(lista)
	ll := s.E.Llamados[:0]
	for _, l := range s.E.Llamados {
		if nid, ok := ids[l.TurnoID]; ok {
			l.TurnoID = nid
			ll = append(ll, l)
		}
	}
	s.E.Llamados = ll
	s.E.Turnos = lista
}

func horaOrden(h string) string {
	if h == "" {
		return "99"
	}
	return h
}

func copiarEstado(n, p *Turno) {
	n.Estado, n.AusenteEn, n.PC = p.Estado, p.AusenteEn, p.PC
	n.LlamadoMesa, n.PasoCaja, n.LlamadoCaja, n.Termino, n.AusenteA = p.LlamadoMesa, p.PasoCaja, p.LlamadoCaja, p.Termino, p.AusenteA
}
