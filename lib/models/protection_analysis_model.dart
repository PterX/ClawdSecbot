/// Risk level from protection agent analysis
enum RiskLevel {
  safe,
  suspicious,
  dangerous,
  critical;

  static RiskLevel fromString(String value) {
    switch (value.toUpperCase()) {
      case 'SAFE':
        return RiskLevel.safe;
      case 'SUSPICIOUS':
        return RiskLevel.suspicious;
      case 'DANGEROUS':
        return RiskLevel.dangerous;
      case 'CRITICAL':
        return RiskLevel.critical;
      default:
        return RiskLevel.safe;
    }
  }

  String get displayName {
    switch (this) {
      case RiskLevel.safe:
        return 'SAFE';
      case RiskLevel.suspicious:
        return 'SUSPICIOUS';
      case RiskLevel.dangerous:
        return 'DANGEROUS';
      case RiskLevel.critical:
        return 'CRITICAL';
    }
  }

  bool get shouldBlock =>
      this == RiskLevel.dangerous || this == RiskLevel.critical;
}

/// Suggested action from protection agent
enum SuggestedAction {
  allow,
  warn,
  block,
  hardBlock;

  static SuggestedAction fromString(String value) {
    switch (value.toUpperCase()) {
      case 'ALLOW':
        return SuggestedAction.allow;
      case 'WARN':
        return SuggestedAction.warn;
      case 'BLOCK':
        return SuggestedAction.block;
      case 'HARD_BLOCK':
        return SuggestedAction.hardBlock;
      default:
        return SuggestedAction.allow;
    }
  }

  String get displayName {
    switch (this) {
      case SuggestedAction.allow:
        return 'ALLOW';
      case SuggestedAction.warn:
        return 'WARN';
      case SuggestedAction.block:
        return 'BLOCK';
      case SuggestedAction.hardBlock:
        return 'HARD_BLOCK';
    }
  }

  bool get shouldBlock =>
      this == SuggestedAction.block || this == SuggestedAction.hardBlock;
}

/// Result from protection agent analysis
class ProtectionAnalysisResult {
  final RiskLevel riskLevel;
  final int confidence;
  final String reason;
  final String maliciousInstructionDetected;
  final SuggestedAction suggestedAction;
  final String traceableQuote;
  final DateTime timestamp;

  ProtectionAnalysisResult({
    required this.riskLevel,
    required this.confidence,
    required this.reason,
    required this.maliciousInstructionDetected,
    required this.suggestedAction,
    required this.traceableQuote,
    DateTime? timestamp,
  }) : timestamp = timestamp ?? DateTime.now();

  factory ProtectionAnalysisResult.fromJson(Map<String, dynamic> json) {
    return ProtectionAnalysisResult(
      riskLevel: RiskLevel.fromString(json['risk_level'] ?? 'SAFE'),
      confidence: json['confidence'] ?? 0,
      reason: json['reason'] ?? '',
      maliciousInstructionDetected:
          json['malicious_instruction_detected'] ?? '',
      suggestedAction: SuggestedAction.fromString(
        json['suggested_action'] ?? 'ALLOW',
      ),
      traceableQuote: json['traceable_quote'] ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'risk_level': riskLevel.displayName,
      'confidence': confidence,
      'reason': reason,
      'malicious_instruction_detected': maliciousInstructionDetected,
      'suggested_action': suggestedAction.displayName,
      'traceable_quote': traceableQuote,
    };
  }

  bool get shouldBlock => suggestedAction.shouldBlock;

  /// Create a safe result for when analysis is not needed
  factory ProtectionAnalysisResult.safe() {
    return ProtectionAnalysisResult(
      riskLevel: RiskLevel.safe,
      confidence: 100,
      reason: 'No analysis performed',
      maliciousInstructionDetected: '',
      suggestedAction: SuggestedAction.allow,
      traceableQuote: '',
    );
  }
}

/// Represents a single message in the conversation being monitored
class ConversationMessage {
  final String role; // "system", "user", "assistant", "tool"
  final String content;
  final dynamic toolCalls;
  final String? toolCallId;

  ConversationMessage({
    required this.role,
    required this.content,
    this.toolCalls,
    this.toolCallId,
  });

  factory ConversationMessage.fromJson(Map<String, dynamic> json) {
    return ConversationMessage(
      role: json['role'] ?? '',
      content: json['content'] ?? '',
      toolCalls: json['tool_calls'],
      toolCallId: json['tool_call_id'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'role': role,
      'content': content,
      if (toolCalls != null) 'tool_calls': toolCalls,
      if (toolCallId != null) 'tool_call_id': toolCallId,
    };
  }
}

/// Request for protection analysis
class ProtectionAnalysisRequest {
  final List<ConversationMessage> messages;
  final String originalUserTask;

  ProtectionAnalysisRequest({
    required this.messages,
    required this.originalUserTask,
  });

  Map<String, dynamic> toJson() {
    return {
      'messages': messages.map((m) => m.toJson()).toList(),
      'original_user_task': originalUserTask,
    };
  }
}

/// API monitoring metrics for a single request/response
class ApiMetrics {
  final int id;
  final DateTime timestamp;
  final int promptTokens;
  final int completionTokens;
  final int totalTokens;
  final int toolCallCount;
  final String model;
  final bool isBlocked;
  final String? riskLevel;

  ApiMetrics({
    this.id = 0,
    required this.timestamp,
    required this.promptTokens,
    required this.completionTokens,
    required this.totalTokens,
    required this.toolCallCount,
    required this.model,
    this.isBlocked = false,
    this.riskLevel,
  });

  factory ApiMetrics.fromJson(Map<String, dynamic> json) {
    return ApiMetrics(
      id: json['id'] ?? 0,
      timestamp: json['timestamp'] != null
          ? DateTime.parse(json['timestamp'])
          : DateTime.now(),
      promptTokens: json['prompt_tokens'] ?? 0,
      completionTokens: json['completion_tokens'] ?? 0,
      totalTokens: json['total_tokens'] ?? 0,
      toolCallCount: json['tool_call_count'] ?? 0,
      model: json['model'] ?? '',
      isBlocked: json['is_blocked'] ?? false,
      riskLevel: json['risk_level'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'timestamp': timestamp.toIso8601String(),
      'prompt_tokens': promptTokens,
      'completion_tokens': completionTokens,
      'total_tokens': totalTokens,
      'tool_call_count': toolCallCount,
      'model': model,
      'is_blocked': isBlocked,
      'risk_level': riskLevel,
    };
  }
}

/// Aggregated API statistics for display
class ApiStatistics {
  final int totalTokens;
  final int totalPromptTokens;
  final int totalCompletionTokens;
  final int totalToolCalls;
  final int requestCount;
  final int blockedCount;
  final List<TokenTrendPoint> tokenTrend;
  final List<ToolCallTrendPoint> toolCallTrend;

  ApiStatistics({
    required this.totalTokens,
    required this.totalPromptTokens,
    required this.totalCompletionTokens,
    required this.totalToolCalls,
    required this.requestCount,
    required this.blockedCount,
    required this.tokenTrend,
    required this.toolCallTrend,
  });

  factory ApiStatistics.empty() {
    return ApiStatistics(
      totalTokens: 0,
      totalPromptTokens: 0,
      totalCompletionTokens: 0,
      totalToolCalls: 0,
      requestCount: 0,
      blockedCount: 0,
      tokenTrend: [],
      toolCallTrend: [],
    );
  }
}

/// Point data for token trend chart
class TokenTrendPoint {
  final DateTime timestamp;
  final int tokens;
  final int promptTokens;
  final int completionTokens;

  TokenTrendPoint({
    required this.timestamp,
    required this.tokens,
    this.promptTokens = 0,
    this.completionTokens = 0,
  });
}

/// Point data for tool call trend chart
class ToolCallTrendPoint {
  final DateTime timestamp;
  final int count;

  ToolCallTrendPoint({required this.timestamp, required this.count});
}

/// Protection statistics for an asset (persisted in database)
class ProtectionStatistics {
  final String assetName;
  final int analysisCount;
  final int messageCount;
  final int warningCount;
  final int blockedCount;
  final int totalTokens;
  final int totalPromptTokens;
  final int totalCompletionTokens;
  final int totalToolCalls;
  final int requestCount;
  final int auditTokens;
  final int auditPromptTokens;
  final int auditCompletionTokens;
  final DateTime updatedAt;

  ProtectionStatistics({
    required this.assetName,
    required this.analysisCount,
    required this.messageCount,
    required this.warningCount,
    required this.blockedCount,
    required this.totalTokens,
    required this.totalPromptTokens,
    required this.totalCompletionTokens,
    required this.totalToolCalls,
    required this.requestCount,
    required this.auditTokens,
    required this.auditPromptTokens,
    required this.auditCompletionTokens,
    required this.updatedAt,
  });

  factory ProtectionStatistics.empty(String assetName) {
    return ProtectionStatistics(
      assetName: assetName,
      analysisCount: 0,
      messageCount: 0,
      warningCount: 0,
      blockedCount: 0,
      totalTokens: 0,
      totalPromptTokens: 0,
      totalCompletionTokens: 0,
      totalToolCalls: 0,
      requestCount: 0,
      auditTokens: 0,
      auditPromptTokens: 0,
      auditCompletionTokens: 0,
      updatedAt: DateTime.now(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'asset_name': assetName,
      'analysis_count': analysisCount,
      'message_count': messageCount,
      'warning_count': warningCount,
      'blocked_count': blockedCount,
      'total_tokens': totalTokens,
      'total_prompt_tokens': totalPromptTokens,
      'total_completion_tokens': totalCompletionTokens,
      'total_tool_calls': totalToolCalls,
      'request_count': requestCount,
      'audit_tokens': auditTokens,
      'audit_prompt_tokens': auditPromptTokens,
      'audit_completion_tokens': auditCompletionTokens,
      'updated_at': updatedAt.toIso8601String(),
    };
  }
}

/// 预计算样式的日志条目
/// 在添加日志时计算颜色,避免 ListView 渲染时重复计算
class LogEntry {
  final String text;
  final int color; // 使用 int 存储颜色值,避免依赖 flutter/material

  LogEntry(this.text) : color = _computeColor(text);

  static int _computeColor(String text) {
    if (text.contains('Error') ||
        text.contains('BLOCKED') ||
        text.contains('CRITICAL')) {
      return 0xFFEF4444; // Red
    } else if (text.contains('Warning') ||
        text.contains('DANGEROUS') ||
        text.contains('SUSPICIOUS')) {
      return 0xFFF59E0B; // Orange
    } else if (text.contains('SAFE') || text.contains('ALLOW')) {
      return 0xFF22C55E; // Green
    } else if (text.contains('[Protection Agent]')) {
      return 0xFF6366F1; // Purple
    }
    return 0xFFB3B3B3; // Default gray (Colors.white70 equivalent)
  }
}
