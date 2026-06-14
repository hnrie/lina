package Api

/*
#include <stdio.h>
#include <Windows.h>
#include <stdbool.h>
#include <stddef.h>
*/
import "C"
import (
	"fmt"
)

func YieldFunc(L *LuaState, callback func(*LuaState) func(L *LuaState) int) int {
	L.PushThread()
	ref := L.Ref(-1)
	L.Pop(1)

	L.Base = L.Top
	L.Status = uint8(LUA_YIELD)
	L.Ci.Flags |= 1

	go func(state *LuaState, ref int) {
		defer func() {
			if r := recover(); r != nil {
				state.PushNil()
				state.PushString(fmt.Sprintf("%v", r))
				Api.ExecutionChannel.Push(Yieldable{
					Type: Yield,
					Yield: YieldData{
						Thread:    state,
						Arguments: 2,
						IsError:   true,
						ErrorMsg:  fmt.Sprintf("%v", r),
					},
				})
				state.Unref(ref)
			}
		}()
		Api.ExecutionChannel.Push(Yieldable{
			Type: Yield,
			Yield: YieldData{
				Thread:    state,
				Arguments: int(callback(state)(state)),
			},
		})
		state.Unref(ref)

	}(L, ref)

	return -0x10
}
