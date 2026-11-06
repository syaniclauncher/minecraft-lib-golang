//go:build windows

package mcl

import (
	"strconv"
	"syscall"
	"unsafe"
)

// osVersion returns the version of the operating system, for example "10.0".
// It is matched against the "version" field of the rules in the version
// metadata.
func osVersion() string {
	key, err := openRegistryKey(`SOFTWARE\Microsoft\Windows NT\CurrentVersion`)
	if err != nil {
		return ""
	}
	defer syscall.RegCloseKey(key)

	major, err := readRegistryDWord(key, "CurrentMajorVersionNumber")
	if err != nil {
		return ""
	}
	minor, err := readRegistryDWord(key, "CurrentMinorVersionNumber")
	if err != nil {
		minor = 0
	}
	return strconv.FormatUint(major, 10) + "." + strconv.FormatUint(minor, 10)
}

// openRegistryKey opens a key of the local machine hive.
func openRegistryKey(name string) (syscall.Handle, error) {
	path, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var handle syscall.Handle
	err = syscall.RegOpenKeyEx(syscall.HKEY_LOCAL_MACHINE, path, 0, syscall.KEY_READ|syscall.KEY_WOW64_64KEY, &handle)
	if err != nil {
		return 0, err
	}
	return handle, nil
}

// readRegistryDWord reads a numeric value of a registry key.
func readRegistryDWord(key syscall.Handle, name string) (uint64, error) {
	valueName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var valueType uint32
	var value uint32
	size := uint32(4)
	err = syscall.RegQueryValueEx(key, valueName, nil, &valueType, (*byte)(unsafe.Pointer(&value)), &size)
	if err != nil {
		return 0, err
	}
	return uint64(value), nil
}
