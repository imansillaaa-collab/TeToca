//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Versión para probar en Linux/Mac durante el desarrollo.

func abrirVentana(url string) { abrirNavegador(url) }

func abrirNavegador(url string) {
	fmt.Println("Abrir:", url)
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
	}
}

func abrirCarpeta(p string) { fmt.Println("Carpeta:", p) }

func prevenirSuspension() {}

type itemBandeja struct {
	Texto string
	Fn    func()
}

func bandeja(titulo string, items []itemBandeja) {
	fmt.Println(titulo, Version, "funcionando. Ctrl+C para salir.")
	select {}
}
