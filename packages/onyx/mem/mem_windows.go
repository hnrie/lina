package mem

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"main/packages/onyx/static"

	"golang.org/x/sys/windows"
)

type (
	ModuleInfo struct {
		BaseAddress uintptr
		Size        uint32
	}
	InstanceType struct {
		Variant_Internal  bool
		Variant_External  bool
		Internal_Injected bool
		External_Injected bool
	}
	Luna struct {
		Pid        uintptr
		Handle     windows.Handle
		Render     uintptr
		Variant    InstanceType
		ModuleInfo []ModuleInfo
	}
)

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
	case string, []byte:
		var initialData []byte
		if str, ok := v.(string); ok {
			initialData = []byte(str)
		} else {
			initialData = v.([]byte)
		}

		if len(size) > 0 && size[0] > uintptr(len(initialData)) {
			data = make([]byte, size[0])
			copy(data, initialData)
		} else {
			data = initialData
		}
	default:
		sz := unsafe.Sizeof(value)
		if len(size) > 0 {
			sz = size[0]
		}
		data = unsafe.Slice((*byte)(unsafe.Pointer(&value)), int(sz))
	}
	if len(data) == 0 {
		return nil
	}
	err := L.WriteRaw(
		addr,
		data,
	)
	if err != nil {
		return err
	}
	return nil
}

func (L *Luna) ReadRaw(addy uintptr, buffer unsafe.Pointer, size uintptr) (uintptr, error) {
	var bytesRead uintptr

	ret, _, err := static.ReadMemory.Call(
		uintptr(L.Handle),
		addy,
		uintptr(buffer),
		size,
		uintptr(unsafe.Pointer(&bytesRead)),
	)
	if ret == 0 {
		return bytesRead, fmt.Errorf("ReadProcessMemory failed: %w", err)
	}

	return bytesRead, nil
}

func (L *Luna) WriteRaw(addy uintptr, value []byte) error {
	var bytesWritten uintptr
	ret, _, err := static.WriteMemory.Call(
		uintptr(L.Handle),
		addy,
		uintptr(unsafe.Pointer(&value[0])),
		uintptr(len(value)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)
	if ret == 0 {
		return fmt.Errorf("WriteProcessMemory failed: %w", err)
	}

	if bytesWritten != uintptr(len(value)) {
		return fmt.Errorf("partial write: %d/%d bytes", bytesWritten, len(value))
	}

	return nil
}

func (L *Luna) VirtualAlloc(addy, size, mem_prot, page_prot uintptr) uintptr {
	addr, _, _ := static.VirtualAlloc.Call(
		uintptr(L.Handle),
		addy,
		size,
		mem_prot,
		page_prot,
	)
	return addr
}

func (L *Luna) GetModuleBase(address uintptr) (uintptr, error) {
	for _, module := range L.EnumModules() {
		if address >= module.BaseAddress && address < module.BaseAddress+uintptr(module.Size) {
			return module.BaseAddress, nil
		}
	}
	return 0, errors.New("module not found")
}

func (L *Luna) EnumModules() []ModuleInfo {
	if len(L.ModuleInfo) > 0 {
		return L.ModuleInfo
	}
	var hMods [1024]windows.Handle
	var cbNeeded uint32
	ret := windows.EnumProcessModules(
		windows.Handle(L.Handle),
		&hMods[0],
		uint32(len(hMods))*uint32(unsafe.Sizeof(hMods[0])),
		&cbNeeded,
	)
	if ret != nil {
		return []ModuleInfo{}
	}
	moduleCount := cbNeeded / uint32(unsafe.Sizeof(hMods[0]))
	for i := 0; i < int(moduleCount); i++ {
		var modInfo windows.ModuleInfo
		err := windows.GetModuleInformation(
			windows.Handle(L.Handle),
			hMods[i],
			&modInfo,
			uint32(unsafe.Sizeof(modInfo)))
		if err != nil {
			continue
		}
		L.ModuleInfo = append(L.ModuleInfo, ModuleInfo{
			BaseAddress: modInfo.BaseOfDll,
			Size:        modInfo.SizeOfImage,
		})
	}
	return L.ModuleInfo
}

func NewLuna(pid uint32) (L *Luna) {
	L = &Luna{}
	if hProcess, err := windows.OpenProcess(
		windows.SYNCHRONIZE|
			windows.PROCESS_QUERY_LIMITED_INFORMATION|
			windows.PROCESS_VM_OPERATION|
			windows.PROCESS_VM_WRITE|
			windows.PROCESS_VM_READ,
		false,
		pid,
	); err == nil {
		L = &Luna{
			Pid:    uintptr(pid),
			Handle: hProcess,
		}
	}
	return
}
