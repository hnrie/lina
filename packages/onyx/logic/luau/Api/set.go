package Api

/*
#include "lua.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"reflect"
	"sync/atomic"
	"unsafe"
)

func (s *LuaState) SetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.lua_setfield(s.cptr(), C.int(C.LUA_GLOBALSINDEX), cName)
}

func (s *LuaState) SetTable(idx int) {
	C.lua_settable(s.cptr(), C.int(idx))
}
func (s *LuaState) NewTable() {
	C.lua_createtable(s.cptr(), 0, 0)
}
func (s *LuaState) SetField(idx int, k string) {
	cK := C.CString(k)
	defer C.free(unsafe.Pointer(cK))
	C.lua_setfield(s.cptr(), C.int(idx), cK)
}
func (s *LuaState) RawSet(idx int) {
	C.lua_rawset(s.cptr(), C.int(idx))
}
func (s *LuaState) RawSetI(idx int, n int) {
	C.lua_rawseti(s.cptr(), C.int(idx), C.int(n))
}
func (s *LuaState) SetMetaTable(idx int) int {
	return int(C.lua_setmetatable(s.cptr(), C.int(idx)))
}
func (s *LuaState) SetFEnv(idx int) int {
	return int(C.lua_setfenv(s.cptr(), C.int(idx)))
}

func (p *LuaState) SetCaps(identity int) {

	capsPtr := (*C.uint64_t)(C.malloc(C.sizeof_uint64_t))                  // create the holder ig.
	*capsPtr = C.uint64_t(IdentityToCapabilities(identity, identity == 8)) // set the capabilities value type shit.
	defer C.free(unsafe.Pointer(capsPtr))                                  // make it release it after the func returns.
	C.set_caps(p.cptr(), *capsPtr)                                         // set the gay shit fyne shits states.
}

func (L *LuaState) SetReadOnly(i int, b bool) {
	C.lua_setreadonly(L.cptr(), C.int(i), func() C.int {
		if b {
			return C.int(1)
		}
		return C.int(0)
	}())
}
func (L *LuaState) GetReadOnly(i int) bool {
	return C.lua_getreadonly(L.cptr(), C.int(i)) == 1
}
func (L *LuaState) SetSafeEnv(i int, b bool) {
	C.lua_setsafeenv(L.cptr(), C.int(i), func() C.int {
		if b {
			return C.int(1)
		}
		return C.int(0)
	}())
}
func (L *LuaState) SandboxThread() {
	C.luaL_sandboxthread(L.cptr())
}
func (L *LuaState) Sandbox() {
	C.luaL_sandbox(L.cptr())
}

func (state *LuaState) ResumeState(ref int, nArgs int) {
	state.Status = uint8(LUA_OK)
	state.Resume(state.MainThread(), nArgs)
	state.Unref(ref)
}

func (state *LuaState) ResumeWithError(ref int, errMsg string) {
	state.Status = uint8(LUA_OK)
	state.PushString(errMsg)
	state.Resume(state.MainThread(), 1)
	state.Unref(ref)
}

var structToTableCounter uint64

func StructToTable(L *LuaState, api any) {
	val := reflect.ValueOf(api)
	if !val.IsValid() {
		L.PushNil()
		return
	}

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			L.PushNil()
			return
		}

		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		L.PushNil()
		return
	}

	id := atomic.AddUint64(&structToTableCounter, 1)
	registryKey := fmt.Sprintf("luna__structtable_%d", id)

	L.NewTable()
	fillStructTable(L, val, registryKey, debugNameFromAPI(api))
}
func debugChildName(parent string, name string) string {
	if parent == "" {
		return name
	}
	if name == "" {
		return parent
	}
	return parent + "." + name
}
func debugNameFromAPI(api any) string {
	if api == nil {
		return ""
	}
	t := reflect.TypeOf(api)
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	if t.Name() == "" {
		return ""
	}
	return t.Name()
}

func (ls *LuaState) PushInstance(i int, ud unsafe.Pointer) int {
	tag := ls.UserDataTag(1)
	ogPayload := (*[2]uintptr)(ud)
	instance, control := ogPayload[0], ogPayload[1]

	payload := (*[2]uintptr)(ls.NewUserData(16, tag))

	if control != 0 {
		atomic.AddInt32((*int32)(unsafe.Pointer(control+8)), 1)
	}

	payload[0] = instance
	payload[1] = control

	if ls.GetMetaTable(1) != 0 {
		ls.SetMetaTable(-2)
	}
	return 1
}

func (ls *LuaState) PushRawInstance(ud unsafe.Pointer) int {
	ogPayload := (*[2]uintptr)(ud)
	instance, control := ogPayload[0], ogPayload[1]

	payload := (*[2]uintptr)(ls.NewUserData(16, 10))

	if control != 0 {
		atomic.AddInt32((*int32)(unsafe.Pointer(control+8)), 1)
	}

	payload[0] = instance
	payload[1] = control

	ls.GetGlobal("workspace")
	if ls.Type(-1) == LUA_TUSERDATA {
		if ls.GetMetaTable(-1) != 0 {
			ls.SetMetaTable(-3)
		}
	}

	ls.Pop(1)

	return 1
}
