package luastate

import (
	. "main/packages/onyx/mem"
)

type (
	LuaState struct {
		Address uintptr
		Luna    *Luna
	}
	WeakThread struct {
		Address uintptr
		Luna    *Luna
	}
	Node struct {
		Address uintptr
		Luna    *Luna
	}
	ThreadRef struct {
		Address uintptr
		Luna    *Luna
	}
)
