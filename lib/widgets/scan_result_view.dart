import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:lucide_icons/lucide_icons.dart';
import 'dart:io';
import '../l10n/app_localizations.dart';
import '../models/asset_model.dart';
import '../models/risk_model.dart';
import '../utils/app_fonts.dart';

/// 扫描结果展示组件
/// 用于展示扫描完成后的资产列表和风险信息
class ScanResultView extends StatelessWidget {
  final ScanResult result;
  final Set<String> protectedAssets;
  final bool isRestoringProtection;
  final VoidCallback onRescan;
  final VoidCallback onViewSkillScanResults;
  final void Function(Asset asset, {required bool isEditMode})
  onShowProtectionConfig;
  final void Function(Asset asset) onShowProtectionMonitor;
  final void Function(RiskInfo risk) onShowMitigation;

  const ScanResultView({
    super.key,
    required this.result,
    required this.protectedAssets,
    required this.isRestoringProtection,
    required this.onRescan,
    required this.onViewSkillScanResults,
    required this.onShowProtectionConfig,
    required this.onShowProtectionMonitor,
    required this.onShowMitigation,
  });

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return SingleChildScrollView(
      key: const ValueKey('completed'),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // 扫描完成标题
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: const Color(0xFF22C55E).withValues(alpha: 0.2),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: const Icon(
                  LucideIcons.checkCircle,
                  color: Color(0xFF22C55E),
                  size: 20,
                ),
              ),
              const SizedBox(width: 12),
              Text(
                l10n.scanComplete,
                style: AppFonts.inter(
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                  color: Colors.white,
                ),
              ),
              const Spacer(),
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: GestureDetector(
                  onTap: onViewSkillScanResults,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 6,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.white.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(
                          LucideIcons.fileSearch,
                          color: Colors.white70,
                          size: 14,
                        ),
                        const SizedBox(width: 6),
                        Text(
                          l10n.viewSkillScanResults,
                          style: AppFonts.inter(
                            fontSize: 12,
                            color: Colors.white70,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              MouseRegion(
                cursor: SystemMouseCursors.click,
                child: GestureDetector(
                  onTap: onRescan,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 6,
                    ),
                    decoration: BoxDecoration(
                      color: Colors.white.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(
                          LucideIcons.refreshCw,
                          color: Colors.white70,
                          size: 14,
                        ),
                        const SizedBox(width: 6),
                        Text(
                          l10n.rescan,
                          style: AppFonts.inter(
                            fontSize: 12,
                            color: Colors.white70,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),

          // 资产信息
          _buildSectionTitle(l10n.detectedAssets, LucideIcons.box),
          const SizedBox(height: 12),
          if (result.assets.isEmpty)
            Padding(
              padding: const EdgeInsets.only(left: 4),
              child: Text(
                l10n.notFound,
                style: AppFonts.inter(color: Colors.white54, fontSize: 13),
              ),
            )
          else
            ...result.assets.map(
              (asset) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _buildAssetCard(
                  context,
                  asset,
                  l10n,
                ).animate().fadeIn().slideX(begin: 0.1, end: 0),
              ),
            ),
          const SizedBox(height: 24),

          // 风险信息
          _buildSectionTitle(
            '${l10n.securityFindings} (${result.risks.length})',
            LucideIcons.alertTriangle,
          ),
          const SizedBox(height: 12),
          if (result.risks.isEmpty)
            _buildNoRisksCard(l10n)
          else
            ...result.risks.asMap().entries.map(
              (entry) => Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: _buildRiskCard(context, entry.value, l10n)
                    .animate(delay: (100 * entry.key).ms)
                    .fadeIn()
                    .slideX(begin: 0.1, end: 0),
              ),
            ),
        ],
      ),
    ).animate().fadeIn(duration: 400.ms);
  }

  Widget _buildSectionTitle(String title, IconData icon) {
    return Row(
      children: [
        Icon(icon, size: 16, color: const Color(0xFF6366F1)),
        const SizedBox(width: 8),
        Text(
          title,
          style: AppFonts.inter(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: Colors.white,
          ),
        ),
      ],
    );
  }

  Widget _buildAssetCard(
    BuildContext context,
    Asset asset,
    AppLocalizations l10n,
  ) {
    final isProtected = protectedAssets.contains(asset.id);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFF6366F1).withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: const Color(0xFF6366F1).withValues(alpha: 0.3),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: const Color(0xFF6366F1).withValues(alpha: 0.2),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(
                  asset.displaySections.isNotEmpty
                      ? LucideIcons.fileJson
                      : LucideIcons.package,
                  color: const Color(0xFF6366F1),
                  size: 18,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  asset.name,
                  style: AppFonts.inter(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: Colors.white,
                  ),
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  asset.type,
                  style: AppFonts.firaCode(fontSize: 10, color: Colors.white70),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (asset.version.isNotEmpty)
            _buildConfigRow(l10n.version, asset.version, Colors.white70),
          if (asset.ports.isNotEmpty)
            _buildConfigRow(l10n.port, asset.ports.join(', '), Colors.white70),
          if (asset.serviceName.isNotEmpty)
            _buildConfigRow(
              l10n.serviceName,
              asset.serviceName,
              Colors.white70,
            ),
          if (asset.processPaths.isNotEmpty)
            _buildConfigRow(
              l10n.processPaths,
              asset.processPaths.join(', '),
              Colors.white70,
            ),
          // Display structured config sections from the plugin
          if (asset.displaySections.isNotEmpty) ...[
            const SizedBox(height: 8),
            const Divider(color: Colors.white12),
            const SizedBox(height: 8),
            _buildDisplaySections(asset.displaySections),
          ],
          // 所有资产都显示防护按钮
          const SizedBox(height: 12),
          _buildProtectionButton(context, asset, isProtected, l10n),
        ],
      ),
    );
  }

  Widget _buildDisplaySections(List<DisplaySection> sections) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (int i = 0; i < sections.length; i++) ...[
          if (i > 0) const SizedBox(height: 12),
          _buildConfigSectionHeader(
            sections[i].title,
            _mapSectionIcon(sections[i].icon),
          ),
          const SizedBox(height: 8),
          for (final item in sections[i].items)
            _buildConfigDetailRow(
              item.label,
              item.value,
              _statusColor(item.status),
            ),
        ],
      ],
    );
  }

  IconData _mapSectionIcon(String iconName) {
    switch (iconName) {
      case 'globe':
        return LucideIcons.globe;
      case 'box':
        return LucideIcons.box;
      case 'file-text':
        return LucideIcons.fileText;
      case 'file':
        return LucideIcons.file;
      case 'shield':
        return LucideIcons.shield;
      case 'key':
        return LucideIcons.key;
      case 'lock':
        return LucideIcons.lock;
      case 'network':
        return LucideIcons.network;
      case 'settings':
        return LucideIcons.settings;
      default:
        return LucideIcons.info;
    }
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'safe':
        return const Color(0xFF22C55E);
      case 'danger':
        return const Color(0xFFEF4444);
      case 'warning':
        return const Color(0xFFF59E0B);
      case 'neutral':
      default:
        return Colors.white70;
    }
  }

  Widget _buildConfigSectionHeader(String title, IconData icon) {
    return Row(
      children: [
        Icon(icon, size: 12, color: Colors.white54),
        const SizedBox(width: 6),
        Text(
          title,
          style: AppFonts.inter(
            fontSize: 11,
            fontWeight: FontWeight.w600,
            color: Colors.white54,
          ),
        ),
      ],
    );
  }

  Widget _buildConfigDetailRow(String label, String value, Color valueColor) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: AppFonts.inter(fontSize: 11, color: Colors.white38),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: AppFonts.firaCode(fontSize: 11, color: valueColor),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildProtectionButton(
    BuildContext context,
    Asset asset,
    bool isProtected,
    AppLocalizations l10n,
  ) {
    if (isProtected) {
      // 已防护资产：显示防护监控和配置按钮
      final isLoading = isRestoringProtection;
      return Row(
        children: [
          MouseRegion(
            cursor: isLoading
                ? SystemMouseCursors.basic
                : SystemMouseCursors.click,
            child: GestureDetector(
              onTap: isLoading ? null : () => onShowProtectionMonitor(asset),
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 8,
                ),
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    colors: isLoading
                        ? [
                            const Color(0xFF22C55E).withValues(alpha: 0.5),
                            const Color(0xFF16A34A).withValues(alpha: 0.5),
                          ]
                        : [const Color(0xFF22C55E), const Color(0xFF16A34A)],
                  ),
                  borderRadius: BorderRadius.circular(8),
                  boxShadow: isLoading
                      ? []
                      : [
                          BoxShadow(
                            color: const Color(
                              0xFF22C55E,
                            ).withValues(alpha: 0.3),
                            blurRadius: 8,
                            offset: const Offset(0, 2),
                          ),
                        ],
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (isLoading)
                      const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                          color: Colors.white,
                          strokeWidth: 2,
                        ),
                      )
                    else
                      const Icon(
                        LucideIcons.shieldCheck,
                        color: Colors.white,
                        size: 14,
                      ),
                    const SizedBox(width: 6),
                    Text(
                      isLoading
                          ? l10n.protectionStarting
                          : l10n.protectionMonitor,
                      style: AppFonts.inter(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: Colors.white,
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(width: 8),
          // 配置按钮（恢复防护期间禁用）
          MouseRegion(
            cursor: isLoading
                ? SystemMouseCursors.basic
                : SystemMouseCursors.click,
            child: GestureDetector(
              onTap: isLoading
                  ? null
                  : () => onShowProtectionConfig(asset, isEditMode: true),
              child: Opacity(
                opacity: isLoading ? 0.4 : 1.0,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(
                      color: Colors.white.withValues(alpha: 0.2),
                    ),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(
                        LucideIcons.settings,
                        color: Colors.white70,
                        size: 14,
                      ),
                      const SizedBox(width: 4),
                      Text(
                        l10n.protectionConfigBtn,
                        style: AppFonts.inter(
                          fontSize: 12,
                          color: Colors.white70,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ],
      );
    } else {
      // 未防护资产：显示一键防护按钮
      return MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          onTap: () => onShowProtectionConfig(asset, isEditMode: false),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFF6366F1), Color(0xFF8B5CF6)],
              ),
              borderRadius: BorderRadius.circular(8),
              boxShadow: [
                BoxShadow(
                  color: const Color(0xFF6366F1).withValues(alpha: 0.3),
                  blurRadius: 8,
                  offset: const Offset(0, 2),
                ),
              ],
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(LucideIcons.shield, color: Colors.white, size: 14),
                const SizedBox(width: 6),
                Text(
                  l10n.oneClickProtection,
                  style: AppFonts.inter(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: Colors.white,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }
  }

  Widget _buildConfigRow(String label, String value, Color valueColor) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 80,
          child: Text(
            label,
            style: AppFonts.inter(fontSize: 12, color: Colors.white54),
          ),
        ),
        Expanded(
          child: Text(
            value,
            style: AppFonts.firaCode(fontSize: 12, color: valueColor),
          ),
        ),
      ],
    );
  }

  Widget _buildNoRisksCard(AppLocalizations l10n) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFF22C55E).withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: const Color(0xFF22C55E).withValues(alpha: 0.3),
        ),
      ),
      child: Row(
        children: [
          const Icon(
            LucideIcons.shieldCheck,
            color: Color(0xFF22C55E),
            size: 24,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.noSecurityIssues,
                  style: AppFonts.inter(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: const Color(0xFF22C55E),
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.secureConfigMessage,
                  style: AppFonts.inter(fontSize: 12, color: Colors.white54),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildRiskCard(
    BuildContext context,
    RiskInfo risk,
    AppLocalizations l10n,
  ) {
    final title = _getRiskTitle(risk, l10n);
    final description = _getRiskDesc(risk, l10n);
    final levelText = _getRiskLevel(risk.level, l10n);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: risk.color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: risk.color.withValues(alpha: 0.3)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: risk.color.withValues(alpha: 0.2),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(risk.icon, color: risk.color, size: 18),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        title,
                        style: AppFonts.inter(
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                          color: Colors.white,
                        ),
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 2,
                      ),
                      decoration: BoxDecoration(
                        color: risk.color.withValues(alpha: 0.2),
                        borderRadius: BorderRadius.circular(4),
                      ),
                      child: Text(
                        levelText,
                        style: AppFonts.inter(
                          fontSize: 10,
                          fontWeight: FontWeight.w600,
                          color: risk.color,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 6),
                Text(
                  description,
                  style: AppFonts.inter(
                    fontSize: 12,
                    color: Colors.white70,
                    height: 1.4,
                  ),
                ),
                const SizedBox(height: 12),
                if (risk.mitigation != null)
                  Align(
                    alignment: Alignment.centerLeft,
                    child: ElevatedButton.icon(
                      onPressed: () => onShowMitigation(risk),
                      icon: const Icon(LucideIcons.wrench, size: 14),
                      label: Text(l10n.mitigate),
                      style: ElevatedButton.styleFrom(
                        backgroundColor: const Color(0xFF6366F1),
                        foregroundColor: Colors.white,
                        textStyle: AppFonts.inter(fontSize: 12),
                        padding: const EdgeInsets.symmetric(
                          horizontal: 12,
                          vertical: 8,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _getRiskTitle(RiskInfo risk, AppLocalizations l10n) {
    switch (risk.id) {
      case 'riskNonLoopbackBinding':
      case 'gateway_bind_unsafe':
        return l10n.riskNonLoopbackBinding;
      case 'riskNoAuth':
      case 'gateway_auth_disabled':
        return l10n.riskNoAuth;
      case 'riskWeakPassword':
      case 'gateway_weak_password':
        return l10n.riskWeakPassword;
      case 'riskAllPluginsAllowed':
        return l10n.riskAllPluginsAllowed;
      case 'riskControlUiEnabled':
        return l10n.riskControlUiEnabled;
      case 'riskRunningAsRoot':
        return l10n.riskRunningAsRoot;
      case 'config_perm_unsafe':
        return l10n.riskConfigPermUnsafe;
      case 'config_dir_perm_unsafe':
        return l10n.riskConfigDirPermUnsafe;
      case 'sandbox_disabled_default':
        return l10n.riskSandboxDisabledDefault;
      case 'sandbox_disabled_agent':
        return l10n.riskSandboxDisabledAgent;
      case 'logging_redact_off':
        return l10n.riskLoggingRedactOff;
      case 'log_dir_perm_unsafe':
        return l10n.riskLogDirPermUnsafe;
      case 'plaintext_secrets':
        return l10n.riskPlaintextSecrets;
      case 'skills_not_scanned':
        return l10n.riskSkillsNotScanned;
      case 'openclaw_1click_rce_vulnerability':
        return l10n.riskOneClickRce;
      default:
        return risk.title;
    }
  }

  String _getRiskDesc(RiskInfo risk, AppLocalizations l10n) {
    switch (risk.id) {
      case 'riskNonLoopbackBinding':
      case 'gateway_bind_unsafe':
        return l10n.riskNonLoopbackBindingDesc(
          risk.args?['bind']?.toString() ?? '',
        );
      case 'riskNoAuth':
      case 'gateway_auth_disabled':
        return l10n.riskNoAuthDesc;
      case 'riskWeakPassword':
      case 'gateway_weak_password':
        return l10n.riskWeakPasswordDesc;
      case 'riskAllPluginsAllowed':
        return l10n.riskAllPluginsAllowedDesc;
      case 'riskControlUiEnabled':
        return l10n.riskControlUiEnabledDesc;
      case 'riskRunningAsRoot':
        return l10n.riskRunningAsRootDesc;
      case 'config_perm_unsafe':
        if (Platform.isWindows) {
          return _getWindowsAclRiskDesc(
            risk,
            l10n: l10n,
            fallback: risk.description,
            defaultLabelEn: 'Config file ACL',
            defaultLabelZh: '\u914d\u7f6e\u6587\u4ef6 ACL',
          );
        }
        return l10n.riskConfigPermUnsafeDesc(
          risk.args?['path']?.toString() ?? '',
          risk.args?['current']?.toString() ?? '',
        );
      case 'config_dir_perm_unsafe':
        if (Platform.isWindows) {
          return _getWindowsAclRiskDesc(
            risk,
            l10n: l10n,
            fallback: risk.description,
            defaultLabelEn: 'Config directory ACL',
            defaultLabelZh: '\u914d\u7f6e\u76ee\u5f55 ACL',
          );
        }
        return l10n.riskConfigDirPermUnsafeDesc(
          risk.args?['path']?.toString() ?? '',
          risk.args?['current']?.toString() ?? '',
        );
      case 'sandbox_disabled_default':
        return l10n.riskSandboxDisabledDefaultDesc;
      case 'sandbox_disabled_agent':
        return l10n.riskSandboxDisabledAgentDesc(
          risk.args?['agent']?.toString() ?? '',
        );
      case 'logging_redact_off':
        return l10n.riskLoggingRedactOffDesc;
      case 'log_dir_perm_unsafe':
        if (Platform.isWindows) {
          return _getWindowsAclRiskDesc(
            risk,
            l10n: l10n,
            fallback: risk.description,
            defaultLabelEn: 'Log directory ACL',
            defaultLabelZh: '\u65e5\u5fd7\u76ee\u5f55 ACL',
          );
        }
        return l10n.riskLogDirPermUnsafeDesc;
      case 'plaintext_secrets':
        return l10n.riskPlaintextSecretsDesc(
          risk.args?['pattern']?.toString() ?? '',
        );
      case 'skills_not_scanned':
        return l10n.riskSkillsNotScannedDesc(
          risk.args?['count'] as int? ?? 0,
          risk.args?['skills']?.toString() ?? '',
        );
      case 'openclaw_1click_rce_vulnerability':
        return l10n.riskOneClickRceDesc(
          risk.args?['current_version']?.toString() ?? 'unknown',
        );
      default:
        return risk.description;
    }
  }

  String _getWindowsAclRiskDesc(
    RiskInfo risk, {
    required AppLocalizations l10n,
    required String fallback,
    required String defaultLabelEn,
    required String defaultLabelZh,
  }) {
    final args = risk.args;
    if (args == null) return fallback;
    final isZh = l10n.localeName.toLowerCase().startsWith('zh');

    final path = args['path']?.toString() ?? '';
    final summaryRaw = args['acl_summary']?.toString() ?? '';
    final summary = _translateAclSummary(summaryRaw, isZh);
    final violationsRaw = args['acl_violations'];

    String violations = '';
    if (violationsRaw is List) {
      violations = violationsRaw
          .map((e) => e.toString())
          .where((e) => e.isNotEmpty)
          .join('; ');
    } else {
      violations = violationsRaw?.toString() ?? '';
    }

    final details = <String>[];
    if (path.isNotEmpty)
      details.add(isZh ? '\u8def\u5f84: $path' : 'Path: $path');
    if (summary.isNotEmpty) {
      details.add(isZh ? '\u6458\u8981: $summary' : 'Summary: $summary');
    }
    if (violations.isNotEmpty) {
      details.add(
        isZh
            ? '\u8fdd\u89c4\u4e3b\u4f53: $violations'
            : 'Violations: $violations',
      );
    }

    if (details.isEmpty) return fallback;
    if (isZh) {
      return '$defaultLabelZh \u6743\u9650\u4e0d\u5b89\u5168\u3002${details.join('\uff1b')}\u3002';
    }
    return '$defaultLabelEn is unsafe. ${details.join(' | ')}.';
  }

  String _translateAclSummary(String summary, bool isZh) {
    if (!isZh) return summary;
    switch (summary.toLowerCase()) {
      case 'acl safe':
        return 'ACL \u5b89\u5168';
      case 'acl has non-whitelisted principal access':
        return '\u5b58\u5728\u975e\u767d\u540d\u5355\u4e3b\u4f53\u8bbf\u95ee\u6743\u9650';
      case 'acl check failed':
        return 'ACL \u68c0\u67e5\u5931\u8d25';
      default:
        return summary;
    }
  }

  String _getRiskLevel(RiskLevel level, AppLocalizations l10n) {
    switch (level) {
      case RiskLevel.low:
        return l10n.riskLevelLow;
      case RiskLevel.medium:
        return l10n.riskLevelMedium;
      case RiskLevel.high:
        return l10n.riskLevelHigh;
      case RiskLevel.critical:
        return l10n.riskLevelCritical;
    }
  }
}
