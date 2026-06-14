package instance

import (
	"main/packages/onyx/mem"
)

type (
	EmbeddedByteCode struct {
		Luna     *mem.Luna
		Instance *Instance
		Bytecode []byte
		Hash     []byte
		Size     int
		Addy     uintptr
	}
	LoadingState int
	Name         uintptr
	Parent       uintptr
	Children     uintptr
	Class        uintptr
	Self         uintptr
	ByteCode     uintptr
	Value        uintptr
	VMState      uintptr
)

type (
	Instance struct {
		_             uintptr    // 0x0
		Self                     // 0x8
		_             uintptr    // 0x10
		cl            Class      // 0x18
		_             [0x48]byte //
		p             Parent     // 0x68
		c             Children   // 0x70
		_             [0x38]byte //
		n             Name       // 0xb0
		_             [0x18]byte // 0xb8
		v             Value      // 0xd0
		_             [0x68]byte // 0xd8
		mbytecode     ByteCode   // 0x140 module script bytecode
		_             [0x28]byte //
		loading_state uintptr    // 0x170
		mvmstate      VMState    // 0x178 module script vmstate
		lvmstate      VMState    // 0x180 localscript vmstate
		_             [0x28]byte //
		lbytecode     ByteCode   // 0x1b0 localscript bytecode
		Luna          *mem.Luna
	}
)

const (
	NotRunYet LoadingState = 0
	Running   LoadingState = 1
	Error     LoadingState = 2
	Success   LoadingState = 3
)
