package main

/*

#include <stdint.h>
#include "packages/api/managers/imgui/hook.hpp"

void InstallCPP_Hook(uintptr_t renderJobAddr);
*/
import "C"
import (
	"main/packages/api/managers/imgui"
	. "main/packages/onyx/logic/luau/Api"
)

//export ProcessQ
func ProcessQ() {
	imgui.ProcessNotifications()
}

//export GoDrawLoop
func GoDrawLoop() {

	imgui.SetNextWindowSize(750, 500)

	imgui.Begin("Luna")

	imgui.Spacing()
	imgui.Separator()
	imgui.Spacing()

	imgui.EditorRender("LuaEditor", -1, -25)

	imgui.Spacing()
	imgui.Separator()

	if imgui.Button("Execute") {
		Api.ExecutionChannel.Push(Yieldable{
			Source: Compile(imgui.EditorGetText(), CompileOptions{
				OptimizationLevel: 1,
				DebugLevel:        2,
			}),
			Type: Execute,
		})
	}

	imgui.End()
}

var inited bool

func D3DHook(renderJob uintptr) {
	if !inited {
		inited = true
		imgui.EditorInit()
		imgui.EditorSetText("-- Welcome to Luna\nprint('Hello World')")
	}
	C.InstallCPP_Hook(C.uintptr_t(renderJob))

}
