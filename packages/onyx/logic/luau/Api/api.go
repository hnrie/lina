package Api

/*
#cgo CFLAGS: -I../VM/include -I../Compiler/include -I../Ast/include -I../Common/include -I../VM/src

#include "stddef.h"
#include "windows.h"
#include <stdlib.h>

#include "lua.h"
#include "luacode.h"
#include "lualib.h"

void luna_register_function(lua_State* L, const char* name);
*/
import "C"
import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	. "main/packages/onyx/instance"
	. "main/packages/onyx/logic"
	. "main/packages/onyx/mem"
	"strings"
	"unsafe"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/xxHash/xxHash32"
	"golang.org/x/sys/windows"
)

type CompileOptions struct {
	OptimizationLevel int
	DebugLevel        int
	VectorLib         string
	VectorCtor        string
	VectorType        string
	MutableGlobals    []string
}

func Compile(source string, opts CompileOptions) []byte {
	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))

	options := C.lua_CompileOptions{
		optimizationLevel: C.int(opts.OptimizationLevel),
		debugLevel:        C.int(opts.DebugLevel),
		typeInfoLevel:     C.int(1),
		coverageLevel:     C.int(2),
	}
	if opts.VectorLib != "" {
		cVectorLib := C.CString(opts.VectorLib)
		defer C.free(unsafe.Pointer(cVectorLib))
		options.vectorLib = cVectorLib
	}
	if opts.VectorCtor != "" {
		cVectorCtor := C.CString(opts.VectorCtor)
		defer C.free(unsafe.Pointer(cVectorCtor))
		options.vectorCtor = cVectorCtor
	}

	if len(opts.MutableGlobals) > 0 {
		ptrSize := unsafe.Sizeof((*C.char)(nil))
		cGlobalsArray := (**C.char)(C.malloc(C.size_t(int(ptrSize) * (len(opts.MutableGlobals) + 1))))
		defer C.free(unsafe.Pointer(cGlobalsArray))
		cGlobalsSlice := unsafe.Slice(cGlobalsArray, len(opts.MutableGlobals)+1)
		for i, global := range opts.MutableGlobals {
			cStr := C.CString(global)
			defer C.free(unsafe.Pointer(cStr))
			cGlobalsSlice[i] = cStr
		}
		cGlobalsSlice[len(opts.MutableGlobals)] = nil
		options.mutableGlobals = cGlobalsArray
	}

	var outSize C.size_t
	cBytecode := C.luau_compile(cSource, C.size_t(len(source)), &options, &outSize)
	if cBytecode == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cBytecode))
	return C.GoBytes(unsafe.Pointer(cBytecode), C.int(outSize))
}

func Compress(bytecode []byte) ([]byte, int) {

	const (
		bytecodeSignature = "RSB1"
		hashSeed          = uint32(42)
		hashMul           = 41
	)
	var (
		enc, _     = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(22)))
		compressed = enc.EncodeAll(bytecode, nil)
		buf        = make([]byte, 0, 8+len(compressed))
		tmp        = make([]byte, 4)
		hashBytes  = make([]byte, 4)
	)

	defer enc.Close()
	binary.LittleEndian.PutUint32(tmp, uint32(len(bytecode)))

	buf = append(
		append(
			buf,
			bytecodeSignature...,
		),
		append(
			tmp,
			compressed...,
		)...,
	)

	binary.LittleEndian.PutUint32(
		hashBytes,
		xxHash32.Checksum(buf, 42),
	)

	for i := 0; i < len(buf); i++ {
		buf[i] ^= byte(
			uint32(hashBytes[i&3]) + uint32(i*hashMul)&
				0xFF)
	}

	return buf, len(buf)
}

func Decompress(instance *Instance) []byte {

	bc := instance.Bytecode()
	source := bc.Bytecode

	if len(source) < 8 {
		return []byte("")
	}

	bytecodeMagic := []byte("RSB1")
	hashBytes := make([]byte, 4)

	copy(hashBytes, source[0:4])
	for i := 0; i < 4; i++ {
		hashBytes[i] ^= bytecodeMagic[i]
		hashBytes[i] -= byte(i * 41)
	}

	deobfuscated := make([]byte, len(source))
	copy(deobfuscated, source)
	for i := 0; i < len(deobfuscated); i++ {
		mask := hashBytes[i%4] + byte(i*41)
		deobfuscated[i] ^= mask
	}

	compressedData := deobfuscated[8:]

	if strings.Contains(instance.ClassName(), "ModuleScript") {
		return compressedData
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return []byte(err.Error())
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return []byte(err.Error())
	}

	return decompressed
}

func Decompile(source []byte) []byte {
	if len(source) < 8 {
		return []byte("")
	}

	bytecodeMagic := []byte("RSB1")
	hashBytes := make([]byte, 4)
	copy(hashBytes, source[0:4])

	for i := 0; i < 4; i++ {
		hashBytes[i] ^= bytecodeMagic[i]
		hashBytes[i] -= byte(i * 41)
	}

	deobfuscated := make([]byte, len(source))
	copy(deobfuscated, source)
	for i := 0; i < len(deobfuscated); i++ {
		mask := hashBytes[i%4] + byte(i*41)
		deobfuscated[i] ^= mask
	}

	compressedData := deobfuscated[8:]

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return []byte(err.Error())
	}
	defer decoder.Close()

	decompressed, err := decoder.DecodeAll(compressedData, nil)
	if err != nil {
		return []byte(err.Error())
	}

	return decompressed
}

func Sign(bytecode []byte) ([]byte, int) {
	var (
		hash, footer, tail = sha256.Sum256(bytecode), make([]byte, 24), [4]byte{}
	)
	binary.LittleEndian.PutUint32(footer[0:4], binary.LittleEndian.Uint32(hash[0:4]))
	binary.LittleEndian.PutUint16(tail[0:2],
		binary.LittleEndian.Uint16(footer[0:2])^0xC432)
	tail[2] = footer[2] ^ 0x6A
	tail[3] = footer[3] ^ 1
	copy(footer[20:24], tail[:])
	copy(footer[4:20], hash[4:20])
	return Compress(append(bytecode, footer...))
}

/*
func (vm *State) RegisterGoFunction(name string, fn func(*State) uintptr) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.lua_pushcclosurek(
		vm.cptr(),
		(C.lua_CFunction)(unsafe.Pointer(windows.NewCallbackCDecl(fn))),
		cName,
		0,
		nil,
	)
	vm.SetGlobal(name)
	fmt.Println("..")
}
*/

//export GoLunaGateway
func GoLunaGateway(L *C.lua_State) uintptr {
	defer func() {
		if r := recover(); r != nil {
			Print(0, "%v", r)
		}
	}()
	name := C.GoString(C.lua_tolstring(L, C.LUA_GLOBALSINDEX-1, nil))
	if fn, ok := registry[name]; ok {
		return uintptr(fn((*LuaState)(unsafe.Pointer(L))))
	}
	return 0
}

func (s *LuaState) UpValueIndex(i int) int {
	return C.LUA_GLOBALSINDEX - i
}

var registry = make(map[string]func(*LuaState) uintptr)

func (vm *LuaState) PushCClosure(fn func(*LuaState) uintptr, fn2 func(*LuaState, uintptr) uintptr, name string, env int) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	var continuation C.lua_Continuation = nil
	if fn2 != nil {
		continuation = (C.lua_Continuation)(unsafe.Pointer(windows.NewCallbackCDecl(fn2)))
	}
	C.lua_pushcclosurek(
		vm.cptr(),
		(C.lua_CFunction)(unsafe.Pointer(windows.NewCallbackCDecl(fn))),
		cName,
		C.int(env),
		continuation,
	)
}

func (vm *LuaState) RegisterFunction(name string, fn func(*LuaState) uintptr) {
	registry[name] = fn
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.luna_register_function(vm.cptr(), cName)
}

func GetLuaState(api *Luna, sc uintptr) *LuaState {
	defer func() {
		if r := recover(); r != nil {
			Print(3, fmt.Sprintf("%v", r))
		}
	}()
	var (
		ptrns = []func(a, b uint32) uint32{
			func(a, b uint32) uint32 { return b - a },
			func(a, b uint32) uint32 { return a ^ b },
			func(a, b uint32) uint32 { return a - b },
			func(a, b uint32) uint32 { return a + b },
		}
		handlr = func(address uintptr, encrypted [2]uint32, op func(a, b uint32) uint32) uintptr {
			low := op(uint32(address), encrypted[0])
			high := op(uint32(address), encrypted[1])
			ptr := (uint64(high) << 32) | uint64(low)
			if ptr != 0 && api.ValidPtr(uintptr(ptr)) && ReadProcessMemory[uintptr](api, uintptr(ptr)) != 0 {
				return uintptr(ptr)
			}
			return 0
		}
	)
	for offset := uintptr(0); offset <= 0x1000; offset += 0x8 {
		address := sc + offset
		encrypted := ReadProcessMemory[[2]uint32](api, address)
		if encrypted[0] == 0 && encrypted[1] == 0 {
			continue
		}
		for _, ptrn := range ptrns {
			if ptr := handlr(address, encrypted, ptrn); ptr != 0 {
				return (*LuaState)(unsafe.Pointer(ptr))
			}
		}
	}
	return nil
}

func NewLuaState(Luna *Sesh) *LuaState {
	dm := Luna.Game.RenderJob.DataModel()
	if dm != nil {
		sc := dm.Traverse("Script Context")
		if sc != nil {
			return GetLuaState(Luna.Luna, uintptr(sc.Self))
		}
	}
	return nil
}
