//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const claveInicio = `Software\Microsoft\Windows\CurrentVersion\Run`
const nombreRegla = "TeToca"

// carpetaInstalacion: C:\TeToca. Si ahí no se puede escribir (PC muy bloqueada),
// se usa la carpeta de programas del usuario.
func carpetaInstalacion() string {
	unidad := os.Getenv("SystemDrive")
	if unidad == "" {
		unidad = "C:"
	}
	d := unidad + `\TeToca`
	if st, err := os.Stat(d); err == nil && st.IsDir() {
		if escribible(d) {
			return d
		}
	} else if err := os.Mkdir(d, 0755); err == nil {
		_ = os.Remove(d) // solo era una prueba: se crea de verdad al instalar
		return d
	}
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return filepath.Join(base, "Programs", "TeToca")
	}
	return d
}

func sinConsola(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}

func inicioActivo(exe string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, claveInicio, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetStringValue("TeToca")
	if err != nil {
		return false
	}
	return mismaRuta(strings.Trim(v, `"`), exe)
}

func ponerInicio(exe string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, claveInicio, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("TeToca", `"`+exe+`"`)
}

func quitarInicio() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, claveInicio, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue("TeToca")
}

func netsh(args ...string) (string, error) {
	cmd := exec.Command("netsh", append([]string{"advfirewall", "firewall"}, args...)...)
	sinConsola(cmd)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// reglaRed: ¿existe nuestra regla del firewall? (se puede consultar sin ser administrador)
func reglaRed() bool {
	_, err := netsh("show", "rule", "name="+nombreRegla)
	return err == nil
}

// permisoRed corre como administrador: borra las reglas que Windows haya creado
// para TeToca (por ejemplo una que bloquea "Redes públicas") y deja una sola que
// permite todo en todas las redes.
func permisoRed(exe string) error {
	_, _ = netsh("delete", "rule", "name=all", "program="+exe)
	_, _ = netsh("delete", "rule", "name="+nombreRegla)
	out, err := netsh("add", "rule", "name="+nombreRegla, "dir=in", "action=allow", "program="+exe, "enable=yes", "profile=any")
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(out))
	}
	return nil
}

func quitarRed(exe string) error {
	_, _ = netsh("delete", "rule", "name=all", "program="+exe)
	_, err := netsh("delete", "rule", "name="+nombreRegla)
	return err
}

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     windows.Handle
}

var errCancelado = errors.New("se canceló el permiso de administrador")

// elevar abre exe con permisos de administrador (Windows muestra su pregunta de
// "¿Querés permitir que esta app haga cambios?") y espera a que termine.
func elevar(exe, args string) error {
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(args)
	dir, _ := windows.UTF16PtrFromString(filepath.Dir(exe))
	info := shellExecuteInfo{fMask: 0x40 | 0x100, lpVerb: verb, lpFile: file, lpParameters: params, lpDirectory: dir} // NOCLOSEPROCESS | NOASYNC
	info.cbSize = uint32(unsafe.Sizeof(info))
	proc := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")
	r, _, e := proc.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		if e == windows.ERROR_CANCELLED {
			return errCancelado
		}
		return e
	}
	if info.hProcess == 0 {
		return nil
	}
	defer windows.CloseHandle(info.hProcess)
	ev, _ := windows.WaitForSingleObject(info.hProcess, uint32((2 * time.Minute).Milliseconds()))
	if ev != windows.WAIT_OBJECT_0 {
		return errors.New("el permiso de red tardó demasiado")
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.hProcess, &code); err == nil && code != 0 {
		return fmt.Errorf("no se pudo cambiar el firewall (código %d)", code)
	}
	return nil
}

// Acceso directo en el Escritorio, para volver a abrir TeToca si alguien lo cierra.
// Se crea con el componente de Windows para accesos directos (IShellLink), igual
// que lo hacen los instaladores comunes.
var (
	clsidShellLink  = windows.GUID{Data1: 0x00021401, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIShellLinkW  = windows.GUID{Data1: 0x000214F9, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIPersistFile = windows.GUID{Data1: 0x0000010B, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	ole32           = windows.NewLazySystemDLL("ole32.dll")
	coCreate        = ole32.NewProc("CoCreateInstance")
)

func metodo(obj uintptr, n int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(n)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func rutaAccesoDirecto() (string, error) {
	esc, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, 0)
	if err != nil {
		return "", err
	}
	return filepath.Join(esc, "TeToca.lnk"), nil
}

func crearAccesoDirecto(exe string) error {
	lnk, err := rutaAccesoDirecto()
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err == nil {
		defer windows.CoUninitialize()
	}
	var sl uintptr
	if r, _, _ := coCreate.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, 1 /*CLSCTX_INPROC_SERVER*/, uintptr(unsafe.Pointer(&iidIShellLinkW)), uintptr(unsafe.Pointer(&sl))); r != 0 {
		return fmt.Errorf("CoCreateInstance 0x%x", r)
	}
	defer metodo(sl, 2)  // Release
	var vivos [][]uint16 // mantiene vivas las cadenas mientras Windows las usa
	defer runtime.KeepAlive(&vivos)
	p := func(s string) uintptr {
		u, _ := windows.UTF16FromString(s)
		vivos = append(vivos, u)
		return uintptr(unsafe.Pointer(&u[0]))
	}
	metodo(sl, 20, p(exe))              // SetPath
	metodo(sl, 9, p(filepath.Dir(exe))) // SetWorkingDirectory
	metodo(sl, 7, p("TeToca"))          // SetDescription
	metodo(sl, 17, p(exe), 0)           // SetIconLocation
	var pf uintptr
	if r := metodo(sl, 0, uintptr(unsafe.Pointer(&iidIPersistFile)), uintptr(unsafe.Pointer(&pf))); r != 0 {
		return fmt.Errorf("IPersistFile 0x%x", r)
	}
	defer metodo(pf, 2)
	if r := metodo(pf, 6, p(lnk), 1); r != 0 { // Save
		return fmt.Errorf("guardar acceso directo 0x%x", r)
	}
	return nil
}

func quitarAccesoDirecto() {
	if lnk, err := rutaAccesoDirecto(); err == nil {
		_ = os.Remove(lnk)
	}
}
