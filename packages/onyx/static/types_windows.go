package static

import (
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	Ps, Kernel, Ntdll, User32, Enum, Modules, ModuleBase, Snapshot = syscall.NewLazyDLL("psapi.dll"), syscall.NewLazyDLL("kernel32.dll"), windows.NewLazySystemDLL("ntdll.dll"), syscall.NewLazyDLL("user32.dll"), Ps.NewProc("EnumProcesses"), Ps.NewProc("EnumProcessModules"), Ps.NewProc("GetModuleBaseNameW"), Kernel.NewProc("CreateToolhelp32Snapshot")
	WriteMemory, ReadMemory, VirtualAlloc, VirtualProtect          = Kernel.NewProc("WriteProcessMemory"), Kernel.NewProc("ReadProcessMemory"), Kernel.NewProc("VirtualAllocEx"), Kernel.NewProc("VirtualProtectEx")
	CreateToolHelp32, Module32FirstW, Module32NextW                = Kernel.NewProc("CreateToolhelp32Snapshot"), Kernel.NewProc("Module32FirstW"), Kernel.NewProc("Module32NextW")
	GetModuleHandleA                                               = Kernel.NewProc("GetModuleHandleA")
	DuplicateHandle                                                = Kernel.NewProc("DuplicateHandle")
	CreateThreadpoolTimer                                          = Kernel.NewProc("CreateThreadpoolTimer")
	ZwSetIoCompletion                                              = Ntdll.NewProc("ZwSetIoCompletion")
	NtQuerySystemInformation                                       = Ntdll.NewProc("NtQuerySystemInformation")
	NtQueryObject                                                  = Ntdll.NewProc("NtQueryObject")
	WaitForSingleObject                                            = Kernel.NewProc("WaitForSingleObject")
	SuspendThread                                                  = Kernel.NewProc("SuspendThread")
	ResumeThread                                                   = Kernel.NewProc("ResumeThread")
	ZwUnmapViewOfSection                                           = Ntdll.NewProc("ZwUnmapViewOfSection")
	GetCurrentThreadId                                             = Kernel.NewProc("GetCurrentThreadId")
	SetForegroundWindow, GetForegroundWindow                       = User32.NewProc("SetForegroundWindow"), User32.NewProc("GetForegroundWindow")
	GetWindowThreadProcessId, AttachThreadInput                    = User32.NewProc("GetWindowThreadProcessId"), User32.NewProc("AttachThreadInput")
	BringWindowToTop, ShowWindow, IsIconic                         = User32.NewProc("BringWindowToTop"), User32.NewProc("ShowWindow"), User32.NewProc("IsIconic")
	EnumWindows, SendInput, PostMessage                            = User32.NewProc("EnumWindows"), User32.NewProc("SendInput"), User32.NewProc("PostMessageW")
	MapVirtualKey, SendMessage                                     = User32.NewProc("MapVirtualKeyW"), User32.NewProc("SendMessageA")
)
