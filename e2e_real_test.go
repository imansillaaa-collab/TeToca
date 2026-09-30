//go:build e2e

package main

import "testing"

// Prueba contra GitHub de verdad: go test -tags e2e -run Real
func TestActualizacionRealGitHub(t *testing.T) {
	Version = "0.9.0"
	buscarActualizacion()
	u := updInfo()
	if !u.Disponible {
		t.Fatalf("no encontró la versión publicada: %+v", u)
	}
	b, err := descargarVerificado(urlExe, urlHash)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("versión %s descargada y verificada (%d bytes)", u.Version, len(b))
}
