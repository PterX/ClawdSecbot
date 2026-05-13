import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:path_provider/path_provider.dart';
import 'package:window_manager/window_manager.dart';

import '../l10n/app_localizations.dart';
import '../pages/audit_log_page.dart';
import '../utils/app_fonts.dart';
import '../utils/app_logger.dart';
import '../utils/locale_utils.dart';
import '../utils/window_animation_helper.dart';
import '../widgets/hide_window_shortcut.dart';

const _appBackground = Color(0xFF0F0F23);

/// Audit Log Window App for multi-window support
class AuditLogWindowApp extends StatefulWidget {
  final String windowId;
  final String locale;
  final String initialAssetName;
  final String initialAssetID;

  const AuditLogWindowApp({
    super.key,
    required this.windowId,
    this.locale = 'en',
    this.initialAssetName = '',
    this.initialAssetID = '',
  });

  @override
  State<AuditLogWindowApp> createState() => _AuditLogWindowAppState();
}

class _AuditLogWindowAppState extends State<AuditLogWindowApp> {
  bool _isWindowShown = false;
  late String _locale;

  @override
  void initState() {
    super.initState();
    _locale = LocaleUtils.normalizeLanguageCode(widget.locale);
    _registerWindowMethodHandler();
    _showWindowAfterFirstFrame();
  }

  /// 注册与主窗口通信：接收主界面语言切换并更新 [MaterialApp.locale]。
  void _registerWindowMethodHandler() {
    WindowController.fromCurrentEngine().then((controller) {
      controller.setWindowMethodHandler((call) async {
        if (call.method == 'updateLanguage') {
          final language = LocaleUtils.normalizeLanguageCode(
            call.arguments?.toString(),
          );
          try {
            appLogger.info('[AuditLogWindow] Received updateLanguage: $language');
            if (!mounted) {
              return null;
            }
            setState(() {
              _locale = language;
            });
          } catch (e) {
            appLogger.error('[AuditLogWindow] Failed to update language', e);
          }
        }
        return null;
      });
    });
  }

  /// Show audit window after first frame to reduce startup flicker.
  void _showWindowAfterFirstFrame() {
    Future<void>(() async {
      await WidgetsBinding.instance.waitUntilFirstFrameRasterized;
      if (!mounted || _isWindowShown) return;
      _isWindowShown = true;
      await WindowAnimationHelper.showWithAnimation();
    });
  }

  Future<String?> _exportMarkdownDesktop({
    required String fileName,
    required String content,
  }) async {
    final outputPath = await _resolveExportPath(fileName: fileName);
    if (outputPath == null || outputPath.trim().isEmpty) {
      return null;
    }
    final file = File(outputPath);
    await file.writeAsString(
      content,
      encoding: const Utf8Codec(allowMalformed: true),
      flush: true,
    );
    return file.path;
  }

  Future<String?> _resolveExportPath({
    required String fileName,
  }) async {
    final l10n = lookupAppLocalizations(Locale(_locale));
    final normalizedFileName = _ensureMarkdownExtension(fileName.trim());
    try {
      final savePath = await FilePicker.platform.saveFile(
        dialogTitle: l10n.auditLogExportDialogTitle,
        fileName: normalizedFileName,
        type: FileType.custom,
        allowedExtensions: const ['md'],
      );
      if (savePath == null || savePath.trim().isEmpty) {
        return null;
      }
      return _ensureMarkdownExtension(savePath.trim());
    } catch (e, st) {
      appLogger.warning(
        '[AuditLogWindow] save dialog unavailable, fallback to local path: $e',
      );
      appLogger.debug('[AuditLogWindow] save dialog stacktrace: $st');
      final fallbackDir = await _resolveFallbackExportDirectory();
      if (fallbackDir == null) {
        return null;
      }
      return _ensureMarkdownExtension(
        '${fallbackDir.path}${Platform.pathSeparator}$normalizedFileName',
      );
    }
  }

  Future<Directory?> _resolveFallbackExportDirectory() async {
    try {
      final downloads = await getDownloadsDirectory();
      if (downloads != null) {
        await downloads.create(recursive: true);
        return downloads;
      }
    } catch (e) {
      appLogger.warning('[AuditLogWindow] getDownloadsDirectory failed: $e');
    }

    try {
      final home = Platform.environment['HOME'];
      if (home != null && home.trim().isNotEmpty) {
        final dir = Directory(home.trim());
        if (await dir.exists()) {
          return dir;
        }
      }
    } catch (e) {
      appLogger.warning('[AuditLogWindow] resolve HOME failed: $e');
    }

    try {
      final docs = await getApplicationDocumentsDirectory();
      await docs.create(recursive: true);
      return docs;
    } catch (e) {
      appLogger.error('[AuditLogWindow] resolve fallback directory failed', e);
      return null;
    }
  }

  String _ensureMarkdownExtension(String path) {
    if (path.toLowerCase().endsWith('.md')) {
      return path;
    }
    return '$path.md';
  }

  @override
  Widget build(BuildContext context) {
    final windowL10n = lookupAppLocalizations(Locale(_locale));
    return MaterialApp(
      title: windowL10n.auditLogTitle,
      debugShowCheckedModeBanner: false,
      locale: Locale(_locale),
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('zh'), Locale('en')],
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF6366F1),
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: _appBackground,
        textTheme: AppFonts.interTextTheme(ThemeData.dark().textTheme),
      ),
      shortcuts: {
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyW):
            const HideWindowIntent(),
      },
      actions: {
        HideWindowIntent: CallbackAction<HideWindowIntent>(
          onInvoke: (_) {
            WindowAnimationHelper.hideWithAnimation();
            return null;
          },
        ),
      },
      home: AuditLogPage(
        windowId: widget.windowId,
        initialAssetName: widget.initialAssetName,
        initialAssetID: widget.initialAssetID,
        onRequestStartDragging: () async {
          try {
            await windowManager.startDragging();
          } catch (_) {}
        },
        onRequestMinimize: () async {
          try {
            await windowManager.minimize();
          } catch (_) {}
        },
        onRequestToggleMaximize: () async {
          try {
            final maximized = await windowManager.isMaximized();
            if (maximized) {
              await windowManager.unmaximize();
            } else {
              await windowManager.maximize();
            }
          } catch (_) {}
        },
        onRequestClose: () async {
          await WindowAnimationHelper.hideWithAnimation();
        },
        onExportMarkdown: _exportMarkdownDesktop,
        initialMaximized: false,
      ),
    );
  }
}
