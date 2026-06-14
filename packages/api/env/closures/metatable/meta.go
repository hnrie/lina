package metatable

import (
	. "main/packages/onyx/logic/luau/Api"
)

type Misc struct {
	GetRawMetaTable func(*LuaState) int `lua:"getrawmetatable"`
	SetRawMetaTable func(*LuaState) int `lua:"setrawmetatable"`
}

func Init(L *LuaState) {
	Register(L, Misc{
		GetRawMetaTable: func(ls *LuaState) int {
			ls.CheckAny(1)
			if ls.GetMetaTable(1) == 0 {
				ls.PushNil()
			}
			return 1
		},
		SetRawMetaTable: func(ls *LuaState) int {
			ls.CheckAny(1)
			ls.CheckType(2, LUA_TTABLE)
			ls.SetMetaTable(1)
			ls.PushValue(1)
			return 1
		},
	})
}
