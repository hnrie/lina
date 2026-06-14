package Api

/*
#include "lua.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

func (s *LuaState) Load(chunkName string, bytecode []byte, env int) error {
	cChunkName := C.CString(chunkName)
	defer C.free(unsafe.Pointer(cChunkName))

	cBytecode := (*C.char)(unsafe.Pointer(&bytecode[0]))
	size := C.size_t(len(bytecode))

	result := C.luau_load(s.cptr(), cChunkName, cBytecode, size, C.int(env))

	if result != 0 {
		errMsg := C.GoString(C.lua_tolstring(s.cptr(), -1, nil))
		C.lua_settop(s.cptr(), -2)
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

func (s *LuaState) PCall(args, results int) error {
	state := s.cptr()
	if result := C.lua_pcall(state, C.int(args), C.int(results), 0); result != 0 {
		errMsg := C.GoString(C.lua_tolstring(state, -1, nil))
		C.lua_settop(state, -2)
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

func (s *LuaState) DoString(source string) error {
	bytecode := Compile(source, CompileOptions{
		OptimizationLevel: 1,
		DebugLevel:        2,
	})
	if err := s.Load("DoString", bytecode, 0); err != nil {
		return err
	}
	return s.PCall(0, 0)
}

func (s *LuaState) Call(nargs, nresults int) {
	C.lua_call(s.cptr(), C.int(nargs), C.int(nresults))
}

func (s *LuaState) Yield(nresults int) int {
	return int(C.lua_yield(s.cptr(), C.int(nresults)))
}

func (s *LuaState) Resume(from *LuaState, narg int) int {
	var fromPtr *C.lua_State
	if from != nil {
		fromPtr = from.cptr()
	}
	return int(C.lua_resume(s.cptr(), fromPtr, C.int(narg)))
}

func (s *LuaState) LuaStatus() int {
	return int(C.lua_status(s.cptr()))
}

func (s *LuaState) GC(what, data int) int {
	return int(C.lua_gc(s.cptr(), C.int(what), C.int(data)))
}

func (s *LuaState) Ref(idx int) int {
	return int(C.lua_ref(s.cptr(), C.int(idx)))
}

func (s *LuaState) Unref(ref int) {
	C.lua_unref(s.cptr(), C.int(ref))
}

type Queue[T any] struct {
	mu    sync.Mutex
	items []T
}

func (q *Queue[T]) Push(item ...T) {
	q.mu.Lock()
	q.items = append(q.items, item...)
	q.mu.Unlock()
}

func (q *Queue[T]) Pop() (T, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		var zero T
		return zero, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *Queue[T]) Clear() {
	q.mu.Lock()
	q.items = q.items[:0]
	q.mu.Unlock()
}
