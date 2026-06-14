package instance

import (
	"fmt"
	"main/packages/onyx/instance/luastate"
	. "main/packages/onyx/mem"
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

func (Name Instance) Name() string {
	var (
		addy = uintptr(Name.n)
		l    = ReadProcessMemory[uint32](Name.Luna, addy+0x14) >> 24
	)
	switch true {
	case l == 0, l > 1000:
		return ReadProcessMemory[string](Name.Luna, addy, 0x100)
	case l > 18 && l < 1000:
		addy = ReadProcessMemory[uintptr](Name.Luna, addy)
	}
	return ReadProcessMemory[string](Name.Luna, addy, uintptr(l))
}

func (Class Instance) ClassName() string {
	var (
		name = ReadProcessMemory[uintptr](Class.Luna, uintptr(Class.cl)+0x8)
		l    = ReadProcessMemory[uint32](Class.Luna, name+0x14) >> 24
	)
	switch true {
	case l == 0, l > 1000:
		return ReadProcessMemory[string](Class.Luna, name, 0x100)
	case l > 18 && l < 1000:
		name = ReadProcessMemory[uintptr](Class.Luna, name)
	}
	return ReadProcessMemory[string](Class.Luna, name, uintptr(l))
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
	Module = new(EmbeddedByteCode)
	Module.Luna = I.Luna
	addy := uintptr(0x0)
	switch I.ClassName() {
	case "ModuleScript":
		addy = uintptr(I.mbytecode)
	case "LocalScript":
		addy = uintptr(I.lbytecode)
	}

	Module.Addy = addy
	Module.Size = ReadProcessMemory[int](I.Luna,
		Module.Addy+0x20,
	)
	if Module.Size > 0x100000 {
		return &EmbeddedByteCode{Luna: I.Luna}
	}

	Module.Hash = ReadProcessMemory[[]byte](I.Luna,
		Module.Addy,
		0xa8,
	)
	Module.Bytecode = ReadProcessMemory[[]byte](I.Luna,
		ReadProcessMemory[uintptr](I.Luna,
			Module.Addy+0x10,
		),
		uintptr(Module.Size),
	)

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
