package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// El sistema de turnos a veces exporta el CSV con los acentos rotos
// (ej. "RODRжGUEZ"). Estos son los reemplazos que vimos en la práctica.
var arreglos = map[rune]rune{'р': 'Ó', 'ж': 'Í', 'щ': 'Ú', 'Е': 'Á', 'Ѕ': 'Ñ', 'Р': 'É', 'е': 'Ñ', 'Ў': 'Ü'}

func arreglarTexto(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 0x0400 && r <= 0x04FF {
			if n, ok := arreglos[r]; ok {
				return n
			}
		}
		if r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
}

func decodificar(b []byte) string {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(b) {
		return arreglarTexto(string(b))
	}
	// Windows-1252 / Latin-1
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return arreglarTexto(string(r))
}

func norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	rep := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n")
	return rep.Replace(s)
}

var espacios = regexp.MustCompile(`\s+`)

func filasHTML(txt string) ([][]string, error) {
	doc, err := html.Parse(strings.NewReader(txt))
	if err != nil {
		return nil, err
	}
	var filas [][]string
	var walk func(n *html.Node)
	texto := func(n *html.Node) string {
		var b strings.Builder
		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.TextNode {
				b.WriteString(n.Data)
				b.WriteString(" ")
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(n)
		return strings.TrimSpace(espacios.ReplaceAllString(b.String(), " "))
	}
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var fila []string
			var celdas func(*html.Node)
			celdas = func(m *html.Node) {
				for c := m.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
						fila = append(fila, texto(c))
						span := 1
						for _, a := range c.Attr {
							if a.Key == "colspan" {
								fmt.Sscanf(a.Val, "%d", &span)
							}
						}
						for i := 1; i < span; i++ {
							fila = append(fila, "")
						}
					} else if c.Type == html.ElementNode {
						celdas(c) // por si hay etiquetas raras en el medio (<b>)
					}
				}
			}
			celdas(n)
			filas = append(filas, fila)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return filas, nil
}

func filasCSV(txt string) ([][]string, error) {
	primera := strings.SplitN(txt, "\n", 2)[0]
	sep := ';'
	if strings.Count(primera, ",") > strings.Count(primera, ";") {
		sep = ','
	}
	if strings.Count(primera, "\t") > strings.Count(primera, string(sep)) {
		sep = '\t'
	}
	r := csv.NewReader(strings.NewReader(txt))
	r.Comma = sep
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	return r.ReadAll()
}

var tramCodigo = regexp.MustCompile(`^\d+\s*-\s*`)

// Importar lee el archivo del sistema de turnos (.xls que en realidad es HTML, o .csv).
func Importar(data []byte) ([]*Turno, string, error) {
	if len(data) > 4 && data[0] == 'P' && data[1] == 'K' {
		return nil, "", UserErr{"Ese archivo es un Excel moderno (.xlsx). Descargalo como .xls o .csv desde el sistema de turnos."}
	}
	if len(data) > 4 && data[0] == 0xD0 && data[1] == 0xCF {
		return nil, "", UserErr{"Ese archivo es un Excel antiguo. Descargalo como .csv desde el sistema de turnos."}
	}
	txt := decodificar(data)
	var filas [][]string
	var err error
	cab := txt
	if len(cab) > 5000 {
		cab = cab[:5000]
	}
	if regexp.MustCompile(`(?i)<t[rd][\s>]`).MatchString(cab) {
		filas, err = filasHTML(txt)
	} else {
		filas, err = filasCSV(txt)
	}
	if err != nil {
		return nil, "", UserErr{"No pude leer el archivo: " + err.Error()}
	}
	hi := -1
	for i, f := range filas {
		tieneP, tieneS := false, false
		for _, c := range f {
			n := norm(c)
			if strings.Contains(n, "precarga") {
				tieneP = true
			}
			if strings.Contains(n, "solicitante") {
				tieneS = true
			}
		}
		if tieneP && tieneS {
			hi = i
			break
		}
	}
	if hi < 0 {
		return nil, "", UserErr{`No encontré las columnas "Solicitante" y "Nro. Precarga". ¿Es el archivo de turnos del día?`}
	}
	h := filas[hi]
	col := func(keys ...string) int {
		for i, c := range h {
			n := norm(c)
			for _, k := range keys {
				if strings.Contains(n, k) {
					return i
				}
			}
		}
		return -1
	}
	cNom, cPre, cHora, cFecha := col("solicitante"), col("precarga"), col("horario", "hora"), col("fecha de turno")
	cTram, cDom, cCanc := col("tramite"), col("dominio"), col("cancelacion")
	get := func(f []string, i int) string {
		if i < 0 || i >= len(f) {
			return ""
		}
		return strings.TrimSpace(f[i])
	}
	var out []*Turno
	fechas := map[string]int{}
	for i, f := range filas[hi+1:] {
		nom := espacios.ReplaceAllString(get(f, cNom), " ")
		if nom == "" {
			continue
		}
		fe := get(f, cFecha)
		if fe != "" {
			fechas[fe]++
		}
		hora := get(f, cHora)
		if len(hora) > 5 {
			hora = hora[:5]
		}
		t := &Turno{
			ID:       fmt.Sprintf("t%d_%s", i, get(f, cPre)),
			Hora:     hora,
			Precarga: get(f, cPre),
			Nombre:   nom,
			Tramite:  tramCodigo.ReplaceAllString(get(f, cTram), ""),
			Dominio:  get(f, cDom),
			Orden:    i,
			Estado:   EstPendiente,
		}
		if c := get(f, cCanc); c != "" && c != "-" {
			t.Estado = EstCancelado
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, "", errors.New("el archivo no tiene turnos")
	}
	fecha, max := "", 0
	keys := make([]string, 0, len(fechas))
	for k := range fechas {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if fechas[k] > max {
			fecha, max = k, fechas[k]
		}
	}
	return out, fecha, nil
}
