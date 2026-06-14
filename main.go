package main

/*
#cgo CFLAGS: -I${SRCDIR}/packages/onyx/logic/luau/VM/include -I${SRCDIR}/packages/onyx/logic/luau/VM/src -I${SRCDIR}/packages/onyx/logic/luau/Compiler/include -I${SRCDIR}/packages/onyx/logic/luau/Compiler/src -I${SRCDIR}/packages/onyx/logic/luau/Ast/include -I${SRCDIR}/packages/onyx/logic/luau/Common/include -I${SRCDIR}/packages/onyx/logic/luau/Ast/src
#cgo CXXFLAGS: -std=c++20 -Wno-dll-attribute-on-redeclaration -I${SRCDIR}/packages/onyx/logic/luau/VM/include -I${SRCDIR}/packages/onyx/logic/luau/VM/src -I${SRCDIR}/packages/onyx/logic/luau/Compiler/include -I${SRCDIR}/packages/onyx/logic/luau/Compiler/src -I${SRCDIR}/packages/onyx/logic/luau/Ast/include -I${SRCDIR}/packages/onyx/logic/luau/Common/include -I${SRCDIR}/packages/onyx/logic/luau/Ast/src

#include <stdio.h>
#include <Windows.h>
#include <stddef.h>
#include <ntstatus.h>
#include <winternl.h>
#include <stdint.h>
#include <stdlib.h>
#include "lualib.h"

extern int errortest(lua_State* L);

*/
import "C"
import (
	"context"
	"time"
	"unsafe"

	"main/packages/api/managers/imgui"
	"main/packages/api/managers/logs"
	. "main/packages/api/managers/tpc"
	. "main/packages/onyx/logic"
	. "main/packages/onyx/logic/luau/Api"
	. "main/packages/onyx/mem"

	"main/packages/api/env/filesystem"

	"golang.org/x/sys/windows"

	"main/packages/api/env"
)

var (
	_index, _namecall LuaCFunction
	_Index, _Namecall func(*LuaState) uintptr
	indexCallback     uintptr
	MaxCapabilities   uintptr = (0x200000000000003F | 0x3FFFFFFFFFFF00) | (1 << 48)
	InGame                    = uintptr(0x830)
)

//export DllMain
func DllMain(hinstDLL uint32, fdwReason uint32, lpvReserved uintptr) bool {
	if fdwReason == 1 {
		go main()
	}
	return true
}

func main() {
	if hProcess, err := windows.OpenProcess(
		windows.PROCESS_VM_OPERATION|
			windows.PROCESS_VM_WRITE|
			windows.PROCESS_VM_READ|
			windows.PROCESS_CREATE_THREAD|
			windows.PROCESS_QUERY_INFORMATION|
			windows.PROCESS_DUP_HANDLE,
		false,
		windows.GetCurrentProcessId(),
	); err == nil {
		Api.Sesh = Session(&Luna{
			Handle: hProcess,
			Pid:    uintptr(windows.GetCurrentProcessId()),
		})

		propName := C.CString("LUNA_SHARED_MEM")
		handle := C.GetProp(C.HWND(unsafe.Pointer(GetHWNDFromPID(windows.GetCurrentProcessId()))), propName)
		if handle != nil {
			Api.Shared = (*Module)(unsafe.Pointer(handle))
		}
		C.free(unsafe.Pointer(propName))
		if Api.Shared != nil {
			dir, tpc := Api.NullTerm()
			filesystem.SetDir(dir)

			logs.Init(dir + "/luna/internal/logs")

			logs.Log("hi")

			go Server(tpc)
		}

		D3DHook(Api.Sesh.Game.RenderJob.Container().Job("RenderJob").Address)

		TeleportHandler(func(ctx context.Context, session TeleportSession) {

			Api.ExecutionChannel.Clear()

			Api.RobloxGlobalState = NewLuaState(Api.Sesh)
			Api.LunaState = Api.RobloxGlobalState.NewThread()

			Api.RobloxGlobalState.Ref(-1)
			Api.RobloxGlobalState.Pop(1)

			if Api.LunaState.Userdata != nil {
				Api.LunaState.Userdata.Identity = 8
				Api.LunaState.Userdata.Capabilities = IdentityToCapabilities(8, true)
			}

			Api.LunaState.NewTable()
			Api.LunaState.SetGlobal("_G")
			Api.LunaState.NewTable()
			Api.LunaState.SetGlobal("shared")

			time.Sleep(time.Millisecond * 25)
			Api.Hook() // hooks whsj
			time.Sleep(time.Millisecond * 25)
			Hook() // hooks the __index/__namecall
			time.Sleep(time.Millisecond * 25)
			env.Env() // setups env
			time.Sleep(time.Millisecond * 25)

			if len(Api.QueuedList) > 0 {
				Api.ExecutionChannel.Push(Api.QueuedList...)
			}

		})
	}
}

type TeleportSession struct {
	DataModel     uintptr
	ScriptContext uintptr
	Name          string
	State         *LuaState
	Teleported    bool
}

var TeleportHandler = func(fn func(ctx context.Context, s TeleportSession)) {

	{
		var last uintptr
		var ingame = false
		detect := func() {
			if Api.Sesh != nil && Api.Sesh.Game.RenderJob != nil {
				if dm := Api.Sesh.Game.RenderJob.DataModel(); dm != nil && dm.Name() != "LuaApp" {
					ingame = true
					addr := uintptr(dm.Self)
					if addr != last {
						last = addr

						Api.LunaState = nil
						Api.RobloxGlobalState = nil
						Api.ExecutionChannel.Clear()
						Api.ScriptContext = 0

						for {
							if ListManager := dm.Traverse(
								"CoreGui", "RobloxGui", "Modules", "PlayerList", "PlayerListManager",
							); ListManager != nil && ListManager.Name() == "PlayerListManager" &&
								len(dm.Traverse("Players").Children()) > 0 &&
								len(dm.Traverse("Workspace").Children()) > 1 {
								break
							}
							time.Sleep(time.Second)
						}

						time.Sleep(time.Millisecond * 500)

						dm = Api.Sesh.Game.RenderJob.DataModel()
						sc := uintptr(dm.Traverse("ScriptContext").Self)

						Api.ScriptContext = sc

						imgui.Message.Success(":)", "welcome to the backrooms.")

						fn(context.Background(), TeleportSession{
							DataModel:     uintptr(dm.Self),
							ScriptContext: sc,
							Name:          dm.Name(),
							State:         NewLuaState(Api.Sesh),
						})

					}
				} else {
					if dm.Name() == "LuaApp" && ingame && last != uintptr(dm.Self) {
						last = uintptr(dm.Self)

						imgui.Message.Success(":)", "wake up to reality.")

						Api.LunaState = nil
						Api.RobloxGlobalState = nil
						Api.ExecutionChannel.Clear()
						Api.QueuedList = []Yieldable{}
						Api.ScriptContext = 0

					}
				}
			}
		}

		detect()

		for {
			time.Sleep(500 * time.Millisecond)
			detect()
		}
	}
}
