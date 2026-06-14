package logic

import (
	"context"
	"sync"
	"time"

	. "main/packages/onyx/mem"
	. "main/packages/onyx/mem/renderview"

	"git.mills.io/prologic/bitcask"
	"github.com/crazywolf132/conduit/client"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/process"
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
		Username, Avatar, GameName string
		Injected, Active           bool
		Luna                       *Luna
		Game                       game            `json:"-"`
		Queue                      []string        `json:"-"`
		RobloxWebSocket            *websocket.Conn `json:"-"`
		WebSocket                  *client.Client  `json:"-"`
		Handle                     uintptr         `json:"-"`
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
	sesh.Handle = sesh.GetHWNDFromPID(uint32(sesh.Luna.Pid))
	return sesh
}

func (p *Sesh) GetHWNDFromPID(pid uint32) uintptr {
	return 0
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

type Config struct {
	TopMost     bool   `json:"topMost"`
	AutoAttach  bool   `json:"autoAttach"`
	Theme       string `json:"theme"`
	AutoExecute bool   `json:"autoExecute"`
	DiscordRPC  bool   `json:"discordRPC"`
	Error       string `json:"error,omitempty"`
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
		if name, err := proc.Name(); err == nil && name == "RobloxPlayer" {
			activePIDs = append(activePIDs, uint32(proc.Pid))
			if directory, err := GetAppBundleByPid(int32(proc.Pid)); err == nil {
				if err := SignForDebugging(directory); err != nil && err.Error() != "process is already signed" {
					logger.LogError("Unable to sign roblox process..")
					continue
				} else if err == nil {
					logger.LogError("Roblox has been patched, please reopen it!")
					proc.Kill()
					continue
				}
			}
		}
	}
	return activePIDs
}
