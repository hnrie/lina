package Api

/*
#include "lua.h"
#include "lualib.h"
*/
import "C"
import (
	"fmt"
	"math"
	"unsafe"
)

func (s *LuaState) cptr() *C.lua_State {
	return (*C.lua_State)(unsafe.Pointer(s))
}

func (s *LuaState) Cptr() *C.lua_State {
	return (*C.lua_State)(unsafe.Pointer(s))
}

func NewState() (*LuaState, error) {
	L := C.luaL_newstate()
	if L == nil {
		return nil, fmt.Errorf("failed to allocate memory for Luau state")
	}
	return (*LuaState)(unsafe.Pointer(L)), nil
}

func (s *LuaState) Close() {
	if s != nil {
		C.lua_close(s.cptr())
		s = nil
	}
}

func (o *TValue) Ttype() int32 {
	return o.Tt
}
func (o *TValue) GcValue() *GCObject {
	return (*GCObject)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) PValue() unsafe.Pointer {
	return unsafe.Pointer(uintptr(o.Value))
}
func (o *TValue) NValue() float64 {
	return math.Float64frombits(uint64(o.Value))
}
func (o *TValue) VValue() *[3]float32 {
	return (*[3]float32)(unsafe.Pointer(&o.Value))
}
func (o *TValue) TsValue() *TString {
	return (*TString)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) UValue() *Udata {
	return (*Udata)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) ClValue() *Closure {
	return (*Closure)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) HValue() *LuaTable {
	return (*LuaTable)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) BValue() bool {
	return int32(o.Value) != 0
}
func (o *TValue) ThValue() *LuaState {
	return (*LuaState)(unsafe.Pointer(uintptr(o.Value)))
}
func (o *TValue) BufValue() unsafe.Pointer {
	return unsafe.Pointer(uintptr(o.Value))
}
func (o *TValue) UpValue() *UpVal {
	return (*UpVal)(unsafe.Pointer(uintptr(o.Value)))
}

const LUAU_BLACKBIT = 2

func (L *LuaState) IsBlack() bool {
	return (L.Marked & (1 << LUAU_BLACKBIT)) != 0
}
func (L *LuaState) BarrierBack(o *GCObject, gclist **GCObject) {
	C.luaC_barrierback(
		L.cptr(),
		(*C.GCObject)(unsafe.Pointer(o)),
		(**C.GCObject)(unsafe.Pointer(gclist)),
	)
}
func (L *LuaState) ThreadBarrier() {
	if L.IsBlack() {
		L.BarrierBack((*GCObject)(unsafe.Pointer(L)), &L.Gclist)
	}
}
