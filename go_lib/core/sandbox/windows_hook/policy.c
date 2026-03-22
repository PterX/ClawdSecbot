#include "policy.h"
#include <cJSON.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>
#include <shlwapi.h>

static void load_string_array(cJSON *arr, char dest[][MAX_PATH_LEN], int *count, int max) {
    *count = 0;
    if (!cJSON_IsArray(arr)) return;
    cJSON *item;
    cJSON_ArrayForEach(item, arr) {
        if (*count >= max) break;
        if (cJSON_IsString(item) && item->valuestring) {
            strncpy(dest[*count], item->valuestring, MAX_PATH_LEN - 1);
            dest[*count][MAX_PATH_LEN - 1] = '\0';
            (*count)++;
        }
    }
}

static void load_ip_array(cJSON *arr, char dest[][64], int *count, int max) {
    *count = 0;
    if (!cJSON_IsArray(arr)) return;
    cJSON *item;
    cJSON_ArrayForEach(item, arr) {
        if (*count >= max) break;
        if (cJSON_IsString(item) && item->valuestring) {
            strncpy(dest[*count], item->valuestring, 63);
            dest[*count][63] = '\0';
            (*count)++;
        }
    }
}

int policy_load(const char *path, SandboxPolicy *out) {
    memset(out, 0, sizeof(*out));
    out->inject_children = true;

    FILE *f = fopen(path, "rb");
    if (!f) return -1;

    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    fseek(f, 0, SEEK_SET);

    if (len <= 0 || len > 1024 * 1024) {
        fclose(f);
        return -2;
    }

    char *buf = (char *)malloc(len + 1);
    if (!buf) { fclose(f); return -3; }
    fread(buf, 1, len, f);
    buf[len] = '\0';
    fclose(f);

    cJSON *root = cJSON_Parse(buf);
    free(buf);
    if (!root) return -4;

    cJSON *val;

    val = cJSON_GetObjectItem(root, "file_policy_type");
    if (val && cJSON_IsString(val)) {
        out->file_policy = (strcmp(val->valuestring, "whitelist") == 0)
                               ? POLICY_WHITELIST : POLICY_BLACKLIST;
    }
    load_string_array(cJSON_GetObjectItem(root, "blocked_paths"),
                      out->blocked_paths, &out->blocked_paths_count, MAX_POLICY_ENTRIES);
    load_string_array(cJSON_GetObjectItem(root, "allowed_paths"),
                      out->allowed_paths, &out->allowed_paths_count, MAX_POLICY_ENTRIES);

    val = cJSON_GetObjectItem(root, "command_policy_type");
    if (val && cJSON_IsString(val)) {
        out->command_policy = (strcmp(val->valuestring, "whitelist") == 0)
                                  ? POLICY_WHITELIST : POLICY_BLACKLIST;
    }
    load_string_array(cJSON_GetObjectItem(root, "blocked_commands"),
                      out->blocked_commands, &out->blocked_commands_count, MAX_POLICY_ENTRIES);
    load_string_array(cJSON_GetObjectItem(root, "allowed_commands"),
                      out->allowed_commands, &out->allowed_commands_count, MAX_POLICY_ENTRIES);

    val = cJSON_GetObjectItem(root, "network_policy_type");
    if (val && cJSON_IsString(val)) {
        out->network_policy = (strcmp(val->valuestring, "whitelist") == 0)
                                  ? POLICY_WHITELIST : POLICY_BLACKLIST;
    }
    load_ip_array(cJSON_GetObjectItem(root, "blocked_ips"),
                  out->blocked_ips, &out->blocked_ips_count, MAX_POLICY_ENTRIES);
    load_ip_array(cJSON_GetObjectItem(root, "allowed_ips"),
                  out->allowed_ips, &out->allowed_ips_count, MAX_POLICY_ENTRIES);

    val = cJSON_GetObjectItem(root, "strict_mode");
    if (val) out->strict_mode = cJSON_IsTrue(val);

    val = cJSON_GetObjectItem(root, "log_only");
    if (val) out->log_only = cJSON_IsTrue(val);

    val = cJSON_GetObjectItem(root, "inject_children");
    if (val) out->inject_children = cJSON_IsTrue(val);

    cJSON_Delete(root);
    return 0;
}

/* Wide-char to narrow for path matching */
static void wchar_to_utf8(const wchar_t *src, char *dst, int maxlen) {
    WideCharToMultiByte(CP_UTF8, 0, src, -1, dst, maxlen, NULL, NULL);
}

static void normalize_windows_path(const char *src, char *dst, size_t dstlen) {
    if (!src || !dst || dstlen == 0) return;
    strncpy(dst, src, dstlen - 1);
    dst[dstlen - 1] = '\0';

    for (char *p = dst; *p; p++) {
        if (*p == '/') *p = '\\';
    }

    if (_strnicmp(dst, "\\\\?\\", 4) == 0 || _strnicmp(dst, "\\\\.\\", 4) == 0) {
        memmove(dst, dst + 4, strlen(dst + 4) + 1);
    } else if (_strnicmp(dst, "\\??\\", 4) == 0) {
        memmove(dst, dst + 4, strlen(dst + 4) + 1);
    }

    size_t n = strlen(dst);
    while (n > 3 && (dst[n - 1] == '\\' || dst[n - 1] == '/')) {
        dst[n - 1] = '\0';
        n--;
    }
}

static int path_matches(const char *pattern, const char *path) {
    char norm_pattern[MAX_PATH_LEN];
    char norm_path[MAX_PATH_LEN];
    normalize_windows_path(pattern, norm_pattern, sizeof(norm_pattern));
    normalize_windows_path(path, norm_path, sizeof(norm_path));

    size_t plen = strlen(norm_pattern);
    if (plen == 0) return 0;
    if (_strnicmp(norm_path, norm_pattern, plen) != 0) return 0;

    /* Prefix boundary check: exact match OR next char is separator. */
    if (strlen(norm_path) == plen) return 1;
    char next = norm_path[plen];
    return next == '\\' || next == '/';
}

PolicyAction policy_check_file(const SandboxPolicy *p, const wchar_t *wpath) {
    if (!wpath) return ACTION_ALLOW;
    char path[MAX_PATH_LEN];
    wchar_to_utf8(wpath, path, MAX_PATH_LEN);

    if (p->file_policy == POLICY_WHITELIST) {
        for (int i = 0; i < p->allowed_paths_count; i++) {
            if (path_matches(p->allowed_paths[i], path)) return ACTION_ALLOW;
        }
        return p->log_only ? ACTION_LOG : ACTION_DENY;
    }

    /* Blacklist */
    for (int i = 0; i < p->blocked_paths_count; i++) {
        if (path_matches(p->blocked_paths[i], path)) {
            return p->log_only ? ACTION_LOG : ACTION_DENY;
        }
    }
    return ACTION_ALLOW;
}

PolicyAction policy_check_command(const SandboxPolicy *p, const wchar_t *wcmd) {
    if (!wcmd) return ACTION_ALLOW;
    char cmd[MAX_PATH_LEN];
    wchar_to_utf8(wcmd, cmd, MAX_PATH_LEN);
    _strlwr(cmd);

    if (p->command_policy == POLICY_WHITELIST) {
        for (int i = 0; i < p->allowed_commands_count; i++) {
            char lower[MAX_PATH_LEN];
            strncpy(lower, p->allowed_commands[i], MAX_PATH_LEN - 1);
            lower[MAX_PATH_LEN - 1] = '\0';
            _strlwr(lower);
            if (strstr(cmd, lower)) return ACTION_ALLOW;
        }
        return p->log_only ? ACTION_LOG : ACTION_DENY;
    }

    for (int i = 0; i < p->blocked_commands_count; i++) {
        char lower[MAX_PATH_LEN];
        strncpy(lower, p->blocked_commands[i], MAX_PATH_LEN - 1);
        lower[MAX_PATH_LEN - 1] = '\0';
        _strlwr(lower);
        if (strstr(cmd, lower)) {
            return p->log_only ? ACTION_LOG : ACTION_DENY;
        }
    }
    return ACTION_ALLOW;
}

PolicyAction policy_check_network(const SandboxPolicy *p, const char *ip, int port) {
    if (!ip) return ACTION_ALLOW;

    if (p->network_policy == POLICY_WHITELIST) {
        for (int i = 0; i < p->allowed_ips_count; i++) {
            if (strcmp(p->allowed_ips[i], ip) == 0) return ACTION_ALLOW;
        }
        return p->log_only ? ACTION_LOG : ACTION_DENY;
    }

    for (int i = 0; i < p->blocked_ips_count; i++) {
        if (strcmp(p->blocked_ips[i], ip) == 0) {
            return p->log_only ? ACTION_LOG : ACTION_DENY;
        }
    }
    return ACTION_ALLOW;
}
