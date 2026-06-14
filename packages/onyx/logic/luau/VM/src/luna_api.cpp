#include "lobject.h"
#include "lstate.h"
#include "lualib.h"
#include "lua.h"
#include <stdint.h>
#include <stdbool.h>
#include <string>
#include <unordered_set>
#include <algorithm>
#include <stdexcept>
#include <mutex>
#include "ldo.h"

typedef struct SWeakObjectRef2_t
{
    uint8_t pad_0[0x28];
    uintptr_t L;
    int32_t ThreadId;
    int32_t ObjectId;
} SWeakObjectRef2_t;

typedef struct lua_debug_result_t
{
    int32_t result;
    int32_t unk_1[4];
} lua_debug_result_t;

typedef int (*ScriptContextResume_t)(
    uintptr_t scriptContext,
    lua_debug_result_t *debug,
    SWeakObjectRef2_t **weak,
    int args,
    bool isError,
    const char *errMsg);

typedef int (*LunaCallback)(lua_State *L, void *ud);
static uintptr_t ScriptContextCallback = 0;
static bool pending_error = false;
static std::mutex error_mutex;

extern "C" void setup_sc_callback(uintptr_t sc)
{
    ScriptContextCallback = sc;
}

extern "C" intptr_t GoLunaGateway(lua_State *L);
extern "C" int go_lua_callback(lua_State *L, void *ud);
extern "C" int GoStepHookPayload();

extern "C" int c_luna_gateway(lua_State *L)
{
    intptr_t results = GoLunaGateway(L);
    bool should_error = false;
    {
        std::lock_guard<std::mutex> lock(error_mutex);
        if (pending_error)
        {
            pending_error = false;
            should_error = true;
        }
    }
    if (should_error)
        lua_error(L);

    if (results == -0x10)
    {
        int top = lua_gettop(L);
        if (top >= 2 && lua_isnil(L, -2) && lua_isstring(L, -1))
        {
            const char *err = lua_tostring(L, -1);
            luaL_error(L, "%s", err);
            return 0;
        }
        return lua_yield(L, 0);
    }

    return (int)results;
}

extern "C" void luna_register_function(lua_State *L, const char *name)
{
    lua_pushstring(L, name);
    lua_pushcclosurek(L, c_luna_gateway, name, 1, NULL);
    lua_setfield(L, LUA_GLOBALSINDEX, name);
}

extern "C" void trigger_luna_error_bridge()
{
    std::lock_guard<std::mutex> lock(error_mutex);
    pending_error = true;
}

typedef uintptr_t (*JobStepFunc)(uintptr_t a, uintptr_t b);
static JobStepFunc OriginalStep = nullptr;

static uintptr_t c_step_hook(uintptr_t a, uintptr_t b)
{
    GoStepHookPayload();
    if (OriginalStep)
        return OriginalStep(a, b);
    return 0;
}

extern "C" void set_original_step(uintptr_t ptr)
{
    OriginalStep = (JobStepFunc)ptr;
}

extern "C" uintptr_t get_hook_ptr()
{
    return (uintptr_t)(&c_step_hook);
}

extern "C" int GoIndex(lua_State *l);
extern "C" int GoNamecall(lua_State *l);

static lua_CFunction index_fn = nullptr;
static lua_CFunction namecall_fn = nullptr;

static int Index(lua_State *l)
{
    int i = GoIndex(l);
    if (i == 0x10)
    {
        luaL_error(l, "use of vuln functions are not allowed in lunas env.");
        return 0;
    }
    if (i == -0x10)
    {
        int top = lua_gettop(l);
        if (top >= 2 && lua_isnil(l, -2) && lua_isstring(l, -1))
        {
            const char *err = lua_tostring(l, -1);
            luaL_error(l, "%s", err);
            return 0;
        }
        return lua_yield(l, 0);
    }
    if (i > 0)
        return i;
    return index_fn(l);
}

static int Namecall(lua_State *l)
{
    int i = GoNamecall(l);
    if (i == 0x10)
    {
        luaL_error(l, "use of vuln functions are not allowed in lunas env.");
        return 0;
    }
    if (i == -0x10)
    {
        int top = lua_gettop(l);
        if (top >= 2 && lua_isnil(l, -2) && lua_isstring(l, -1))
        {
            const char *err = lua_tostring(l, -1);
            luaL_error(l, "%s", err);
            return 0;
        }

        return lua_yield(l, 0);
    }
    if (i > 0)
        return i;
    return namecall_fn(l);
}

LUA_API int Hook_Calls(lua_State *l)
{
    lua_getglobal(l, "game");
    luaL_getmetafield(l, -1, "__index");
    Closure *index_closure = (Closure *)(lua_topointer(l, -1));
    if (index_closure->c.f != Index)
    {
        index_fn = index_closure->c.f;
        index_closure->c.f = Index;
    }
    lua_pop(l, 1);

    luaL_getmetafield(l, -1, "__namecall");
    Closure *namecall_closure = (Closure *)(lua_topointer(l, -1));
    if (namecall_closure->c.f != Namecall)
    {
        namecall_fn = namecall_closure->c.f;
        namecall_closure->c.f = Namecall;
    }
    lua_pop(l, 2);
    return 0;
}

extern "C"
{
    static void cgo_newcclosure_handler_run(lua_State *L, void *ud)
    {
        luaD_call(L, (StkId)(ud), LUA_MULTRET);
    }
    static int cgo_newcclosure_handler(lua_State *L)
    {
        lua_pushvalue(L, lua_upvalueindex(1));
        lua_insert(L, 1);
        StkId func = L->base;
        L->ci->flags |= LUA_CALLINFO_HANDLE;
        L->baseCcalls++;
        int status = luaD_pcall(L, cgo_newcclosure_handler_run, func, savestack(L, func), 0);
        L->baseCcalls--;
        if (status == LUA_ERRRUN)
        {
            lua_error(L);
            return 0;
        }
        expandstacklimit(L, L->top);
        if (status == 0 && (L->status == LUA_YIELD || L->status == LUA_BREAK))
        {
            return -1;
        }
        return lua_gettop(L);
    }
    static int cgo_newcclosure_cont(lua_State *L, int status)
    {
        if (status != LUA_OK)
        {
            lua_error(L);
        }
        return lua_gettop(L);
    }
    void bridge_push_newcclosure(void *state)
    {
        lua_State *L = (lua_State *)state;
        lua_pushcclosurek(L, cgo_newcclosure_handler, "newcclosure", 1, cgo_newcclosure_cont);
    }
}