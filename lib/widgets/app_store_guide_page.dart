import 'package:flutter/material.dart';
import '../utils/app_fonts.dart';
import '../l10n/app_localizations.dart';
import '../services/model_config_database_service.dart';
import '../models/llm_config_model.dart';
import '../services/protection_service.dart';
import '../services/protection_database_service.dart';
import '../utils/app_logger.dart';

/// App Store 版本引导页面（简化版）
/// HTTP 端点已移除,直接启动代理服务
class AppStoreGuidePage extends StatefulWidget {
  final VoidCallback onFinish;

  const AppStoreGuidePage({super.key, required this.onFinish});

  @override
  State<AppStoreGuidePage> createState() => _AppStoreGuidePageState();
}

class _AppStoreGuidePageState extends State<AppStoreGuidePage> {
  final ProtectionService _protectionService = ProtectionService();
  String? _error;
  bool _proxyStarted = false;

  @override
  void initState() {
    super.initState();
    _startProxy();
  }

  /// 启动代理服务
  Future<void> _startProxy() async {
    try {
      // 安全模型配置作为顶层参数传递给 Go，bot 模型由 Go 内部独立加载
      final dbService = ModelConfigDatabaseService();
      final securityModelConfig = await dbService.getSecurityModelConfig();

      if (securityModelConfig == null || !securityModelConfig.isValid) {
        if (!mounted) return;
        final l10n = AppLocalizations.of(context)!;
        if (mounted) {
          setState(() {
            _error = l10n.configureAiModelFirst;
          });
        }
        return;
      }

      final proxyResult = await _protectionService.startProtectionProxy(
        securityModelConfig,
        ProtectionRuntimeConfig(),
      );
      if (proxyResult['success'] != true) {
        if (mounted) {
          setState(() {
            _error = proxyResult['error']?.toString() ?? 'Proxy start failed';
          });
        }
        return;
      }

      await ProtectionDatabaseService().saveProtectionState(
        enabled: true,
        providerName: proxyResult['provider_name'],
        proxyPort: proxyResult['port'],
        originalBaseUrl: proxyResult['original_base_url'],
      );

      if (mounted) {
        setState(() {
          _proxyStarted = true;
        });
      }

      // 自动完成引导
      Future.delayed(const Duration(milliseconds: 500), () {
        if (mounted) {
          widget.onFinish();
        }
      });
    } catch (e) {
      appLogger.error('[AppStoreGuidePage] Failed to start proxy', e);
      if (mounted) {
        setState(() {
          _error = e.toString();
        });
      }
    }
  }

  @override
  void dispose() {
    _protectionService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.appStoreGuideTitle,
          style: AppFonts.inter(
            fontSize: 18,
            fontWeight: FontWeight.w600,
            color: Colors.white,
          ),
        ),
        const SizedBox(height: 8),
        Text(
          l10n.appStoreGuideDesc,
          style: AppFonts.inter(fontSize: 13, color: Colors.white70),
        ),
        const SizedBox(height: 24),
        if (_error != null)
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.red.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.red.withValues(alpha: 0.3)),
            ),
            child: Row(
              children: [
                const Icon(Icons.error_outline, color: Colors.red, size: 20),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    _error!,
                    style: AppFonts.inter(
                      fontSize: 12,
                      color: Colors.red.shade300,
                    ),
                  ),
                ),
              ],
            ),
          )
        else if (_proxyStarted)
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.green.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.green.withValues(alpha: 0.3)),
            ),
            child: Row(
              children: [
                const Icon(
                  Icons.check_circle_outline,
                  color: Colors.green,
                  size: 20,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    '防护代理已启动',
                    style: AppFonts.inter(
                      fontSize: 12,
                      color: Colors.green.shade300,
                    ),
                  ),
                ),
              ],
            ),
          )
        else
          Center(
            child: Column(
              children: [
                const CircularProgressIndicator(),
                const SizedBox(height: 16),
                Text(
                  '正在启动防护代理...',
                  style: AppFonts.inter(fontSize: 13, color: Colors.white70),
                ),
              ],
            ),
          ),
      ],
    );
  }
}
