package Api

/*
#include "lua.h"
#include "lualib.h"
#include <stdlib.h>

void trigger_luna_error_bridge();

*/
import "C"
import (
	"fmt"
	"unsafe"
)

func (s *LuaState) PushNumber(n float64) {
	C.lua_pushnumber(s.cptr(), C.double(n))
}

func (s *LuaState) PushBoolean(b bool) {
	val := C.int(0)
	if b {
		val = 1
	}
	C.lua_pushboolean(s.cptr(), val)
}

func (s *LuaState) PushString(str string) {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	C.lua_pushstring(s.cptr(), cStr)
}

func (s *LuaState) PushNil() {
	C.lua_pushnil(s.cptr())
}

func (s *LuaState) PushInteger(n int) {
	C.lua_pushinteger(s.cptr(), C.int(n))
}

func (s *LuaState) PushUnsigned(n uint) {
	C.lua_pushunsigned(s.cptr(), C.uint(n))
}

func (s *LuaState) PushLString(str string) {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	C.lua_pushlstring(s.cptr(), cStr, C.size_t(len(str)))
}

func (s *LuaState) PushLightUserData(p unsafe.Pointer) {
	C.lua_pushlightuserdatatagged(s.cptr(), p, 0)
}

func (s *LuaState) PushThread() int {
	return int(C.lua_pushthread(s.cptr()))
}

func (s *LuaState) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))
	C.luaL_where(s.cptr(), 1)
	C.lua_pushstring(s.cptr(), cMessage)
	C.lua_concat(s.cptr(), 2)
	C.trigger_luna_error_bridge()
	panic(message)
}

func (ls *LuaState) ArgError(arg int, extraMsg string) {
	ls.Error(fmt.Sprintf("invalid argument #%d (%s)", arg, extraMsg))
}

func (ls *LuaState) TypeError(arg int, expectedType string) {
	actualType := ls.TypeName(ls.Type(arg))
	ls.Error(fmt.Sprintf("invalid argument #%d (expected %s, got %s)", arg, expectedType, actualType))
}

func (s *LuaState) ToNumber(index int) float64 {
	return float64(C.lua_tonumberx(s.cptr(), C.int(index), nil))
}

func (s *LuaState) ToBoolean(index int) bool {
	return C.lua_toboolean(s.cptr(), C.int(index)) != 0
}

func (s *LuaState) ToString(index int) string {
	return C.GoString(C.lua_tolstring(s.cptr(), C.int(index), nil))
}

func (s *LuaState) Pop(n int) {
	C.lua_settop(s.cptr(), C.int(-n-1))
}

func (ls *LuaState) PushClosure(cl *Closure) {
	ls.Top.Value = Value(uintptr(unsafe.Pointer(cl)))
	ls.Top.Tt = int32(LUA_TFUNCTION)
	ls.Top = (StkId)(unsafe.Add(unsafe.Pointer(ls.Top), unsafe.Sizeof(TValue{})))
}
