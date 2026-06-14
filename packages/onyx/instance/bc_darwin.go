package instance

import (
	. "main/packages/onyx/mem"
)

func (Bytecode *EmbeddedByteCode) Set(new []byte, size int) {
	if Bytecode.Addy > 0x1000 {
		alloc := Bytecode.Luna.VirtualAlloc(0, uintptr(size))
		WriteProcessMemory(Bytecode.Luna, alloc, new, uintptr(size))
		WriteProcessMemory(Bytecode.Luna, Bytecode.Addy+0x10, alloc)
		WriteProcessMemory(Bytecode.Luna, Bytecode.Addy+0x18, size)
	}
}
