//go:build windows

package win32

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const clsctxLocalServer = 0x4

var (
	procCoCreateInstance              = ole32.NewProc("CoCreateInstance")
	applicationActivationManagerCLSID = windows.GUID{
		Data1: 0x45ba127d, Data2: 0x10a8, Data3: 0x46ea,
		Data4: [8]byte{0x8a, 0xb7, 0x56, 0xea, 0x90, 0x78, 0x94, 0x3c},
	}
	applicationActivationManagerIID = windows.GUID{
		Data1: 0x2e941141, Data2: 0x7f97, Data3: 0x4756,
		Data4: [8]byte{0xba, 0x1d, 0x9d, 0xec, 0xde, 0x89, 0x4a, 0x3d},
	}
)

type applicationActivationManager struct {
	vtbl *applicationActivationManagerVtbl
}

type applicationActivationManagerVtbl struct {
	queryInterface      uintptr
	addRef              uintptr
	release             uintptr
	activateApplication uintptr
}

func activateApplication(appUserModelID string) (uint32, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	result, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if result != 0 && result != 1 && uint32(result) != rpcEChangedMode {
		return 0, fmt.Errorf("CoInitializeEx failed with HRESULT 0x%08x", uint32(result))
	}
	if result == 0 || result == 1 {
		defer procCoUninitialize.Call()
	}

	var manager *applicationActivationManager
	result, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&applicationActivationManagerCLSID)),
		0,
		clsctxLocalServer,
		uintptr(unsafe.Pointer(&applicationActivationManagerIID)),
		uintptr(unsafe.Pointer(&manager)),
	)
	if result != 0 {
		return 0, fmt.Errorf("CoCreateInstance failed with HRESULT 0x%08x", uint32(result))
	}
	defer syscall.SyscallN(manager.vtbl.release, uintptr(unsafe.Pointer(manager)))

	id, err := windows.UTF16PtrFromString(appUserModelID)
	if err != nil {
		return 0, err
	}
	var pid uint32
	result, _, _ = syscall.SyscallN(
		manager.vtbl.activateApplication,
		uintptr(unsafe.Pointer(manager)),
		uintptr(unsafe.Pointer(id)),
		0, // No app-specific arguments.
		0, // AO_NONE.
		uintptr(unsafe.Pointer(&pid)),
	)
	if result != 0 {
		return 0, fmt.Errorf("ActivateApplication failed with HRESULT 0x%08x", uint32(result))
	}
	return pid, nil
}
