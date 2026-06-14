package mem

/*
#include <stdlib.h>
extern char* __cxa_demangle(const char* mangled_name,
                            char* output_buffer,
                            size_t* length,
                            int* status);
char* demangle(const char* name) {
    int status = 0;
    return __cxa_demangle(name, NULL, NULL, &status);
}
*/
import "C"
import (
	"fmt"
	. "main/packages/onyx/mem"
	"unsafe"
)

func RTTIInformation(process *Luna, object uintptr) (string, error) {
	var (
		demangle = func(mangled string) (string, error) {
			cstr := C.CString(mangled)
			defer C.free(unsafe.Pointer(cstr))
			out := C.demangle(cstr)
			if out == nil {
				return "", fmt.Errorf("failed to demangle %s", mangled)
			}
			defer C.free(unsafe.Pointer(out))
			return C.GoString(out), nil
		}
	)

	vtable := ReadProcessMemory[uintptr](process, object)
	if vtable == 0 {
		return "", fmt.Errorf("vtable is null")
	}
	typeinfoPtr := ReadProcessMemory[uintptr](process, vtable-8)
	if typeinfoPtr == 0 {
		return "", fmt.Errorf("typeinfoPtr is null")
	}
	candidate := ReadProcessMemory[uintptr](process, typeinfoPtr+8)
	if candidate > 0x1000 {
		return demangle(ReadProcessMemory[string](process, candidate, 128))
	}
	return "", fmt.Errorf("candidate is null")

}
