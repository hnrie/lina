package closures

/*
void bridge_push_newcclosure(void* state);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	. "main/packages/onyx/logic/luau/Api"

	"golang.org/x/sys/windows"
)

type ClosureError struct {
	Op  string
	Err string
}

func (e ClosureError) Error() string {
	return fmt.Sprintf("closure: %s: %s", e.Op, e.Err)
}

type LuaMiscOps struct {
	RestoreFunction   func(*LuaState) int `lua:"restorefunction"`
	HookFunction      func(*LuaState) int `lua:"hookfunction" alias:"replaceclosure,hookfunc"`
	NewCClosure       func(*LuaState) int `lua:"newcclosure"`
	HookMetaMethod    func(*LuaState) int `lua:"hookmetamethod"`
	CloneFunction     func(*LuaState) int `lua:"clonefunction"`
	IsCClosure        func(*LuaState) int `lua:"iscclosure"`
	IsLClosure        func(*LuaState) int `lua:"islclosure"`
	CloneRef          func(*LuaState) int `lua:"cloneref" alias:"clonereference"`
	CheckCaller       func(*LuaState) int `lua:"checkcaller"`
	GetNamecallMethod func(*LuaState) int `lua:"getnamecallmethod"`
	SetnamecallMethod func(*LuaState) int `lua:"setnamecallmethod"`
	SetReadOnly       func(*LuaState) int `lua:"setreadonly"`
	IsReadOnly        func(*LuaState) int `lua:"isreadonly"`
}

type HookInfo struct {
	Hook          *Closure
	OriginalCCF   LuaCFunction
	OriginalLData *ClosureData
}

type ClosureData struct {
	Env       *LuaTable
	Stacksize uint8
	Proto     *Proto
	Uprefs    []TValue
}

type ClosureCache[V any] struct {
	mu    sync.RWMutex
	cache map[*Closure]V
}

func (c *ClosureCache[V]) Get(key *Closure) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	v, ok := c.cache[key]
	return v, ok
}

func (c *ClosureCache[V]) Set(key *Closure, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = value
}

func (c *ClosureCache[V]) Delete(key *Closure) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.cache, key)
}

func NewClosureCache[V any]() *ClosureCache[V] {
	return &ClosureCache[V]{
		cache: make(map[*Closure]V),
	}
}

type HookDispatcher struct {
	hooks        *ClosureCache[*HookInfo]
	callbackOnce sync.Once
	callback     LuaCFunction
}

func NewHookDispatcher() *HookDispatcher {
	return &HookDispatcher{
		hooks: NewClosureCache[*HookInfo](),
	}
}

func (h *HookDispatcher) getCallback() LuaCFunction {
	h.callbackOnce.Do(func() {
		h.callback = LuaCFunction(unsafe.Pointer(windows.NewCallbackCDecl(h.dispatcher)))
	})
	return h.callback
}

func (h *HookDispatcher) dispatcher(ls *LuaState) uintptr {
	var Current *Closure = ls.CurrentClosure()
	
	if Current == nil {
		ls.Error("hook dispatcher: current closure is nil")
		return 0
	}

	HookInfo, Ok := h.hooks.Get(Current)
	if !Ok {
		ls.Error("hook dispatcher: hook target not found")
		return 0
	}

	ls.PushClosure(HookInfo.Hook)
	ls.Insert(1)

	var Nargs int = ls.GetTop() - 1
	var BaseBefore int = ls.GetTop() - Nargs

	ls.Call(Nargs, LUA_MULTRET)

	return uintptr(ls.GetTop() - BaseBefore + 1)
}

type ClosureOps struct {
	hookDispatcher *HookDispatcher
	newCCache      *ClosureCache[*Closure]
	executorCache  *ClosureCache[bool]
}

func NewClosureOps() *ClosureOps {
	return &ClosureOps{
		hookDispatcher: NewHookDispatcher(),
		newCCache:      NewClosureCache[*Closure](),
		executorCache:  NewClosureCache[bool](),
	}
}

func (c *ClosureOps) Init(L *LuaState) {
	Register(L, LuaMiscOps{
		RestoreFunction:   c.RestoreFunction,
		CloneRef:          c.CloneRef,
		HookFunction:      c.HookFunction,
		HookMetaMethod:    c.HookMetaMethod,
		NewCClosure:       c.NewCClosure,
		CloneFunction:     c.CloneFunction,
		IsCClosure:        c.IsCClosure,
		IsLClosure:        c.IsLClosure,
		CheckCaller:       c.CheckCaller,
		GetNamecallMethod: c.GetNamecallMethod,
		SetnamecallMethod: c.SetNamecallMethod,
		SetReadOnly:       c.SetReadOnly,
		IsReadOnly:        c.IsReadOnly,
	})
}

func (c *ClosureOps) RestoreFunction(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)

	var Function *Closure = ls.ToObject(1).ClValue()
	
	if Function == nil {
		ls.Error(ClosureError{Op: "restorefunction", Err: "invalid closure"}.Error())
		return 0
	}

	HookInfo, Ok := c.hookDispatcher.hooks.Get(Function)
	
	if !Ok {
		ls.Error(ClosureError{Op: "restorefunction", Err: "function is not hooked"}.Error())
		return 0
	}

	if HookInfo.OriginalCCF != 0 {
		Function.AsC().F = HookInfo.OriginalCCF
		c.hookDispatcher.hooks.Delete(Function)
		return 0
	}

	if HookInfo.OriginalLData != nil {
		var OrigData *ClosureData = HookInfo.OriginalLData
		
		Function.Env = OrigData.Env
		Function.Stacksize = OrigData.Stacksize
		Function.AsL().P = OrigData.Proto

		copy(Function.UpValues(), OrigData.Uprefs)

		c.hookDispatcher.hooks.Delete(Function)
		return 0
	}

	ls.Error(ClosureError{Op: "restorefunction", Err: "invalid hook data"}.Error())
	return 0
}

func (c *ClosureOps) CloneRef(ls *LuaState) int {
	ls.CheckType(1, LUA_TUSERDATA)
	return ls.PushInstance(1, ls.ToUserData(1))
}

func (c *ClosureOps) HookFunction(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)
	ls.CheckType(2, LUA_TFUNCTION)

	var Target *Closure = ls.ToObject(1).ClValue()
	var Hook *Closure = ls.ToObject(2).ClValue()

	if ClosureTt(Target) != 0 {
		ls.CloneCFunction(1)
	} else {
		ls.CloneFunction(1)
	}

	var TtT int = ClosureTt(Target)

	if TtT != 0 {
		c.hookDispatcher.hooks.Set(Target, &HookInfo{
			Hook:        Hook,
			OriginalCCF: Target.AsC().F,
		})
		Target.AsC().F = c.hookDispatcher.getCallback()
	} else {
		c.patchLClosure(Target, Hook)
	}

	return 1
}

func (c *ClosureOps) patchLClosure(target, hook *Closure) {
	var OriginalData *ClosureData = &ClosureData{
		Env: target.Env, Stacksize: target.Stacksize, Proto: target.AsL().P,
		Uprefs: make([]TValue, target.NUpvalues),
	}
	
	copy(OriginalData.Uprefs, target.UpValues())
	c.hookDispatcher.hooks.Set(target, &HookInfo{Hook: hook, OriginalLData: OriginalData})

	if ClosureTt(hook) == 0 {
		target.Env = hook.Env
		target.Stacksize = hook.Stacksize
		target.AsL().P = hook.AsL().P
		copy(target.UpValues(), hook.UpValues())
	} else {
		target.IsC = 1
		target.AsC().F = c.hookDispatcher.getCallback()
		for i := 0; i < int(target.NUpvalues); i++ {
			if i < int(hook.NUpvalues) {
				*target.GetUpval(i) = *hook.GetUpval(i)
			}
		}
	}
}

func (c *ClosureOps) copyClosureUpvalues(ls *LuaState, function, hook *Closure, functionNups, hookNups int) int {

	var OriginalData *ClosureData = &ClosureData{
		Env:       function.Env,
		Stacksize: function.Stacksize,
		Proto:     function.AsL().P,
		Uprefs:    make([]TValue, functionNups),
	}
	
	copy(OriginalData.Uprefs, function.UpValues()[:functionNups])

	var HInfo *HookInfo = &HookInfo{
		Hook:          hook,
		OriginalLData: OriginalData,
	}
	
	c.hookDispatcher.hooks.Set(function, HInfo)

	function.Env = hook.Env
	function.Stacksize = hook.Stacksize
	function.AsL().P = hook.AsL().P

	for i := 0; i < functionNups; i++ {
		if i < hookNups {
			*function.GetUpval(i) = *hook.GetUpval(i)
		} else {
			*function.GetUpval(i) = TValue{Tt: LUA_TNIL}
		}
	}

	return 1
}

func (c *ClosureOps) hookLWithCClosure(ls *LuaState, function, hook *Closure) int {
	OriginalLClosure, IsFromLClosure := c.newCCache.Get(hook)

	var LHookNups int
	var HookToUse *Closure

	if IsFromLClosure {
		LHookNups = int(OriginalLClosure.NUpvalues)
		HookToUse = OriginalLClosure
	} else {
		ls.Error(ClosureError{
			Op:  "hookfunction",
			Err: "cannot hook Lua function with raw C closure (not created via newcclosure) - use newcclosure first",
		}.Error())
		return 0
	}

	var FunctionNups int = int(function.NUpvalues)

	var OrigData *ClosureData = &ClosureData{
		Env:       function.Env,
		Stacksize: function.Stacksize,
		Proto:     function.AsL().P,
		Uprefs:    make([]TValue, FunctionNups),
	}
	
	copy(OrigData.Uprefs, function.UpValues()[:FunctionNups])

	var HInfo *HookInfo = &HookInfo{
		Hook:          HookToUse,
		OriginalLData: OrigData,
	}
	
	c.hookDispatcher.hooks.Set(function, HInfo)

	function.Env = HookToUse.Env
	function.Stacksize = HookToUse.Stacksize
	function.AsL().P = HookToUse.AsL().P

	for i := 0; i < FunctionNups; i++ {
		if i < LHookNups {
			*function.GetUpval(i) = *HookToUse.GetUpval(i)
		} else {
			*function.GetUpval(i) = TValue{Tt: LUA_TNIL}
		}
	}

	return 1
}

func (c *ClosureOps) HookMetaMethod(ls *LuaState) int {
	ls.CheckAny(1)
	ls.CheckType(2, LUA_TSTRING)
	ls.CheckType(3, LUA_TFUNCTION)

	var Method string = ls.ToString(2)

	if ls.GetMetaTable(1) == 0 {
		ls.PushNil()
		return 1
	}

	var Mt int = ls.GetTop()
	var OldReadOnly bool = ls.GetReadOnly(Mt)

	ls.GetField(Mt, Method)
	
	if ls.IsNil(-1) {
		ls.Pop(2)
		ls.PushNil()
		return 1
	}

	var OldFuncIndex int = ls.GetTop()

	ls.SetReadOnly(Mt, false)
	ls.PushValue(3)
	ls.SetField(Mt, Method)
	ls.SetReadOnly(Mt, OldReadOnly)

	ls.PushValue(OldFuncIndex)
	return 1
}

func (c *ClosureOps) NewCClosure(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)

	var Function *Closure = ls.ToObject(1).ClValue()

	if ClosureTt(Function) != 0 {
		ls.PushValue(1)
		return 1
	}

	ls.PushValue(1)
	C.bridge_push_newcclosure(unsafe.Pointer(ls.Cptr()))
	
	var NewCC *Closure = ls.ToObject(-1).ClValue()
	ls.Ref(-1)

	c.newCCache.Set(NewCC, Function)

	return 1
}

func (c *ClosureOps) CloneFunction(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)

	var Original *Closure = ls.ToObject(1).ClValue()
	var Tt int = ClosureTt(Original)

	if LClosureTarget, Ok := c.newCCache.Get(Original); Ok {
		ls.PushClosure(LClosureTarget)
		C.bridge_push_newcclosure(unsafe.Pointer(ls.Cptr()))
		var Clone *Closure = ls.ToObject(-1).ClValue()
		c.newCCache.Set(Clone, LClosureTarget)
		return 1
	}

	if Tt == 1 {
		ls.CloneCFunction(1)
		return 1
	}

	if Tt == 0 {
		ls.CloneFunction(1)
		var Clone *Closure = ls.ToObject(-1).ClValue()
		c.executorCache.Set(Clone, true)
		return 1
	}

	ls.Error(ClosureError{Op: "clonefunction", Err: "invalid closure type"}.Error())
	return 0
}

func (c *ClosureOps) IsCClosure(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)
	ls.PushBoolean(ls.IsCClosure(1))
	return 1
}

func (c *ClosureOps) IsLClosure(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)
	ls.PushBoolean(ls.IsLClosure(1))
	return 1
}

func (c *ClosureOps) CheckCaller(ls *LuaState) int {
	if ls.Userdata != nil {
		ls.PushBoolean(ls.Userdata.Source.Expired())
		return 1
	}
	ls.PushBoolean(false)
	return 1
}

func (c *ClosureOps) GetNamecallMethod(ls *LuaState) int {
	if ls.Namecall != nil {
		ls.PushString(C.GoStringN((*C.char)(unsafe.Pointer(&ls.Namecall.Data)), C.int(ls.Namecall.Len)))
		return 1
	}
	ls.PushNil()
	return 1
}

func (c *ClosureOps) SetNamecallMethod(ls *LuaState) int {
	if ls.Namecall != nil {
		ls.Namecall = ls.ToObject(1).TsValue()
	}
	return 1
}

func (c *ClosureOps) SetReadOnly(ls *LuaState) int {
	ls.CheckType(1, LUA_TTABLE)
	ls.CheckType(2, LUA_TBOOLEAN)
	ls.SetReadOnly(1, ls.ToBoolean(2))
	return 0
}

func (c *ClosureOps) IsReadOnly(ls *LuaState) int {
	ls.CheckType(1, LUA_TTABLE)
	ls.PushBoolean(ls.GetReadOnly(1))
	return 1
}

func Init(L *LuaState) {
	var Ops *ClosureOps = NewClosureOps()
	Ops.Init(L)
}