package Api

import "unsafe"

type Vmvalue struct {
	Get func(addr uintptr) uintptr
	Set func(addr uintptr, val uintptr)
}

func mem(addr uintptr) *uintptr {
	return (*uintptr)(unsafe.Pointer(addr))
}

var VMValue1 = Vmvalue{
	Get: func(p uintptr) uintptr { return *mem(p) - p },
	Set: func(p uintptr, val uintptr) { *mem(p) = val + p },
}

var VMValue2 = Vmvalue{
	Get: func(p uintptr) uintptr { return p - *mem(p) },
	Set: func(p uintptr, val uintptr) { *mem(p) = p - val },
}

var VMValue3 = Vmvalue{
	Get: func(p uintptr) uintptr { return p ^ *mem(p) },
	Set: func(p uintptr, val uintptr) { *mem(p) = p ^ val },
}

var VMValue4 = Vmvalue{
	Get: func(p uintptr) uintptr { return p + *mem(p) },
	Set: func(p uintptr, val uintptr) { *mem(p) = val - p },
}

var (
	CLOSURE_CONT_ENC      = VMValue1
	CLOSURE_DEBUGNAME_ENC = VMValue2
	LSTATE_STACKSIZE_ENC  = VMValue1
	PROTO_ABSLINEINFO_ENC = VMValue1
	PROTO_DEBUGINSN_ENC   = VMValue2
	PROTO_DEBUGNAME_ENC   = VMValue2
	PROTO_LINEINFO_ENC    = VMValue4
	PROTO_LOCVARS_ENC     = VMValue1
	PROTO_SOURCE_ENC      = VMValue1
	PROTO_TYPEINFO_ENC    = VMValue2
	PROTO_UPVALUES_ENC    = VMValue1
	PROTO_USERDATA_ENC    = VMValue4
	TSTRING_HASH_ENC      = VMValue3
	UDATA_META_ENC        = VMValue3
)
