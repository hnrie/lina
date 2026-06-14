@echo off

set CC=clang
set CXX=clang++

set "ROOT_DIR=%~dp0"
set "LUAU_DIR=%ROOT_DIR%packages\onyx\logic\luau"
set "BUILD_DIR=%LUAU_DIR%\cmake_win"
set "WIN_OUT=%LUAU_DIR%\windows"

rmdir /s /q "%BUILD_DIR%" 2>nul
rmdir /s /q "%WIN_OUT%" 2>nul
mkdir "%BUILD_DIR%"
mkdir "%WIN_OUT%"
cd /d "%BUILD_DIR%"

cmake .. -G "Ninja" ^
    -DCMAKE_C_COMPILER=clang ^
    -DCMAKE_CXX_COMPILER=clang++ ^
    -DCMAKE_CXX_STANDARD=20 ^
    -DLUAU_BUILD_TESTS=OFF ^
    -DLUAU_EXTERN_C=ON ^
    -DCMAKE_BUILD_TYPE=Release ^
    -DCMAKE_ARCHIVE_OUTPUT_DIRECTORY="%WIN_OUT%"
if %ERRORLEVEL% NEQ 0 goto :fail

cmake --build . --target Luau.VM Luau.Compiler Luau.Bytecode Luau.Ast Luau.Common
if %ERRORLEVEL% NEQ 0 goto :fail

cd /d "%ROOT_DIR%"
go clean -cache

set CGO_ENABLED=1
set CGO_LDFLAGS_ALLOW=.*
set "WIN_OUT_FWD=%WIN_OUT:\=/%"
set "CGO_LDFLAGS=-static -L%WIN_OUT_FWD% -lLuau.VM -lLuau.Compiler -lLuau.Bytecode -lLuau.Ast -lLuau.Common -ldbghelp -ld3d11 -ld3dcompiler -lgdi32 -ldwmapi -limm32"

go1.24.13 build -buildmode=c-shared --buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o "C:\Users\Luna\Desktop\luna\onyx\mem\injector\dll\luna.dll" .
if %ERRORLEVEL% NEQ 0 goto :fail

pause
exit /b 0

:fail
pause
exit /b 1