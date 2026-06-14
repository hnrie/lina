package aob

import (
	. "main/packages/onyx/mem"
)

const WildcardByte = '?'

type (
	OptimizedCompiledPattern struct {
		FullPattern []byte
		Mask        []byte
		Pivot       []byte
		PivotOffset int
	}
	ScanConfig struct {
		MaxRegionSize      int
		Limit              int
		EndAtOne           bool
		Debug              bool
		AllowedProtections []uint32
	}
	MemoryReg struct {
		base  uintptr
		size  uintptr
		state uint32
		prot  uint32
		alloc uint32
	}
	MatcherFunc[T any] func(process *Luna, address uintptr) (T, bool)
	WalkResult[T any]  struct {
		BaseAddress  uintptr
		Offsets      []uintptr
		FinalValue   T
		FoundAddress uintptr
	}
	walkNode struct {
		addr   uintptr
		offset uintptr
		parent *walkNode
		depth  int
	}
)
