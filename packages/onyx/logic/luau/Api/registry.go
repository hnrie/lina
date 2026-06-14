package Api

/*
#include "lua.h"
#include "luacode.h"
#include "lualib.h"

int go_lua_callback(lua_State* L, void* ud);
void push_c_trampoline(lua_State* L, void* ud);
void setup_go_callback(void* fn_ptr);
void luna_register_function(lua_State* L, const char* name);
*/
import "C"
import (
	"bytes"
	"fmt"
	"reflect"
	"runtime/cgo"
	"strings"
	"sync"
	"unsafe"

	. "main/packages/onyx/logic"

	"main/packages/api/managers/logs"
)

const (
	TCClosure                     ClosureType   = 0
	TLClosure                     ClosureType   = 1
	NewCClosure                   ClosureType   = 2
	Execute                       ExecutionType = 0
	Yield                         ExecutionType = 1
	RegisterLibrary               ExecutionType = 2
	TASK_AUTH                     string        = "RTC-AUTH"
	AUTHED                        string        = "CTR-AUTHED"
	EXECUTE                       string        = "CTR-EXECUTE"
	COMPILE                       string        = "RTC-COMPILE"
	OP_CODE_TELEPORTED                          = 0x01
	OP_CODE_GAME_JOINED                         = 0x02
	OP_CODE_EXECUTION_ALLOWED                   = 0x03
	OP_CODE_EXECUTION_NOT_ALLOWED               = 0x04
	OP_CODE_SUCCESFUL_EXECUTION                 = 0x05
)

type (
	ClosureType   int
	ExecutionType int
	Luau          struct {
		Sesh              *Sesh
		Shared            *Module
		RobloxGlobalState *LuaState
		LunaState         *LuaState
		ExecutionChannel  Queue[Yieldable]
		Error             uintptr
		ScriptContext     uintptr
		QueuedList        []Yieldable
	}
	Yieldable struct {
		Source   []byte
		Type     ExecutionType
		Yield    YieldData
		Register RegisterData
	}
	YieldData struct {
		Thread    *LuaState
		Arguments int
		IsError   bool
		ErrorMsg  string
	}
	RegisterData struct {
		Name  string
		Func  func(*LuaState) uintptr
		IsCpp bool
		Cpp   unsafe.Pointer
	}
	Module struct {
		Pipe    [256]byte
		Dllpath [256]byte
	}
	Response struct {
		Type     string `json:"type"`
		Status   string `json:"status"`
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}
)

var (
	executionMu sync.Mutex
	Api         = Luau{}
	ExecuteCode = func(source []byte) {
		if Api.LunaState == nil {
			return
		}

		execution_state := Api.LunaState.NewThread()
		Api.LunaState.Pop(1)

		execution_state.SandboxThread()
		execution_state.PushValue(LUA_GLOBALSINDEX)
		execution_state.SetGlobal("_ENV")

		if execution_state.Userdata != nil {
			execution_state.Userdata.Identity = 8
			execution_state.Userdata.Capabilities = IdentityToCapabilities(8, true)
		}

		if status := execution_state.Load("@luna", source, 0); status != nil {
			Print(3, "%v", status.Error())
			execution_state.Pop(1)
		} else {

			execution_state.SetCaps(8)

			execution_state.GetGlobal("task")
			execution_state.GetField(-1, "spawn")
			execution_state.Insert(-3)
			execution_state.Pop(1)

			if err := execution_state.PCall(1, 0); err != nil {
				Print(3, "%v", err.Error())
				execution_state.Pop(1)
			}
		}
	}
)

func (p *Luau) NullTerm() (path, pipe string) {
	defer func() { recover() }()
	if p == nil {
		return "", ""
	}
	bytesToString := func(b []byte) string {
		n := bytes.IndexByte(b, 0)
		if n == -1 {
			n = len(b)
		}
		return string(b[:n])
	}
	path, pipe = bytesToString(p.Shared.Dllpath[:]), bytesToString(p.Shared.Pipe[:])
	return path, pipe
}

func wrapInt(name string, fn func(*LuaState) int) func(*LuaState) uintptr {
	return func(state *LuaState) (ret uintptr) {
		logs.Log("called %s", name)
		defer func() {
			if r := recover(); r != nil {
				state.PushNil()
				state.PushString(fmt.Sprintf("%v", r))
				ret = 2
			}
		}()
		return uintptr(fn(state))
	}
}

func wrapUintptr(name string, fn func(*LuaState) uintptr) func(*LuaState) uintptr {
	return func(state *LuaState) (ret uintptr) {
		logs.Log("called %s", name)
		defer func() {
			if r := recover(); r != nil {
				state.PushNil()
				state.PushString(fmt.Sprintf("%v", r))
				ret = 2
			}
		}()
		return fn(state)
	}
}

func pushValue(L *LuaState, val reflect.Value, registryKey, debugName string) {
	if !val.IsValid() {
		L.PushNil()
		return
	}

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			L.PushNil()
			return
		}

		pushValue(L, val.Elem(), registryKey, debugName)
		return
	}

	switch val.Kind() {
	case reflect.Func:
		switch fn := val.Interface().(type) {
		case func(*LuaState) int:
			L.RegisterFunction(registryKey, wrapInt(debugName, fn))
			L.GetGlobal(registryKey)

		case func(*LuaState) uintptr:
			L.RegisterFunction(registryKey, wrapUintptr(debugName, fn))
			L.GetGlobal(registryKey)

		default:
			L.PushNil()
		}

	case reflect.String:
		L.PushString(val.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		L.PushInteger(int(val.Int()))

	case reflect.Bool:
		L.PushBoolean(val.Bool())

	case reflect.Map:
		L.NewTable()

		iter := val.MapRange()
		for iter.Next() {
			k := fmt.Sprintf("%v", iter.Key().Interface())

			L.PushString(k)
			pushValue(
				L,
				iter.Value(),
				registryKey+"_"+k,
				debugChildName(debugName, k),
			)
			L.SetTable(-3)
		}

	case reflect.Struct:
		L.NewTable()
		fillStructTable(L, val, registryKey, debugName)

	default:
		L.PushNil()
	}
}

func parseNames(field reflect.StructField) []string {
	var names []string
	if luaTag := field.Tag.Get("lua"); luaTag != "" {
		for _, name := range strings.Split(luaTag, ",") {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				names = append(names, trimmed)
			}
		}
	} else {
		names = append(names, field.Name)
	}
	if aliasTag := field.Tag.Get("alias"); aliasTag != "" {
		for _, alias := range strings.Split(aliasTag, ",") {
			if trimmed := strings.TrimSpace(alias); trimmed != "" {
				names = append(names, trimmed)
			}
		}
	}
	return names
}

func setNestedField(L *LuaState, tableIndex int, path string) {
	parts := strings.Split(path, ".")
	absTableIndex := tableIndex
	if tableIndex < 0 && tableIndex > -10000 {
		absTableIndex = L.GetTop() + tableIndex + 1
	}
	valIndex := L.GetTop()
	L.PushValue(absTableIndex)
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		L.PushString(parts[i])
		L.GetTable(-2)
		if L.Type(-1) == 0 {
			L.Pop(1)
			L.NewTable()
			L.PushString(parts[i])
			L.PushValue(-2)
			L.SetTable(-4)
		}
		L.Remove(-2)
	}
	L.PushString(parts[len(parts)-1])
	L.PushValue(valIndex)
	L.SetTable(-3)
	L.Pop(1)
	L.Remove(valIndex)
}

func fillStructTable(L *LuaState, val reflect.Value, prefix, debugPrefix string) {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			fillStructTable(L, fieldVal, prefix, debugPrefix)
			continue
		}
		names := parseNames(field)
		safeName := strings.ReplaceAll(names[0], ".", "_")
		registryKey := prefix + "__" + safeName
		fieldDebugName := debugChildName(debugPrefix, names[0])
		pushValue(L, fieldVal, registryKey, fieldDebugName)
		for _, n := range names {
			L.PushValue(-1)
			setNestedField(L, -3, n)
		}
		L.Pop(1)
	}
}

func Register(L *LuaState, api any) {
	val := reflect.ValueOf(api)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)
		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			Register(L, fieldVal.Interface())
			continue
		}
		names := parseNames(field)
		safeName := strings.ReplaceAll(names[0], ".", "_")
		registryKey := "luna__" + safeName
		pushValue(L, fieldVal, registryKey, names[0])
		for _, n := range names {
			L.PushValue(-1)
			setNestedField(L, LUA_GLOBALSINDEX, n)
		}
		L.Pop(1)
	}
}

//export go_lua_callback
func go_lua_callback(L *C.lua_State, ud unsafe.Pointer) (ret C.int) {
	h := cgo.Handle(ud)
	fn := h.Value().(func(*LuaState) int)
	defer func() {
		if r := recover(); r != nil {
			ls := (*LuaState)(unsafe.Pointer(L))
			ls.PushNil()
			ls.PushString(fmt.Sprint(r))
			ret = 2
		}
	}()
	ret = C.int(fn((*LuaState)(unsafe.Pointer(L))))
	return
}

//export free_go_handle
func free_go_handle(ud unsafe.Pointer) {
	h := cgo.Handle(ud)
	h.Delete()
}

func (p *LuaState) IsDead(gco *GCObject) bool {
	const (
		WHITE0BIT = 0
		WHITE1BIT = 1
		BLACKBIT  = 2
		WHITEBITS = (1 << WHITE0BIT) | (1 << WHITE1BIT)
	)
	otherWhite := p.Global.Currentwhite ^ WHITEBITS
	return (gco.Marked & otherWhite & WHITEBITS) != 0
}

func (p *LuaState) CloneFunction(arg int) {
	C.lua_clonefunction(p.cptr(), C.int(arg))
}

func (p *LuaState) CloneCFunction(arg int) {
	C.lua_clonecfunction(p.cptr(), C.int(arg))
}

func RegisterDebugTable(L *LuaState, d any, tableIdx int) {
	L.PushValue(tableIdx)

	infoVal := reflect.ValueOf(d)
	infoType := infoVal.Type()

	for i := 0; i < infoVal.NumField(); i++ {
		field := infoType.Field(i)
		fieldVal := infoVal.Field(i)

		names := parseNames(field)
		if len(names) == 0 || !fieldVal.CanInterface() {
			continue
		}

		fn, ok := fieldVal.Interface().(func(*LuaState) int)
		if !ok {
			continue
		}

		L.PushGoFunction("debug."+names[0], fn)
		L.SetField(-2, names[0])

		for _, alias := range getAliases(field) {
			if alias != names[0] {
				L.PushValue(-1)
				L.SetField(-3, alias)
			}
		}
	}

	L.Pop(1)
}

func getAliases(field reflect.StructField) []string {
	var aliases []string
	if aliasTag := field.Tag.Get("alias"); aliasTag != "" {
		for _, a := range strings.Split(aliasTag, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				aliases = append(aliases, trimmed)
			}
		}
	}
	return aliases
}

func (L *LuaState) PushGoFunction(name string, fn func(*LuaState) int) {
	L.RegisterFunction("__temp_debug_fn", wrapInt("debug_temp", fn))
	L.GetGlobal("__temp_debug_fn")
	L.PushNil()
	L.SetGlobal("__temp_debug_fn")
}
