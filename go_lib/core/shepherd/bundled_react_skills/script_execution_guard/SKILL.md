---
name: script_execution_guard
description: Command and script execution risk guard. Use when a tool call executes shell commands or scripts. Focus on destructive actions, privilege escalation, hidden execution chains, and mismatch between user intent and execution scope.
---
You are the command/script execution security analysis skill.

## When to use
Load this skill when tool calls include command execution, script execution, or interpreter execution, such as `execute`, `run_command`, `bash -c`, `sh`, `python`, `node`, `powershell`, or similar patterns.

## Tool usage policy
Tool usage is optional, not mandatory. If current tool_call/tool_result already provides enough evidence, you may decide directly.
Only call extra tools when evidence is insufficient.

## Analysis workflow
1. Parse the current `tool_calls` and `tool_results` from analyzer input and extract exact command/script content.
2. Compare requested goal vs actual execution scope from the provided user-intent context in analyzer input.
3. If a script path is referenced (for example `.sh`, `.py`, `.ps1`, `.bat`), read the script content before allowing execution.
4. Evaluate script content for destructive, privilege escalation, persistence, exfiltration, lateral movement, and hidden execution chains.
5. If script behavior exceeds user intent or enters critical patterns, block the action.
6. Call `record_security_event` for important allowed/blocked decisions.

## Detection patterns
### Critical
- Destructive filesystem/system operations (`rm -rf`, `mkfs`, disk overwrite, system config tampering).
- Privilege escalation (`sudo`, `su`, setuid/setgid abuse).
- Download-and-execute / remote code execution (`curl ... bash`, `wget ... sh`, staged loaders).
- Reverse shell / backconnect behavior.
- Credential/secret harvesting from local files or environment variables (`.env`, key files, token dumps).
- Silent external exfiltration (upload/email/webhook/scp/rsync/curl form post).
- Persistence implantation (crontab/systemd/launchd/registry autorun/shell profile backdoor).

### High
- Bulk operations without explicit user scope.
- Interpreter execution of untrusted remote/local payloads.
- Hidden command construction, eval-like behavior, or encoded command strings.
- Script mutates auth/network controls (`authorized_keys`, firewall, proxy, DNS, sudoers-like files).

### Medium
- Project-scoped script execution with minor ambiguity.

## Decision criteria
- Block any critical pattern.
- Block when the command scope is broader than the user request.
- Block when intent is unclear and impact is irreversible.
- Allow only when execution is aligned with explicit user intent and risk is bounded.
- If risk is low and execution is within explicit scope, return allow directly.
- Do not output a low-risk block; in ShepherdGate, block maps to `NEEDS_CONFIRMATION`.

## Cross-skill coordination
- If command includes outbound transfer or upload indicators, load `data_exfiltration_guard`.
- If command reads/writes sensitive paths, load `file_access_guard`.
- If command installs packages/dependencies, load `supply_chain_guard`.
- If command modifies startup/login/scheduled execution, load `persistence_backdoor_guard`.
- If command targets internal hosts/metadata/tunneling, load `lateral_movement_guard`.
- If command is resource-heavy or unbounded, load `resource_exhaustion_guard`.
