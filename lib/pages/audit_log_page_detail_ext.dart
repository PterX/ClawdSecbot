part of 'audit_log_page.dart';

extension _AuditLogPageDetailExt on _AuditLogPageState {
  /// Replaces illegal filename characters with `_` for cross-platform safety.
  String _safeFileNameSegment(String input) {
    final normalized = input.trim();
    if (normalized.isEmpty) return 'log';
    return normalized.replaceAll(RegExp(r'[\\/:*?"<>|]'), '_');
  }

  List<_RawMessageItem> _buildRawMessagesWithToolFallback(AuditLog log) {
    final l10n = AppLocalizations.of(context)!;
    final nonSystemMessages =
        log.messages.where((msg) => msg.role.toLowerCase() != 'system').toList()
          ..sort((a, b) => a.index.compareTo(b.index));

    if (nonSystemMessages.isEmpty) {
      return _buildRawMessagesWithoutMessageTimeline(log);
    }

    final hasExplicitToolMessages = nonSystemMessages.any((msg) {
      final role = msg.role.toLowerCase();
      return role == 'tool' || role == 'toolcall' || role == 'tool_call';
    });

    final timeline = <_RawMessageItem>[];
    int toolIndex = 0;

    for (final msg in nonSystemMessages) {
      final role = msg.role.toLowerCase();

      if (!hasExplicitToolMessages &&
          role == 'assistant' &&
          toolIndex < log.toolCalls.length) {
        timeline.addAll(_buildRawToolTimeline(log.toolCalls.skip(toolIndex)));
        toolIndex = log.toolCalls.length;
      }

      final roleLabel = msg.role.isNotEmpty
          ? '${msg.role[0].toUpperCase()}${msg.role.substring(1)}'
          : 'Unknown';
      final content = msg.content.trim().isNotEmpty
          ? msg.content.trim()
          : (l10n.auditLogContentEmptyPlaceholder);
      timeline.add(_RawMessageItem(roleLabel: roleLabel, content: content));
    }

    if (!hasExplicitToolMessages && toolIndex < log.toolCalls.length) {
      timeline.addAll(_buildRawToolTimeline(log.toolCalls.skip(toolIndex)));
    }

    return timeline;
  }

  List<_RawMessageItem> _buildRawMessagesWithoutMessageTimeline(AuditLog log) {
    final timeline = <_RawMessageItem>[];
    final requestContent = log.requestContent.trim();
    if (requestContent.isNotEmpty) {
      timeline.add(_RawMessageItem(roleLabel: 'User', content: requestContent));
    }

    timeline.addAll(_buildRawToolTimeline(log.toolCalls));

    final output = (log.outputContent ?? '').trim();
    if (output.isNotEmpty) {
      timeline.add(_RawMessageItem(roleLabel: 'Assistant', content: output));
    }
    return timeline;
  }

  List<_RawMessageItem> _buildRawToolTimeline(Iterable<AuditToolCall> calls) {
    final timeline = <_RawMessageItem>[];
    for (final tc in calls) {
      timeline.add(
        _RawMessageItem(roleLabel: 'ToolCall', content: _buildToolCallRaw(tc)),
      );
      final result = (tc.result ?? '').trim();
      if (result.isNotEmpty) {
        timeline.add(_RawMessageItem(roleLabel: 'ToolResult', content: result));
      }
    }
    return timeline;
  }

  String _buildToolCallRaw(AuditToolCall tc) {
    final l10n = AppLocalizations.of(context)!;
    final args = tc.arguments.trim().isNotEmpty ? tc.arguments.trim() : '{}';
    return l10n.auditLogToolCallRawLine(tc.name, args);
  }

  Widget _buildLogDetail() {
    final l10n = AppLocalizations.of(context)!;
    final log = _selectedLog!;
    final rawMessages = _buildRawMessagesWithToolFallback(log);

    return Container(
      margin: const EdgeInsets.fromLTRB(12, 0, 12, 12),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Colors.white.withValues(alpha: 0.1)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
              ),
            ),
            child: Row(
              children: [
                Text(
                  l10n.auditLogDetail,
                  style: AppFonts.inter(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: Colors.white,
                  ),
                ),
                const Spacer(),
                Tooltip(
                  message: l10n.auditLogExport,
                  child: IconButton(
                    icon: const Icon(LucideIcons.download, size: 16),
                    color: Colors.white54,
                    onPressed: _exportLogDetail,
                  ),
                ),
                IconButton(
                  icon: const Icon(LucideIcons.x, size: 16),
                  color: Colors.white54,
                  onPressed: () => _selectLog(null),
                ),
              ],
            ),
          ),
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildDetailRow(l10n.auditLogId, log.id),
                  _buildDetailRow(
                    l10n.auditLogTimestamp,
                    _formatTimestamp(log.timestamp),
                  ),
                  _buildDetailRow(
                    l10n.auditLogRequestId,
                    log.requestId,
                  ),
                  if (log.model != null)
                    _buildDetailRow(l10n.auditLogModel, log.model!),
                  _buildDetailRow(
                    l10n.auditLogAction,
                    _getActionText(log.action),
                  ),
                  if (log.hasRisk) ...[
                    _buildDetailRow(
                      l10n.auditLogRiskLevel,
                      log.riskLevel ?? 'N/A',
                    ),
                    _buildDetailRow(
                      l10n.auditLogRiskReason,
                      log.riskReason ?? 'N/A',
                    ),
                  ],
                  _buildDetailRow(
                    l10n.auditLogDuration,
                    '${log.durationMs}ms',
                  ),
                  if (log.totalTokens != null)
                    _buildDetailRow(
                      l10n.auditLogTokens,
                      '${log.totalTokens}',
                    ),

                  const SizedBox(height: 16),
                  _buildSectionTitleWithCopy(
                    rawMessages.isNotEmpty
                        ? l10n.auditLogSectionRawWithCount(rawMessages.length)
                        : l10n.auditLogSectionRaw,
                    _buildRawSectionText(log),
                  ),
                  if (rawMessages.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Container(
                      width: double.infinity,
                      padding: const EdgeInsets.fromLTRB(10, 10, 10, 2),
                      decoration: BoxDecoration(
                        color: Colors.black.withValues(alpha: 0.22),
                        borderRadius: BorderRadius.circular(10),
                        border: Border.all(
                          color: Colors.white.withValues(alpha: 0.08),
                        ),
                      ),
                      child: Column(
                        children: [
                          for (int i = 0; i < rawMessages.length; i++)
                            _buildConversationTimelineItem(
                              log: log,
                              item: rawMessages[i],
                              index: i,
                              total: rawMessages.length,
                            ),
                        ],
                      ),
                    ),
                  ] else ...[
                    _buildCodeBlock(log.requestContent),
                  ],

                  if (log.toolCalls.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    _buildSectionTitleWithCopy(
                      l10n.auditLogSectionActionsWithCount(log.toolCalls.length),
                      _buildActionSectionText(log),
                    ),
                    ...log.toolCalls.map((tc) => _buildToolCallItem(tc)),
                  ],

                  const SizedBox(height: 16),
                  _buildSectionTitleWithCopy(
                    _relatedEvents.isNotEmpty
                        ? l10n.auditLogSectionEventsWithCount(
                            _relatedEvents.length,
                          )
                        : l10n.auditLogSectionEventsHeading,
                    _relatedEvents.isNotEmpty
                        ? _buildEventSectionText(_relatedEvents)
                        : '',
                  ),
                  if (_relatedEvents.isNotEmpty)
                    ..._relatedEvents.map((evt) => _buildSecurityEventCard(evt))
                  else
                    Padding(
                      padding: const EdgeInsets.only(top: 4),
                      child: Text(
                        l10n.auditLogNoRelatedSecurityEvents,
                        style: AppFonts.inter(
                          fontSize: 12,
                          color: Colors.white38,
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Builds one security event card.
  Widget _buildSecurityEventCard(SecurityEvent evt) {
    final l10n = AppLocalizations.of(context)!;
    final blocked = evt.isBlocked;
    final accent = blocked ? Colors.red : Colors.amber;
    final typeLabel = switch (evt.eventType) {
      'blocked' => l10n.auditLogEventBadgeBlocked,
      'needs_confirmation' => l10n.auditLogEventBadgeNeedsConfirmation,
      'tool_execution' => l10n.auditLogEventBadgeToolExecution,
      _ => localizeSecurityEventType(
        evt.eventType,
        l10n,
      ),
    };
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: accent.withValues(alpha: blocked ? 0.1 : 0.05),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: accent.withValues(alpha: 0.3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                blocked ? LucideIcons.shieldAlert : LucideIcons.alertTriangle,
                size: 14,
                color: accent,
              ),
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                decoration: BoxDecoration(
                  color: accent.withValues(alpha: 0.2),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  typeLabel,
                  style: AppFonts.firaCode(
                    fontSize: 9,
                    fontWeight: FontWeight.w600,
                    color: accent,
                  ),
                ),
              ),
              if (evt.riskType.isNotEmpty) ...[
                const SizedBox(width: 6),
                Flexible(
                  child: Text(
                    localizeSecurityRiskType(
                      evt.riskType,
                      l10n,
                    ),
                    style: AppFonts.firaCode(
                      fontSize: 10,
                      color: Colors.white54,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
              const Spacer(),
              Text(
                evt.source,
                style: AppFonts.firaCode(fontSize: 9, color: Colors.white38),
              ),
            ],
          ),
          const SizedBox(height: 6),
          SelectableText(
            localizeSecurityActionDesc(
              evt.actionDesc,
              l10n,
            ),
            style: AppFonts.inter(fontSize: 11, color: Colors.white70),
          ),
          if (evt.detail.isNotEmpty) ...[
            const SizedBox(height: 4),
            SelectableText(
              evt.detail,
              style: AppFonts.firaCode(fontSize: 10, color: Colors.white38),
            ),
          ],
        ],
      ),
    );
  }

  String _getActionText(String action) {
    final l10n = AppLocalizations.of(context)!;
    switch (action.toUpperCase()) {
      case 'BLOCK':
      case 'HARD_BLOCK':
        return l10n.auditLogActionBlock;
      case 'WARN':
        return l10n.auditLogActionWarn;
      case 'ALLOW':
      default:
        return l10n.auditLogActionAllow;
    }
  }

  Widget _buildDetailRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: AppFonts.inter(fontSize: 12, color: Colors.white54),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: AppFonts.firaCode(fontSize: 12, color: Colors.white),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSectionTitleWithCopy(String title, String copyText) {
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Text(
            title,
            style: AppFonts.inter(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: const Color(0xFF6366F1),
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(LucideIcons.copy, size: 16),
            color: Colors.white54,
            tooltip: l10n.appStoreGuideCopy,
            onPressed: () => _copyText(copyText),
          ),
        ],
      ),
    );
  }

  Widget _buildCodeBlock(String content) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(8),
      ),
      child: SelectableText(
        content,
        style: AppFonts.firaCode(fontSize: 11, color: Colors.white70),
      ),
    );
  }

  _RawTimelineStyle _rawTimelineStyleForRole(String roleLabel) {
    switch (roleLabel.toLowerCase()) {
      case 'user':
        return const _RawTimelineStyle(
          color: Color(0xFF22C55E),
          icon: LucideIcons.user,
        );
      case 'assistant':
        return const _RawTimelineStyle(
          color: Color(0xFF6366F1),
          icon: LucideIcons.bot,
        );
      case 'toolcall':
      case 'tool call':
        return const _RawTimelineStyle(
          color: Color(0xFFF59E0B),
          icon: LucideIcons.wrench,
        );
      case 'toolresult':
      case 'tool result':
      case 'tool':
        return const _RawTimelineStyle(
          color: Color(0xFFEC4899),
          icon: LucideIcons.flaskConical,
        );
      default:
        return const _RawTimelineStyle(
          color: Colors.white70,
          icon: LucideIcons.circle,
        );
    }
  }

  String _buildRawTimelineTickLabel({
    required AuditLog log,
    required int index,
    required int total,
  }) {
    final l10n = AppLocalizations.of(context)!;
    if (index == 0) {
      return l10n.auditLogTimelineStart;
    }
    if (index == total - 1) {
      return l10n.auditLogTimelineEnd;
    }
    if (log.durationMs <= 0 || total <= 1) {
      return l10n.auditLogTimelineStep(index + 1);
    }
    final offsetMs = ((log.durationMs * index) / (total - 1)).round();
    if (offsetMs >= 1000) {
      final seconds = offsetMs / 1000;
      final secondsText = (seconds % 1 == 0)
          ? seconds.toStringAsFixed(0)
          : seconds.toStringAsFixed(1);
      return 'T+$secondsText s';
    }
    return 'T+$offsetMs ms';
  }

  /// Builds one timeline node for the raw message chain.
  Widget _buildConversationTimelineItem({
    required AuditLog log,
    required _RawMessageItem item,
    required int index,
    required int total,
  }) {
    final style = _rawTimelineStyleForRole(item.roleLabel);
    final isLast = index == total - 1;
    final tickLabel = _buildRawTimelineTickLabel(
      log: log,
      index: index,
      total: total,
    );

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      child: IntrinsicHeight(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            SizedBox(
              width: 88,
              child: Column(
                children: [
                  Text(
                    tickLabel,
                    style: AppFonts.firaCode(
                      fontSize: 10,
                      color: Colors.white54,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Container(
                    width: 22,
                    height: 22,
                    decoration: BoxDecoration(
                      color: style.color.withValues(alpha: 0.16),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: style.color.withValues(alpha: 0.42),
                      ),
                    ),
                    child: Icon(style.icon, size: 12, color: style.color),
                  ),
                  if (!isLast)
                    Expanded(
                      child: Container(
                        width: 2,
                        margin: const EdgeInsets.only(top: 4, bottom: 2),
                        decoration: BoxDecoration(
                          color: style.color.withValues(alpha: 0.22),
                          borderRadius: BorderRadius.circular(10),
                        ),
                      ),
                    ),
                ],
              ),
            ),
            Expanded(
              child: Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: Colors.black.withValues(alpha: 0.3),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: style.color.withValues(alpha: 0.24),
                  ),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Text(
                          item.roleLabel,
                          style: AppFonts.inter(
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                            color: style.color,
                          ),
                        ),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 6,
                            vertical: 1,
                          ),
                          decoration: BoxDecoration(
                            color: style.color.withValues(alpha: 0.14),
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(
                              color: style.color.withValues(alpha: 0.26),
                            ),
                          ),
                          child: Text(
                            '#${index + 1}',
                            style: AppFonts.firaCode(
                              fontSize: 10,
                              color: style.color,
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 6),
                    SelectableText(
                      item.content,
                      style: AppFonts.firaCode(
                        fontSize: 11,
                        color: Colors.white70,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildToolCallItem(AuditToolCall tc) {
    final l10n = AppLocalizations.of(context)!;
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: tc.isSensitive
            ? Colors.red.withValues(alpha: 0.1)
            : Colors.black.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(
          color: tc.isSensitive
              ? Colors.red.withValues(alpha: 0.3)
              : Colors.white.withValues(alpha: 0.1),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                LucideIcons.wrench,
                size: 14,
                color: tc.isSensitive ? Colors.red : const Color(0xFF6366F1),
              ),
              const SizedBox(width: 6),
              Text(
                tc.name,
                style: AppFonts.firaCode(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: Colors.white,
                ),
              ),
              if (tc.isSensitive) ...[
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 4,
                    vertical: 1,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.red.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    l10n.auditLogSensitive,
                    style: AppFonts.firaCode(fontSize: 9, color: Colors.red),
                  ),
                ),
              ],
            ],
          ),
          if (tc.arguments.isNotEmpty) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  '${l10n.auditLogToolArguments}:',
                  style: AppFonts.inter(fontSize: 10, color: Colors.white54),
                ),
                const SizedBox(width: 6),
                IconButton(
                  icon: const Icon(LucideIcons.copy, size: 14),
                  color: Colors.white54,
                  tooltip: l10n.appStoreGuideCopy,
                  onPressed: () => _copyText(tc.arguments),
                ),
              ],
            ),
            const SizedBox(height: 4),
            _buildCodeBlock(tc.arguments),
          ],
          if (tc.result != null && tc.result!.isNotEmpty) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  '${l10n.auditLogToolResult}:',
                  style: AppFonts.inter(fontSize: 10, color: Colors.white54),
                ),
                const SizedBox(width: 6),
                IconButton(
                  icon: const Icon(LucideIcons.copy, size: 14),
                  color: Colors.white54,
                  tooltip: l10n.appStoreGuideCopy,
                  onPressed: () => _copyText(tc.result!),
                ),
              ],
            ),
            const SizedBox(height: 4),
            _buildCodeBlock(tc.result!),
          ],
        ],
      ),
    );
  }
}

class _RawTimelineStyle {
  final Color color;
  final IconData icon;

  const _RawTimelineStyle({required this.color, required this.icon});
}
