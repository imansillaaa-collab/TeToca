//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
)

// En Linux/Mac (solo para desarrollo) no hay instalación.

// Para probar la instalación en Linux: TETOCA_CARPETA=/tmp/algo
func carpetaInstalacion() string          { return os.Getenv("TETOCA_CARPETA") }
func sinConsola(cmd *exec.Cmd)            {}
func inicioActivo(exe string) bool        { return false }
func ponerInicio(exe string) error        { return nil }
func quitarInicio() error                 { return nil }
func reglaRed() bool                      { return false }
func permisoRed(exe string) error         { return nil }
func quitarRed(exe string) error          { return nil }
func elevar(exe, args string) error       { return errors.New("no disponible") }
func crearAccesoDirecto(exe string) error { return nil }
func quitarAccesoDirecto()                {}
