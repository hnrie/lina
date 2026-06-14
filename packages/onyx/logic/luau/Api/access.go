package Api

/*
#include "lua.h"
*/
import "C"
import "unsafe"

func (s *LuaState) Type(idx int) int {
	return int(C.lua_type(s.cptr(), C.int(idx)))
}

func (s *LuaState) TypeName(tp int) string {
	return C.GoString(C.lua_typename(s.cptr(), C.int(tp)))
}

func (s *LuaState) IsNumber(idx int) bool {
	return C.lua_isnumber(s.cptr(), C.int(idx)) != 0
}

func (s *LuaState) IsString(idx int) bool {
	return C.lua_isstring(s.cptr(), C.int(idx)) != 0
}

func (s *LuaState) IsCFunction(idx int) bool {
	return C.lua_iscfunction(s.cptr(), C.int(idx)) != 0
}

func (s *LuaState) IsFunction(idx int) bool {
	return C.lua_type(s.cptr(), C.int(idx)) == LUA_TFUNCTION
}

func (s *LuaState) IsUserData(idx int) bool {
	return C.lua_isuserdata(s.cptr(), C.int(idx)) != 0
}

func (s *LuaState) IsNil(idx int) bool {
	return C.lua_type(s.cptr(), C.int(idx)) == C.LUA_TNIL
}

func (s *LuaState) Equal(idx1, idx2 int) bool {
	return C.lua_equal(s.cptr(), C.int(idx1), C.int(idx2)) != 0
}

func (s *LuaState) RawEqual(idx1, idx2 int) bool {
	return C.lua_rawequal(s.cptr(), C.int(idx1), C.int(idx2)) != 0
}

func (s *LuaState) LessThan(idx1, idx2 int) bool {
	return C.lua_lessthan(s.cptr(), C.int(idx1), C.int(idx2)) != 0
}

func (s *LuaState) ToInteger(idx int) int {
	return int(C.lua_tointegerx(s.cptr(), C.int(idx), nil))
}

func (s *LuaState) ToUnsigned(idx int) uint {
	return uint(C.lua_tounsignedx(s.cptr(), C.int(idx), nil))
}

func (s *LuaState) ObjLen(idx int) int {
	return int(C.lua_objlen(s.cptr(), C.int(idx)))
}

func (s *LuaState) ToUserData(idx int) unsafe.Pointer {
	return C.lua_touserdata(s.cptr(), C.int(idx))
}

func (s *LuaState) ToThread(idx int) *LuaState {
	return (*LuaState)(unsafe.Pointer(C.lua_tothread(s.cptr(), C.int(idx))))
}

func (s *LuaState) ToPointer(idx int) unsafe.Pointer {
	return unsafe.Pointer(C.lua_topointer(s.cptr(), C.int(idx)))
}
