package luastate

import (
	"fmt"
	. "main/packages/onyx/mem"
	"unsafe"
)

var (
	NodePointer  uintptr = 0x20
	Next         uintptr = 0x18
	First        uintptr = 0x8
	State        uintptr = 0x8
	Identity     uintptr = 0x0
	Capabilities uintptr = 0x0
)

func (Instance *LuaState) Iterate(callback func(callback WeakThread)) {
	node := Instance.Node()
	for ref := node.First(); ref.Address > 0x1000; ref = ref.Next() {
		callback(*ref)
	}
}
func (Instance *LuaState) Threads() (data []*WeakThread) {
	node := Instance.Node()
	for ref := node.First(); ref.Address > 0x1000; ref = ref.Next() {
		data = append(data, ref)
	}
	return
}
func (Instance *Node) First() *WeakThread {
	return &WeakThread{Luna: Instance.Luna, Address: ReadProcessMemory[uintptr](Instance.Luna,
		Instance.Address+First)}
}
func (Instance *WeakThread) Next() *WeakThread {
	return &WeakThread{Luna: Instance.Luna, Address: ReadProcessMemory[uintptr](Instance.Luna,
		Instance.Address+Next)}
}
func (Instance *LuaState) Node() *Node {
	return &Node{Luna: Instance.Luna, Address: ReadProcessMemory[uintptr](Instance.Luna,
		Instance.Address+NodePointer)}
}
func (Instance *WeakThread) ToLuaContainer() uintptr {
	return ReadProcessMemory[uintptr](Instance.Luna, Instance.Address+State)
}
func (Instance *WeakThread) ThreadRef() *ThreadRef {
	return &ThreadRef{Address: ReadProcessMemory[uintptr](Instance.Luna,
		Instance.Address+NodePointer), Luna: Instance.Luna}
}
func (Instance *ThreadRef) Thread() uintptr {
	return ReadProcessMemory[uintptr](Instance.Luna,
		Instance.Address+State)
}

func (Instance *LuaState) SetIdentity(identity uintptr) {
	caps := IdentityToCapabilities(uint32(identity))
	Instance.Iterate(func(callback WeakThread) {
		if thread := callback.ToLuaContainer(); thread != 0 {

			fmt.Printf("0w0 0x%x\n", thread)
			WriteProcessMemory(Instance.Luna, thread+Identity, identity)
			WriteProcessMemory(Instance.Luna, thread+Capabilities, caps)
		}
	})
}

func (Instance *LuaState) GetIdentity() uint16 {
	return ReadProcessMemory[uint16](Instance.Luna,
		Instance.Node().First().ToLuaContainer()+Identity)
}

type S struct {
	ThreadCount   int32
	ScriptContext uintptr
	ScriptVmState uintptr
}
type RBXExtraSpace struct {
	Previous         *RBXExtraSpace
	Count            uintptr
	Next             *RBXExtraSpace
	SharedExtraSpace *S
	Pad32            [16]byte
	Identity         int32
	Pad0034          [16]byte
	Capabilities     uintptr
	Script           [2]uintptr
	Actor            [2]uintptr
	Continuations    uintptr
	GlobalActorState bool
}

type LocalToRemoteMap = map[*RBXExtraSpace]uintptr

func CreateLuaState(L *Luna, address uintptr) (*RBXExtraSpace, LocalToRemoteMap) {
	visited := make(map[uintptr]*RBXExtraSpace)
	localToRemote := make(LocalToRemoteMap)
	head := readStateRecursive(L, address, visited, localToRemote)
	return head, localToRemote
}

func readStateRecursive(L *Luna, address uintptr,
	visited map[uintptr]*RBXExtraSpace,
	localToRemote LocalToRemoteMap) *RBXExtraSpace {
	if address == 0 {
		return nil
	}
	if existingState, ok := visited[address]; ok {
		return existingState
	}
	remoteStateData := ReadProcessMemory[RBXExtraSpace](L, address)
	localState := new(RBXExtraSpace)
	*localState = remoteStateData
	visited[address] = localState
	localToRemote[localState] = address
	if localState.SharedExtraSpace != nil {
		remoteSharedAddr := uintptr(unsafe.Pointer(localState.SharedExtraSpace))
		localShared := new(S)
		*localShared = ReadProcessMemory[S](L, remoteSharedAddr)
		localState.SharedExtraSpace = localShared
	}
	remoteNextAddr := uintptr(unsafe.Pointer(remoteStateData.Next))
	localState.Next = readStateRecursive(L, remoteNextAddr, visited, localToRemote)
	remotePrevAddr := uintptr(unsafe.Pointer(remoteStateData.Previous))
	localState.Previous = readStateRecursive(L, remotePrevAddr, visited, localToRemote)
	return localState
}
