package logic

import (
	"sync"
)

var (
	Sessions  []*Sesh
	SessionMu sync.Mutex
)

func RemoveSession(pid uint32) {
	SessionMu.Lock()
	defer SessionMu.Unlock()
	for i, session := range Sessions {
		if session.Luna.Pid == uintptr(pid) {
			Sessions = append(Sessions[:i], Sessions[i+1:]...)
			break
		}
	}
}

func UpdateInstance(L *Sesh) {
	for i, luna := range Sessions {
		if luna == L {
			Sessions[i] = L
		}
	}
}
