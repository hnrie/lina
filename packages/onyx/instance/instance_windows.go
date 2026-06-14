package instance

import (
	"fmt"
	"main/packages/onyx/instance/luastate"
	. "main/packages/onyx/mem"
	"time"
	"unsafe"
)

func NewInstance(L *Luna, a uintptr) *Instance {
	instance := ReadProcessMemory[Instance](L, a)
	instance.Luna = L
	return &instance
}

func (I *Instance) VMState() VMState {
	switch I.ClassName() {
	case "ModuleScript":
		return I.mvmstate
	case "LocalScript":
		return I.lvmstate
	}
	return 0
}

func (Instance *Instance) LuaState() *luastate.LuaState {
	if i := Instance.VMState(); i != 0 {
		return &luastate.LuaState{
			Address: ReadProcessMemory[uintptr](Instance.Luna, uintptr(i)+0x8),
			Luna:    Instance.Luna,
		}

	}
	return &luastate.LuaState{Luna: Instance.Luna}
}

func (I *Instance) LoadingState() LoadingState {
	if class := I.ClassName(); class != "ModuleScript" && class != "LocalScript" {
		return -1
	}
	fmt.Println(I.loading_state)
	return LoadingState(I.loading_state)
}

func (I *Instance) SetLoadingState(state LoadingState) {
	if class := I.ClassName(); class != "ModuleScript" && class != "LocalScript" {
		return
	}
	WriteProcessMemory(I.Luna, unsafe.Offsetof(I.loading_state)+uintptr(I.Self), uintptr(state))
}
func (Instance *Instance) WaitForLoadingState(state LoadingState) bool {
	for range 50 {
		time.Sleep(time.Millisecond * 1)
		if Instance.LoadingState() == state {
			return true
		}
	}
	return false
}

func (Name Instance) Name() string {
	var (
		buffer [0x100]byte = ReadProcessMemory[[0x100]byte](Name.Luna, uintptr(Name.n))
		l                  = ReadProcessMemory[uint16](Name.Luna, uintptr(Name.n)+0x10)
	)
	if l > 0x100 {
		l = 0x10
	}
	if l >= 16 {
		buffer = ReadProcessMemory[[0x100]byte](Name.Luna, ReadProcessMemory[uintptr](Name.Luna, uintptr(Name.n)))
	}
	return string(buffer[:l])
}

func (Class Instance) ClassName() string {
	var (
		name               = ReadProcessMemory[uintptr](Class.Luna, uintptr(Class.cl)+0x8)
		buffer [0x100]byte = ReadProcessMemory[[0x100]byte](Class.Luna, name)
	)
	size := min(ReadProcessMemory[uint16](Class.Luna, name+0x10), 0x100)
	return string(buffer[:size])
}

func (Parent Instance) Parent() *Instance {
	p := ReadProcessMemory[Instance](Parent.Luna, uintptr(Parent.p))
	p.Luna = Parent.Luna
	return &p
}

func (Children Instance) Children() (c []*Instance) {
	type container struct {
		inst uintptr
		_    [8]byte
	}
	var (
		children_ptr = ReadProcessMemory[uintptr](Children.Luna, uintptr(Children.c))
		size         = uintptr(uintptr(int((ReadProcessMemory[uintptr](Children.Luna, uintptr(Children.c)+0x10)-children_ptr)/unsafe.Sizeof(container{}))) * unsafe.Sizeof(container{}))
	)
	for _, container := range ReadProcessMemory[[]container](Children.Luna, children_ptr, size) {
		if inst := NewInstance(Children.Luna, container.inst); inst != nil && inst.Self > 0x1000 {
			c = append(c, inst)
		}
	}
	return
}

func (L *Instance) Traverse(names ...string) *Instance {
	if L != nil && L.Self > 0x1000 && len(names) > 0 {
		var (
			cached    *Instance            = L
			instances map[string]*Instance = make(map[string]*Instance)
			cache     func()               = func() {
				for _, value := range cached.Children() {
					if name := value.Name(); name != "" {
						instances[name] = value
						instances[value.ClassName()] = value
					}
				}
			}
		)

		cache()

		for _, name := range names {
			if ok, found := instances[name]; found {
				cached = ok
				if len(names) == 1 {
					return cached
				}
				instances = make(map[string]*Instance)
				cache()
			}
		}
		return cached
	}
	return &Instance{Luna: L.Luna}
}

func (I *Instance) BytecodeAddy() uintptr {
	switch I.ClassName() {
	case "ModuleScript":
		return uintptr(I.Self) + unsafe.Offsetof(I.mbytecode)
	case "LocalScript":
		return uintptr(I.Self) + unsafe.Offsetof(I.lbytecode)
	}
	return 0
}

func (I *Instance) Bytecode() (Module *EmbeddedByteCode) {
	Module = &EmbeddedByteCode{Luna: I.Luna}

	var addy uintptr
	switch I.ClassName() {
	case "ModuleScript":
		addy = uintptr(I.mbytecode)
	case "LocalScript":
		addy = uintptr(I.lbytecode)
	default:
		return
	}

	if addy == 0 {
		return
	}

	Module.Addy = addy
	Module.Size = ReadProcessMemory[int](I.Luna, addy+0x20)

	if Module.Size <= 0 || Module.Size > 0x100000 {
		return
	}

	Module.Hash = ReadProcessMemory[[]byte](I.Luna, addy, 0xa8)

	bcPtr := ReadProcessMemory[uintptr](I.Luna, addy+0x10)

	if bcPtr == 0 {
		return
	}

	Module.Bytecode = ReadProcessMemory[[]byte](I.Luna, bcPtr, uintptr(Module.Size))

	return
}

func (I *Instance) UniverseID() (id uintptr) {
	if I.ClassName() == "DataModel" {
		return uintptr(I.mvmstate)
	}
	return
}
func (Bytecode *EmbeddedByteCode) Restore() {
	if Bytecode.Addy > 0x1000 {
		WriteProcessMemory(Bytecode.Luna, Bytecode.Addy, Bytecode.Hash, 0xa8)
		WriteProcessMemory(Bytecode.Luna, ReadProcessMemory[uintptr](Bytecode.Luna, Bytecode.Addy+0x10), Bytecode.Bytecode, uintptr(Bytecode.Size))
	}
}
