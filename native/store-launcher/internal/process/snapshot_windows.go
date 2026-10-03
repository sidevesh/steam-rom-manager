//go:build windows

package process

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

type WindowsSource struct{}

func (WindowsSource) Snapshot() ([]Info, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			return nil, nil
		}
		return nil, err
	}

	processes := make([]Info, 0, 256)
	for {
		info := Info{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Name:      windows.UTF16ToString(entry.ExeFile[:]),
		}
		populateDetails(&info)
		processes = append(processes, info)

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, err
		}
	}
	return processes, nil
}

func populateDetails(info *Info) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, info.PID)
	if err != nil {
		return
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err == nil {
		info.Executable = windows.UTF16ToString(buffer[:size])
	}

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err == nil {
		info.CreationTime = uint64(creation.Nanoseconds())
	}
}
