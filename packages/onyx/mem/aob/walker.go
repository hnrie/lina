package aob

import (
	. "main/packages/onyx/mem"
	. "main/packages/onyx/mem/rtti"
	"slices"
	"strings"
	"unsafe"
)

func PointerWalk[T any](process any, startAddr uintptr, maxDepth, searchSize, stepSize int, matcher MatcherFunc[T]) *WalkResult[T] {
	Luna := process.(*Luna)

	if startAddr == 0 || maxDepth <= 0 {
		return nil
	}
	queue := []*walkNode{{addr: startAddr, depth: 0, parent: nil}}
	visited := make(map[uintptr]bool)
	visited[startAddr] = true
	readBuf := make([]byte, searchSize)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.depth >= maxDepth {
			continue
		}
		bytesRead, err := Luna.ReadRaw(current.addr, unsafe.Pointer(&readBuf[0]), uintptr(searchSize))
		if err != nil || bytesRead == 0 {
			continue
		}
		limit := min(int(bytesRead), searchSize)

		for offset := 0; offset < limit; offset += stepSize {
			if offset+8 > limit {
				break
			}
			ptrVal := *(*uintptr)(unsafe.Pointer(&readBuf[offset]))
			if ptrVal < 0x10000 || ptrVal > 0x7FFFFFFFFFFF {
				continue
			}
			if visited[ptrVal] {
				continue
			}
			if val, ok := matcher(Luna, ptrVal); ok {
				return &WalkResult[T]{
					BaseAddress:  startAddr,
					Offsets:      reconstructPath(current, uintptr(offset)),
					FinalValue:   val,
					FoundAddress: ptrVal,
				}
			}
			visited[ptrVal] = true
			queue = append(queue, &walkNode{
				addr:   ptrVal,
				offset: uintptr(offset),
				parent: current,
				depth:  current.depth + 1,
			})
		}
	}
	return nil
}

func reconstructPath(node *walkNode, finalOffset uintptr) []uintptr {
	var path []uintptr
	path = append(path, finalOffset)
	curr := node
	for curr != nil && curr.parent != nil {
		path = append(path, curr.offset)
		curr = curr.parent
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func RTTIScanMatcher(find ...string) MatcherFunc[string] {
	cache := make(map[uintptr]string)
	return func(process *Luna, address uintptr) (string, bool) {
		if cachedName, ok := cache[address]; ok {
			for _, s := range find {
				if strings.Contains(cachedName, s) {
					return cachedName, true
				}
			}
			return "", false
		}
		name, err := RTTIInformation(process, address)
		if err != nil {
			return "", false
		}
		cache[address] = name
		for _, s := range find {
			if strings.Contains(name, s) {
				return name, true
			}
		}
		return "", false
	}
}

func StringScanMatcher(find ...string) MatcherFunc[string] {
	return func(process *Luna, address uintptr) (string, bool) {
		val_ := ReadProcessMemory[[256]byte](process, address)
		val := string(val_[:])
		if idx := strings.IndexByte(val, 0); idx != -1 {
			val = val[:idx]
		}
		if slices.Contains(find, val) {
			return val, true
		}
		return "", false
	}
}

func NumberScanMatcher[N int | int32 | int64 | uintptr](find ...N) MatcherFunc[N] {
	return func(process *Luna, address uintptr) (N, bool) {
		val := ReadProcessMemory[N](process, address)
		if slices.Contains(find, val) {
			return val, true
		}
		return 0, false
	}
}
