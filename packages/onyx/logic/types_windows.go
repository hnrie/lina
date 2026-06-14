package logic

import (
	"context"
	"sync"
	"syscall"
	"time"
	"unsafe"

	. "main/packages/onyx/mem"
	. "main/packages/onyx/mem/renderview"
	. "main/packages/onyx/static"

	"git.mills.io/prologic/bitcask"
	"github.com/crazywolf132/conduit/client"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"
)

var (
	m     sync.Mutex
	s     sync.Mutex
	ts    sync.Mutex
	Conns map[int64]*Connections = make(map[int64]*Connections)
)

type (
	game struct {
		RenderJob *Render
	}
	Sesh struct {
		Luna                       *Luna
		Username, Avatar, GameName string
		Injected, Active           bool
		Game                       game            `json:"-"`
		Queue                      []string        `json:"-"`
		RobloxWebSocket            *websocket.Conn `json:"-"`
		WebSocket                  *client.Client  `json:"-"`
		Handle                     windows.HWND    `json:"-"`
	}
)

func Session(luna *Luna) *Sesh {
	render := RenderView(luna)
	sesh := &Sesh{
		Luna: luna,
		Game: game{
			RenderJob: &render,
		},
	}
	sesh.Handle = GetHWNDFromPID(uint32(sesh.Luna.Pid))
	return sesh
}

func GetHWNDFromPID(pid uint32) windows.HWND {
	var foundHwnd windows.HWND
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var procID uint32
		GetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&procID)))
		if procID == pid {
			foundHwnd = windows.HWND(hwnd)
			return 0
		}
		return 1
	})
	EnumWindows.Call(cb, 0)
	return foundHwnd
}

type Resp struct {
	Error   string `json:"error"`
	Success bool   `json:"success"`
}

type meta struct {
	Size int64
	Hash uint64
	When time.Time
	Path string
}

type App struct {
	ctx      context.Context
	dir      string
	Database *bitcask.Bitcask
	Config   Config
	configMu sync.RWMutex
}

type Connections struct {
	Pid       int64
	Username  string
	Avatar    string
	Execution bool
	InGame    bool
	Conn      *websocket.Conn
}

type Tab struct {
	Tabs []TabContents `json:"tabsv2"`
}

type TabContents struct {
	Content     string `json:"content"`
	TabName     string `json:"name"`
	TabUUID     string `json:"id"`
	TabSelected bool   `json:"selected"`
	MadeAt      int64  `json:"time"`
	Position    int64  `json:"position"`
}

type ScriptItem struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
}

type Task struct {
	Type   string `json:"type"`
	Source string `json:"source"`
}

func GetRunningPIDs() []uint32 {
	var activePIDs []uint32
	p, _ := process.Processes()
	for _, proc := range p {
		if name, err := proc.Name(); err == nil && name == "RobloxPlayerBeta.exe" {
			activePIDs = append(activePIDs, uint32(proc.Pid))
		}
	}
	return activePIDs
}
