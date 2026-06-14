package websocket

import (
	"net/http"
	"strings"
	"sync"
	"time"

	. "main/packages/onyx/logic/luau/Api"

	gws "github.com/gorilla/websocket"
)

type Library struct {
	WebSocket WebSocketTable `lua:"WebSocket"`
}

type WebSocketTable struct {
	Connect func(*LuaState) int `lua:"connect"`
}

type WebSocketObject struct {
	Send      func(*LuaState) int `lua:"Send"`
	Close     func(*LuaState) int `lua:"Close"`
	OnMessage EventTable          `lua:"OnMessage"`
	OnClose   EventTable          `lua:"OnClose"`
}

type EventTable struct {
	Connect func(*LuaState) int `lua:"Connect"`
	Once    func(*LuaState) int `lua:"Once"`
	Wait    func(*LuaState) int `lua:"Wait"`
}

type Client struct {
	Ls        *LuaState
	Conn      *gws.Conn
	Closed    bool
	Mutex     sync.Mutex
	OnMessage *Event
	OnClose   *Event
}

type Event struct {
	Mutex     sync.Mutex
	Callbacks map[int]*EventCallback
	NextID    int
}

type EventCallback struct {
	Ref  int
	Once bool
	Dead bool
}

type QueuedEvent struct {
	Event *Event
	Args  []string
}

var (
	queueMu       sync.Mutex
	queues        = make(map[*LuaState][]QueuedEvent)
	pollerMu      sync.Mutex
	activePollers = make(map[*LuaState]bool)
)

func Init(L *LuaState) {
	Register(L, Library{
		WebSocket: WebSocketTable{
			Connect: connect,
		},
	})
}

func connect(Ls *LuaState) int {
	var Url string = strings.TrimSpace(Ls.CheckString(1))
	var Lower string = strings.ToLower(Url)

	if !strings.HasPrefix(Lower, "ws://") && !strings.HasPrefix(Lower, "wss://") {
		Ls.Error("invalid protocol, expected ws:// or wss://")
		return 0
	}

	var Dialer gws.Dialer = gws.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
	}

	Conn, _, err := Dialer.Dial(Url, nil)
	if err != nil {
		Ls.Error("%v", err.Error())
		return 0
	}

	var ClientPtr *Client = &Client{
		Ls:        Ls,
		Conn:      Conn,
		OnMessage: NewEvent(),
		OnClose:   NewEvent(),
	}

	go ClientPtr.readLoop()

	StructToTable(Ls, WebSocketObject{
		Send: func(Ls *LuaState) int {
			return ClientPtr.Send(Ls)
		},
		Close: func(Ls *LuaState) int {
			return ClientPtr.Close(Ls)
		},
		OnMessage: ClientPtr.OnMessage.Table(),
		OnClose:   ClientPtr.OnClose.Table(),
	})

	pollerMu.Lock()
	if !activePollers[Ls] {
		activePollers[Ls] = true
		go func(State *LuaState) {
			for {
				time.Sleep(time.Millisecond * 100)
				Poll(State)
			}
		}(Ls)
	}
	pollerMu.Unlock()

	return 1
}

func (C *Client) readLoop() {
	for {
		MessageType, Data, err := C.Conn.ReadMessage()
		if err != nil {
			C.Mutex.Lock()
			var WasClosed bool = C.Closed
			C.Closed = true
			C.Mutex.Unlock()

			if !WasClosed {
				queueEvent(C.Ls, C.OnClose, err.Error())
				_ = C.Conn.Close()
			}
			return
		}

		if MessageType == gws.TextMessage || MessageType == gws.BinaryMessage {
			queueEvent(C.Ls, C.OnMessage, string(Data))
		}
	}
}

func (C *Client) Send(Ls *LuaState) int {
	var Data string

	if Ls.IsString(2) {
		Data = Ls.ToString(2)
	} else if Ls.IsString(1) {
		Data = Ls.ToString(1)
	} else {
		Ls.TypeError(1, "string")
		return 0
	}

	C.Mutex.Lock()
	defer C.Mutex.Unlock()

	if C.Closed {
		Ls.Error("websocket is closed")
		return 0
	}

	if err := C.Conn.WriteMessage(gws.TextMessage, []byte(Data)); err != nil {
		Ls.Error("%v", err.Error())
		return 0
	}

	return 0
}

func (C *Client) Close(Ls *LuaState) int {
	C.Mutex.Lock()

	if C.Closed {
		C.Mutex.Unlock()
		return 0
	}

	C.Closed = true

	_ = C.Conn.WriteControl(
		gws.CloseMessage,
		gws.FormatCloseMessage(gws.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)

	_ = C.Conn.Close()
	C.Mutex.Unlock()

	queueEvent(C.Ls, C.OnClose, "closed")
	return 0
}

func NewEvent() *Event {
	return &Event{
		Callbacks: map[int]*EventCallback{},
	}
}

func (E *Event) Table() EventTable {
	return EventTable{
		Connect: func(Ls *LuaState) int {
			return E.Connect(Ls, false)
		},
		Once: func(Ls *LuaState) int {
			return E.Connect(Ls, true)
		},
		Wait: func(Ls *LuaState) int {
			Ls.Error("event Wait is not implemented")
			return 0
		},
	}
}

func (E *Event) Connect(Ls *LuaState, Once bool) int {
	var Arg int = 1
	if !Ls.IsFunction(Arg) && Ls.IsFunction(2) {
		Arg = 2
	}

	if !Ls.IsFunction(Arg) {
		Ls.TypeError(Arg, "function")
		return 0
	}

	Ls.PushValue(Arg)
	var Ref int = Ls.Ref(-1)

	E.Mutex.Lock()
	E.NextID++
	var Id int = E.NextID

	var Cb *EventCallback = &EventCallback{
		Ref:  Ref,
		Once: Once,
	}

	E.Callbacks[Id] = Cb
	E.Mutex.Unlock()

	StructToTable(Ls, struct {
		Disconnect func(*LuaState) int `lua:"Disconnect"`
	}{
		Disconnect: func(Ls *LuaState) int {
			E.Mutex.Lock()
			if CbObj, Ok := E.Callbacks[Id]; Ok && !CbObj.Dead {
				CbObj.Dead = true
				Ls.Unref(CbObj.Ref)
				delete(E.Callbacks, Id)
			}
			E.Mutex.Unlock()
			return 0
		},
	})

	return 1
}

func queueEvent(Ls *LuaState, EventPtr *Event, Args ...string) {
	if EventPtr == nil || Ls == nil {
		return
	}

	queueMu.Lock()
	queues[Ls] = append(queues[Ls], QueuedEvent{
		Event: EventPtr,
		Args:  Args,
	})
	queueMu.Unlock()
}

func Poll(Ls *LuaState) int {
	queueMu.Lock()
	var Items []QueuedEvent = queues[Ls]
	delete(queues, Ls)
	queueMu.Unlock()

	for _, Item := range Items {
		fireEvent(Ls, Item.Event, Item.Args...)
	}
	return 0
}

func fireEvent(Ls *LuaState, EventPtr *Event, Args ...string) {
	if EventPtr == nil {
		return
	}

	EventPtr.Mutex.Lock()
	var Callbacks []*EventCallback = make([]*EventCallback, 0, len(EventPtr.Callbacks))

	for Id, Cb := range EventPtr.Callbacks {
		if Cb.Dead {
			continue
		}
		Callbacks = append(Callbacks, Cb)
		if Cb.Once {
			Cb.Dead = true
			delete(EventPtr.Callbacks, Id)
		}
	}
	EventPtr.Mutex.Unlock()

	for _, Cb := range Callbacks {
		Ls.GetRef(Cb.Ref)
		for _, Arg := range Args {
			Ls.PushString(Arg)
		}
		if err := Ls.PCall(len(Args), 0); err != nil {
			Print(3, "websocket callback error: %v", err.Error())
			Ls.Pop(1)
		}
		if Cb.Once {
			Ls.Unref(Cb.Ref)
		}
	}
}
