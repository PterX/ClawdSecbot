//go:build windows

package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"go_lib/core/logging"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procVirtualAllocEx    = kernel32.NewProc("VirtualAllocEx")
	procWriteProcessMemory = kernel32.NewProc("WriteProcessMemory")
	procCreateRemoteThread = kernel32.NewProc("CreateRemoteThread")
	procOpenProcess       = kernel32.NewProc("OpenProcess")
	procCloseHandle       = kernel32.NewProc("CloseHandle")
	procGetModuleHandleW  = kernel32.NewProc("GetModuleHandleW")
	procGetProcAddress    = kernel32.NewProc("GetProcAddress")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procThread32First     = kernel32.NewProc("Thread32First")
	procThread32Next      = kernel32.NewProc("Thread32Next")
	procOpenThread        = kernel32.NewProc("OpenThread")
	procResumeThread      = kernel32.NewProc("ResumeThread")
)

const (
	processAllAccess      = 0x001F0FFF
	memCommit             = 0x00001000
	memReserve            = 0x00002000
	memRelease            = 0x00008000
	pageReadwrite         = 0x04
	createSuspended       = 0x00000004
	th32csSnapthread      = 0x00000004
	threadSuspendResume   = 0x0002
	threadQueryInfo       = 0x0040
	invalidHandleValue    = ^uintptr(0)
)

// threadEntry32 matches the Windows THREADENTRY32 structure
type threadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

// hookDLLSearchPaths lists where to look for the sandbox hook DLL
var hookDLLSearchPaths = []string{
	"sandbox_hook.dll",
	filepath.Join("plugins", "sandbox_hook.dll"),
}

func firstExistingPath(candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			abs, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return abs
			}
			return candidate
		}
	}
	return ""
}

// findHookDLL searches for sandbox_hook.dll in standard locations and the policy directory
func findHookDLL(policyDir string) string {
	// Check explicit environment override first.
	if p := os.Getenv("SANDBOX_HOOK_DLL"); p != "" {
		if found := firstExistingPath(p); found != "" {
			return found
		}
	}

	// Check relative to executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		if found := firstExistingPath(
			filepath.Join(exeDir, "sandbox_hook.dll"),
			filepath.Join(exeDir, "plugins", "sandbox_hook.dll"),
		); found != "" {
			return found
		}

		// Walk up parent directories and probe "<parent>/plugins/sandbox_hook.dll".
		current := exeDir
		for i := 0; i < 8; i++ {
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			if found := firstExistingPath(
				filepath.Join(parent, "sandbox_hook.dll"),
				filepath.Join(parent, "plugins", "sandbox_hook.dll"),
			); found != "" {
				return found
			}
			current = parent
		}
	}

	// Check policy directory
	if policyDir != "" {
		if found := firstExistingPath(filepath.Join(policyDir, "sandbox_hook.dll")); found != "" {
			return found
		}
	}

	// Check standard paths
	for _, p := range hookDLLSearchPaths {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	return ""
}

// injectDLL injects a DLL into a target process using CreateRemoteThread + LoadLibraryW
func injectDLL(pid int, dllPath string) error {
	absDLL, err := filepath.Abs(dllPath)
	if err != nil {
		return fmt.Errorf("failed to resolve DLL path: %w", err)
	}
	if _, err := os.Stat(absDLL); err != nil {
		return fmt.Errorf("hook DLL not found: %s", absDLL)
	}

	hProcess, _, err := procOpenProcess.Call(
		uintptr(processAllAccess),
		0,
		uintptr(pid),
	)
	if hProcess == 0 {
		return fmt.Errorf("OpenProcess failed for PID %d: %v", pid, err)
	}
	defer procCloseHandle.Call(hProcess)

	// Get LoadLibraryW address from kernel32.dll
	k32Name, _ := syscall.UTF16PtrFromString("kernel32.dll")
	hKernel32, _, _ := procGetModuleHandleW.Call(uintptr(unsafe.Pointer(k32Name)))
	if hKernel32 == 0 {
		return fmt.Errorf("GetModuleHandle(kernel32.dll) failed")
	}

	loadLibName, _ := syscall.BytePtrFromString("LoadLibraryW")
	loadLibAddr, _, _ := procGetProcAddress.Call(hKernel32, uintptr(unsafe.Pointer(loadLibName)))
	if loadLibAddr == 0 {
		return fmt.Errorf("GetProcAddress(LoadLibraryW) failed")
	}

	// Convert DLL path to UTF-16
	dllPathUTF16, err := syscall.UTF16FromString(absDLL)
	if err != nil {
		return fmt.Errorf("UTF16 conversion failed: %w", err)
	}
	dllPathSize := uintptr(len(dllPathUTF16) * 2)

	// Allocate memory in target process
	remoteMem, _, err := procVirtualAllocEx.Call(
		hProcess,
		0,
		dllPathSize,
		uintptr(memCommit|memReserve),
		uintptr(pageReadwrite),
	)
	if remoteMem == 0 {
		return fmt.Errorf("VirtualAllocEx failed: %v", err)
	}

	// Write DLL path to target process memory
	var bytesWritten uintptr
	ret, _, err := procWriteProcessMemory.Call(
		hProcess,
		remoteMem,
		uintptr(unsafe.Pointer(&dllPathUTF16[0])),
		dllPathSize,
		uintptr(unsafe.Pointer(&bytesWritten)),
	)
	if ret == 0 {
		return fmt.Errorf("WriteProcessMemory failed: %v", err)
	}

	// Create remote thread to call LoadLibraryW with the DLL path
	var threadID uint32
	hThread, _, err := procCreateRemoteThread.Call(
		hProcess,
		0,
		0,
		loadLibAddr,
		remoteMem,
		0,
		uintptr(unsafe.Pointer(&threadID)),
	)
	if hThread == 0 {
		return fmt.Errorf("CreateRemoteThread failed: %v", err)
	}
	defer procCloseHandle.Call(hThread)

	// Wait for the remote thread to finish loading the DLL (max 10 seconds)
	syscall.WaitForSingleObject(syscall.Handle(hThread), 10000)

	logging.Info("[Sandbox] Injected hook DLL into PID %d (thread %d)", pid, threadID)
	return nil
}

// resumeProcessThreads resumes all suspended threads of a process
func resumeProcessThreads(pid int) error {
	hSnap, _, err := procCreateToolhelp32Snapshot.Call(
		uintptr(th32csSnapthread),
		0,
	)
	if hSnap == invalidHandleValue {
		return fmt.Errorf("CreateToolhelp32Snapshot failed: %v", err)
	}
	defer procCloseHandle.Call(hSnap)

	var te threadEntry32
	te.Size = uint32(unsafe.Sizeof(te))

	ret, _, err := procThread32First.Call(hSnap, uintptr(unsafe.Pointer(&te)))
	if ret == 0 {
		return fmt.Errorf("Thread32First failed: %v", err)
	}

	resumedCount := 0
	for {
		if te.OwnerProcessID == uint32(pid) {
			hThread, _, _ := procOpenThread.Call(
				uintptr(threadSuspendResume|threadQueryInfo),
				0,
				uintptr(te.ThreadID),
			)
			if hThread != 0 {
				procResumeThread.Call(hThread)
				procCloseHandle.Call(hThread)
				resumedCount++
			}
		}

		te.Size = uint32(unsafe.Sizeof(te))
		ret, _, _ = procThread32Next.Call(hSnap, uintptr(unsafe.Pointer(&te)))
		if ret == 0 {
			break
		}
	}

	logging.Info("[Sandbox] Resumed %d threads for PID %d", resumedCount, pid)
	return nil
}
