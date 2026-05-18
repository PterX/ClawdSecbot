import '../l10n/app_localizations.dart';
import '../models/audit_log_model.dart';
import '../models/security_event_model.dart';

/// 生成审计日志详情的 Markdown 文本（.md 导出），文案随 [l10n] 语言变化。
String buildAuditLogMarkdownContent({
  required AppLocalizations l10n,
  required AuditLog log,
  required List<SecurityEvent> relatedEvents,
  required String rawText,
  required String actionText,
  required String eventText,
}) {
  final title = l10n.auditLogMarkdownDetailExportTitle;
  final safeRaw = _escapeMarkdown(rawText);
  final safeAction = _escapeMarkdown(actionText);
  final safeEvent = _escapeMarkdown(
    relatedEvents.isEmpty ? l10n.auditLogMarkdownNoRelatedEventsBody : eventText,
  );
  return '''
# $title

${l10n.auditLogMarkdownSectionMeta}
- ID: ${_escapeMarkdown(log.id)}
- Request ID: ${_escapeMarkdown(log.requestId)}
- Timestamp: ${_escapeMarkdown(log.timestamp.toIso8601String())}
- Asset: ${_escapeMarkdown(log.assetName)} (${_escapeMarkdown(log.assetID)})
- Model: ${_escapeMarkdown(log.model ?? '')}
- Action: ${_escapeMarkdown(log.action)}
- Risk Level: ${_escapeMarkdown(log.riskLevel ?? '')}
- Risk Reason: ${_escapeMarkdown(log.riskReason ?? '')}

${l10n.auditLogMarkdownSectionRaw}
```text
$safeRaw
```

${l10n.auditLogMarkdownSectionActions}
```text
$safeAction
```

${l10n.auditLogMarkdownSectionEvents}
```text
$safeEvent
```
''';
}

/// Markdown 转义，避免与格式语法冲突。
String _escapeMarkdown(String input) {
  return input
      .replaceAll('```', '\\`\\`\\`')
      .replaceAll('\r\n', '\n')
      .replaceAll('\r', '\n');
}
