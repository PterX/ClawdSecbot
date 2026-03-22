#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <stdarg.h>
#include <dlfcn.h>
#include <errno.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <netdb.h>

// 沙箱策略结构 - 支持黑名单/白名单模式、域名拦截
typedef struct {
    int log_only;

    int file_policy_whitelist;
    char **blocked_paths;
    size_t blocked_paths_count;
    char **allowed_paths;
    size_t allowed_paths_count;

    int network_policy_whitelist;
    char **blocked_ips;
    size_t blocked_ips_count;
    char **allowed_ips;
    size_t allowed_ips_count;
    char **blocked_domains;
    size_t blocked_domains_count;
    char **allowed_domains;
    size_t allowed_domains_count;

    int command_policy_whitelist;
    char **blocked_cmds;
    size_t blocked_cmds_count;
    char **allowed_cmds;
    size_t allowed_cmds_count;
} sandbox_policy_t;

static sandbox_policy_t g_policy;
static int g_policy_loaded = 0;

// ---------------- 日志输出 ----------------

// 输出沙箱拦截事件日志(action=BLOCK/LOG_ONLY, type=FILE/NET/DNS/CMD)
static void log_event(const char *action, const char *type, const char *target) {
    if (!type) type = "";
    if (!target) target = "";
    fprintf(stderr, "[botsec-sandbox] ACTION=%s TYPE=%s TARGET=%s\n",
            action ? action : "UNKNOWN", type, target);
}

// 输出沙箱运行信息日志，支持 printf 格式化
static void log_info(const char *fmt, ...) {
    if (!fmt) return;
    fprintf(stderr, "[botsec-sandbox] ");
    va_list ap;
    va_start(ap, fmt);
    vfprintf(stderr, fmt, ap);
    va_end(ap);
    fprintf(stderr, "\n");
}

// ---------------- 工具函数 ----------------

// 释放字符串数组及其中每个元素的内存
static void free_string_array(char **arr, size_t count) {
    if (!arr) return;
    for (size_t i = 0; i < count; ++i) {
        free(arr[i]);
    }
    free(arr);
}

// 安全版 strndup，拷贝最多 n 个字节并追加 '\0'
static char *strndup_safe(const char *s, size_t n) {
    char *p = (char *)malloc(n + 1);
    if (!p) return NULL;
    memcpy(p, s, n);
    p[n] = '\0';
    return p;
}

// 在 JSON buffer 中查找键对应的值起始位置
// key 参数不含引号 (如 "blocked_paths"), 函数会检查 JSON 中的 "key": value 格式
static const char *find_key_start(const char *buf, const char *key) {
    if (!buf || !key) return NULL;

    size_t key_len = strlen(key);
    const char *p = buf;
    while ((p = strstr(p, key)) != NULL) {
        if (p > buf && p[-1] == '"' && p[key_len] == '"') {
            const char *colon = strchr(p + key_len + 1, ':');
            if (!colon) { p += key_len; continue; }
            colon++;
            while (*colon == ' ' || *colon == '\t' || *colon == '\n' || *colon == '\r') {
                colon++;
            }
            return colon;
        }
        p += key_len;
    }
    return NULL;
}

// 解析 JSON 中的布尔字段 (true/false)
static int parse_bool_field(const char *buf, const char *key, int *out) {
    const char *v = find_key_start(buf, key);
    if (!v || !out) return -1;
    if (strncmp(v, "true", 4) == 0) { *out = 1; return 0; }
    if (strncmp(v, "false", 5) == 0) { *out = 0; return 0; }
    return -1;
}

// 解析策略类型字段，返回 1 表示白名单，0 表示黑名单(默认)
static int parse_policy_type(const char *buf, const char *key) {
    const char *v = find_key_start(buf, key);
    if (!v || *v != '"') return 0;
    return (strncmp(v + 1, "whitelist", 9) == 0) ? 1 : 0;
}

// 解析 JSON 中的字符串数组字段 (如 "key": ["a", "b"])
static int parse_string_array(const char *buf, const char *key, char ***out_arr, size_t *out_len) {
    if (!out_arr || !out_len) return -1;
    *out_arr = NULL;
    *out_len = 0U;

    const char *v = find_key_start(buf, key);
    if (!v) return -1;

    const char *start = strchr(v, '[');
    if (!start) return -1;
    const char *end = strchr(start, ']');
    if (!end || end <= start) return -1;

    size_t count = 0;
    const char *p = start;
    while (p < end) {
        const char *q = strchr(p, '"');
        if (!q || q >= end) break;
        const char *r = strchr(q + 1, '"');
        if (!r || r >= end) break;
        count++;
        p = r + 1;
    }

    if (count == 0) return 0;

    char **arr = (char **)calloc(count, sizeof(char *));
    if (!arr) return -1;

    size_t idx = 0;
    p = start;
    while (p < end && idx < count) {
        const char *q = strchr(p, '"');
        if (!q || q >= end) break;
        const char *r = strchr(q + 1, '"');
        if (!r || r >= end) break;

        size_t len = (size_t)(r - (q + 1));
        arr[idx] = strndup_safe(q + 1, len);
        if (!arr[idx]) {
            free_string_array(arr, idx);
            return -1;
        }
        idx++;
        p = r + 1;
    }

    *out_arr = arr;
    *out_len = idx;
    return 0;
}

// ---------------- 策略加载 ----------------

// 从指定路径的 JSON 文件加载沙箱策略到全局结构体
static void load_policy_from_file(const char *path) {
    FILE *f = fopen(path, "r");
    if (!f) {
        log_info("policy file not found: %s, sandbox disabled", path);
        return;
    }

    if (fseek(f, 0, SEEK_END) != 0) { fclose(f); return; }
    long size = ftell(f);
    if (size <= 0) { fclose(f); return; }
    if (fseek(f, 0, SEEK_SET) != 0) { fclose(f); return; }

    char *buf = (char *)malloc((size_t)size + 1);
    if (!buf) { fclose(f); return; }

    size_t n = fread(buf, 1, (size_t)size, f);
    fclose(f);
    buf[n] = '\0';

    memset(&g_policy, 0, sizeof(g_policy));

    int log_only = 0;
    if (parse_bool_field(buf, "log_only", &log_only) == 0) {
        g_policy.log_only = log_only;
    }

    g_policy.file_policy_whitelist = parse_policy_type(buf, "file_policy_type");
    g_policy.network_policy_whitelist = parse_policy_type(buf, "network_policy_type");
    g_policy.command_policy_whitelist = parse_policy_type(buf, "command_policy_type");

    parse_string_array(buf, "blocked_paths", &g_policy.blocked_paths, &g_policy.blocked_paths_count);
    parse_string_array(buf, "allowed_paths", &g_policy.allowed_paths, &g_policy.allowed_paths_count);
    parse_string_array(buf, "blocked_ips", &g_policy.blocked_ips, &g_policy.blocked_ips_count);
    parse_string_array(buf, "allowed_ips", &g_policy.allowed_ips, &g_policy.allowed_ips_count);
    parse_string_array(buf, "blocked_domains", &g_policy.blocked_domains, &g_policy.blocked_domains_count);
    parse_string_array(buf, "allowed_domains", &g_policy.allowed_domains, &g_policy.allowed_domains_count);
    parse_string_array(buf, "blocked_commands", &g_policy.blocked_cmds, &g_policy.blocked_cmds_count);
    parse_string_array(buf, "allowed_commands", &g_policy.allowed_cmds, &g_policy.allowed_cmds_count);

    g_policy_loaded = 1;
    free(buf);

    log_info("policy loaded: file=%s(%zu/%zu) net=%s(%zu/%zu) domain=(%zu/%zu) cmd=%s(%zu/%zu) log_only=%d",
             g_policy.file_policy_whitelist ? "whitelist" : "blacklist",
             g_policy.blocked_paths_count, g_policy.allowed_paths_count,
             g_policy.network_policy_whitelist ? "whitelist" : "blacklist",
             g_policy.blocked_ips_count, g_policy.allowed_ips_count,
             g_policy.blocked_domains_count, g_policy.allowed_domains_count,
             g_policy.command_policy_whitelist ? "whitelist" : "blacklist",
             g_policy.blocked_cmds_count, g_policy.allowed_cmds_count,
             g_policy.log_only);
}

// ---------------- 策略匹配 ----------------

// 检查 path 是否以 prefix 开头
static int has_prefix(const char *path, const char *prefix) {
    if (!path || !prefix) return 0;
    size_t n = strlen(prefix);
    if (strlen(path) < n) return 0;
    return strncmp(path, prefix, n) == 0;
}

// 白名单模式下始终允许的系统路径前缀
static int is_system_path(const char *path) {
    static const char *prefixes[] = {
        "/lib/", "/lib64/", "/usr/lib/", "/usr/lib64/",
        "/proc/", "/dev/", "/sys/",
        "/etc/ld.so", "/etc/resolv.conf", "/etc/nsswitch.conf",
        "/etc/hosts", "/etc/ssl/", "/etc/ca-certificates/",
        "/tmp/", "/run/",
        NULL
    };
    for (int i = 0; prefixes[i]; ++i) {
        if (has_prefix(path, prefixes[i])) return 1;
    }
    return 0;
}

// 判断文件路径是否被策略拦截 (支持黑名单/白名单模式)
static int is_path_blocked(const char *path) {
    if (!g_policy_loaded || !path) return 0;

    if (g_policy.file_policy_whitelist) {
        if (g_policy.allowed_paths_count == 0) return 0;
        if (is_system_path(path)) return 0;
        for (size_t i = 0; i < g_policy.allowed_paths_count; ++i) {
            if (has_prefix(path, g_policy.allowed_paths[i])) return 0;
        }
        return 1;
    }

    for (size_t i = 0; i < g_policy.blocked_paths_count; ++i) {
        if (has_prefix(path, g_policy.blocked_paths[i])) return 1;
    }
    return 0;
}

// 判断 IP 地址是否被策略拦截 (支持黑名单/白名单模式)
static int is_ip_blocked(const char *ip) {
    if (!g_policy_loaded || !ip) return 0;

    if (g_policy.network_policy_whitelist) {
        if (g_policy.allowed_ips_count == 0) return 0;
        for (size_t i = 0; i < g_policy.allowed_ips_count; ++i) {
            if (strcmp(ip, g_policy.allowed_ips[i]) == 0) return 0;
        }
        return 1;
    }

    for (size_t i = 0; i < g_policy.blocked_ips_count; ++i) {
        if (strcmp(ip, g_policy.blocked_ips[i]) == 0) return 1;
    }
    return 0;
}

// 域名匹配: 精确匹配或后缀匹配 (e.g. pattern=baidu.com 匹配 www.baidu.com)
static int domain_matches(const char *domain, const char *pattern) {
    if (!domain || !pattern) return 0;
    if (strcasecmp(domain, pattern) == 0) return 1;
    size_t dlen = strlen(domain);
    size_t plen = strlen(pattern);
    if (dlen > plen + 1) {
        const char *suffix = domain + (dlen - plen);
        if (suffix[-1] == '.' && strcasecmp(suffix, pattern) == 0) return 1;
    }
    return 0;
}

// 判断域名是否被策略拦截 (支持黑名单/白名单模式，含后缀匹配)
static int is_domain_blocked(const char *domain) {
    if (!g_policy_loaded || !domain) return 0;

    if (g_policy.network_policy_whitelist) {
        if (g_policy.allowed_domains_count == 0 && g_policy.allowed_ips_count == 0) return 0;
        for (size_t i = 0; i < g_policy.allowed_domains_count; ++i) {
            if (domain_matches(domain, g_policy.allowed_domains[i])) return 0;
        }
        return 1;
    }

    for (size_t i = 0; i < g_policy.blocked_domains_count; ++i) {
        if (domain_matches(domain, g_policy.blocked_domains[i])) return 1;
    }
    return 0;
}

// 判断命令是否被策略拦截 (支持黑名单/白名单模式，子串匹配)
static int is_cmd_blocked(const char *cmd) {
    if (!g_policy_loaded || !cmd) return 0;

    if (g_policy.command_policy_whitelist) {
        if (g_policy.allowed_cmds_count == 0) return 0;
        for (size_t i = 0; i < g_policy.allowed_cmds_count; ++i) {
            if (g_policy.allowed_cmds[i] && strstr(cmd, g_policy.allowed_cmds[i]) != NULL) return 0;
        }
        return 1;
    }

    for (size_t i = 0; i < g_policy.blocked_cmds_count; ++i) {
        if (g_policy.blocked_cmds[i] && strstr(cmd, g_policy.blocked_cmds[i]) != NULL) return 1;
    }
    return 0;
}

// ---------------- 生命周期 ----------------

// 库加载时自动执行: 从环境变量读取策略文件路径并加载
__attribute__((constructor))
static void sandbox_init(void) {
    const char *policy_path = getenv("SANDBOX_POLICY_FILE");
    if (!policy_path || policy_path[0] == '\0') {
        log_info("SANDBOX_POLICY_FILE not set, sandbox disabled");
        return;
    }

    log_info("loading policy from: %s", policy_path);
    load_policy_from_file(policy_path);
    if (g_policy_loaded) {
        log_info("sandbox policy loaded successfully");
    } else {
        log_info("sandbox policy load FAILED");
    }
}

// 库卸载时自动执行: 释放策略结构体内存
__attribute__((destructor))
static void sandbox_fini(void) {
    free_string_array(g_policy.blocked_paths, g_policy.blocked_paths_count);
    free_string_array(g_policy.allowed_paths, g_policy.allowed_paths_count);
    free_string_array(g_policy.blocked_ips, g_policy.blocked_ips_count);
    free_string_array(g_policy.allowed_ips, g_policy.allowed_ips_count);
    free_string_array(g_policy.blocked_domains, g_policy.blocked_domains_count);
    free_string_array(g_policy.allowed_domains, g_policy.allowed_domains_count);
    free_string_array(g_policy.blocked_cmds, g_policy.blocked_cmds_count);
    free_string_array(g_policy.allowed_cmds, g_policy.allowed_cmds_count);
}

// ---------------- 系统调用拦截 ----------------

static int (*real_open_fn)(const char *, int, ...) = NULL;
static int (*real_openat_fn)(int, const char *, int, ...) = NULL;
static int (*real_connect_fn)(int, const struct sockaddr *, socklen_t) = NULL;
static ssize_t (*real_sendto_fn)(int, const void *, size_t, int,
                                 const struct sockaddr *, socklen_t) = NULL;
static ssize_t (*real_sendmsg_fn)(int, const struct msghdr *, int) = NULL;
static int (*real_system_fn)(const char *) = NULL;
static int (*real_execve_fn)(const char *, char *const [], char *const []) = NULL;
static int (*real_getaddrinfo_fn)(const char *, const char *,
                                  const struct addrinfo *, struct addrinfo **) = NULL;

// 从 sockaddr 提取 IP 并检查是否被网络策略拦截，被拦截返回 1
static int check_sockaddr_blocked(const struct sockaddr *addr, socklen_t addrlen,
                                  char *ip_out, size_t ip_out_size) {
    if (!addr || !g_policy_loaded || !ip_out) return 0;
    ip_out[0] = '\0';

    if (addr->sa_family == AF_INET && addrlen >= (socklen_t)sizeof(struct sockaddr_in)) {
        const struct sockaddr_in *in = (const struct sockaddr_in *)addr;
        inet_ntop(AF_INET, &in->sin_addr, ip_out, ip_out_size);
    } else if (addr->sa_family == AF_INET6 && addrlen >= (socklen_t)sizeof(struct sockaddr_in6)) {
        const struct sockaddr_in6 *in6 = (const struct sockaddr_in6 *)addr;
        inet_ntop(AF_INET6, &in6->sin6_addr, ip_out, ip_out_size);
    }

    if (ip_out[0] != '\0' && is_ip_blocked(ip_out)) {
        return 1;
    }
    return 0;
}

// 拦截 open() 系统调用，检查文件路径是否被策略禁止
int open(const char *pathname, int flags, ...) {
    if (!real_open_fn) {
        real_open_fn = (int (*)(const char *, int, ...))dlsym(RTLD_NEXT, "open");
    }

    if (pathname && is_path_blocked(pathname)) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "FILE", pathname);
        if (!g_policy.log_only) {
            errno = EACCES;
            return -1;
        }
    }

    va_list ap;
    va_start(ap, flags);
    int ret;
    if (flags & O_CREAT) {
        int mode_int = va_arg(ap, int);
        mode_t mode = (mode_t)mode_int;
        ret = real_open_fn(pathname, flags, mode);
    } else {
        ret = real_open_fn(pathname, flags);
    }
    va_end(ap);
    return ret;
}

// 拦截 openat() 系统调用，覆盖现代 glibc 内部使用 openat 的场景
int openat(int dirfd, const char *pathname, int flags, ...) {
    if (!real_openat_fn) {
        real_openat_fn = (int (*)(int, const char *, int, ...))dlsym(RTLD_NEXT, "openat");
    }

    if (pathname && is_path_blocked(pathname)) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "FILE", pathname);
        if (!g_policy.log_only) {
            errno = EACCES;
            return -1;
        }
    }

    va_list ap;
    va_start(ap, flags);
    int ret;
    if (flags & O_CREAT) {
        int mode_int = va_arg(ap, int);
        mode_t mode = (mode_t)mode_int;
        ret = real_openat_fn(dirfd, pathname, flags, mode);
    } else {
        ret = real_openat_fn(dirfd, pathname, flags);
    }
    va_end(ap);
    return ret;
}

// 拦截 connect() 系统调用，提取目标 IP 地址并检查是否被策略禁止
int connect(int sockfd, const struct sockaddr *addr, socklen_t addrlen) {
    if (!real_connect_fn) {
        real_connect_fn = (int (*)(int, const struct sockaddr *, socklen_t))dlsym(RTLD_NEXT, "connect");
    }

    char ip[INET6_ADDRSTRLEN] = {0};
    if (check_sockaddr_blocked(addr, addrlen, ip, sizeof(ip))) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "NET", ip);
        if (!g_policy.log_only) {
            errno = ECONNREFUSED;
            return -1;
        }
    }

    return real_connect_fn(sockfd, addr, addrlen);
}

// 拦截 sendto() 系统调用，覆盖 ICMP ping / UDP 等使用 sendto 直接发包的场景
ssize_t sendto(int sockfd, const void *buf, size_t len, int flags,
               const struct sockaddr *dest_addr, socklen_t addrlen) {
    if (!real_sendto_fn) {
        real_sendto_fn = (ssize_t (*)(int, const void *, size_t, int,
                                      const struct sockaddr *, socklen_t))
                         dlsym(RTLD_NEXT, "sendto");
    }

    if (dest_addr) {
        char ip[INET6_ADDRSTRLEN] = {0};
        if (check_sockaddr_blocked(dest_addr, addrlen, ip, sizeof(ip))) {
            log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "NET-SENDTO", ip);
            if (!g_policy.log_only) {
                errno = ECONNREFUSED;
                return -1;
            }
        }
    }

    return real_sendto_fn(sockfd, buf, len, flags, dest_addr, addrlen);
}

// 拦截 sendmsg() 系统调用，覆盖通过 msghdr 指定目标地址发包的场景
ssize_t sendmsg(int sockfd, const struct msghdr *msg, int flags) {
    if (!real_sendmsg_fn) {
        real_sendmsg_fn = (ssize_t (*)(int, const struct msghdr *, int))
                          dlsym(RTLD_NEXT, "sendmsg");
    }

    if (msg && msg->msg_name && msg->msg_namelen > 0) {
        char ip[INET6_ADDRSTRLEN] = {0};
        if (check_sockaddr_blocked((const struct sockaddr *)msg->msg_name,
                                   msg->msg_namelen, ip, sizeof(ip))) {
            log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "NET-SENDMSG", ip);
            if (!g_policy.log_only) {
                errno = ECONNREFUSED;
                return -1;
            }
        }
    }

    return real_sendmsg_fn(sockfd, msg, flags);
}

// 拦截 getaddrinfo() DNS 解析，检查域名是否被策略禁止
int getaddrinfo(const char *node, const char *service,
                const struct addrinfo *hints, struct addrinfo **res) {
    if (!real_getaddrinfo_fn) {
        real_getaddrinfo_fn = (int (*)(const char *, const char *,
                                       const struct addrinfo *, struct addrinfo **))
                              dlsym(RTLD_NEXT, "getaddrinfo");
    }

    if (node && g_policy_loaded && is_domain_blocked(node)) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "DNS", node);
        if (!g_policy.log_only) {
            return EAI_FAIL;
        }
    }

    return real_getaddrinfo_fn(node, service, hints, res);
}

// 拦截 system() 调用，检查命令是否被策略禁止
int system(const char *command) {
    if (!real_system_fn) {
        real_system_fn = (int (*)(const char *))dlsym(RTLD_NEXT, "system");
    }

    if (command && is_cmd_blocked(command)) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "CMD", command);
        if (!g_policy.log_only) {
            errno = EPERM;
            return -1;
        }
    }

    return real_system_fn(command);
}

// 拦截 execve() 系统调用，检查执行的命令是否被策略禁止
int execve(const char *filename, char *const argv[], char *const envp[]) {
    if (!real_execve_fn) {
        real_execve_fn = (int (*)(const char *, char *const [], char *const []))dlsym(RTLD_NEXT, "execve");
    }

    const char *cmd = filename;
    if (argv && argv[0]) {
        cmd = argv[0];
    }

    if (cmd && is_cmd_blocked(cmd)) {
        log_event(g_policy.log_only ? "LOG_ONLY" : "BLOCK", "CMD", cmd);
        if (!g_policy.log_only) {
            errno = EPERM;
            return -1;
        }
    }

    return real_execve_fn(filename, argv, envp);
}
