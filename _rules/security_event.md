# 安全事件规范（SecurityEvent）

> 适用范围：防护监控界面的"拦截次数"计数与"安全事件"列表的数据生成路径。
> 本文档描述触发口径与职责边界；`_rules/global.md` §7 仅保留索引。

## 1. 目标与定位

- `SecurityEvent`：防护链路中每一次"被防护信号"的离散记录，作为监控界面事件面板的权威数据源。
- 约束：**"拦截次数" ≤ "安全事件数量"**，即拦截计数每递增一次，事件列表必然存在对应记录。

## 2. 广义事件边界

以下防护信号必须生成 `SecurityEvent`（广义定义，Q1 口径）：

1. ShepherdGate 启发式命中（`isSensitiveByRules` / `detectCriticalCommand` / `detectToolResultInjection`）
2. ShepherdGate ReAct agent 深度分析的非 `Allowed` 决策
3. ShepherdGate post-validation override 强制拦截（即 LLM `Allowed=true` 但 Go 侧因提示词注入风险强制改写）
4. 代理层 token 配额命中的阻断（会话级 / 每日级）
5. 沙箱命中后的阻断（`SANDBOX_BLOCKED` 分支）

## 3. 单一权威写入点（强约束）

- **唯一写入点**：`go_lib/core/proxy/proxy_protection.go::emitSecurityEvent`，在 `ProxyProtection` 的决策汇聚点调用。
- **同步语义**：`emitSecurityEvent` 必须与 `blockedCount++` / `warningCount++` 位于同一代码块，禁止拆分到异步路径，保证监控面板两侧数据一致。
- **禁止写入点**：
  - ShepherdGate 内部（`shepherd_react_analyzer.go` 启发式分支）**不得**直接调用 `AddSecurityEvent`，结果由 `ShepherdDecision` 向上透传、由代理层统一落库。
  - 沙箱 hook 审计日志（`sandbox.StartHookLogWatcherByKey`）**不得**再被插件网关消费为 `SecurityEvent`。沙箱命中事件仅通过代理层的 `SANDBOX_BLOCKED` 分支记录。
  - `record_security_event` 工具（ReAct agent 可调用）是"观察性补充"，不作为拦截计数的来源，仅承载额外上下文（允许轻度重复，不计入"拦截次数"）。

## 4. 字段口径

| 字段 | 来源 |
| --- | --- |
| `Source` | 固定为 `"react_agent"`（代理层写入点），沿用既有 UI 徽章分类 |
| `EventType` | 严格按决策语义：`"blocked"`（硬拦截、配额、沙箱）/ `"needs_confirmation"`（ShepherdGate `NEEDS_CONFIRMATION` 决策，待用户确认）；观察性工具可写 `"tool_execution"` 等 |
| `ActionDesc` | 优先使用 `decision.ActionDesc`，缺省时回退到 `decision.Reason` |
| `RiskType` | 优先使用 `decision.RiskType`，缺省时回退到 `decision.Status` / 分支常量（`QUOTA` / `SANDBOX_BLOCKED`） |
| `Detail` | `decision.Reason`；post-validation override 额外前置 `post_validation_override \| ` 标签 |
| `RequestID` | 绑定当前请求上下文的 `requestID`，便于与 `TruthRecord` / `AuditChain` 交叉 |

## 5. Post-validation Override 识别

- `shepherd.PostValidationOverrideTag` 常量定义于 `go_lib/core/shepherd/shepherd_gate.go`。
- 代理层通过 `strings.Contains(decision.Reason, shepherd.PostValidationOverrideTag)` 识别并在 `Detail` 加前缀，避免修改 `ShepherdDecision` 结构。

## 6. Fail-open 约定

- `AddSecurityEvent` 内部任何持久化或推送失败都不得阻断代理主流程。
- 监控/审计异常与代理决策解耦，延续 `_rules/audit_chain.md` §2 的 fail-open 原则。

## 7. 职责边界表

| 层 | 职责 | 是否写 SecurityEvent |
| --- | --- | --- |
| ShepherdGate 启发式 | 返回 `ReactRiskDecision` | 否（由代理汇聚） |
| ShepherdGate ReAct agent | 返回 `ReactRiskDecision` | 否（由代理汇聚） |
| ShepherdGate post-validation | 覆写 `Allowed=false` 并追加 tag | 否（由代理汇聚） |
| `ProxyProtection` 决策汇聚 | `blockedCount++` + `emitSecurityEvent` | **是（唯一权威写入点）** |
| 沙箱 hook 审计日志 | 本地审计文件落盘 | 否（不再被消费为 `SecurityEvent`） |
| `record_security_event` 工具 | agent 观察性补录 | 是（观察性，不计入拦截次数） |
