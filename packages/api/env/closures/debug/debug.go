package debug

import (
	"strings"
	"unsafe"

	. "main/packages/onyx/logic/luau/Api"
)

type Info struct {
	GetInfo      func(*LuaState) int `lua:"debug.getinfo"`
	GetRegistry  func(*LuaState) int `lua:"debug.getregistry" alias:"debug.getreg"`
	GetUpvalues  func(*LuaState) int `lua:"debug.getupvalues"`
	GetUpvalue   func(*LuaState) int `lua:"debug.getupvalue"`
	SetUpvalue   func(*LuaState) int `lua:"debug.setupvalue"`
	GetConstants func(*LuaState) int `lua:"debug.getconstants"`
	GetConstant  func(*LuaState) int `lua:"debug.getconstant"`
	SetConstant  func(*LuaState) int `lua:"debug.setconstant"`
	GetProto     func(*LuaState) int `lua:"debug.getproto"`
	GetProtos    func(*LuaState) int `lua:"debug.getprotos"`
	GetStack     func(*LuaState) int `lua:"debug.getstack"`
	SetStack     func(*LuaState) int `lua:"debug.setstack"`
}

func pushTValue(Ls *LuaState, Val *TValue) {
	Ls.Top.Value = Val.Value
	Ls.Top.Extra = Val.Extra
	Ls.Top.Tt = Val.Tt
	Ls.Top = (StkId)(unsafe.Add(unsafe.Pointer(Ls.Top), unsafe.Sizeof(TValue{})))
}

func getFunction(Ls *LuaState, Arg int, AllowC bool) *Closure {
	if Ls.IsFunction(Arg) {
		var Cl *Closure = Ls.ToObject(Arg).ClValue()
		if !AllowC && Cl.IsC == 1 {
			Ls.ArgError(Arg, "luau function expected")
			return nil
		}
		return Cl
	} else if Ls.IsNumber(Arg) {
		var Level int = Ls.ToInteger(Arg)
		if Level <= 0 {
			Ls.ArgError(Arg, "level out of range")
			return nil
		}
		var Info LuaDebug
		if !Ls.GetInfo(Level, "f", &Info) {
			Ls.ArgError(Arg, "invalid level")
			return nil
		}
		if !Ls.IsFunction(-1) {
			Ls.ArgError(Arg, "level does not point to a function")
			return nil
		}
		var Cl *Closure = Ls.ToObject(-1).ClValue()
		Ls.Pop(1)

		if !AllowC && Cl.IsC == 1 {
			Ls.ArgError(Arg, "level points to c function")
			return nil
		}
		return Cl
	}
	Ls.ArgError(Arg, "function or number")
	return nil
}

func Init(L *LuaState) {
	var MainThread *LuaState = Api.RobloxGlobalState
	MainThread.GetGlobal("debug")

	if MainThread.Type(-1) == LUA_TTABLE {
		L.NewTable()
		MainThread.PushNil()
		for MainThread.Next(-2) {
			var Key string = MainThread.ToString(-2)
			var ValPtr *TValue = MainThread.ToObject(-1)
			L.PushString(Key)
			pushTValue(L, ValPtr)
			L.SetTable(-3)
			MainThread.Pop(1)
		}
		L.SetGlobal("debug")
	}
	MainThread.Pop(1)

	Register(L, Info{
		GetInfo: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			if !(Ls.IsFunction(1) || Ls.IsNumber(1)) {
				Ls.ArgError(1, "function or number")
				return 0
			}
			var Level int = 0
			if Ls.IsNumber(1) {
				Level = Ls.ToInteger(1)
			} else {
				Level = -Ls.GetTop()
			}
			var Info LuaDebug
			var Desc string = Ls.OptString(2, "sluanf")
			if !Ls.GetInfo(Level, Desc, &Info) {
				Ls.ArgError(1, "invalid level")
				return 0
			}

			Ls.NewTable()
			if strings.Contains(Desc, "s") {
				Ls.PushString(Info.Source)
				Ls.SetField(-2, "source")
				Ls.PushString(Info.ShortSrc)
				Ls.SetField(-2, "short_src")
				Ls.PushString(Info.What)
				Ls.SetField(-2, "what")
				Ls.PushInteger(Info.LineDefined)
				Ls.SetField(-2, "linedefined")
			}
			if strings.Contains(Desc, "l") {
				Ls.PushInteger(Info.CurrentLine)
				Ls.SetField(-2, "currentline")
			}
			if strings.Contains(Desc, "u") {
				Ls.PushInteger(int(Info.NUpvals))
				Ls.SetField(-2, "nups")
			}
			if strings.Contains(Desc, "a") {
				Ls.PushInteger(int(Info.IsVarArg))
				Ls.SetField(-2, "is_vararg")
				Ls.PushInteger(int(Info.NParams))
				Ls.SetField(-2, "numparams")
			}
			if strings.Contains(Desc, "n") {
				Ls.PushString(Info.Name)
				Ls.SetField(-2, "name")
			}
			if strings.Contains(Desc, "f") {
				Ls.PushValue(-2)
				Ls.SetField(-2, "func")
				Ls.Remove(-2)
			}
			return 1
		},

		GetRegistry: func(Ls *LuaState) int {
			Ls.RawCheckStack(1)
			Ls.ThreadBarrier()
			Ls.PushValue(LUA_REGISTRYINDEX)
			return 1
		},

		GetUpvalues: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Cl *Closure = getFunction(Ls, 1, true)
			if Cl == nil {
				return 0
			}
			Ls.NewTable()
			for I, Upval := range Cl.UpValues() {
				Ls.RawCheckStack(1)
				pushTValue(Ls, &Upval)
				Ls.RawSetI(-2, I+1)
			}
			return 1
		},

		GetUpvalue: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Cl *Closure = getFunction(Ls, 1, true)
			if Cl == nil {
				return 0
			}
			var Idx int = Ls.ToInteger(2)
			if Cl.NUpvalues <= 0 {
				Ls.ArgError(1, "function has no upvalues")
				return 0
			}
			if Idx < 1 || Idx > int(Cl.NUpvalues) {
				Ls.PushNil()
				return 1
			}
			pushTValue(Ls, &Cl.UpValues()[Idx-1])
			return 1
		},

		SetUpvalue: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			Ls.CheckAny(3)
			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var Idx int = Ls.ToInteger(2)
			if Cl.NUpvalues <= 0 {
				Ls.ArgError(1, "function has no upvalues")
				return 0
			}
			if Idx < 1 || Idx > int(Cl.NUpvalues) {
				Ls.ArgError(2, "index out of range")
				return 0
			}

			var NewVal *TValue = Ls.ToObject(3)
			var Upval *TValue = Cl.GetUpval(Idx - 1)

			Print(0, "%v %v %v", Upval, NewVal, Cl.NUpvalues)

			Upval.Value = NewVal.Value
			Upval.Tt = NewVal.Tt

			Ls.PushBoolean(true)
			return 1
		},

		GetConstants: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var P *Proto = Cl.AsL().P
			Ls.CreateTable(int(P.Sizek), 0)

			var KSlice []TValue = unsafe.Slice(P.K, P.Sizek)
			for i := 0; i < int(P.Sizek); i++ {
				var K *TValue = &KSlice[i]
				if K.Tt == LUA_TNIL || K.Tt == LUA_TFUNCTION || K.Tt == LUA_TTABLE {
					Ls.PushNil()
				} else {
					pushTValue(Ls, K)
				}
				Ls.RawSetI(-2, i+1)
			}
			return 1
		},

		GetConstant: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var Idx int = Ls.ToInteger(2)
			var P *Proto = Cl.AsL().P

			if P.Sizek <= 0 {
				Ls.ArgError(1, "function has no constants")
				return 0
			}
			if Idx < 1 || Idx > int(P.Sizek) {
				Ls.ArgError(2, "index out of range")
				return 0
			}

			var KSlice []TValue = unsafe.Slice(P.K, P.Sizek)
			var K *TValue = &KSlice[Idx-1]

			if K.Tt == LUA_TNIL || K.Tt == LUA_TTABLE || K.Tt == LUA_TFUNCTION {
				Ls.PushNil()
				return 1
			}

			pushTValue(Ls, K)
			return 1
		},

		SetConstant: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			Ls.CheckAny(3)
			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var Idx int = Ls.ToInteger(2)
			var P *Proto = Cl.AsL().P

			if P.Sizek <= 0 {
				Ls.ArgError(1, "function has no constants")
				return 0
			}
			if Idx < 1 || Idx > int(P.Sizek) {
				Ls.ArgError(2, "index out of range")
				return 0
			}

			var KSlice []TValue = unsafe.Slice(P.K, P.Sizek)
			var K *TValue = &KSlice[Idx-1]
			var NewVal *TValue = Ls.ToObject(3)

			if K.Tt != LUA_TFUNCTION && K.Tt != LUA_TTABLE {
				if K.Tt == NewVal.Tt {
					K.Value = NewVal.Value
				}
			}
			return 0
		},

		GetProto: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			Ls.CheckType(2, LUA_TNUMBER)
			var Active bool = Ls.OptBoolean(3, false)

			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var Idx int = Ls.ToInteger(2)
			var P *Proto = Cl.AsL().P

			if Idx <= 0 || Idx > int(P.Sizep) {
				Ls.ArgError(2, "index out of bounds")
				return 0
			}

			var PSlice []*Proto = unsafe.Slice(P.P, P.Sizep)
			var WantedProto *Proto = PSlice[Idx-1]

			if !Active {
				var Bytecode []byte = Compile("return", CompileOptions{
					OptimizationLevel: 1,
					DebugLevel:        2,
				})
				Ls.Load("getproto", Bytecode, 0)
				var NewCl *Closure = Ls.ToObject(-1).ClValue()

				NewCl.NUpvalues = Cl.NUpvalues
				NewCl.Env = Cl.Env
				NewCl.AsL().P = WantedProto
			} else {
				var G *GlobalState = Ls.Global
				var ObjectCount int = 0
				Ls.NewTable()

				for Page := G.Allgcopages; Page != nil; Page = Page.Listnext {
					var Start, End uintptr
					var BlockSize int32
					Start, End, _, BlockSize = Page.GetPageWalkInfo()
					for Pos := Start; Pos != End; Pos += uintptr(BlockSize) {
						var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))
						if Gco.Tt != LUA_TFUNCTION || Ls.IsDead(Gco) {
							continue
						}

						var GcClosure *Closure = (*Closure)(unsafe.Pointer(Pos))
						if GcClosure.IsC == 0 && GcClosure.AsL().P == WantedProto {
							ObjectCount++
							Ls.Top.Value = Value(uintptr(unsafe.Pointer(GcClosure)))
							Ls.Top.Tt = int32(LUA_TFUNCTION)
							Ls.Top = (StkId)(unsafe.Add(unsafe.Pointer(Ls.Top), unsafe.Sizeof(TValue{})))
							Ls.RawSetI(-2, ObjectCount)
						}
					}
				}
			}
			return 1
		},

		GetProtos: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Cl *Closure = getFunction(Ls, 1, false)
			if Cl == nil {
				return 0
			}

			var P *Proto = Cl.AsL().P
			var PSlice []*Proto = unsafe.Slice(P.P, P.Sizep)

			Ls.NewTable()
			for i := 0; i < int(P.Sizep); i++ {
				var WantedProto *Proto = PSlice[i]

				var Bytecode []byte = Compile("return", CompileOptions{
					OptimizationLevel: 1,
					DebugLevel:        2,
				})
				Ls.Load("getprotos", Bytecode, 0)
				var NewCl *Closure = Ls.ToObject(-1).ClValue()

				NewCl.NUpvalues = Cl.NUpvalues
				NewCl.Env = Cl.Env
				NewCl.AsL().P = WantedProto

				Ls.RawSetI(-2, i+1)
			}
			return 1
		},

		GetStack: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Level int = 0
			if Ls.IsNumber(1) {
				Level = Ls.ToInteger(1)
				if Level <= 0 {
					Ls.ArgError(1, "level out of range")
					return 0
				}
			} else if Ls.IsFunction(1) {
				Level = -Ls.GetTop()
			} else {
				Ls.ArgError(1, "function or number")
				return 0
			}

			var Info LuaDebug
			if !Ls.GetInfo(Level, "f", &Info) {
				Ls.ArgError(1, "invalid level")
				return 0
			}

			var Cl *Closure = Ls.ToObject(-1).ClValue()
			if Cl == nil || Cl.IsC == 1 {
				Ls.ArgError(1, "luau function expected")
				return 0
			}
			Ls.Pop(1)

			var CiPtr uintptr = uintptr(unsafe.Pointer(Ls.Ci)) - uintptr(Level)*unsafe.Sizeof(CallInfo{})
			var Ci *CallInfo = (*CallInfo)(unsafe.Pointer(CiPtr))

			if Ls.IsNumber(2) {
				var Idx int = Ls.ToInteger(2) - 1
				var StackLen int = int(uintptr(unsafe.Pointer(Ci.Top))-uintptr(unsafe.Pointer(Ci.Base))) / int(unsafe.Sizeof(TValue{}))
				if Idx < 0 || Idx >= StackLen {
					Ls.ArgError(2, "index out of range")
					return 0
				}

				var ValPtr *TValue = (*TValue)(unsafe.Pointer(uintptr(unsafe.Pointer(Ci.Base)) + uintptr(Idx)*unsafe.Sizeof(TValue{})))
				pushTValue(Ls, ValPtr)
			} else {
				var Idx int = 1
				Ls.NewTable()

				var Curr uintptr = uintptr(unsafe.Pointer(Ci.Base))
				var Top uintptr = uintptr(unsafe.Pointer(Ci.Top))

				for Curr < Top {
					var ValPtr *TValue = (*TValue)(unsafe.Pointer(Curr))
					Ls.PushInteger(Idx)
					pushTValue(Ls, ValPtr)
					Ls.SetTable(-3)

					Idx++
					Curr += unsafe.Sizeof(TValue{})
				}
			}
			return 1
		},

		SetStack: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var Level int = 0
			if Ls.IsNumber(1) {
				Level = Ls.ToInteger(1)
				if Level <= 0 {
					Ls.ArgError(1, "level out of range")
					return 0
				}
			} else if Ls.IsFunction(1) {
				Level = -Ls.GetTop()
			} else {
				Ls.ArgError(1, "function or number")
				return 0
			}

			var Info LuaDebug
			if !Ls.GetInfo(Level, "f", &Info) {
				Ls.ArgError(1, "invalid level")
				return 0
			}

			var Cl *Closure = Ls.ToObject(-1).ClValue()
			if Cl == nil || Cl.IsC == 1 {
				Ls.ArgError(1, "luau function expected")
				return 0
			}
			Ls.Pop(1)
			Ls.CheckAny(3)

			var CiPtr uintptr = uintptr(unsafe.Pointer(Ls.Ci)) - uintptr(Level)*unsafe.Sizeof(CallInfo{})
			var Ci *CallInfo = (*CallInfo)(unsafe.Pointer(CiPtr))

			var Idx int = Ls.ToInteger(2) - 1
			var StackLen int = int(uintptr(unsafe.Pointer(Ci.Top))-uintptr(unsafe.Pointer(Ci.Base))) / int(unsafe.Sizeof(TValue{}))

			if Idx < 0 || Idx >= StackLen {
				Ls.ArgError(2, "index out of range")
				return 0
			}

			var ValPtr *TValue = (*TValue)(unsafe.Pointer(uintptr(unsafe.Pointer(Ci.Base)) + uintptr(Idx)*unsafe.Sizeof(TValue{})))
			var NewVal *TValue = Ls.ToObject(3)

			if ValPtr.Tt != NewVal.Tt {
				Ls.ArgError(3, "new value type does not match previous value type")
				return 0
			}

			ValPtr.Value = NewVal.Value
			return 0
		},
	})
}
