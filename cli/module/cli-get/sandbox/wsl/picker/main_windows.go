//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	bifReturnOnlyFSDirs  = 0x0001
	bifEditBox           = 0x0010
	bifNewDialogStyle    = 0x0040
	bifNoNewFolderButton = 0x0200
	bifUseNewUI          = bifEditBox | bifNewDialogStyle
	maxFolderPathChars   = 32768
	bffmInitialized      = 1
	bffmSetSelectionW    = 0x467
)

var errPickerCanceled = errors.New("picker canceled")

var (
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	ole32                   = syscall.NewLazyDLL("ole32.dll")
	procSHBrowseForFolderW  = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListEx")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSendMessageW        = user32.NewProc("SendMessageW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procGetConsoleWindow    = kernel32.NewProc("GetConsoleWindow")
	procOleInitialize       = ole32.NewProc("OleInitialize")
	procOleUninitialize     = ole32.NewProc("OleUninitialize")
	procCoTaskMemFree       = ole32.NewProc("CoTaskMemFree")
)

type browseInfo struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

func main() {
	hideConsoleWindow()
	path, err := pickWindowsFolder()
	if err != nil {
		if errors.Is(err, errPickerCanceled) {
			os.Exit(1)
		}
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(2)
	}
	fmt.Fprint(os.Stdout, path)
}

func browseFolderCallback(hwnd, msg, _ uintptr, data uintptr) uintptr {
	if msg == bffmInitialized && data != 0 {
		procSendMessageW.Call(hwnd, bffmSetSelectionW, 1, data)
	}
	return 0
}

func hideConsoleWindow() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	const swHide = 0
	procShowWindow.Call(hwnd, swHide)
}

func pickWindowsFolder() (string, error) {
	hr, _, callErr := procOleInitialize.Call(0)
	switch hr {
	case 0, 1:
		defer procOleUninitialize.Call()
	default:
		if callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", fmt.Errorf("ole initialize failed: 0x%x", hr)
	}

	title, err := syscall.UTF16PtrFromString("CLI_SANDBOX 请选择允许访问的 Windows 目录")
	if err != nil {
		return "", err
	}
	var displayName [maxFolderPathChars]uint16
	owner, _, _ := procGetForegroundWindow.Call()
	initialPath, err := syscall.UTF16PtrFromString(defaultWindowsPickerDirectory())
	if err != nil {
		return "", err
	}
	info := browseInfo{
		hwndOwner:      owner,
		pszDisplayName: &displayName[0],
		lpszTitle:      title,
		ulFlags:        bifReturnOnlyFSDirs | bifUseNewUI | bifNoNewFolderButton,
		lpfn:           syscall.NewCallback(browseFolderCallback),
		lParam:         uintptr(unsafe.Pointer(initialPath)),
	}

	pidl, _, callErr := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(initialPath)
	if pidl == 0 {
		return "", errPickerCanceled
	}
	defer procCoTaskMemFree.Call(pidl)

	buf := make([]uint16, maxFolderPathChars)
	ok, _, pathErr := procSHGetPathFromIDList.Call(
		pidl,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if ok == 0 {
		if pathErr != syscall.Errno(0) {
			return "", pathErr
		}
		return "", errors.New("failed to resolve selected folder path")
	}
	path := syscall.UTF16ToString(buf)
	if path == "" {
		return "", errors.New("selected folder path is empty")
	}
	return path, nil
}

func defaultWindowsPickerDirectory() string {
	return `C:\`
}
