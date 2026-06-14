package instance

import (
	. "main/packages/onyx/mem"

	"golang.org/x/sys/windows"
)

func (Bytecode *EmbeddedByteCode) Set(new []byte, size int) {
	if Bytecode.Addy > 0x1000 {
		alloc := Bytecode.Luna.VirtualAlloc(0, uintptr(size), uintptr(windows.MEM_COMMIT|windows.MEM_RESERVE), uintptr(windows.PAGE_EXECUTE_READWRITE))
		WriteProcessMemory(Bytecode.Luna, alloc, new, uintptr(size))
		WriteProcessMemory(Bytecode.Luna, Bytecode.Addy+0x10, alloc)
		WriteProcessMemory(Bytecode.Luna, Bytecode.Addy+0x20, size)
	}
}
