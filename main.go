package main

import (
	"context"
	"embed"
	"net"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

const signalPort = "127.0.0.1:39455"

func alreadyRunning() bool {

	name, _ := syscall.UTF16PtrFromString("ProtectChildrenSystemMutex")

	handle, _, lastErr := procCreateMutex.Call(
		0,
		1,
		uintptr(unsafe.Pointer(name)),
	)

	if handle == 0 {
		return false
	}

	if lastErr == syscall.ERROR_ALREADY_EXISTS {
		return true
	}

	return false
}

func sendShowSignal() {
	conn, err := net.Dial("tcp", signalPort)
	if err != nil {
		return
	}
	conn.Write([]byte("SHOW"))
	conn.Close()
}

func startSignalServer(app *App) {

	ln, err := net.Listen("tcp", signalPort)
	if err != nil {
		return
	}

	go func() {

		for {

			conn, err := ln.Accept()
			if err != nil {
				continue
			}

			buf := make([]byte, 10)
			conn.Read(buf)

			if string(buf[:4]) == "SHOW" {

				if app.ctx != nil {

					runtime.WindowShow(app.ctx)
					runtime.WindowUnminimise(app.ctx)

				}

			}

			conn.Close()

		}

	}()

}

func main() {

	if alreadyRunning() {

		sendShowSignal()
		return

	}

	app := NewApp()

	go startSignalServer(app)

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
		OnStartup: func(ctx context.Context) {

			app.startup(ctx)

			time.Sleep(200 * time.Millisecond)

		},
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
