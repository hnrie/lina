package Api

/*
#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>
#include "lua.h"
#include "lualib.h"

typedef struct live_thread_ref
{
    int __atomic_refs;
    lua_State* th;
    int thread_id;
    int ref_id;
} live_thread_ref;

typedef struct weak_thread_ref_t
{
    uint8_t pad_0[16];
    struct weak_thread_ref_t* previous;
    struct weak_thread_ref_t* next;
    live_thread_ref* liveThreadRef;
    struct
    {
        uint8_t pad_0[8];
        struct weak_thread_ref_t* wtr;
    }* node;
    uint8_t pad_1[8];
} weak_thread_ref_t;

typedef struct lua_debug_result_t
{
    int32_t result;
    int32_t unk_1[4];
} lua_debug_result_t;

typedef struct node_t
{
    uint8_t pad_0[8];
    weak_thread_ref_t* wtr;
} node_t;

typedef struct weak_lua_thread_reference_t
{
    int32_t references;
    lua_State* thread;
    int32_t thread_reference;
    int32_t object_id;
    int32_t unk_1;
    int32_t unk_2;
} weak_lua_thread_reference_t;

typedef struct SWeakObjectRef2
{
    uint8_t pad_0[0x28];
    uintptr_t L;
    int32_t ThreadId;
    int32_t ObjectId;
} SWeakObjectRef2;

typedef struct SFunctionScriptSlotImpl
{
    uintptr_t VFTable;
    uint8_t pad_0[0x38];
    uintptr_t WaitRef;
    uint8_t pad_1[0x20];
    uintptr_t FunctionRef;
    uintptr_t pad_2;
    uint8_t pad_3[0x1D];
    bool IsOnce;
} SFunctionScriptSlotImpl;

typedef struct SSlotBase
{
    int32_t Strong;
    int32_t Weak;
    uintptr_t DoCall;
    uintptr_t Next;
    uintptr_t Atomic;
    uintptr_t Owner;
    uintptr_t DestroySignal;
    uintptr_t Storage0;
    uintptr_t Storage1;
} SSlotBase;

typedef struct SSignalConnectionBridge
{
    uintptr_t ISlot;
    uintptr_t SharedRefISlot;
    uintptr_t Pad0;
    uintptr_t Pad1;
} SSignalConnectionBridge;
*/
import "C"
import (
	"syscall"
	"unsafe"
)

const (
	LUA_EXTRA_SIZE        = 1
	LUA_SIZECLASSES       = 40
	LUA_MEMORY_CATEGORIES = 256
	LUA_UTAG_LIMIT        = 128
	LUA_LUTAG_LIMIT       = 128
	TM_N                  = 21
	LUA_IDSIZE            = 256
	LUA_MINSTACK          = 20
	LUAI_MAXCALLS         = 20000
	LUAI_MAXCCALLS        = 200
	LUA_BUFFERSIZE        = 512
	LUA_MINSTRTABSIZE     = 32
	LUA_MAXCAPTURES       = 32
	LUA_MULTRET           = -1

	LUAI_MAXCSTACK    = 8000
	LUA_REGISTRYINDEX = -LUAI_MAXCSTACK - 2000
	LUA_ENVIRONINDEX  = -LUAI_MAXCSTACK - 2001
	LUA_GLOBALSINDEX  = -LUAI_MAXCSTACK - 2002

	LUA_CALLINFO_RETURN = 1 << 0
	LUA_CALLINFO_HANDLE = 1 << 1
	LUA_CALLINFO_NATIVE = 1 << 2
)

const (
	LUA_OK int = iota
	LUA_YIELD
	LUA_ERRRUN
	LUA_ERRSYNTAX
	LUA_ERRMEM
	LUA_ERRERR
	LUA_BREAK
)

const (
	TM_INDEX = iota
	TM_NEWINDEX
	TM_MODE
	TM_NAMECALL
	TM_CALL
	TM_ITER
	TM_LEN
	TM_EQ
	TM_ADD
	TM_SUB
	TM_MUL
	TM_DIV
	TM_IDIV
	TM_MOD
	TM_POW
	TM_UNM
	TM_LT
	TM_LE
	TM_CONCAT
	TM_TYPE
	TM_METATABLE
)

const (
	LUA_TNIL = iota
	LUA_TBOOLEAN
	LUA_TLIGHTUSERDATA
	LUA_TNUMBER
	LUA_TINTEGER
	LUA_TVECTOR
	LUA_TSTRING
	LUA_TTABLE
	LUA_TFUNCTION
	LUA_TUSERDATA
	LUA_TTHREAD
	LUA_TBUFFER
	LUA_TCLASS
	LUA_TOBJECT
	LUA_TDEADKEY
	LUA_TPROTO
	LUA_TUPVAL
	LUA_T_COUNT = LUA_TDEADKEY
)

type Instruction uint32
type LuaCFunction uintptr

func (L LuaCFunction) Call(ls *LuaState) int {
	ret, _, _ := syscall.SyscallN(uintptr(L), uintptr(unsafe.Pointer(ls)))
	return int(ret)
}

type LuaContinuation uintptr
type LuaAlloc uintptr
type Value uint64
type StkId *TValue

type WeakThreadRefNodule = C.weak_thread_ref_t
type LuaDebugResult = C.lua_debug_result_t
type WeakThreadRef = C.weak_lua_thread_reference_t
type SWeakObjectRef2 = C.SWeakObjectRef2

type LuaDebug struct {
	Name        string
	What        string
	Source      string
	ShortSrc    string
	LineDefined int
	CurrentLine int
	NUpvals     uint8
	NParams     uint8
	IsVarArg    int8
	UserData    unsafe.Pointer
	SsBuf       [LUA_IDSIZE]byte
}

type StdPtrControlBlock struct {
	VTable    uintptr
	UseCount  int32
	WeakCount int32
}

type StdWeakPtr struct {
	Ptr uintptr
	Rep uintptr
}

func (w *StdWeakPtr) Expired() bool {
	if w.Rep == 0 {
		return true
	}
	return (*StdPtrControlBlock)(unsafe.Pointer(w.Rep)).UseCount == 0
}

func (w *StdWeakPtr) Lock() (uintptr, bool) {
	if w.Expired() {
		return 0, false
	}
	return w.Ptr, true
}

func (La *LuaState) LuaDebug() LuaDebugResult {
	return LuaDebugResult{result: 0}
}

func (La *LuaState) WeakThread() SWeakObjectRef2 {
	ref := SWeakObjectRef2{
		L:        C.uintptr_t(uintptr(unsafe.Pointer(La.cptr()))),
		ThreadId: 0,
		ObjectId: 0,
	}
	La.PushThread()
	ref.ObjectId = C.int32_t(La.Ref(-1))
	La.Pop(1)
	return ref
}

type CommonHeader struct {
	Memcat uint8
	Tt     uint8
	Marked uint8
}

type GCObject struct {
	Memcat uint8
	Tt     uint8
	Marked uint8
	_      [5]byte
	Gclist *GCObject
}

type TValue struct {
	Value Value
	Extra [LUA_EXTRA_SIZE]int32
	Tt    int32
}

func (t *TValue) Type() int32     { return t.Tt }
func (t *TValue) Gc() uintptr     { return uintptr(t.Value) }
func (t *TValue) SetGc(p uintptr) { t.Value = Value(p) }

type TKey struct {
	Value  Value
	Extra  [LUA_EXTRA_SIZE]int32
	TtNext uint32
}

type LuaNode struct {
	Val TValue
	Key TKey
}

type CallInfo struct {
	Base     StkId
	Func     StkId
	Top      StkId
	Savedpc  *Instruction
	Nresults int32
	Flags    uint32
}

type TString struct {
	Memcat uint8
	Tt     uint8
	Marked uint8
	_      uint8
	Flag   int16
	Atom   int16
	Next   *TString
	Hash   uint32
	Len    uint32
	Data   [1]byte
}

func (s *TString) String() string {
	if s == nil || s.Len == 0 {
		return ""
	}
	return unsafe.String(&s.Data[0], s.Len)
}

type StringTable struct {
	Hash **TString
	Nuse uint32
	Size int32
}

type LuaTable struct {
	Memcat    uint8
	Tt        uint8
	Marked    uint8
	Lsizenode uint8
	Safeenv   uint8
	Nodemask8 uint8
	Tmcache   uint8
	Readonly  uint8
	Sizearray int32
	Lastfree  int32
	Metatable *LuaTable
	Array     *TValue
	Gclist    *GCObject
	Node      *LuaNode
}

type LuaState struct {
	Memcat       uint8
	Tt           uint8
	Marked       uint8
	Status       uint8
	Activememcat uint8
	Isactive     bool
	Singlestep   bool
	_            uint8
	NCcalls      uint16
	BaseCcalls   uint16
	Cachedslot   int32
	EndCi        *CallInfo
	BaseCi       *CallInfo
	Gclist       *GCObject
	Global       *GlobalState
	Base         StkId
	Top          StkId
	Stack        StkId
	Ci           *CallInfo
	StackLast    StkId
	Namecall     *TString
	Gt           *LuaTable
	Userdata     *RobloxExtraSpace
	Openupval    *UpVal
	Stacksize    int32
	SizeCi       int32
}

type Closure struct {
	Memcat    uint8
	Tt        uint8
	Marked    uint8
	Stacksize uint8
	NUpvalues uint8
	IsC       uint8
	Preload   uint8
	_         uint8
	Usage     uint64
	Gclist    *GCObject
	Env       *LuaTable
	Union     [0x28]byte
}

type CClosure struct {
	DebugName uintptr
	Cont      uintptr
	F         LuaCFunction
	Upvals    [1]TValue
}

type LClosure struct {
	P      *Proto
	Uprefs [1]TValue
}

func (cl *Closure) UpValues() []TValue {
	if cl.NUpvalues == 0 {
		return nil
	}
	if cl.IsC == 1 {
		return unsafe.Slice(&cl.AsC().Upvals[0], int(cl.NUpvalues))
	}
	return unsafe.Slice(&cl.AsL().Uprefs[0], int(cl.NUpvalues))
}

func (cl *Closure) AsC() *CClosure {
	return (*CClosure)(unsafe.Pointer(&cl.Union[0]))
}

func (cl *Closure) AsL() *LClosure {
	return (*LClosure)(unsafe.Pointer(&cl.Union[0]))
}

type Proto struct {
	Memcat          uint8
	Tt              uint8
	Marked          uint8
	Maxstacksize    uint8
	Flags           uint8
	Numparams       uint8
	Nups            uint8
	IsVararg        uint8
	Source          uintptr
	Debuginsn       uintptr
	Debugname       uintptr
	Gclist          *GCObject
	Execdata        uintptr
	Exectarget      uintptr
	Codeentry       *Instruction
	Userdata        uintptr
	P               **Proto
	Lineinfo        uintptr
	Upvalues        uintptr
	Locvars         uintptr
	K               *TValue
	Code            *Instruction
	Abslineinfo     uintptr
	Typeinfo        uintptr
	Linedefined     int32
	Sizelocvars     int32
	Bytecodeid      int32
	Sizetypeinfo    int32
	Sizelineinfo    int32
	Sizek           int32
	Sizep           int32
	Sizecode        int32
	Sizeupvalues    int32
	Linegaplog2     int32
	Feedbackvec     uintptr
	Feedbackvecsize uint32
	Funid           uint32
}

type UpVal struct {
	CommonHeader
	Markedopen   uint8
	_            [4]byte
	V            *TValue
	UnionStorage [24]byte
}

type LuaPage struct {
	Listprev   *LuaPage
	Listnext   *LuaPage
	Prev       *LuaPage
	Next       *LuaPage
	PageSize   int32
	BlockSize  int32
	FreeList   uintptr
	FreeNext   int32
	BusyBlocks int32
	_          [0x8]byte
	Data       [1]byte
}

type LuaCallbacks struct {
	Userdata            uintptr
	Interrupt           uintptr
	Panic               uintptr
	OnAllocate          uintptr
	UserThread          uintptr
	DebugBreak          uintptr
	DebugProtectedError uintptr
	DebugStep           uintptr
	DebugInterrupt      uintptr
	UserAtom            uintptr
}

type GlobalState struct {
	GCthreshold       uintptr
	Totalbytes        uintptr
	Weak              *GCObject
	Grayagain         *GCObject
	Gray              *GCObject
	Currentwhite      uint8
	Gcstate           uint8
	_                 [6]byte
	Frealloc          uintptr
	Ud                uintptr
	Gcstepsize        int32
	Gcstepmul         int32
	Gcgoal            int32
	_                 int32
	Strt              StringTable
	Sweepgcopage      *LuaPage
	Allpages          *LuaPage
	Mainthread        *LuaState
	Allgcopages       *LuaPage
	Freepages         [LUA_SIZECLASSES]*LuaPage
	Uvhead            UpVal
	Freegcopages      [LUA_SIZECLASSES]*LuaPage
	Tmname            [TM_N]*TString
	Ttname            [14]*TString
	Mt                [14]*LuaTable
	Pseudotemp        TValue
	Registry          TValue
	Registryfree      int32
	_                 int32
	Errorjmp          uintptr
	Cb                LuaCallbacks
	Rngstate          uint64
	Ptrenckey         [4]uint64
	Ecb               [10]uintptr
	Ecbdata           [512]uint8
	Udatadirect       [130]uintptr
	Memcatbytes       [256]uintptr
	Udatagc           [128]uintptr
	Udatamt           [128]*LuaTable
	Lightuserdataname [128]*TString
	Udatadirectfields [130]*LuaTable
	Gcstats           [0xB8]byte
	Lastprotoid       uint32
	_                 uint32
	Gcmetrics         [0x100]byte
}

type Udata struct {
	Tt        uint8
	Memcat    uint8
	Marked    uint8
	Tag       uint8
	Len       int32
	Metatable uintptr
	Data      unsafe.Pointer
}

type Shared struct {
	_             [0x20]byte
	ScriptContext uintptr
}

type RobloxExtraSpace struct {
	Next         *RobloxExtraSpace
	Container    uintptr
	Prev         *RobloxExtraSpace
	Shared       [0x10]byte
	_            [0x8]byte
	Identity     uintptr
	_            [0x10]byte
	Source       StdWeakPtr
	Capabilities uintptr
	_            [32]byte
	Actor        StdWeakPtr
}
