#pragma once
#ifdef __cplusplus

#include <windows.h>
#include <string>
#include <cstdint>
#include <mutex>
#include <fstream>
#include "vmvalues.hpp"

#define CLOSURE_CONT_ENC VMValue3
#define CLOSURE_DEBUGNAME_ENC VMValue1
#define LSTATE_STACKSIZE_ENC VMValue3
#define PROTO_ABSLINEINFO_ENC VMValue3
#define PROTO_DEBUGINSN_ENC VMValue4
#define PROTO_DEBUGNAME_ENC VMValue1
#define PROTO_LINEINFO_ENC VMValue2
#define PROTO_LOCVARS_ENC VMValue3
#define PROTO_SOURCE_ENC VMValue1
#define PROTO_TYPEINFO_ENC VMValue4
#define PROTO_UPVALUES_ENC VMValue1
#define PROTO_USERDATA_ENC VMValue4
#define TSTRING_HASH_ENC VMValue3
#define UDATA_META_ENC VMValue3

inline uintptr_t maxCapabilities = (0x200000000000003FLL | 0x3FFFFFFFFFFF00LL) | (1ULL << 48ULL);

extern "C"
{
#endif
    void luna_log_info(const char *message);
    void luna_log_error(int code, const char *message);
    void luna_init_logger(const char *path);
    typedef struct lua_State lua_State;

#ifdef __cplusplus
}

namespace luna
{
    namespace roblox
    {
        template <typename T>
        inline T rebase_address(std::uintptr_t address)
        {
            size_t base = (size_t)GetModuleHandleA(NULL);
            return address != 0 ? (T)(address + base) : (T)NULL;
        }
        namespace luau
        {
            using luaD_throw_t = void(__fastcall *)(lua_State *L, int status);
            using luau_execute_t = void(__fastcall *)(lua_State *lua_state);
            using push_instance_t = void(__fastcall *)(lua_State *, int);
            using luaC_step_t = size_t(__fastcall*)(lua_State*, bool);

            static std::uintptr_t opcode = rebase_address<std::uintptr_t>(0x6056C90);
            static std::uintptr_t luaO_nilobject = rebase_address<std::uintptr_t>(0x67AE440);
            static std::uintptr_t luaH_dummynode = rebase_address<std::uintptr_t>(0x67AE2E8);
            static luaC_step_t luaC_step = rebase_address<luaC_step_t>(0x453D620);
            static luau_execute_t luau_execute = rebase_address<luau_execute_t>(0x454ACD0);
            static luaD_throw_t luaD_throw = rebase_address<luaD_throw_t>(0x4537A00);
        }
    }
}
#endif