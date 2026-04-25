import '../l10n/app_localizations.dart';

bool _isZh(AppLocalizations l10n) =>
    l10n.localeName.toLowerCase().startsWith('zh');

String localizeSecurityEventType(String raw, AppLocalizations l10n) {
  switch (raw.trim().toLowerCase()) {
    case 'blocked':
      return l10n.eventBlocked;
    case 'needs_confirmation':
      return l10n.riskTypeNeedsConfirmation;
    case 'tool_execution':
      return l10n.eventToolExecution;
    case 'warning':
      return l10n.eventTypeWarning;
    case 'other':
      return l10n.eventOther;
    default:
      return raw;
  }
}

String localizeSecurityRiskType(String raw, AppLocalizations l10n) {
  final isZh = _isZh(l10n);
  switch (raw.trim().toUpperCase()) {
    case 'PROMPT_INJECTION_DIRECT':
      return isZh ? '直接提示词注入' : 'Direct Prompt Injection';
    case 'PROMPT_INJECTION_INDIRECT':
      return isZh ? '间接提示词注入' : 'Indirect Prompt Injection';
    case 'SENSITIVE_DATA_EXFILTRATION':
      return isZh ? '敏感数据外泄' : 'Sensitive Data Exfiltration';
    case 'HIGH_RISK_OPERATION':
      return isZh ? '高危操作' : 'High-Risk Operation';
    case 'PRIVILEGE_ABUSE':
      return isZh ? '权限滥用' : 'Privilege Abuse';
    case 'UNEXPECTED_CODE_EXECUTION':
      return isZh ? '非预期代码执行' : 'Unexpected Code Execution';
    case 'CONTEXT_POISONING':
      return isZh ? '上下文污染' : 'Context Poisoning';
    case 'SUPPLY_CHAIN_RISK':
      return isZh ? '供应链风险' : 'Supply Chain Risk';
    case 'HUMAN_TRUST_EXPLOITATION':
      return isZh ? '人类信任利用' : 'Human Trust Exploitation';
    case 'CASCADING_FAILURE':
      return isZh ? '级联故障风险' : 'Cascading Failure Risk';
    case 'QUOTA':
      return l10n.riskTypeQuota;
    case 'SANDBOX_BLOCKED':
      return l10n.riskTypeSandboxBlocked;
    case 'NEEDS_CONFIRMATION':
      return l10n.riskTypeNeedsConfirmation;
    default:
      return raw;
  }
}
