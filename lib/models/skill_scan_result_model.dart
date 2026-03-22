/// Skill AI 安全分析结果
class SkillAnalysisResult {
  /// 是否安全
  final bool safe;

  /// 风险等级 (low, medium, high, critical)
  final String riskLevel;

  /// 发现的问题列表
  final List<SkillSecurityIssue> issues;

  /// 分析摘要
  final String summary;

  /// 原始输出（可选）
  final String rawOutput;

  const SkillAnalysisResult({
    required this.safe,
    this.riskLevel = 'unknown',
    this.issues = const [],
    this.summary = '',
    this.rawOutput = '',
  });

  /// 从 JSON 创建
  factory SkillAnalysisResult.fromJson(Map<String, dynamic> json) {
    return SkillAnalysisResult(
      safe: json['safe'] ?? false,
      riskLevel: json['risk_level'] ?? 'unknown',
      issues:
          (json['issues'] as List<dynamic>?)
              ?.map(
                (e) => SkillSecurityIssue.fromJson(e as Map<String, dynamic>),
              )
              .toList() ??
          [],
      summary: json['summary'] ?? '',
      rawOutput: json['raw_output'] ?? '',
    );
  }

  /// 转换为 JSON
  Map<String, dynamic> toJson() {
    return {
      'safe': safe,
      'risk_level': riskLevel,
      'issues': issues.map((e) => e.toJson()).toList(),
      'summary': summary,
      if (rawOutput.isNotEmpty) 'raw_output': rawOutput,
    };
  }

  @override
  String toString() {
    return 'SkillAnalysisResult(safe: $safe, riskLevel: $riskLevel, issues: ${issues.length})';
  }
}

/// Skill 安全分析发现的问题
class SkillSecurityIssue {
  /// 问题类型
  final String type;

  /// 严重程度
  final String severity;

  /// 文件路径
  final String file;

  /// 问题描述
  final String description;

  /// 证据
  final String evidence;

  const SkillSecurityIssue({
    required this.type,
    this.severity = 'medium',
    this.file = '',
    this.description = '',
    this.evidence = '',
  });

  /// 从 JSON 创建
  factory SkillSecurityIssue.fromJson(Map<String, dynamic> json) {
    return SkillSecurityIssue(
      type: json['type'] ?? '',
      severity: json['severity'] ?? 'medium',
      file: json['file'] ?? '',
      description: json['description'] ?? '',
      evidence: json['evidence'] ?? '',
    );
  }

  /// 转换为 JSON
  Map<String, dynamic> toJson() {
    return {
      'type': type,
      'severity': severity,
      'file': file,
      'description': description,
      'evidence': evidence,
    };
  }

  @override
  String toString() {
    return 'SkillSecurityIssue(type: $type, severity: $severity, file: $file)';
  }
}
