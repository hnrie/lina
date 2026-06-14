package scripts

import "C"
import (
	"crypto/sha512"
	"encoding/hex"
	"strings"
	"unsafe"

	"main/packages/api/managers/logs"
	. "main/packages/onyx/instance"
	. "main/packages/onyx/logic/luau/Api"
	. "main/packages/onyx/mem"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

type Misc struct {
	GetGC             func(*LuaState) int `lua:"getgc"`
	GetREnv           func(*LuaState) int `lua:"getrenv"`
	GetCallingScript  func(*LuaState) int `lua:"getcallingscript"`
	GetScriptClosure  func(*LuaState) int `lua:"getscriptclosure" alias:"getscriptfunction"`
	FilterGC          func(*LuaState) int `lua:"filtergc"`
	GetThreadIdentity func(*LuaState) int `lua:"getthreadidentity" alias:"getidentity,getthreadcontext"`
	SetThreadIdentity func(*LuaState) int `lua:"setthreadidentity" alias:"setidentity,setthreadcontext"`
	GetLoadedModules  func(*LuaState) int `lua:"getloadedmodules"`
	GetScriptBytecode func(*LuaState) int `lua:"getscriptbytecode" alias:"dumpstring"`
	GetScripts        func(*LuaState) int `lua:"getscripts"`
	GetRunningScripts func(*LuaState) int `lua:"getrunningscripts"`
	GetScriptHash     func(*LuaState) int `lua:"getscripthash"`
	CompareInstances  func(*LuaState) int `lua:"compareinstances"`
	GetNilInstances   func(*LuaState) int `lua:"getnilinstances"`
	IsRbxActive       func(*LuaState) int `lua:"isrbxactive"`

	GetHiddenProperty func(*LuaState) int `lua:"gethiddenproperty"`
	SetHiddenProperty func(*LuaState) int `lua:"sethiddenproperty"`

	IsScriptable  func(*LuaState) int `lua:"isscriptable"`
	SetScriptable func(*LuaState) int `lua:"setscriptable"`
}

func pushTValue(Ls *LuaState, Val *TValue) {
	Ls.Top.Value = Val.Value
	Ls.Top.Extra = Val.Extra
	Ls.Top.Tt = Val.Tt
	Ls.Top = (StkId)(unsafe.Add(unsafe.Pointer(Ls.Top), unsafe.Sizeof(TValue{})))
}

func tableHasKeys(Ls *LuaState, TargetTable, KeysTable int) bool {
	var Passed bool = true
	Ls.PushNil()
	for Ls.Next(KeysTable) {
		Ls.PushValue(-1)
		Ls.RawGet(TargetTable)
		if Ls.IsNil(-1) {
			Passed = false
			Ls.Pop(3)
			return false
		}
		Ls.Pop(2)
	}
	return Passed
}

func tableHasValues(Ls *LuaState, TargetTable, ValuesTable int) bool {
	Ls.PushNil()
	for Ls.Next(ValuesTable) {
		var ReqValIdx int = Ls.GetTop()
		var Found bool = false
		Ls.PushNil()
		for Ls.Next(TargetTable) {
			if Ls.Equal(-1, ReqValIdx) {
				Found = true
				Ls.Pop(2)
				break
			}
			Ls.Pop(1)
		}
		if !Found {
			Ls.Pop(2)
			return false
		}
		Ls.Pop(1)
	}
	return true
}

func tableHasKeyValuePairs(Ls *LuaState, TargetTable, KvTable int) bool {
	Ls.PushNil()
	for Ls.Next(KvTable) {
		var ReqKeyIdx int = Ls.GetTop() - 1
		var ReqValIdx int = Ls.GetTop()
		Ls.PushValue(ReqKeyIdx)
		Ls.RawGet(TargetTable)
		if !Ls.Equal(-1, ReqValIdx) {
			Ls.Pop(3)
			return false
		}
		Ls.Pop(2)
	}
	return true
}

func tableHasMetatable(Ls *LuaState, TargetTable, ReqMtIdx int) bool {
	if Ls.GetMetaTable(TargetTable) != 0 {
		var Passed bool = Ls.Equal(-1, ReqMtIdx)
		Ls.Pop(1)
		return Passed
	}
	return false
}

func functionHasConstants(Ls *LuaState, FuncIdx, ConstsTable int) bool {
	var Cl *Closure = Ls.ToObject(FuncIdx).ClValue()
	if Cl == nil || Cl.IsC == 1 {
		return false
	}

	var P *Proto = Cl.AsL().P
	var KSlice []TValue = unsafe.Slice(P.K, int(P.Sizek))

	Ls.PushNil()
	for Ls.Next(ConstsTable) {
		var ReqConstIdx int = Ls.GetTop()
		var Found bool = false

		for i := 0; i < int(P.Sizek); i++ {
			pushTValue(Ls, &KSlice[i])
			if Ls.Equal(-1, ReqConstIdx) {
				Found = true
				Ls.Pop(1)
				break
			}
			Ls.Pop(1)
		}

		if !Found {
			Ls.Pop(2)
			return false
		}
		Ls.Pop(1)
	}
	return true
}

func functionHasUpvalues(Ls *LuaState, FuncIdx, UpsTable int) bool {
	var Cl *Closure = Ls.ToObject(FuncIdx).ClValue()
	if Cl == nil || Cl.IsC == 1 {
		return false
	}

	var Upvals []TValue = Cl.UpValues()

	Ls.PushNil()
	for Ls.Next(UpsTable) {
		var ReqUpIdx int = Ls.GetTop()
		var Found bool = false

		for i := 0; i < int(Cl.NUpvalues); i++ {
			pushTValue(Ls, &Upvals[i])
			if Ls.Equal(-1, ReqUpIdx) {
				Found = true
				Ls.Pop(1)
				break
			}
			Ls.Pop(1)
		}

		if !Found {
			Ls.Pop(2)
			return false
		}
		Ls.Pop(1)
	}
	return true
}

func Init(L *LuaState) {
	Register(L, Misc{
		GetScriptClosure: func(Ls *LuaState) int {
			var UserData unsafe.Pointer = Ls.ToUserData(1)
			if UserData == nil {
				Ls.PushNil()
				return 1
			}

			var Script uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(UserData))
			if Script == 0 {
				Ls.PushNil()
				return 1
			}

			var Bytecode []byte = Decompress(NewInstance(Api.Sesh.Luna, Script))
			if len(Bytecode) == 0 {
				Ls.PushNil()
				return 1
			}

			var Thread *LuaState = Ls.NewThread()
			Thread.SandboxThread()

			if Thread.Userdata != nil {
				Thread.Userdata.Identity = 8
				Thread.Userdata.Capabilities = IdentityToCapabilities(8, true)
			}

			Ls.PushValue(1)
			Ls.XMove(Thread, 1)
			Thread.SetGlobal("script")

			if err := Thread.Load("getscriptclosure", Bytecode, 0); err != nil {
				Ls.PushNil()
				return 1
			}

			Thread.SetCaps(8)
			Thread.XMove(Ls, 1)

			return 1
		},
		GetREnv: func(Ls *LuaState) int {
			var Roblox *LuaState = Ls.Global.Mainthread
			var Cloned *LuaTable = Ls.Clone(Roblox.Gt)

			Ls.RawCheckStack(1)
			Ls.ThreadBarrier()
			Roblox.ThreadBarrier()

			Ls.Top.Value = Value(uintptr(unsafe.Pointer(Cloned)))
			Ls.Top.Tt = int32(LUA_TTABLE)
			Ls.Top = (StkId)(unsafe.Add(unsafe.Pointer(Ls.Top), unsafe.Sizeof(TValue{})))

			Ls.RawGetI(LUA_REGISTRYINDEX, 2)
			Ls.SetField(-2, "_G")
			Ls.RawGetI(LUA_REGISTRYINDEX, 4)
			Ls.SetField(-2, "shared")

			return 1
		},
		GetCallingScript: func(Ls *LuaState) int {
			if Ls.Userdata.Source.Expired() {
				Ls.PushNil()
			} else {
				Ls.PushRawInstance(unsafe.Pointer(&Ls.Userdata.Source))
			}
			return 1
		},
		GetGC: func(Ls *LuaState) int {
			var Incld bool = Ls.ToBoolean(1)
			var Indx int = 1
			var Curr *LuaPage = Ls.Global.Allgcopages

			Ls.NewTable()

			for Curr != nil {
				Start, End, Busy, Blocksize := Curr.GetPageWalkInfo()
				for Pos := Start; Pos < End; Pos += uintptr(Blocksize) {
					var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))
					if Gco.Tt != uint8(LUA_TNIL) && !Ls.IsDead(Gco) {
						if Gco.Tt == uint8(LUA_TFUNCTION) || Gco.Tt == uint8(LUA_TUSERDATA) || Gco.Tt == uint8(LUA_TTHREAD) || Gco.Tt == uint8(LUA_TBUFFER) || (Gco.Tt == uint8(LUA_TTABLE) && Incld) {
							Ls.CheckStack(2)
							Ls.PushInteger(Indx)
							Ls.Top.Value = Value(uintptr(unsafe.Pointer(Gco)))
							Ls.Top.Tt = int32(Gco.Tt)
							Ls.Top = StkId(unsafe.Pointer(uintptr(unsafe.Pointer(Ls.Top)) + unsafe.Sizeof(TValue{})))
							Ls.SetTable(-3)
							Indx++
						}
						Busy--
						if Busy <= 0 {
							break
						}
					}
				}
				Curr = Curr.Listnext
			}
			return 1
		},
		FilterGC: func(Ls *LuaState) int {
			Ls.CheckAny(1)
			var FilterOptions int = 2
			var ReturnOne bool = Ls.OptBoolean(3, false)

			if Ls.IsFunction(1) {
				Ls.NewTable()
				var MatchesIndex int = Ls.GetTop()
				var MatchCount int = 0

				Ls.GetGlobal("getgc")
				if Ls.IsNil(-1) {
					Ls.Pop(1)
					Ls.Error("getgc function not available")
					return 0
				}

				Ls.PushBoolean(true)
				Ls.Call(1, 1)

				var GcTable int = Ls.GetTop()
				Ls.PushNil()

				for Ls.Next(GcTable) {
					Ls.PushValue(1)
					Ls.PushValue(-2)

					if err := Ls.PCall(1, 1); err == nil {
						if Ls.ToBoolean(-1) {
							Ls.Pop(1)
							Ls.PushValue(-1)
							MatchCount++
							Ls.RawSetI(MatchesIndex, MatchCount)
						} else {
							Ls.Pop(1)
						}
					}

					Ls.Pop(1)
				}

				Ls.Pop(1)
				if ReturnOne && MatchCount > 0 {
					Ls.RawGetI(MatchesIndex, 1)
					return 1
				}
				return 1
			}

			var FilterType string = Ls.CheckString(1)
			Ls.NewTable()
			var MatchesIndex int = Ls.GetTop()
			var MatchCount int = 0

			if FilterType == "table" {
				Ls.GetGlobal("getgc")
				Ls.PushBoolean(true)
				Ls.Call(1, 1)

				var GcTable int = Ls.GetTop()
				Ls.PushNil()

				for Ls.Next(GcTable) {
					if Ls.IsTable(-1) {
						var Passed bool = true
						var CurrentTable int = Ls.GetTop()

						if Passed && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Keys")
							if !Ls.IsNil(-1) {
								Passed = tableHasKeys(Ls, CurrentTable, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Values")
							if !Ls.IsNil(-1) {
								Passed = tableHasValues(Ls, CurrentTable, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "KeyValuePairs")
							if !Ls.IsNil(-1) {
								Passed = tableHasKeyValuePairs(Ls, CurrentTable, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Metatable")
							if !Ls.IsNil(-1) {
								Passed = tableHasMetatable(Ls, CurrentTable, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed {
							if ReturnOne {
								Ls.Pop(1)
								return 1
							}
							Ls.PushValue(CurrentTable)
							MatchCount++
							Ls.RawSetI(MatchesIndex, MatchCount)
						}
					}
					Ls.Pop(1)
				}
				Ls.Pop(1)

			} else if FilterType == "function" {
				var IgnoreExecutor bool = true
				if Ls.IsTable(FilterOptions) {
					Ls.GetField(FilterOptions, "IgnoreExecutor")
					if !Ls.IsNil(-1) {
						IgnoreExecutor = Ls.ToBoolean(-1)
					}
					Ls.Pop(1)
				}

				Ls.GetGlobal("getgc")
				Ls.PushBoolean(false)
				Ls.Call(1, 1)

				var GcTable int = Ls.GetTop()
				Ls.PushNil()

				for Ls.Next(GcTable) {
					if Ls.IsFunction(-1) {
						var Passed bool = true
						var CurrentFunc int = Ls.GetTop()
						var IsCClosure bool = Ls.IsCFunction(CurrentFunc)

						if Passed && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Name")
							if !Ls.IsNil(-1) {
								var RequiredName string = Ls.ToString(-1)
								var Info LuaDebug
								Ls.PushValue(CurrentFunc)
								if Ls.GetInfo(-Ls.GetTop(), "n", &Info) {
									Passed = strings.Contains(Info.Name, RequiredName)
								} else {
									Passed = false
								}
								Ls.Pop(1)
							}
							Ls.Pop(1)
						}

						if Passed && IgnoreExecutor {
							if IsCClosure {
								Passed = false
							}
						}

						if IsCClosure && Ls.IsTable(FilterOptions) && Passed {
							Ls.GetField(FilterOptions, "Hash")
							var HasHash bool = !Ls.IsNil(-1)
							Ls.Pop(1)

							Ls.GetField(FilterOptions, "Constants")
							var HasConstants bool = !Ls.IsNil(-1)
							Ls.Pop(1)

							Ls.GetField(FilterOptions, "Upvalues")
							var HasUpvalues bool = !Ls.IsNil(-1)
							Ls.Pop(1)

							if HasHash || HasConstants || HasUpvalues {
								Passed = false
							}
						}

						if Passed && !IsCClosure && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Hash")
							if !Ls.IsNil(-1) {
								Passed = false
							}
							Ls.Pop(1)
						}

						if Passed && !IsCClosure && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Constants")
							if !Ls.IsNil(-1) {
								Passed = functionHasConstants(Ls, CurrentFunc, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed && !IsCClosure && Ls.IsTable(FilterOptions) {
							Ls.GetField(FilterOptions, "Upvalues")
							if !Ls.IsNil(-1) {
								Passed = functionHasUpvalues(Ls, CurrentFunc, Ls.GetTop())
							}
							Ls.Pop(1)
						}

						if Passed {
							if ReturnOne {
								Ls.Pop(1)
								return 1
							}
							Ls.PushValue(CurrentFunc)
							MatchCount++
							Ls.RawSetI(MatchesIndex, MatchCount)
						}
					}
					Ls.Pop(1)
				}
				Ls.Pop(1)
			} else {
				Ls.Error("Expected type 'function' or 'table', got '%s'", FilterType)
			}

			if ReturnOne {
				Ls.PushNil()
				return 1
			}
			return 1
		},
		GetThreadIdentity: func(Ls *LuaState) int {
			if Ls.Userdata != nil {
				Ls.PushNumber(float64(Ls.Userdata.Identity))
				return 1
			}
			Ls.PushNumber(0)
			return 0
		},
		SetThreadIdentity: func(Ls *LuaState) int {
			var Identity float64 = Ls.CheckNumber(1)
			if Ls.Userdata != nil {
				Ls.Userdata.Identity = uintptr(Identity)
				Ls.Userdata.Capabilities = IdentityToCapabilities(int(Identity), Identity >= 7)
			}
			return 0
		},
		GetLoadedModules: func(Ls *LuaState) int {
			Ls.NewTable()
			var ResultTableIdx int = Ls.GetTop()

			var FoundModules map[uintptr]bool = make(map[uintptr]bool)
			var ItemsFound int = 1

			var Curr *LuaPage = Ls.Global.Allgcopages
			for Curr != nil {
				Start, End, Busy, Blocksize := Curr.GetPageWalkInfo()
				for Pos := Start; Pos < End; Pos += uintptr(Blocksize) {
					var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))
					if Gco.Tt == uint8(LUA_TFUNCTION) && !Ls.IsDead(Gco) {
						Ls.CheckStack(4)
						Ls.Top.Value = Value(uintptr(unsafe.Pointer(Gco)))
						Ls.Top.Tt = int32(LUA_TFUNCTION)
						Ls.Top = StkId(unsafe.Pointer(uintptr(unsafe.Pointer(Ls.Top)) + unsafe.Sizeof(TValue{})))

						Ls.GetFEnv(-1)
						if !Ls.IsNil(-1) {
							Ls.GetField(-1, "script")
							if !Ls.IsNil(-1) {
								var UserData unsafe.Pointer = Ls.ToUserData(-1)
								if UserData != nil {
									var ScriptAddr uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(UserData))
									if ScriptAddr != 0 {
										var Inst Instance = NewInstance(Api.Sesh.Luna, ScriptAddr)
										if Inst.ClassName() == "ModuleScript" {
											if !FoundModules[ScriptAddr] {
												FoundModules[ScriptAddr] = true
												Ls.PushValue(-1)
												Ls.RawSetI(ResultTableIdx, ItemsFound)
												ItemsFound++
											}
										}
									}
								}
							}
							Ls.Pop(1)
						}
						Ls.Pop(2)
					}
					Busy--
					if Busy <= 0 {
						break
					}
				}
				Curr = Curr.Listnext
			}
			return 1
		},
		GetScripts: func(Ls *LuaState) int {
			Ls.NewTable()
			var ResultTableIdx int = Ls.GetTop()

			var FoundScripts map[uintptr]bool = make(map[uintptr]bool)
			var ItemsFound int = 1

			var Curr *LuaPage = Ls.Global.Allgcopages
			for Curr != nil {
				Start, End, Busy, Blocksize := Curr.GetPageWalkInfo()
				for Pos := Start; Pos < End; Pos += uintptr(Blocksize) {
					var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))
					if Gco.Tt == uint8(LUA_TUSERDATA) && !Ls.IsDead(Gco) {
						Ls.CheckStack(2)
						Ls.Top.Value = Value(uintptr(unsafe.Pointer(Gco)))
						Ls.Top.Tt = int32(LUA_TUSERDATA)
						Ls.Top = StkId(unsafe.Pointer(uintptr(unsafe.Pointer(Ls.Top)) + unsafe.Sizeof(TValue{})))

						var UserData unsafe.Pointer = Ls.ToUserData(-1)
						if UserData != nil {
							var ScriptAddr uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(UserData))
							if ScriptAddr > 0x10000 {
								var Inst Instance = NewInstance(Api.Sesh.Luna, ScriptAddr)
								var ClassName string = Inst.ClassName()

								var ShouldInclude bool = false

								switch ClassName {
								case "LocalScript", "ModuleScript":
									ShouldInclude = true
								case "Script":
									Ls.GetField(-1, "RunContext")
									if Ls.IsUserData(-1) {
										Ls.GetField(-1, "Name")
										if Ls.IsString(-1) && Ls.ToString(-1) == "Client" {
											ShouldInclude = true
										}
										Ls.Pop(1)
									}
									Ls.Pop(1)
								}
								if ShouldInclude && !FoundScripts[ScriptAddr] {
									FoundScripts[ScriptAddr] = true
									Ls.PushValue(-1)
									Ls.RawSetI(ResultTableIdx, ItemsFound)
									ItemsFound++
								}
							}
						}
						Ls.Pop(1)
					}

					Busy--
					if Busy <= 0 {
						break
					}
				}
				Curr = Curr.Listnext
			}
			return 1
		},
		GetRunningScripts: func(Ls *LuaState) int {
			Ls.NewTable()
			var ResultTableIdx int = Ls.GetTop()

			var FoundScripts map[uintptr]bool = make(map[uintptr]bool)
			var ItemsFound int = 1

			var Curr *LuaPage = Ls.Global.Allgcopages
			for Curr != nil {
				Start, End, Busy, Blocksize := Curr.GetPageWalkInfo()
				for Pos := Start; Pos < End; Pos += uintptr(Blocksize) {
					var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))

					if Gco.Tt == uint8(LUA_TTHREAD) && !Ls.IsDead(Gco) {
						var Th *LuaState = (*LuaState)(unsafe.Pointer(Gco))

						if Th.Userdata != nil && !Th.Userdata.Source.Expired() {
							Ls.CheckStack(2)
							Ls.PushRawInstance(unsafe.Pointer(&Th.Userdata.Source))

							if Ls.IsUserData(-1) {
								var UserData unsafe.Pointer = Ls.ToUserData(-1)
								if UserData != nil {
									var ScriptAddr uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(UserData))
									if ScriptAddr > 0x10000 {
										var Inst Instance = NewInstance(Api.Sesh.Luna, ScriptAddr)
										if Inst.ClassName() != "CoreScript" {
											if !FoundScripts[ScriptAddr] {
												FoundScripts[ScriptAddr] = true
												Ls.PushValue(-1)
												Ls.RawSetI(ResultTableIdx, ItemsFound)
												ItemsFound++
											}
										}
									}
								}
							}
							Ls.Pop(1)
						}
					}

					Busy--
					if Busy <= 0 {
						break
					}
				}
				Curr = Curr.Listnext
			}
			return 1
		},
		GetScriptBytecode: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)

			var Modules Instance = NewInstance(Api.Sesh.Luna, ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(Ls.ToPointer(1))))

			switch Modules.ClassName() {
			case "ModuleScript", "LocalScript":
				Ls.PushString(string(Decompress(Modules)))
				return 1
			}
			return 0
		},
		GetScriptHash: func(Ls *LuaState) int {
			Ls.CheckAny(1)

			var UserData unsafe.Pointer = Ls.ToUserData(1)
			if UserData == nil {
				Ls.PushNil()
				return 1
			}

			var ScriptAddr uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(UserData))
			if ScriptAddr == 0 {
				Ls.PushNil()
				return 1
			}

			var Inst Instance = NewInstance(Api.Sesh.Luna, ScriptAddr)
			var ClassName string = Inst.ClassName()

			if ClassName != "LocalScript" && ClassName != "ModuleScript" && ClassName != "Script" {
				Ls.PushNil()
				return 1
			}

			var RawBytecode []byte = Inst.Bytecode().Bytecode
			if len(RawBytecode) == 0 {
				Ls.PushNil()
				return 1
			}

			var Hash [64]byte = sha512.Sum384(RawBytecode)
			Ls.PushString(hex.EncodeToString(Hash[:]))
			return 1
		},
		CompareInstances: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)
			Ls.CheckType(2, LUA_TUSERDATA)

			if Ls.TypeName(1) != "Instance" || Ls.TypeName(2) != "Instance" {
				Ls.ArgError(1, "instance")
			}

			Ls.PushBoolean(Ls.ToPointer(1) == Ls.ToPointer(2))
			return 1
		},
		GetNilInstances: func(Ls *LuaState) int {
			Ls.NewTable()
			var ResultTableIdx int = Ls.GetTop()
			var ItemsFound int = 1

			Ls.GetGlobal("game")
			if Ls.IsNil(-1) {
				Ls.Pop(1)
				return 1
			}

			Ls.GetMetaTable(-1)
			var InstanceMtIdx int = Ls.GetTop()

			var OldThreshold uintptr = Ls.Global.GCthreshold
			Ls.Global.GCthreshold = ^uintptr(0)
			defer func() { Ls.Global.GCthreshold = OldThreshold }()

			var Curr *LuaPage = Ls.Global.Allgcopages
			for Curr != nil {
				Start, End, Busy, Blocksize := Curr.GetPageWalkInfo()
				for Pos := Start; Pos < End; Pos += uintptr(Blocksize) {
					var Gco *GCObject = (*GCObject)(unsafe.Pointer(Pos))
					if Gco.Tt == uint8(LUA_TUSERDATA) && !Ls.IsDead(Gco) {
						Ls.CheckStack(4)
						Ls.Top.Value = Value(uintptr(unsafe.Pointer(Gco)))
						Ls.Top.Tt = int32(LUA_TUSERDATA)
						Ls.Top = StkId(unsafe.Pointer(uintptr(unsafe.Pointer(Ls.Top)) + unsafe.Sizeof(TValue{})))

						if Ls.GetMetaTable(-1) != 0 {
							if Ls.Equal(-1, InstanceMtIdx) {
								Ls.Pop(1)
								Ls.GetField(-1, "Parent")
								if Ls.IsNil(-1) {
									Ls.PushValue(-2)
									Ls.RawSetI(ResultTableIdx, ItemsFound)
									ItemsFound++
								}
								Ls.Pop(1)
							} else {
								Ls.Pop(1)
							}
						}
						Ls.Pop(1)
					}
					Busy--
					if Busy <= 0 {
						break
					}
				}
				Curr = Curr.Listnext
			}
			Ls.SetTop(ResultTableIdx)
			return 1
		},
		IsRbxActive: func(Ls *LuaState) int {
			WindowName, _ := windows.UTF16PtrFromString("Roblox")
			Hwnd, _, _ := procFindWindowW.Call(
				0,
				uintptr(unsafe.Pointer(WindowName)),
			)
			FgHwnd, _, _ := procGetForegroundWindow.Call()
			Ls.PushBoolean(Hwnd != 0 && Hwnd == FgHwnd)
			return 0
		},
		GetHiddenProperty: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)
			Ls.CheckType(2, LUA_TSTRING)

			var PropName string = Ls.ToString(2)
			var UserDataPtr uintptr = uintptr(Ls.ToUserData(1))

			logs.Log("[GetHiddenProperty] Requested property: '%s'", PropName)

			if UserDataPtr == 0 {
				logs.Log("[GetHiddenProperty] Error: UserData pointer is 0")
				Ls.PushNil()
				Ls.PushBoolean(false)
				return 2
			}

			var InstanceAddr uintptr = ReadProcessMemory[uintptr](Api.Sesh.Luna, UserDataPtr)
			if InstanceAddr == 0 {
				logs.Log("[GetHiddenProperty] Error: Instance address is 0")
				Ls.PushNil()
				Ls.PushBoolean(false)
				return 2
			}

			var Inst Instance = NewInstance(Api.Sesh.Luna, InstanceAddr)
			var Prop *PropertyDescriptor = Inst.NewPropertyDescriptorContainer().Get(PropName)

			if Prop == nil || Prop.Address == 0 {
				logs.Log("[GetHiddenProperty] Error: Property '%s' does not exist on instance %x", PropName, InstanceAddr)
				Ls.Error("Property '%s' does not exist", PropName)
				return 0
			}

			var WasHidden bool = Prop.IsHiddenValue()
			var TypeNum int = Prop.TypeNumber()
			var Getter uintptr = Prop.Getter()
			var Impl uintptr = Prop.GetSetImpl()

			logs.Log("[GetHiddenProperty] Resolved '%s' | Hidden: %v | TypeNum: %d | Getter: %x | Impl: %x",
				PropName, WasHidden, TypeNum, Getter, Impl)

			if Getter != 0 && Impl != 0 {
				switch TypeNum {
				case ReflectionType_Bool:
					logs.Log("[GetHiddenProperty] Routing via PureGo (Bool)")
					var GetBool func(Impl uintptr, Inst uintptr) bool
					purego.RegisterFunc(&GetBool, Getter)

					var Result bool = GetBool(Impl, InstanceAddr)
					logs.Log("[GetHiddenProperty] PureGo result: %v", Result)

					Ls.PushBoolean(Result)
					Ls.PushBoolean(WasHidden)
					return 2

				case ReflectionType_Int:
					logs.Log("[GetHiddenProperty] Routing via PureGo (Int)")
					var GetInt func(Impl uintptr, Inst uintptr) int32
					purego.RegisterFunc(&GetInt, Getter)

					var Result int32 = GetInt(Impl, InstanceAddr)
					logs.Log("[GetHiddenProperty] PureGo result: %d", Result)

					Ls.PushInteger(int(Result))
					Ls.PushBoolean(WasHidden)
					return 2

				case ReflectionType_Float:
					logs.Log("[GetHiddenProperty] Routing via PureGo (Float)")
					var GetFloat func(Impl uintptr, Inst uintptr) float32
					purego.RegisterFunc(&GetFloat, Getter)

					var Result float32 = GetFloat(Impl, InstanceAddr)
					logs.Log("[GetHiddenProperty] PureGo result: %f", Result)

					Ls.PushNumber(float64(Result))
					Ls.PushBoolean(WasHidden)
					return 2

				case ReflectionType_SystemAddress:
					logs.Log("[GetHiddenProperty] Routing via PureGo (SystemAddress)")
					type sysAddr struct {
						PeerId int32
						Pad    [12]byte
					}
					var Result sysAddr
					var GetSysAddr func(Impl uintptr, Res *sysAddr, Inst uintptr)
					purego.RegisterFunc(&GetSysAddr, Getter)

					GetSysAddr(Impl, &Result, InstanceAddr)
					logs.Log("[GetHiddenProperty] PureGo result (PeerID): %d", Result.PeerId)

					Ls.PushInteger(int(Result.PeerId))
					Ls.PushBoolean(WasHidden)
					return 2
				}
			}

			logs.Log("[GetHiddenProperty] Routing via Lua Fallback (TypeNum: %d)", TypeNum)

			Prop.SetScriptable(true)
			Ls.GetField(1, PropName)
			Prop.SetScriptable(false)

			Ls.PushBoolean(WasHidden)

			logs.Log("[GetHiddenProperty] Successfully executed Lua Fallback")
			return 2
		},
		SetHiddenProperty: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)
			Ls.CheckType(2, LUA_TSTRING)
			Ls.CheckAny(3)

			var PropName string = Ls.ToString(2)
			var UserDataPtr uintptr = uintptr(Ls.ToPointer(1))

			var Inst Instance = NewInstance(Api.Sesh.Luna, ReadProcessMemory[uintptr](Api.Sesh.Luna, UserDataPtr))
			var Prop *PropertyDescriptor = Inst.NewPropertyDescriptorContainer().Get(PropName)

			if Prop == nil || Prop.Address == 0 {
				Ls.Error("Property '%s' does not exist", PropName)
				return 0
			}

			var WasScriptable bool = Prop.IsScriptable()
			Prop.SetScriptable(true)

			Ls.PushValue(1)
			Ls.PushValue(3)
			Ls.PushValue(3)

			Ls.SetField(1, PropName)

			Prop.SetScriptable(WasScriptable)
			Ls.PushBoolean(!WasScriptable)

			return 1
		},
		IsScriptable: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)
			Ls.CheckType(2, LUA_TSTRING)

			Ls.PushBoolean(
				NewInstance(Api.Sesh.Luna, ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(Ls.ToPointer(1)))).
					NewPropertyDescriptorContainer().Get(Ls.ToString(2)).IsScriptable(),
			)
			return 1
		},
		SetScriptable: func(Ls *LuaState) int {
			Ls.CheckType(1, LUA_TUSERDATA)
			Ls.CheckType(2, LUA_TSTRING)
			Ls.CheckBoolean(3)

			Ls.PushBoolean(
				NewInstance(Api.Sesh.Luna, ReadProcessMemory[uintptr](Api.Sesh.Luna, uintptr(Ls.ToPointer(1)))).
					NewPropertyDescriptorContainer().Get(Ls.ToString(2)).SetScriptable(Ls.ToBoolean(3)),
			)
			return 1
		},
	})
}

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
)
