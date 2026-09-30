//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"fyne.io/systray"
)

func buscarNavegador() string {
	var cands []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA", "ProgramW6432"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands, filepath.Join(base, `Google\Chrome\Application\chrome.exe`))
	}
	for _, env := range []string{"ProgramFiles(x86)", "ProgramFiles", "ProgramW6432"} {
		base := os.Getenv(env)
		if base == "" {
			continue
		}
		cands = append(cands, filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`))
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// abrirVentana abre TeToca como una ventana propia (sin barra de direcciones).
func abrirVentana(url string) {
	if nav := buscarNavegador(); nav != "" {
		cmd := exec.Command(nav, "--app="+url, "--start-maximized", "--disable-features=TranslateUI", "--autoplay-policy=no-user-gesture-required")
		if err := cmd.Start(); err == nil {
			return
		}
	}
	abrirNavegador(url)
}

func abrirNavegador(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
}

func abrirCarpeta(p string) {
	_ = exec.Command("explorer", p).Start()
}

// prevenirSuspension evita que la PC central se duerma mientras TeToca está abierto.
func prevenirSuspension() {
	const esContinuous, esSystemRequired = 0x80000000, 0x00000001
	k := syscall.NewLazyDLL("kernel32.dll")
	p := k.NewProc("SetThreadExecutionState")
	if p.Find() == nil {
		_, _, _ = p.Call(uintptr(esContinuous | esSystemRequired))
	}
}

type itemBandeja struct {
	Texto string
	Fn    func()
}

func bandeja(titulo string, items []itemBandeja) {
	systray.Run(func() {
		if b, err := webFS.ReadFile("web/icono.ico"); err == nil {
			systray.SetIcon(b)
		}
		systray.SetTitle("TeToca")
		systray.SetTooltip(titulo + " " + Version)
		for _, it := range items {
			m := systray.AddMenuItem(it.Texto, "")
			fn := it.Fn
			go func() {
				for range m.ClickedCh {
					fn()
				}
			}()
		}
		systray.AddSeparator()
		salir := systray.AddMenuItem("Cerrar TeToca", "")
		go func() {
			<-salir.ClickedCh
			systray.Quit()
		}()
	}, func() {
		logf("cerrado desde la bandeja")
		os.Exit(0)
	})
}
