package rtti

/*

#include <windows.h>
#include <dbghelp.h>
#include <stdlib.h>
#include <string.h>

char* demangleMSVC(const char* mangled, DWORD flags) {
    char* buffer = (char*)malloc(8192);
    if (!buffer) return NULL;
    DWORD len = UnDecorateSymbolName(mangled, buffer, 8192, flags);
    if (len == 0) {
        free(buffer);
        return NULL;
    }
    char* result = (char*)malloc(len + 1);
    if (result) {
        memcpy(result, buffer, len);
        result[len] = '\0';
    }
    free(buffer);
    return result;
}
*/
import "C"
import (
	"fmt"
	. "main/packages/onyx/mem"
	"strings"
	"sync"
	"unsafe"
)

// #cgo LDFLAGS: -ldbghelp

var dbghelpMu sync.Mutex

func RTTIInformation(process *Luna, address uintptr) (string, error) {
	type (
		RTTICompleteObjectLocator struct {
			Signature       uint32
			Offset          uint32
			CdOffset        uint32
			TypeDescriptor  uint32
			ClassDescriptor uint32
			Self            uint32
		}
	)
	if address == 0 {
		return "", fmt.Errorf("invalid address")
	}
	vTableAddress := ReadProcessMemory[uintptr](process, address)
	if vTableAddress == 0 {
		return "", fmt.Errorf("invalid vtable")
	}
	colPtrAddress := vTableAddress - 8
	colAddress := ReadProcessMemory[uintptr](process, colPtrAddress)
	if colAddress == 0 {
		return "", fmt.Errorf("invalid COL pointer")
	}
	col := ReadProcessMemory[RTTICompleteObjectLocator](process, colAddress)
	var moduleBase uintptr
	if col.Signature == 1 && col.Self != 0 {
		moduleBase = colAddress - uintptr(col.Self)
	} else {
		var err error
		moduleBase, err = process.GetModuleBase(colAddress)
		if err != nil {
			return "", err
		}
	}
	typeDescAddr := moduleBase + uintptr(col.TypeDescriptor)
	nameBuf := ReadProcessMemory[[128]byte](process, typeDescAddr+16)
	var buf = string(nameBuf[:])
	if idx := strings.IndexByte(buf, 0); idx != -1 {
		buf = buf[:idx]
	}
	if buf == "" {
		return "", fmt.Errorf("empty RTTI name")
	}
	return Demangle([]byte(buf))
}

func Demangle(nameBytes []byte) (string, error) {
	end := 0
	for i, b := range nameBytes {
		if b == 0 {
			end = i
			break
		}
	}
	if end == 0 {
		end = len(nameBytes)
	}
	mangled := string(nameBytes[:end])

	if len(mangled) >= 4 && mangled[0] == '.' && mangled[1] == '?' && mangled[2] == 'A' {
		regularMangled := mangled[1:]
		input := C.CString(regularMangled)
		defer C.free(unsafe.Pointer(input))

		dbghelpMu.Lock()
		defer dbghelpMu.Unlock()

		flags := C.UNDNAME_NO_ARGUMENTS
		cResult := C.demangleMSVC(input, C.DWORD(flags))
		if cResult != nil {
			result := C.GoString(cResult)
			C.free(unsafe.Pointer(cResult))
			result = fixRTTIDemangling(result)
			return result, nil
		}
		flags = C.UNDNAME_COMPLETE
		cResult = C.demangleMSVC(input, C.DWORD(flags))
		if cResult != nil {
			result := C.GoString(cResult)
			C.free(unsafe.Pointer(cResult))
			result = fixRTTIDemangling(result)
			return result, nil
		}
		return extractRTTIName(mangled), nil
	}
	if len(mangled) > 0 && mangled[0] == '?' {
		input := C.CString(mangled)
		defer C.free(unsafe.Pointer(input))

		dbghelpMu.Lock()
		defer dbghelpMu.Unlock()

		flags := C.UNDNAME_COMPLETE
		cResult := C.demangleMSVC(input, C.DWORD(flags))
		if cResult != nil {
			defer C.free(unsafe.Pointer(cResult))
			result := C.GoString(cResult)
			return result, nil
		}
	}
	return mangled, nil
}

func fixRTTIDemangling(demangled string) string {
	demangled = strings.TrimPrefix(demangled, "?? ")
	if len(demangled) >= 2 && demangled[0:2] == "AV" && len(demangled) > 2 {
		thirdChar := demangled[2]
		if (thirdChar >= 'A' && thirdChar <= 'Z') || thirdChar == '_' {
			demangled = demangled[2:]
		}
	}
	demangled = removeTypePrefix(demangled)
	return demangled
}

func removeTypePrefix(name string) string {
	prefixes := []string{"class ", "struct ", "enum ", "union ", " "}
	for {
		changed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(name, prefix) {
				name = strings.TrimPrefix(name, prefix)
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return name
}

func extractRTTIName(mangled string) string {
	if len(mangled) < 6 || !strings.HasSuffix(mangled, "@@") {
		return mangled
	}
	return strings.ReplaceAll(
		strings.ReplaceAll(
			mangled[4:len(mangled)-2],
			"@",
			"::"),
		"::::",
		"::")
}
