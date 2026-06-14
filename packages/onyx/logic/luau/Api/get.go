package Api

/*
#include <stdbool.h>
#include "lua.h"
#include "lualib.h"
#include "lmem.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"
)

func (s *LuaState) GetTable(idx int) {
	C.lua_gettable(s.cptr(), C.int(idx))
}

func (s *LuaState) GetField(idx int, k string) {
	cName := C.CString(k)
	defer C.free(unsafe.Pointer(cName))
	C.lua_getfield(s.cptr(), C.int(idx), cName)
}

func (s *LuaState) GetGlobal(name string) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.lua_getfield(s.cptr(), C.int(C.LUA_GLOBALSINDEX), cName)
}

func (s *LuaState) RawGet(idx int) {
	C.lua_rawget(s.cptr(), C.int(idx))
}

func (s *LuaState) RawGetI(idx int, n int) {
	C.lua_rawgeti(s.cptr(), C.int(idx), C.int(n))
}

func (s *LuaState) CreateTable(narr, nrec int) {
	C.lua_createtable(s.cptr(), C.int(narr), C.int(nrec))
}

func (s *LuaState) NewUserData(sz uintptr, tag int) unsafe.Pointer {
	return C.lua_newuserdatatagged(s.cptr(), C.size_t(sz), C.int(tag))
}

func (s *LuaState) NewUserDataWithMetatable(sz uintptr, tag int) unsafe.Pointer {
	return C.lua_newuserdatataggedwithmetatable(s.cptr(), C.size_t(sz), C.int(tag))
}

func (s *LuaState) GetMetaTable(idx int) int {
	return int(C.lua_getmetatable(s.cptr(), C.int(idx)))
}

func (s *LuaState) GetMetaField(idx int, k string) int {
	cK := C.CString(k)
	defer C.free(unsafe.Pointer(cK))
	return int(C.luaL_getmetafield(s.cptr(), C.int(idx), cK))
}

func (ls *LuaState) UserDataTag(i int) int {
	return int(C.lua_userdatatag(ls.cptr(), C.int(i)))
}

func (ls *LuaState) DPCall(fn, fn2 unsafe.Pointer, i int) int {
	return int(C.luaD_pcall(ls.cptr(), (C.Pfunc)(fn), fn2, C.ptrdiff_t(ls.SaveStack(fn2)), C.ptrdiff_t(i)))
}

func (ls *LuaState) DCall(fn unsafe.Pointer, i int) {
	C.luaD_call(ls.cptr(), (C.StkId)(fn), C.int(i))
}

func (ls *LuaState) ExpandStackLimit(p *TValue) {
	if uintptr(unsafe.Pointer(p)) > uintptr(unsafe.Pointer(ls.StackLast)) {
		ls.Error("Stack overflow: p exceeds stack_last")
	}
	if uintptr(unsafe.Pointer(ls.Ci.Top)) < uintptr(unsafe.Pointer(p)) {
		ls.Ci.Top = p
	}
}

func (ls *LuaState) ClValue(index int) *Closure {
	if tObj := ls.Index2Addr(index); tObj != nil && tObj.Tt == LUA_TFUNCTION && tObj.Value != 0 {
		return (*Closure)(unsafe.Pointer(uintptr(tObj.Value)))
	}
	return nil
}

func (ls *LuaState) SaveStack(p unsafe.Pointer) int {
	return int(uintptr(p) - uintptr(unsafe.Pointer(ls.Stack)))
}

func (s *LuaState) GetFEnv(idx int) {
	C.lua_getfenv(s.cptr(), C.int(idx))
}

func (s *LuaState) GetRef(ref int) {
	C.lua_rawgeti(s.cptr(), C.LUA_REGISTRYINDEX, C.int(ref))
}

func stkIdAdd(ptr StkId, n int) StkId {
	return (*TValue)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(n)*unsafe.Sizeof(TValue{})))
}
func (L *LuaState) Pseudo2Addr(idx int) *TValue {
	switch idx {
	case LUA_REGISTRYINDEX:
		return &L.Global.Registry
	case LUA_ENVIRONINDEX:
		funcObj := L.Ci.Func
		cl := (*Closure)(unsafe.Pointer(uintptr(funcObj.Value)))
		env := cl.Env
		L.Global.Pseudotemp.Tt = LUA_TTABLE
		L.Global.Pseudotemp.Value = Value(uintptr(unsafe.Pointer(env)))
		return &L.Global.Pseudotemp
	case LUA_GLOBALSINDEX:
		L.Global.Pseudotemp.Tt = LUA_TTABLE
		L.Global.Pseudotemp.Value = Value(uintptr(unsafe.Pointer(L.Gt)))
		return &L.Global.Pseudotemp
	default:
		funcObj := L.Ci.Func
		cl := (*Closure)(unsafe.Pointer(uintptr(funcObj.Value)))
		targetIdx := LUA_GLOBALSINDEX - idx
		if cl.IsC == 1 {
			if targetIdx <= int(cl.NUpvalues) {
				return stkIdAdd(&cl.UpValues()[0], targetIdx-1)
			}
		}
		return nil
	}
}

func (L *LuaPage) GetPageWalkInfo() (start, end uintptr, busy, blockSize int) {
	var cStart *C.char
	var cEnd *C.char
	var cBusy C.int
	var cBlockSize C.int
	C.luaM_getpagewalkinfo((*C.lua_Page)(unsafe.Pointer(L)), &cStart, &cEnd, &cBusy, &cBlockSize)
	return uintptr(unsafe.Pointer(cStart)),
		uintptr(unsafe.Pointer(cEnd)),
		int(cBusy),
		int(cBlockSize)
}

func (L *LuaState) Index2Addr(idx int) *TValue {
	if idx > 0 {
		o := stkIdAdd(L.Base, idx-1)
		if uintptr(unsafe.Pointer(o)) >= uintptr(unsafe.Pointer(L.Top)) {
			return nil
		}
		return o
	} else if idx > LUA_REGISTRYINDEX {
		return stkIdAdd(L.Top, idx)
	} else {
		return L.Pseudo2Addr(idx)
	}
}

func (L *LuaState) ToObject(idx int) *TValue {
	return L.Index2Addr(idx)
}

func (L *Luau) PushThread(La *LuaState) C.int {
	return C.int(C.lua_pushthread(La.cptr()))
}

func (L *Luau) Ref(state *LuaState, index int) C.int {
	return C.int(C.lua_ref(state.cptr(), C.int(index)))
}

func TableToStruct[T any](ls *LuaState, idx int) (T, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	if ls.Type(idx) != LUA_TTABLE {
		return result, fmt.Errorf("expected table at index %d", idx)
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		name := field.Name
		if tag := field.Tag.Get("lua"); tag != "" {
			name = tag
		}
		ls.GetField(idx, name)
		switch fieldVal.Kind() {
		case reflect.String:
			if ls.IsString(-1) {
				fieldVal.SetString(ls.ToString(-1))
			}
		case reflect.Bool:
			fieldVal.SetBool(ls.ToBoolean(-1))
		case reflect.Int, reflect.Int32, reflect.Int64:
			fieldVal.SetInt(int64(ls.ToInteger(-1)))
		case reflect.Float32, reflect.Float64:
			fieldVal.SetFloat(ls.ToNumber(-1))
		case reflect.Map:
			if ls.Type(-1) == LUA_TTABLE {
				m := reflect.MakeMap(fieldVal.Type())
				ls.PushNil()
				for ls.Next(-2) {
					k := ls.ToString(-2)
					v := ls.ToString(-1)
					m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
					ls.Pop(1)
				}
				fieldVal.Set(m)
			}
		case reflect.Ptr:
			if ls.IsString(-1) {
				s := ls.ToString(-1)
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
				fieldVal.Elem().SetString(s)
			}
		}

		ls.Pop(1)
	}

	return result, nil
}

func (cl *Closure) GetUpval(index int) *TValue {
	return &cl.UpValues()[index]
}

func (ls *LuaState) CurrentClosure() *Closure {
	if ls.Ci == nil || ls.Ci.Func == nil {
		return nil
	}
	return (*Closure)(unsafe.Pointer(uintptr(ls.Ci.Func.Value)))
}

func (ls *LuaState) CheckType(arg int, expectedType int) {
	obj := ls.ToObject(arg)
	actualType := -1
	if obj != nil {
		actualType = int(obj.Tt)
	}
	if actualType != expectedType {
		ls.Error("bad argument #%d (%s expected, got %s)",
			arg,
			ls.TypeName(expectedType),
			ls.TypeName(actualType))
	}
}

func (ls *LuaState) CheckAny(arg int) {
	obj := ls.ToObject(arg)
	actualType := -1
	if obj != nil {
		actualType = int(obj.Tt)
	}
	if actualType == -1 {
		ls.Error("missing argument #%d", arg)
	}
}

func (s *LuaState) AbsIndex(idx int) int {
	if idx > 0 || idx <= LUA_REGISTRYINDEX {
		return idx
	}
	return s.GetTop() + idx + 1
}

func (s *LuaState) IsTable(idx int) bool {
	return s.Type(idx) == LUA_TTABLE
}

func (s *LuaState) IsNilOrNone(idx int) bool {
	return s.Type(idx) <= LUA_TNIL
}

func (s *LuaState) IsThread(idx int) bool {
	return s.Type(idx) == LUA_TTHREAD
}

func (s *LuaState) CheckString(arg int) string {
	if !s.IsString(arg) {
		s.TypeError(arg, "string")
	}
	return s.ToString(arg)
}

func (s *LuaState) CheckNumber(arg int) float64 {
	if !s.IsNumber(arg) {
		s.TypeError(arg, "number")
	}
	return s.ToNumber(arg)
}

func (s *LuaState) CheckInteger(arg int) int {
	if !s.IsNumber(arg) {
		s.TypeError(arg, "integer")
	}
	return s.ToInteger(arg)
}

func (s *LuaState) CheckBoolean(arg int) bool {
	if s.Type(arg) != LUA_TBOOLEAN {
		s.TypeError(arg, "boolean")
	}
	return s.ToBoolean(arg)
}

func (s *LuaState) CheckTable(arg int) {
	if s.Type(arg) != LUA_TTABLE {
		s.TypeError(arg, "table")
	}
}

func (s *LuaState) OptString(arg int, def string) string {
	if s.IsNilOrNone(arg) {
		return def
	}
	return s.CheckString(arg)
}

func (s *LuaState) OptNumber(arg int, def float64) float64 {
	if s.IsNilOrNone(arg) {
		return def
	}
	return s.CheckNumber(arg)
}

func (s *LuaState) OptInteger(arg int, def int) int {
	if s.IsNilOrNone(arg) {
		return def
	}
	return s.CheckInteger(arg)
}

func (s *LuaState) GetInfo(level int, desc string, a *LuaDebug) bool {
	cDesc := C.CString(desc)
	defer C.free(unsafe.Pointer(cDesc))

	var cDebug C.lua_Debug

	result := C.lua_getinfo(s.cptr(), C.int(level), cDesc, &cDebug)

	if result == 0 {
		return false
	}

	if a != nil {
		if cDebug.name != nil {
			a.Name = C.GoString(cDebug.name)
		}
		if cDebug.what != nil {
			a.What = C.GoString(cDebug.what)
		}
		if cDebug.source != nil {
			a.Source = C.GoString(cDebug.source)
		}
		if cDebug.short_src != nil {
			a.ShortSrc = C.GoString(cDebug.short_src)
		}
		a.LineDefined = int(cDebug.linedefined)
		a.CurrentLine = int(cDebug.currentline)
		a.NUpvals = uint8(cDebug.nupvals)
		a.NParams = uint8(cDebug.nparams)
		a.IsVarArg = int8(cDebug.isvararg)
		a.UserData = unsafe.Pointer(cDebug.userdata)
	}

	return true
}

func (s *LuaState) OptBoolean(arg int, def bool) bool {
	if s.IsNilOrNone(arg) {
		return def
	}
	return s.CheckBoolean(arg)
}

func (s *LuaState) RawGetField(idx int, k string) {
	s.PushString(k)
	s.RawGet(idx - 1)
}

func (s *LuaState) SetTableString(idx int, k, v string) {
	s.PushString(k)
	s.PushString(v)
	s.SetTable(idx - 2)
}

func (s *LuaState) SetTableInt(idx int, k string, v int) {
	s.PushString(k)
	s.PushInteger(v)
	s.SetTable(idx - 2)
}

func (s *LuaState) SetTableBool(idx int, k string, v bool) {
	s.PushString(k)
	s.PushBoolean(v)
	s.SetTable(idx - 2)
}

func (s *LuaState) ForEach(idx int, fn func()) {
	idx = s.AbsIndex(idx)
	s.PushNil()
	for s.Next(idx) {
		fn()
		s.Pop(1)
	}
}

func (s *LuaState) GetUpvalue(funcIdx, upvalIdx int) *TValue {
	cl := s.ToObject(funcIdx).ClValue()
	if cl == nil || int(cl.NUpvalues) <= upvalIdx {
		return nil
	}
	return &cl.UpValues()[upvalIdx]
}

func (s *LuaState) IsLClosure(idx int) bool {
	obj := s.ToObject(idx)
	if obj == nil || obj.Tt != int32(LUA_TFUNCTION) {
		return false
	}
	cl := obj.ClValue()
	return cl != nil && cl.IsC == 0
}

func (s *LuaState) IsCClosure(idx int) bool {
	obj := s.ToObject(idx)
	if obj == nil || obj.Tt != int32(LUA_TFUNCTION) {
		return false
	}
	cl := obj.ClValue()
	return cl != nil && cl.IsC == 1
}

func (s *LuaState) ObjectType(idx int) int32 {
	o := s.ToObject(idx)
	if o == nil {
		return int32(LUA_TNIL)
	}
	return o.Tt
}

func (s *LuaState) IsDeadKey(idx int) bool {
	o := s.ToObject(idx)
	return o != nil && o.Tt == int32(LUA_TDEADKEY)
}

func (s *LuaState) Clone(H *LuaTable) *LuaTable {
	return (*LuaTable)(unsafe.Pointer(C.luaH_clone(s.cptr(), (*C.LuaTable)(unsafe.Pointer(H)))))
}
