import 'package:flutter/foundation.dart';
import '../../models/llm_config_model.dart';

/// Protection BLoC State
class ProtectionState {
  final bool isRunning;
  final bool isLoading;
  final String? error;
  final int? proxyPort;
  final String? proxyUrl;
  final String? providerName;

  // Statistics
  final int analysisCount;
  final int blockedCount;
  final int warningCount;
  final int totalTokens;

  // Audit mode
  final bool auditOnly;

  const ProtectionState({
    this.isRunning = false,
    this.isLoading = false,
    this.error,
    this.proxyPort,
    this.proxyUrl,
    this.providerName,
    this.analysisCount = 0,
    this.blockedCount = 0,
    this.warningCount = 0,
    this.totalTokens = 0,
    this.auditOnly = false,
  });

  ProtectionState copyWith({
    bool? isRunning,
    bool? isLoading,
    String? error,
    int? proxyPort,
    String? proxyUrl,
    String? providerName,
    int? analysisCount,
    int? blockedCount,
    int? warningCount,
    int? totalTokens,
    bool? auditOnly,
  }) {
    return ProtectionState(
      isRunning: isRunning ?? this.isRunning,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      proxyPort: proxyPort ?? this.proxyPort,
      proxyUrl: proxyUrl ?? this.proxyUrl,
      providerName: providerName ?? this.providerName,
      analysisCount: analysisCount ?? this.analysisCount,
      blockedCount: blockedCount ?? this.blockedCount,
      warningCount: warningCount ?? this.warningCount,
      totalTokens: totalTokens ?? this.totalTokens,
      auditOnly: auditOnly ?? this.auditOnly,
    );
  }
}

/// Protection BLoC Events
abstract class ProtectionEvent {}

/// 启动防护事件
class StartProtection extends ProtectionEvent {
  final SecurityModelConfig securityConfig;
  final ProtectionRuntimeConfig runtimeConfig;
  final int? port;
  StartProtection(this.securityConfig, this.runtimeConfig, {this.port});
}

class StopProtection extends ProtectionEvent {}

/// 更新运行时配置事件
class UpdateConfig extends ProtectionEvent {
  final SecurityModelConfig securityConfig;
  final ProtectionRuntimeConfig runtimeConfig;
  UpdateConfig(this.securityConfig, this.runtimeConfig);
}

class SetAuditOnly extends ProtectionEvent {
  final bool auditOnly;
  SetAuditOnly(this.auditOnly);
}

class RefreshStatus extends ProtectionEvent {}

class ResetStatistics extends ProtectionEvent {}

/// Protection BLoC
class ProtectionBloc extends ChangeNotifier {
  final ProtectionState _state = const ProtectionState();

  ProtectionState get state => _state;

  void dispatch(ProtectionEvent event) {
    // Handle events
  }

  // Getters for convenience
  bool get isRunning => _state.isRunning;
  bool get isLoading => _state.isLoading;
  String? get error => _state.error;
}
