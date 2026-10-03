//go:build windows

package win32

import (
	"fmt"
	"os/exec"
	"runtime"
	"unsafe"

	"github.com/SteamGridDB/steam-rom-manager/native/store-launcher/internal/cli"
	"golang.org/x/sys/windows"
)

const (
	coinitApartmentThreaded = 0x2
	rpcEChangedMode         = 0x80010106
	swShowNormal            = 1
)

var (
	ole32              = windows.NewLazySystemDLL("ole32.dll")
	shell32            = windows.NewLazySystemDLL("shell32.dll")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	procCoUninitialize = ole32.NewProc("CoUninitialize")
	procShellExecuteW  = shell32.NewProc("ShellExecuteW")
)

type ShellLauncher struct{}

func (ShellLauncher) Launch(options cli.Options) (uint32, error) {
	if options.AppUserModelID != "" {
		return activateApplication(options.AppUserModelID)
	}
	if options.LaunchExecutable != "" {
		command := exec.Command(options.LaunchExecutable, options.LaunchArguments...)
		command.Dir = options.LaunchDirectory
		if err := command.Start(); err != nil {
			return 0, err
		}
		if err := command.Process.Release(); err != nil {
			return 0, err
		}
		return 0, nil
	}
	return 0, openURI(options.URI)
}

func openURI(uri string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	result, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if result != 0 && result != 1 && uint32(result) != rpcEChangedMode {
		return fmt.Errorf("CoInitializeEx failed with HRESULT 0x%08x", uint32(result))
	}
	if result == 0 || result == 1 {
		defer procCoUninitialize.Call()
	}

	operation, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	instance, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafePointer(operation)),
		uintptr(unsafePointer(target)),
		0,
		0,
		swShowNormal,
	)
	if instance <= 32 {
		return fmt.Errorf("ShellExecuteW failed with code %d", instance)
	}
	return nil
}

func unsafePointer(value *uint16) uintptr {
	return uintptr(unsafe.Pointer(value))
}
