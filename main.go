package main

import (
	"embed"
	"os"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func main() {
	// PREVENT DOUBLE PROCESS (Single Instance Lock)
	mutexName := "ProtectChildrenSystemMutex"
	ptr, _ := syscall.UTF16PtrFromString(mutexName)
	ret, _, _ := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(ptr)))
	if ret == 0 || syscall.GetLastError() == syscall.ERROR_ALREADY_EXISTS {
		// If another instance is running, exit immediately
		os.Exit(0)
	}

	app := NewApp()
	isHidden := false
	for _, arg := range os.Args {
		if arg == "--autostart" {
			isHidden = true
			break
		}
	}

	err := wails.Run(&options.App{
		Title:         "Protect Children System",
		Width:         824,
		Height:        468,
		DisableResize: true,
		StartHidden:   isHidden,
		OnStartup:     app.startup,
		OnBeforeClose: app.OnBeforeClose,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
