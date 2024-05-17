package showConsol

import (
	"syscall"
	"time"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procShowWindow       = user32.NewProc("ShowWindow")
	consoleVisible       int
)

const (
	VK_CONTROL = 0x11
	VK_SHIFT   = 0x10
	VK_F4      = 0x73
	VK_F3      = 0x72
)

func toggleConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	if consoleVisible == 0 {
		procShowWindow.Call(hwnd, 1)
		consoleVisible = 1
	} else {
		procShowWindow.Call(hwnd, 0)
		consoleVisible = 0
	}
}

func isKeyPressed(keyCode uintptr) bool {
	ret, _, _ := procGetAsyncKeyState.Call(keyCode)
	return ret&0x8000 != 0
}

func CheckHotkey(restartFunc func()) {
	consoleVisible = 1
	toggleConsole()
	for {
		if isKeyPressed(VK_CONTROL) && isKeyPressed(VK_SHIFT) && isKeyPressed(VK_F4) {
			toggleConsole()
		}
		if isKeyPressed(VK_CONTROL) && isKeyPressed(VK_SHIFT) && isKeyPressed(VK_F3) {
			restartFunc()
		}
		time.Sleep(100 * time.Millisecond)
	}
}
