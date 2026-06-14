package Api

/*
#include "windows.h"
*/
import "C"
import (
	"fmt"
	. "main/packages/onyx/static"
	"syscall"

	"unsafe"
)

var (
	Rebase = func(offset uintptr) uintptr {
		ret, _, _ := GetModuleHandleA.Call(uintptr(unsafe.Pointer(nil)))
		return ret + offset
	}
	scroff = uintptr(0x7F8)
	print  = Rebase(0x1E08380)
	scr    = Rebase(0x1D79260)

	Print = func(type_m int, message string, args ...any) {
		if message == "" {
			return
		}
		if type_m > 3 || type_m < 0 {
			type_m = 0
		}
		var msg []byte = []byte(message)
		if len(args) > 0 {
			msg = []byte("")
			msg = fmt.Appendf(msg, message, args...)
		}
		syscall.SyscallN(
			print,
			uintptr(type_m),
			uintptr(unsafe.Pointer(&msg[0])),
			0,
		)
	}
)

func IdentityToCapabilities(identity int, isMax bool) uintptr {
	var result uint64

	switch int64(identity) {
	case 0, 2:
		result = 0
	case 1, 4:
		result = 0x2000000000000003
	case 3:
		result = 0x300000000000000B
	case 5:
		result = 0x2000000000000001
	case 6:
		result = 0x700000000000000B
	case 7, 8:
		result = 0x200000000000003F
	case 9:
		result = 12
	case 10:
		result = 0x6000000000000003
	case 11:
		result = 0x2000000000000000
	case 12:
		result = 0x1000000000000000
	default:
		result = 0
	}

	base := result | 0x3FFFFFFFFFFF00

	if isMax {
		return uintptr(base | (1 << 48))
	}

	return uintptr(base)
}

func ScriptContextResume(Debug *LuaDebugResult, Weak **SWeakObjectRef2, args int, err bool, errmsg string) int {
	var (
		ptr uintptr = 0
	)
	if errmsg != "" {
		var msg []byte = []byte(errmsg)
		ptr = uintptr(unsafe.Pointer(&msg[0]))
	}

	ret, _, _ := syscall.SyscallN(
		scr,
		Api.ScriptContext+uintptr(scroff),
		uintptr(unsafe.Pointer(Debug)),
		uintptr(unsafe.Pointer(Weak)),
		uintptr(args),
		func() uintptr {
			if err {
				return 1
			}
			return 0
		}(),
		ptr,
	)
	return int(ret)
}
