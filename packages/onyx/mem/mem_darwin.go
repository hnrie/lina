package mem

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <mach/mach.h>
#include <mach/mach_vm.h>
#include <sys/types.h>
#include <mach-o/dyld_images.h>
#include <stdio.h>
#include <stdlib.h>
#include <libproc.h>
#include <string.h>
#include <errno.h>

kern_return_t get_task_for_pid_func(int pid, mach_port_name_t *task) {
    return task_for_pid(mach_task_self(), pid, task);
}

kern_return_t mach_vm_read_overwrite_func(mach_port_name_t task, mach_vm_address_t address, mach_vm_size_t size, mach_vm_address_t data, mach_vm_size_t *outSize) {
    return mach_vm_read_overwrite(task, address, size, data, outSize);
}

kern_return_t mach_vm_write_func(mach_port_name_t task, mach_vm_address_t address, mach_vm_size_t size, mach_vm_address_t data) {
    return mach_vm_write(task, address, (vm_offset_t)data, size);
}
*/
import "C"
import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"
)

const (
	VM_PROT_READ           = C.VM_PROT_READ
	VM_PROT_WRITE          = C.VM_PROT_WRITE
	VM_PROT_EXECUTE        = C.VM_PROT_EXECUTE
	PAGE_EXECUTE_READWRITE = uint32(0x40)
	MEM_COMMIT             = 0x1000
	MEM_RESERVE            = 0x2000
	MEM_MAPPED             = 0x4000
	MEM_RELEASE            = 0x8000
)

type ModuleInfo struct {
	BaseOfDll   uintptr
	SizeOfImage uint32
}

type InstanceType struct {
	Variant_Internal  bool
	Variant_External  bool
	Internal_Injected bool
	External_Injected bool
}

type Luna struct {
	Pid        uintptr
	Handle     C.mach_port_name_t
	Render     uintptr
	Variant    InstanceType
	ModuleInfo []ModuleInfo
}

type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       uintptr
	RegionSize        uintptr
	Protect           uintptr
	AllocationProtect uintptr
	State             uint32
	Type              uint32
	Shared            bool
	Reserved          bool
	Offset            uint64
}

func NewLuna(pid uint32) *Luna {
	L := &Luna{}
	var task C.mach_port_name_t

	kr := C.get_task_for_pid_func(C.int(pid), &task)
	if kr == C.KERN_SUCCESS {
		L = &Luna{
			Pid:    uintptr(pid),
			Handle: task,
		}
	}
	return L
}

func ReadProcessMemory[T any](L *Luna, addr uintptr, maxSize ...uintptr) T {
	var result T

	defer func() { recover() }()

	t := reflect.TypeOf(result)
	k := t.Kind()

	if k == reflect.String {
		size := uintptr(0x1000)
		if len(maxSize) > 0 {
			size = maxSize[0]
		}
		data := make([]byte, size)
		bytesRead, _ := L.ReadRaw(
			addr,
			unsafe.Pointer(&data[0]),
			size,
		)
		data = data[:bytesRead]
		if i := bytes.IndexByte(data, 0); i != -1 {
			data = data[:i]
		}
		return any(string(data)).(T)

	} else if k == reflect.Slice {
		elemType := t.Elem()

		if elemType.Kind() == reflect.Uint8 {
			size := uintptr(0x1000)
			if len(maxSize) > 0 {
				size = maxSize[0]
			}
			data := make([]byte, size)

			bytesRead, _ := L.ReadRaw(
				addr,
				unsafe.Pointer(&data[0]),
				size,
			)

			return any(data[:bytesRead]).(T)
		} else {
			if len(maxSize) == 0 {
				return result
			}
			size := maxSize[0]
			elemSize := elemType.Size()
			numElements := int(size / elemSize)
			newSlice := reflect.MakeSlice(t, numElements, numElements)

			L.ReadRaw(
				addr,
				newSlice.UnsafePointer(),
				size,
			)

			return newSlice.Interface().(T)
		}
	} else {
		size := t.Size()
		if size == 0 {
			return result
		}

		L.ReadRaw(
			addr,
			unsafe.Pointer(&result),
			size,
		)

		return result
	}
}

func WriteProcessMemory[T any](L *Luna, addr uintptr, value T, size ...uintptr) error {
	var data []byte
	switch v := any(value).(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		sz := unsafe.Sizeof(value)
		if len(size) > 0 {
			sz = size[0]
		}
		data = unsafe.Slice((*byte)(unsafe.Pointer(&value)), int(sz))
	}

	if len(size) > 0 {
		targetSize := size[0]
		if targetSize > 0 && targetSize < uintptr(len(data)) {
			data = data[:targetSize]
		} else if targetSize > uintptr(len(data)) {
			padding := make([]byte, targetSize-uintptr(len(data)))
			data = append(data, padding...)
		}
	}

	if len(data) == 0 {
		return nil
	}
	return L.WriteRaw(addr, data)
}

func (L *Luna) ReadRaw(addy uintptr, buffer unsafe.Pointer, size uintptr) (uintptr, error) {
	var outSize C.mach_vm_size_t
	kr := C.mach_vm_read_overwrite_func(
		L.Handle,
		C.mach_vm_address_t(addy),
		C.mach_vm_size_t(size),
		C.mach_vm_address_t(uintptr(buffer)),
		&outSize,
	)

	if kr != C.KERN_SUCCESS {
		return 0, fmt.Errorf("mach_vm_read failed with error code %d", kr)
	}

	return uintptr(outSize), nil
}

func (L *Luna) WriteRaw(addy uintptr, value []byte) error {
	kr := C.mach_vm_write_func(
		L.Handle,
		C.mach_vm_address_t(addy),
		C.mach_vm_size_t(len(value)),
		C.mach_vm_address_t(uintptr(unsafe.Pointer(&value[0]))),
	)

	if kr != C.KERN_SUCCESS {
		return fmt.Errorf("mach_vm_write failed with error code %d", kr)
	}

	return nil
}

func (L *Luna) VirtualAlloc(addy, size uintptr) uintptr {
	var remoteAddr C.mach_vm_address_t = C.mach_vm_address_t(addy)

	kr := C.mach_vm_allocate(
		L.Handle,
		&remoteAddr,
		C.mach_vm_size_t(size),
		C.int(1),
	)

	if kr != C.KERN_SUCCESS {
		return 0
	}

	C.mach_vm_protect(
		L.Handle,
		remoteAddr,
		C.mach_vm_size_t(size),
		0,
		C.VM_PROT_READ|C.VM_PROT_WRITE,
	)

	return uintptr(remoteAddr)
}

func (L *Luna) VirtualQuery(address uintptr) (*MEMORY_BASIC_INFORMATION, error) {
	var regionAddress C.mach_vm_address_t = C.mach_vm_address_t(address)
	var regionSize C.mach_vm_size_t
	var info C.vm_region_basic_info_data_64_t
	var count C.mach_msg_type_number_t = C.mach_msg_type_number_t(C.VM_REGION_BASIC_INFO_COUNT_64)
	var objectName C.mach_port_t

	kr := C.mach_vm_region(
		L.Handle,
		&regionAddress,
		&regionSize,
		C.VM_REGION_BASIC_INFO_64,
		C.vm_region_info_t(unsafe.Pointer(&info)),
		&count,
		&objectName,
	)

	if kr != C.KERN_SUCCESS {
		return nil, fmt.Errorf("mach_vm_region failed with error code %d", kr)
	}

	var memType uint32 = 0x4000
	if info.shared != 0 {
		memType = MEM_MAPPED
	}

	return &MEMORY_BASIC_INFORMATION{
		BaseAddress:       uintptr(regionAddress),
		RegionSize:        uintptr(regionSize),
		Protect:           uintptr(info.protection),
		AllocationProtect: uintptr(info.max_protection),
		State:             uint32(MEM_COMMIT),
		Type:              memType,
		Shared:            info.shared != 0,
		Reserved:          info.reserved != 0,
	}, nil
}

func (L *Luna) GetModuleBase(address uintptr) (uintptr, error) {
	if len(L.ModuleInfo) > 0 {
		return L.ModuleInfo[0].BaseOfDll, nil
	}
	return 0, errors.New("module enumeration not fully implemented on macOS")
}

func (L *Luna) EnumModules() []ModuleInfo {
	return L.ModuleInfo
}

func SignForDebugging(appPath string) error {
	var HasDebugEntitlement = func(appPath string) bool {
		cmd := exec.Command("codesign", "-d", "--entitlements", "-", "--xml", appPath)
		var out bytes.Buffer
		cmd.Stdout = &out
		_ = cmd.Run()
		output := out.String()
		return strings.Contains(output, "com.apple.security.get-task-allow") &&
			strings.Contains(output, "<true/>")
	}
	if HasDebugEntitlement(appPath) {
		return errors.New("process is already signed")
	}
	fmt.Printf("[-] Preparing to patch: %s\n", appPath)
	tmpFile, err := os.CreateTemp("", "entitlements-*.plist")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	plistPath := tmpFile.Name()
	cmdExtract := exec.Command("codesign", "-d", "--entitlements", "-", "--xml", appPath)
	cmdExtract.Stdout = tmpFile
	if err := cmdExtract.Run(); err != nil {
		defaultPlist := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict></dict></plist>`)
		os.WriteFile(plistPath, defaultPlist, 0644)
	}
	tmpFile.Close()
	for _, key := range []string{
		"com.apple.developer.team-identifier",
		"com.apple.application-identifier",
		"keychain-access-groups",
		"com.apple.developer.icloud-container-identifiers",
		"com.apple.developer.ubiquity-kvstore-identifier",
	} {
		exec.Command("/usr/libexec/PlistBuddy", "-c", "Delete :"+key, plistPath).Run()
	}
	exec.Command("/usr/libexec/PlistBuddy", "-c", "Delete :com.apple.security.get-task-allow", plistPath).Run()
	if err := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :com.apple.security.get-task-allow bool true", plistPath).Run(); err != nil {
		return fmt.Errorf("failed to add debug entitlement: %v", err)
	}
	exec.Command("/usr/libexec/PlistBuddy", "-c", "Delete :com.apple.security.cs.disable-library-validation", plistPath).Run()
	exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :com.apple.security.cs.disable-library-validation bool true", plistPath).Run()
	fmt.Println("[-] Resigning application...")
	cmdSign := exec.Command("codesign", "--force", "--deep", "--options", "runtime",
		"--sign", "-", "--entitlements", plistPath, appPath)
	if out, err := cmdSign.CombinedOutput(); err != nil {
		return fmt.Errorf("codesign failed: %s", string(out))
	}
	fmt.Println("[+] Success! App is now debuggable.")
	return nil
}

func GetAppBundleByPid(pid int32) (string, error) {
	buffer := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	ret, err := C.proc_pidpath(C.int(pid), unsafe.Pointer(&buffer[0]), C.uint32_t(len(buffer)))
	if ret <= 0 {
		if err != nil {
			return "", err
		}
		return "", errors.New("failed to retrieve process path")
	}
	fullExecutablePath := C.GoString((*C.char)(unsafe.Pointer(&buffer[0])))
	currentPath := fullExecutablePath
	for {
		if strings.HasSuffix(currentPath, ".app") {
			return currentPath, nil
		}
		parent := filepath.Dir(currentPath)
		if parent == currentPath || parent == "." || parent == "/" {
			break
		}
		currentPath = parent
	}
	return filepath.Dir(fullExecutablePath), nil
}
