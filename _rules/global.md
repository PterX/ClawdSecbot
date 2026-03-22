# ClawSecbot 全局开发规范

**产品名称:** ClawSecbot —— 面向 Bot 类端侧智能体的桌面安全防护软件

## 核心架构

- **前后端分离:** Flutter Desktop(UI+状态) + Go(全业务逻辑) + FFI 通信
- **单体集成:** 所有插件和 core 编译为单一动态库(botsec.dylib/so/dll),共享路径/数据库/日志实例
- **跨平台:** 兼容 Windows/Linux/macOS,macOS 支持 x86_64 和 arm64

## 代码规范

- 英文注释,英文日志;单文件 ≤1500 行
- 禁止无用代码、重复实现、未经要求的逻辑
- 禁止过度封装导致语义模糊：单一职责、语义清晰、组合优于继承
- 强制国际化(i18n);Flutter 生产环境必须使用 appLogger
- Go 层业务逻辑需要有单元测试
- 除非明确要求，否则不用向前兼容，不用做数据迁移

## FFI 通信规范

### 动态库加载
- `NativeLibraryService` 单例加载 dylib,应用启动时 `initialize()` 一次,缓存 `dylib/libraryPath/freeString`
- 主 Isolate 服务通过 `NativeLibraryService().dylib` 获取已缓存实例
- 后台 Isolate 通过 `DynamicLibrary.open(libPath)` 重新打开(OS 保证同一句柄)

### 数据协议
- 统一 JSON 输入/输出,响应格式: `{"success": bool, "data": ..., "error": ...}`
- Go 端必须使用 `json.Marshal` 序列化,禁止 `fmt.Sprintf` 拼接
- 复用 `core/ffi.go` 的 `toCString` 辅助函数

### 回调机制
- Go -> Dart 使用 `NativeCallable.listener` 实现回调
- `MessageBridgeService` 负责注册回调、分发日志/指标/状态流
- 回调优先,轮询(200ms)降级

### CGO 约束
- 所有 `//export` 函数必须在 `package main` 中,symbol 不重复
- `.h` 头文件为编译产物,已加入 `.gitignore`

## Go 层架构

```
go_lib/
├── main.go              # dylib 入口,导出所有 FFI
├── core/                # 核心公共包
│   ├── plugin.go        # BotPlugin 接口
│   ├── plugin_manager.go # 插件管理器
│   ├── path_manager.go  # 路径管理器
│   ├── ffi.go           # FFI 辅助函数
│   ├── logging/         # 日志模块
│   ├── repository/      # 数据访问层(CRUD)
│   ├── service/         # 业务服务层
│   ├── scanner/         # 资产扫描引擎
│   ├── sandbox/         # 沙箱策略模块
│   └── callback_bridge/ # FFI 回调桥接
├── plugins/openclaw/    # Openclaw 插件
├── skillagent/          # Skill Agent 独立包
└── chatmodel-routing/   # LLM 协议转换模块
```

### 插件规范
- 实现 `BotPlugin` 接口: `GetID/GetAssetName/ScanAssets/AssessRisks/Start|Stop|GetProtectionStatus`
- `init()` 中自动注册到全局 `PluginManager`
- UI 层通过 core 聚合方法调用,不直接与插件交互
- 插件随软件启动初始化、退出卸载

### 路径管理
- `PathManager` 集中管理所有路径
- Flutter 传入 `workspaceDir/homeDir` -> Go 派生 `logDir/backupDir/policyDir/dbPath`

## Flutter 层架构

### 核心服务
- `NativeLibraryService`: 动态库加载唯一入口
- `PluginService`: 插件管理
- `ProtectionService`: 防护代理管理
- `ProtectionMonitorService`: 防护监控(日志/指标/事件流)
- `MessageBridgeService`: FFI 回调桥接
- 各 `*DatabaseService`: 业务域数据库 FFI 封装

### 关键约束
- 所有数据库操作通过 FFI 在 Go 层完成,Flutter 禁止直接操作 DB

## 数据库规范

- **分层:** `repository`(数据访问) -> `service`(业务逻辑) -> `main.go` FFI(C 字符串转换)
- **FFI 位置:** 数据库相关 FFI(如 db_ffi.go)必须位于 `core/` 目录
- **调用链路:** Flutter -> main.go FFI -> core/service -> core/repository
- 每个业务域独立 repository/service 文件
- DB 为全局实例,启动时初始化,退出时关闭
- 表结构初始化保持幂等(`CREATE TABLE IF NOT EXISTS`)
- ShepherdGate 规则优先从 SQLite 查询,无记录则用默认规则

## 日志规范

- 执行流程需有 info 日志记录
- 日志必须精简有效,仅保留: ① 链路追踪标识 ② 关键操作节点 ③ 真实错误信息
- 禁止调试杂音、冗余状态打印
- Go 使用 `core/logging`,Flutter 使用 `appLogger`

## 业务规范

### 资产与插件
- 插件通过 `GetAssetName` 导出资产名称(如 'openclaw')
- 每个 Bot 有独立资产属性(工作区路径/配置文件/命令行名称/端口/进程路径)

### 防护代理
- 代理 Bot 与 LLM 接口请求,分析内容判断风险
- 检测到风险时模拟返回风险提示,要求用户确认
- 代理实例归属于资产插件实例,支持持久化到数据库

### 协议转换
- 代理接收 OpenAI 标准协议请求,转换为目标 LLM 格式转发
- 解析目标 LLM 返回,转换为 OpenAI 格式返回
- 每种 provider 在 `chatmodel-routing/providers/` 独立实现
- 支持非流式/流式/reasoning/toolcall/usage 等数据解析和转换
- **Provider 路由:** Flutter UI 选择 -> 存入数据库 `provider` 字段 -> Flutter 读取自动解析为 `AgentConfig.type` -> Go 层 `BotModelConfig.Provider` 接收 -> `adapter.NormalizeProviderName()` 标准化 -> `createForwardingProvider()` switch 匹配实例化

### 审计日志
- 必录字段: request_id/timestamp/model/request_content/tool_calls/output_content/has_risk/risk_level/risk_reason/action/token 用量/duration
- Go 端内存缓存,Flutter 端定期(每 25 次轮询周期)通过 `GetPendingAuditLogs` 同步到 SQLite

## 沙箱规范(macOS)

### 策略文件路径
- 沙箱策略文件必须放在 `~/.botsec/policies/`(用户主目录)
- 原因: `sandbox-exec` 需在沙箱生效前读取策略文件,应用沙盒目录内文件不可访问
- 数据库文件使用 `getApplicationSupportDirectory()`,与策略文件路径分离

### Seatbelt 策略生成
- **黑名单模式:** `deny` 规则必须放在 `(allow default)` 之前
- **白名单模式:** `(deny default)` 在前 -> 必要系统 allow 规则 -> 用户 allow 规则
- 路径使用绝对路径 `(subpath "/absolute/path")`,不使用 `(home)` 可移植语法
- 引用 HOME 目录需在策略顶部显式声明 `(define user-homedir (param "HOME"))`

### 进程监控自愈
- 通过进程监控器检测用户手动启动的网关进程,发现后终止并使用 `sandbox-exec` 在 Seatbelt 策略下重启,实现沙箱防护自动恢复
- 沙箱操作需先检查 plist 状态保证幂等

## Windows 平台规范

### 沙箱实现(MinHook)
- Windows 沙箱通过 `sandbox_hook.dll` 实现用户态 API Hook,等效于 macOS Seatbelt / Linux LD_PRELOAD
- Hook 库: MinHook (轻量内联 Hook),拦截 `CreateFileW`/`CreateProcessW`/`connect`/`WSAConnect`
- 注入流程: `CREATE_SUSPENDED` → `CreateRemoteThread(LoadLibraryW)` → `ResumeThread`
- 子进程继承: `CreateProcessW` Hook 自动向子进程注入相同 DLL,实现沙箱链式传递
- 策略格式与 Linux PreloadConfig 兼容,通过 `SANDBOX_POLICY_FILE` 环境变量传递
- 审计日志写入 `SANDBOX_LOG_FILE`,由 Go 层 `HookLogWatcher` 轮询并接入 SecurityEvent 管线
- `isSandboxSupportedOnPlatform()` 在找到 `sandbox_hook.dll` 时返回 true
- 失败降级: DLL 注入失败时终止进程并上报错误,不允许"伪保护"状态

### 网关管理
- 沙箱可用时: 通过 SandboxManager 以 Hook 模式启动,支持策略执行与自愈重启
- 沙箱不可用时: 回退到直接进程 start/stop,无服务注册
- 进程管理使用 `taskkill`/`tasklist`/`wmic` 替代 Unix `pgrep`/`ps`/`SIGTERM`

### 桌面 UI 一致性
- 窗口标题栏: 与 macOS 对齐,使用 `TitleBarStyle.hidden` + Flutter 自定义标题栏。Windows 在 `win32_window.cpp` 使用无边框样式(无原生标题栏/系统菜单),仅保留 Flutter 绘制的标题栏与 -/x 按钮
- 子窗口(审计/监控): Windows 使用对称 `horizontal: 16` padding,macOS 保留 `left: 78` 为红黄绿预留
- 子窗口标题栏包含最小化/关闭按钮(macOS 由原生红黄绿处理)
- 托盘图标: `images/tray_icon.ico`,应用图标: `windows/runner/resources/app_icon.ico`
- 菜单项结构与 Linux/macOS 一致

### 平台抽象规范
- Go 平台相关函数通过文件名后缀(`_windows.go`/`_darwin.go`/`_linux.go`)或 build tag 分离
- 进程信号操作统一使用 `sandbox/process_{unix,windows}.go` 中的 `gracefulTerminate`/`KillProcess` 等函数
- 信号重置使用 `signal_{unix,windows}.go` 中的 `resetSignals()`,Windows 不支持 SIGTSTP
- Dart 层获取用户主目录使用 `Platform.environment['HOME'] ?? Platform.environment['USERPROFILE']` 双 fallback
- 文件可执行权限检查需跳过 Windows(`runtime.GOOS != "windows"`)
- `platformPostStart()` 为各平台的进程启动后钩子,Windows 用于 DLL 注入+恢复线程

### 路径与环境变量
- 策略目录: `%USERPROFILE%\.botsec\policies\`(与 Unix `~/.botsec/policies/` 对应)
- 环境变量: 使用 `USERPROFILE` 替代 `HOME`,Go 层 `os.UserHomeDir()` 自动适配
- Hook DLL 搜索路径: 可执行文件同目录 > 策略目录 > 工作目录

## 构建与 SCM

### 构建脚本
- `scripts/build_go.sh`: 构建 Go 安全引擎库
- `scripts/build_openclaw_plugin.sh`: 构建 Openclaw 插件
- `scripts/build_macos_release.sh`: macOS 发布构建
- `scripts/build_windows_release.ps1`: Windows 发布构建(PowerShell,含 hook DLL CMake 构建)
- `scripts/build_windows_from_mac.sh`: 通过 Parallels VM 从 Mac 构建 Windows 包
- `scripts/build_linux_release.sh`: Linux 发布构建(DEB/RPM)
- `scripts/generate_icons.sh`: 从 `icon_1024.png` 生成所有平台图标(含 Windows app_icon.ico)
- `chatmodel-routing` 作为 `go_lib/` 下的独立模块存在
- Hook DLL 源码: `go_lib/core/sandbox/windows_hook/`,使用 CMake + FetchContent(MinHook + cJSON)

### Git 管理
- cgo 生成的 `.h` 头文件为编译产物,已加入 `.gitignore`
- `go_lib/` 下的编译产物(.dylib/.so/.dll)不纳入版本管理
