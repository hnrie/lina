package Api

/*
#include "lua.h"
#include <stdint.h>
#include <stdlib.h>

extern void set_original_step(uintptr_t ptr);
extern uintptr_t get_hook_ptr();
void setup_sc_callback(uintptr_t sc);
*/
import "C"
import (
	"sync/atomic"
	"unsafe"
)

func (S *Luau) Hook() {

	C.setup_sc_callback(C.uintptr_t(Api.ScriptContext))

	if Api.Sesh == nil {
		return
	}

	job := Api.Sesh.Game.RenderJob.Container().Job("Heartbeat")
	if job == nil || job.Address == 0 {
		return
	}

	jobVFtableAddr := *(*uintptr)(unsafe.Pointer(job.Address))
	jobVFtable := (*[9]uintptr)(unsafe.Pointer(jobVFtableAddr))
	newVFtable := (*[9]uintptr)(C.malloc(C.size_t(9 * unsafe.Sizeof(uintptr(0)))))

	for i := 0; i < 9; i++ {
		newVFtable[i] = jobVFtable[i]
	}

	C.set_original_step(C.uintptr_t(jobVFtable[1]))

	newVFtable[1] = uintptr(C.get_hook_ptr())

	*(*uintptr)(unsafe.Pointer(job.Address)) = uintptr(unsafe.Pointer(newVFtable))

}

var inStepHook atomic.Bool

//export GoStepHookPayload
func GoStepHookPayload() C.int {
	if !inStepHook.CompareAndSwap(false, true) {
		return 0
	}
	defer inStepHook.Store(false)

	defer func() {
		if r := recover(); r != nil {
			Print(3, "%v", r)
		}
	}()

	if Api.ExecutionChannel.Len() == 0 {
		return 0
	}

	yield, ok := Api.ExecutionChannel.Pop()
	if !ok {
		return 0
	}

	switch yield.Type {
	case Execute:
		ExecuteCode(yield.Source)

	case Yield:
		if yield.Yield.Thread == nil {
			return 0
		}

		debug := yield.Yield.Thread.LuaDebug()
		weak := yield.Yield.Thread.WeakThread()
		thread := &weak

		ScriptContextResume(&debug, &thread, yield.Yield.Arguments, false, "")
	}

	return 0
}
