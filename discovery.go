package main

import (
	"encoding/json"
	"net"
	"os"
	"time"
)

const PuertoDescubrir = 8766

type Central struct {
	Host    string `json:"host"`
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	Version string `json:"version"`
}

// responderDescubrimiento contesta a los puestos que buscan la PC central.
func responderDescubrimiento() {
	pc, err := net.ListenPacket("udp4", ":"+itoa(PuertoDescubrir))
	if err != nil {
		logf("no pude escuchar descubrimiento: %v", err)
		return
	}
	h, _ := os.Hostname()
	buf := make([]byte, 512)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}
		if string(buf[:n]) != "TETOCA?" {
			continue
		}
		b, _ := json.Marshal(Central{Host: h, Port: Puerto, Version: Version})
		_, _ = pc.WriteTo(b, addr)
	}
}

func direccionesBroadcast() []string {
	out := []string{"255.255.255.255"}
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			n, ok := a.(*net.IPNet)
			if !ok || n.IP.To4() == nil {
				continue
			}
			ip := n.IP.To4()
			m := n.Mask
			if len(m) == 16 {
				m = m[12:]
			}
			b := make(net.IP, 4)
			for k := 0; k < 4; k++ {
				b[k] = ip[k] | ^m[k]
			}
			out = append(out, b.String())
		}
	}
	return out
}

// BuscarCentrales grita en la red "¿hay un TeToca?" y junta las respuestas.
func BuscarCentrales(espera time.Duration) []Central {
	pc, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil
	}
	defer pc.Close()
	for _, d := range direccionesBroadcast() {
		addr := &net.UDPAddr{IP: net.ParseIP(d), Port: PuertoDescubrir}
		_, _ = pc.WriteTo([]byte("TETOCA?"), addr)
	}
	vistos := map[string]bool{}
	var out []Central
	fin := time.Now().Add(espera)
	buf := make([]byte, 1024)
	for time.Now().Before(fin) {
		_ = pc.SetReadDeadline(fin)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			break
		}
		var c Central
		if json.Unmarshal(buf[:n], &c) != nil {
			continue
		}
		c.IP = addr.(*net.UDPAddr).IP.String()
		if !vistos[c.IP] {
			vistos[c.IP] = true
			out = append(out, c)
		}
	}
	return out
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
