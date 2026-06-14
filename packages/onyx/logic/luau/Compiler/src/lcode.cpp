// This file is part of the Luau programming language and is licensed under MIT License; see LICENSE.txt for details
#include "luacode.h"

#include "Luau/Compiler.h"

#include <string.h>

#include "../../luau.h"
#include <Luau/BytecodeUtils.h>
#include <Luau/BytecodeBuilder.h>

struct : Luau::BytecodeEncoder
{
    void encode(uint32_t *data, size_t count)
    {
        const auto lookupTable = reinterpret_cast<const uint8_t*>(luna::roblox::luau::opcode);
        for (auto i = 0u; i < count;)
        {
            uint8_t op = LUAU_INSN_OP(data[i]);
            uint8_t newOp = op * 227;
            newOp = lookupTable[newOp];
            data[i] = (newOp) | (data[i] & ~0xff);
            i += Luau::getOpLength(static_cast<LuauOpcode>(op));
        }
    }
} enc;

char* luau_compile(const char* source, size_t size, lua_CompileOptions* options, size_t* outsize)
{
    LUAU_ASSERT(outsize);

    Luau::CompileOptions opts;

    if (options)
    {
        static_assert(sizeof(lua_CompileOptions) == sizeof(Luau::CompileOptions), "C and C++ interface must match");
        memcpy(static_cast<void*>(&opts), options, sizeof(opts));
    }

    std::string result = compile(std::string(source, size), opts, {}, &enc);

    char* copy = static_cast<char*>(malloc(result.size()));
    if (!copy)
        return nullptr;

    memcpy(copy, result.data(), result.size());
    *outsize = result.size();
    return copy;
}

void luau_set_compile_constant_nil(lua_CompileConstant* constant)
{
    Luau::setCompileConstantNil(constant);
}

void luau_set_compile_constant_boolean(lua_CompileConstant* constant, int b)
{
    Luau::setCompileConstantBoolean(constant, b != 0);
}

void luau_set_compile_constant_number(lua_CompileConstant* constant, double n)
{
    Luau::setCompileConstantNumber(constant, n);
}

void luau_set_compile_constant_integer64(lua_CompileConstant* constant, int64_t l)
{
    Luau::setCompileConstantInteger64(constant, l);
}

void luau_set_compile_constant_vector(lua_CompileConstant* constant, float x, float y, float z, float w)
{
    Luau::setCompileConstantVector(constant, x, y, z, w);
}

void luau_set_compile_constant_string(lua_CompileConstant* constant, const char* s, size_t l)
{
    Luau::setCompileConstantString(constant, s, l);
}
