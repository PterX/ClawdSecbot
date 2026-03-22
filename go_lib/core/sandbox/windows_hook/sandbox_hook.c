#include "hooks.h"
#include "policy.h"
#include <stdio.h>

static SandboxPolicy g_sandbox_policy;
static BOOL g_initialized = FALSE;

BOOL APIENTRY DllMain(HMODULE hModule, DWORD reason, LPVOID lpReserved) {
    switch (reason) {
    case DLL_PROCESS_ATTACH: {
        DisableThreadLibraryCalls(hModule);

        char policyPath[MAX_PATH] = {0};
        DWORD len = GetEnvironmentVariableA("SANDBOX_POLICY_FILE", policyPath, MAX_PATH);
        if (len == 0 || len >= MAX_PATH) {
            OutputDebugStringA("[sandbox_hook] SANDBOX_POLICY_FILE not set, hook inactive\n");
            break;
        }

        if (policy_load(policyPath, &g_sandbox_policy) != 0) {
            OutputDebugStringA("[sandbox_hook] Failed to load policy file\n");
            break;
        }

        if (hooks_install(&g_sandbox_policy) != 0) {
            OutputDebugStringA("[sandbox_hook] Failed to install hooks\n");
            break;
        }

        g_initialized = TRUE;
        OutputDebugStringA("[sandbox_hook] Initialized successfully\n");
        break;
    }
    case DLL_PROCESS_DETACH:
        if (g_initialized) {
            hooks_remove();
            g_initialized = FALSE;
        }
        break;
    }
    return TRUE;
}
