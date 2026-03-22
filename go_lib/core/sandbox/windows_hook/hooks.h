#ifndef SANDBOX_HOOKS_H
#define SANDBOX_HOOKS_H

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include "policy.h"

/* Initialize all API hooks. Returns 0 on success. */
int hooks_install(const SandboxPolicy *policy);

/* Remove all hooks and clean up. */
void hooks_remove(void);

/* Get the path to this DLL (for child process injection). */
const wchar_t *hooks_get_dll_path(void);

#endif /* SANDBOX_HOOKS_H */
