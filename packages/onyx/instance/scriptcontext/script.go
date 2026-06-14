package scriptcontext

import (
	"fmt"
	. "main/packages/onyx/instance"
	"main/packages/onyx/instance/luastate"
	. "main/packages/onyx/mem"
)

type (
	ScriptContext struct {
		Instance *Instance
	}
)

func NewScriptContext(i *Instance) *ScriptContext {
	sc := i.Traverse("Script Context")
	return &ScriptContext{
		Instance: sc,
	}
}

func (I *ScriptContext) offsets() uintptr {
	for i := 0x700; i < 0x1000; i += 0x8 {
		ptr := ReadProcessMemory[uintptr](I.Instance.Luna, uintptr(I.Instance.Self)+uintptr(i))
		a := fmt.Sprintf("%x", ReadProcessMemory[uint32](I.Instance.Luna, ReadProcessMemory[uintptr](I.Instance.Luna, ptr)+0x10))
		b := fmt.Sprintf("%x", ReadProcessMemory[uint32](I.Instance.Luna, ReadProcessMemory[uintptr](I.Instance.Luna, ptr+0x8)+0x10))
		if len(a) > 2 && len(b) > 2 && a[:len(a)-2] == "ffffff" && b[:len(b)-2] == "ffffff" {
			return uintptr(i)
		}
	}
	return 0
}

var offset uintptr

func (sc *ScriptContext) LuaStates() (luaStates map[uintptr]bool) {
	if offset == 0 {
		offset = sc.offsets()
	}
	luaStates = make(map[uintptr]bool)
	luastate := ReadProcessMemory[uintptr](sc.Instance.Luna, uintptr(sc.Instance.Self)+offset)
	for {
		lua := ReadProcessMemory[uintptr](sc.Instance.Luna, uintptr(luastate))
		if _, ok := luaStates[lua]; ok {
			break
		}
		luaStates[lua] = true
		luastate = lua
	}
	return
}

func (sc *ScriptContext) SetMaxCaps(luaStates map[uintptr]bool, identity uintptr) {
	caps := luastate.IdentityToCapabilities(uint32(identity))
	for lua, _ := range luaStates {
		fmt.Printf("sigma, 0x%x\n", lua)
		WriteProcessMemory(sc.Instance.Luna, lua+0x10, caps)
		WriteProcessMemory(sc.Instance.Luna, lua+0x18, caps)
	}
}
