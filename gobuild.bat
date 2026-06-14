@echo off

del /f /q "C:\Users\Luna\Desktop\luna\onyx\mem\injector\dll\luna.dll" 2>nul
del /f /q "C:\Users\Luna\Desktop\luna\onyx\mem\injector\dll\luna.h" 2>nul

set "ROOT_DIR=%~dp0"
set "LUAU_DIR=%ROOT_DIR%packages\onyx\logic\luau"
set "BUILD_DIR=%LUAU_DIR%\cmake_win"
set "WIN_OUT=%LUAU_DIR%\windows"
set CGO_ENABLED=1
set CGO_LDFLAGS_ALLOW=.*
set GOTOOLCHAIN=local
set "PATH=C:\Users\Luna\sdk\go1.24.13\bin;%USERPROFILE%\go\bin;%PATH%"
set "WIN_OUT_FWD=%WIN_OUT:\=/%"
set "CGO_LDFLAGS=-static -L%WIN_OUT_FWD% -lLuau.VM -lLuau.Compiler -lLuau.Bytecode -lLuau.Ast -lLuau.Common -ldbghelp -ld3d11 -ld3dcompiler -lgdi32 -ldwmapi -limm32"

garble -literals -tiny -seed=random build -buildmode=c-shared -buildvcs=false -trimpath -ldflags="-s -w -buildid= -extldflags=-Wl,--gc-sections" -o "C:\Users\Luna\Desktop\luna\onyx\mem\injector\dll\luna.dll" .

pause