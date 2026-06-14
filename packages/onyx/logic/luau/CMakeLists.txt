# This file is part of the Luau programming language and is licensed under MIT License; see LICENSE.txt for details
if(EXT_PLATFORM_STRING)
    include(EXTLuau.cmake)
    return()
endif()

cmake_minimum_required(VERSION 3.10)

# Turned off options that are not needed
option(LUAU_BUILD_CLI "Build CLI" OFF)
option(LUAU_BUILD_TESTS "Build tests" OFF)
option(LUAU_BUILD_WEB "Build Web module" OFF)
option(LUAU_WERROR "Warnings as errors" OFF)
option(LUAU_STATIC_CRT "Link with the static CRT (/MT)" OFF)
option(LUAU_EXTERN_C "Use extern C for all APIs" OFF)

cmake_policy(SET CMP0054 NEW)
cmake_policy(SET CMP0091 NEW)

if(LUAU_STATIC_CRT)
    cmake_minimum_required(VERSION 3.15)
    set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
endif()

project(Luau LANGUAGES CXX C)

# Add only the requested libraries
add_library(Luau.Common STATIC)
add_library(Luau.Ast STATIC)
add_library(Luau.Compiler STATIC)
add_library(Luau.VM STATIC)
add_library(Luau.Bytecode STATIC)

# Proxy target to make it possible to depend on private VM headers
add_library(Luau.VM.Internals INTERFACE)

include(Sources.cmake)

# Luau.Common
target_include_directories(Luau.Common PUBLIC Common/include)

# Luau.Bytecode
target_include_directories(Luau.Bytecode PUBLIC Bytecode/include)
target_link_libraries(Luau.Bytecode PUBLIC Luau.Common)

# Luau.Ast
target_compile_features(Luau.Ast PUBLIC cxx_std_17)
target_include_directories(Luau.Ast PUBLIC Ast/include)
target_link_libraries(Luau.Ast PUBLIC Luau.Common)

# Luau.Compiler
target_compile_features(Luau.Compiler PUBLIC cxx_std_17)
target_include_directories(Luau.Compiler PUBLIC Compiler/include)
target_link_libraries(Luau.Compiler PUBLIC Luau.Ast Luau.Bytecode) # Added Luau.Bytecode here

# Luau.VM
target_compile_features(Luau.VM PRIVATE cxx_std_11)
target_include_directories(Luau.VM PUBLIC VM/include)
target_link_libraries(Luau.VM PUBLIC Luau.Common)

# Luau.VM.Internals
target_include_directories(Luau.VM.Internals INTERFACE VM/src)

set(LUAU_OPTIONS)

if(MSVC)
    list(APPEND LUAU_OPTIONS /D_CRT_SECURE_NO_WARNINGS) # We need to use the portable CRT functions.
    list(APPEND LUAU_OPTIONS "/we4018") # Signed/unsigned mismatch
    list(APPEND LUAU_OPTIONS "/we4388") # Also signed/unsigned mismatch
else()
    list(APPEND LUAU_OPTIONS -Wall) # All warnings
    list(APPEND LUAU_OPTIONS -Wimplicit-fallthrough)
    list(APPEND LUAU_OPTIONS -Wsign-compare) # This looks to be included in -Wall for GCC but not clang
endif()

if (CMAKE_CXX_COMPILER_ID STREQUAL "MSVC")
    list(APPEND LUAU_OPTIONS /MP) # Distribute single project compilation across multiple cores
endif()

if (CMAKE_CXX_COMPILER_ID STREQUAL "GNU")
    # Some gcc versions treat var in `if (type var = val)` as unused
    # Some gcc versions treat variables used in constexpr if blocks as unused
    list(APPEND LUAU_OPTIONS -Wno-unused)
    # some gcc versions warn maybe uninitialized on optional<string> members on structs
    list(APPEND LUAU_OPTIONS -Wno-maybe-uninitialized)
endif()

if(LUAU_WERROR)
    if(MSVC)
        list(APPEND LUAU_OPTIONS /WX) # Warnings are errors
    else()
        list(APPEND LUAU_OPTIONS -Werror) # Warnings are errors
    endif()
endif()

target_compile_options(Luau.Ast PRIVATE ${LUAU_OPTIONS})
target_compile_options(Luau.VM PRIVATE ${LUAU_OPTIONS})

if(LUAU_EXTERN_C)
    target_compile_definitions(Luau.VM PUBLIC LUA_API=extern\"C\")
    target_compile_definitions(Luau.Compiler PUBLIC LUACODE_API=extern\"C\")
endif()

if (CMAKE_CXX_COMPILER_ID STREQUAL "MSVC" AND MSVC_VERSION GREATER_EQUAL 1924)
    # disable partial redundancy elimination which regresses interpreter codegen substantially in VS2022:
    # https://developercommunity.visualstudio.com/t/performance-regression-on-a-complex-interpreter-lo/1631863
    set_source_files_properties(VM/src/lvmexecute.cpp PROPERTIES COMPILE_FLAGS /d2ssa-pre-)
endif()

if (NOT MSVC)
    # disable support for math_errno which allows compilers to lower sqrt() into a single CPU instruction
    target_compile_options(Luau.VM PRIVATE -fno-math-errno)
endif()

# embed .natvis inside the library debug information
if(MSVC)
    target_link_options(Luau.Ast INTERFACE /NATVIS:${CMAKE_CURRENT_SOURCE_DIR}/tools/natvis/Ast.natvis)
    target_link_options(Luau.VM INTERFACE /NATVIS:${CMAKE_CURRENT_SOURCE_DIR}/tools/natvis/VM.natvis)
endif()

# make .natvis visible inside the solution
if(MSVC_IDE)
    target_sources(Luau.Ast PRIVATE tools/natvis/Ast.natvis)
    target_sources(Luau.VM PRIVATE tools/natvis/VM.natvis)
endif()

# On Windows and Android threads are provided, on Linux/Mac/iOS we use pthreads
add_library(osthreads INTERFACE)
if(CMAKE_SYSTEM_NAME MATCHES "Linux|Darwin|iOS")
    target_link_libraries(osthreads INTERFACE "-lpthread")
endif ()