import 'dart:io';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:tray_manager/tray_manager.dart';
import '../../config/build_config.dart';
import '../../l10n/app_localizations.dart';
import '../../providers/locale_provider.dart';
import '../../utils/app_logger.dart';
import '../main_page.dart';

/// 托盘管理 Mixin
/// 负责系统托盘图标初始化、菜单构建和事件处理
mixin MainPageTrayMixin on State<MainPage>, TrayListener {
  // ============ 需要 MainPage 提供的状态和方法 ============
  bool get launchAtStartupEnabled;
  set launchAtStartupEnabled(bool value);

  void showAuditLogWindow();
  Future<void> toggleLaunchAtStartup();
  Future<void> showClearDataConfirmDialog();
  Future<void> showRestoreConfigConfirmDialog();
  Future<void> reauthorizeDirectory();
  Future<void> applyProtectionLanguage(String language);
  Future<void> showWindow();

  bool _trayAvailable = false;

  // ============ 托盘方法 ============

  /// 初始化系统托盘图标
  Future<void> initTray() async {
    try {
      final iconPath = Platform.isWindows
          ? 'images/tray_icon.ico'
          : 'images/tray_icon.png';
      appLogger.info('[Tray] Setting icon: $iconPath (platform=${Platform.operatingSystem})');

      final iconFile = File(iconPath);
      if (await iconFile.exists()) {
        appLogger.info('[Tray] Icon file found at resolved path: ${iconFile.absolute.path}');
      } else {
        appLogger.warning('[Tray] Icon file NOT found at: ${iconFile.absolute.path}');
      }

      await trayManager.setIcon(
        iconPath,
        isTemplate: Platform.isMacOS,
      );
      _trayAvailable = true;
      appLogger.info('[Tray] Icon set successfully, initializing menu');
      await updateTray();
    } catch (e) {
      _trayAvailable = false;
      appLogger.error('[Tray] Initialization error', e);
    }
  }

  /// 更新托盘菜单
  Future<void> updateTray() async {
    if (!_trayAvailable) return;
    final l10n = AppLocalizations.of(context);
    if (l10n == null) return;

    final currentLocale = Localizations.localeOf(context).languageCode;

    // setToolTip 在 Linux 上可能不被支持，添加异常处理
    if (!Platform.isLinux) {
      try {
        await trayManager.setToolTip(l10n.appTitle);
      } catch (e) {
        // 忽略 setToolTip 错误
      }
    }
    Menu menu = Menu(
      items: [
        MenuItem(key: 'show', label: l10n.showWindow),
        MenuItem(key: 'audit_log', label: l10n.auditLog),
        MenuItem.separator(),
        // 开机启动选项（仅非 App Store 版本）
        if (!BuildConfig.isAppStore)
          MenuItem.checkbox(
            key: 'launch_at_startup',
            label: l10n.launchAtStartup,
            checked: launchAtStartupEnabled,
          ),
        if (!BuildConfig.isAppStore) MenuItem.separator(),
        // 清空数据选项
        MenuItem(key: 'clear_data', label: l10n.clearData),
        // 恢复配置选项
        MenuItem(key: 'restore_config', label: l10n.restoreConfig),
        MenuItem.separator(),
        // 仅在 macOS App Store 版本显示重新授权选项
        if (Platform.isMacOS && BuildConfig.requiresDirectoryAuth)
          MenuItem(key: 'reauthorize', label: '重新授权目录'),
        if (Platform.isMacOS && BuildConfig.requiresDirectoryAuth)
          MenuItem.separator(),
        MenuItem.checkbox(
          key: 'lang_zh',
          label: '中文',
          checked: currentLocale == 'zh',
        ),
        MenuItem.checkbox(
          key: 'lang_en',
          label: 'English',
          checked: currentLocale == 'en',
        ),
        MenuItem.separator(),
        MenuItem(key: 'exit', label: l10n.exit),
      ],
    );
    await trayManager.setContextMenu(menu);
  }

  @override
  void onTrayIconMouseDown() {
    showWindow();
  }

  @override
  void onTrayIconRightMouseDown() {
    if (!_trayAvailable) return;
    trayManager.popUpContextMenu();
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) async {
    if (menuItem.key == 'show') {
      showWindow();
    } else if (menuItem.key == 'exit') {
      handleExitFromTray();
    } else if (menuItem.key == 'lang_zh') {
      await context.read<LocaleProvider>().setLocale(const Locale('zh'));
      await applyProtectionLanguage('zh');
    } else if (menuItem.key == 'lang_en') {
      await context.read<LocaleProvider>().setLocale(const Locale('en'));
      await applyProtectionLanguage('en');
    } else if (menuItem.key == 'audit_log') {
      showAuditLogWindow();
    } else if (menuItem.key == 'reauthorize') {
      reauthorizeDirectory();
    } else if (menuItem.key == 'launch_at_startup') {
      toggleLaunchAtStartup();
    } else if (menuItem.key == 'clear_data') {
      showClearDataConfirmDialog();
    } else if (menuItem.key == 'restore_config') {
      showRestoreConfigConfirmDialog();
    }
  }

  /// 处理托盘退出事件（需要 MainPage 实现具体逻辑）
  void handleExitFromTray();
}
