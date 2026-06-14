package env

import (
	"main/packages/api/env/closures"
	"main/packages/api/env/closures/debug"
	"main/packages/api/env/closures/metatable"
	"main/packages/api/env/crypt"
	"main/packages/api/env/filesystem"
	"main/packages/api/env/misc"
	"main/packages/api/env/scripts"
	"main/packages/api/env/websocket"
	. "main/packages/onyx/logic/luau/Api"
)

func Env() {
	state := Api.LunaState

	state.RegisterFunction("getgenv", func(ls *LuaState) uintptr {
		if ls == state {
			ls.PushValue(LUA_GLOBALSINDEX)
			return 1
		}
		state.PushValue(LUA_GLOBALSINDEX)
		state.XMove(ls, 1)
		return 1
	})

	state.RegisterFunction("luna_internal_httpget", HttpGet)
	state.RegisterFunction("luna_internal_getobjects", GetObjects)
	{
		misc.Init(state)
		scripts.Init(state)
		closures.Init(state)
		filesystem.Init(state)
		websocket.Init(state)
		metatable.Init(state)
		crypt.Init(state)
		debug.Init(state)
	}

}
