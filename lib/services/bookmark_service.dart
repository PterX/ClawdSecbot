import 'dart:io';
import 'package:flutter/services.dart';
import '../config/build_config.dart';
import '../utils/app_logger.dart';

/// Service for managing Security-Scoped Bookmarks on macOS
/// Handles directory authorization for sandbox file access
class BookmarkService {
  static final BookmarkService _instance = BookmarkService._internal();
  factory BookmarkService() => _instance;
  BookmarkService._internal();

  static const _channel = MethodChannel(
    'com.clawdbot.guard/security_scoped_bookmark',
  );

  String? _authorizedPath;
  bool _isAccessing = false;

  /// Get the currently authorized path
  String? get authorizedPath => _authorizedPath;

  /// Check if we're currently accessing the authorized directory
  bool get isAccessing => _isAccessing;

  /// Check if we have a stored bookmark (persistent authorization)
  Future<bool> hasStoredBookmark() async {
    // Personal build or non-macOS platforms don't need bookmark
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return true;

    try {
      final result = await _channel.invokeMethod<bool>('hasStoredBookmark');
      return result ?? false;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error checking bookmark', e);
      return false;
    }
  }

  /// Get the path from stored bookmark (if valid)
  Future<String?> getBookmarkedPath() async {
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return null;

    try {
      final result = await _channel.invokeMethod<String?>('getBookmarkedPath');
      _authorizedPath = result;
      return result;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error getting bookmarked path', e);
      return null;
    }
  }

  /// Show directory picker and store the bookmark
  /// Returns the selected path or null if cancelled
  Future<String?> selectAndStoreDirectory() async {
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return null;

    try {
      final result = await _channel.invokeMethod<String?>(
        'selectAndStoreDirectory',
      );
      if (result != null) {
        _authorizedPath = result;
      }
      return result;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error selecting directory', e);
      return null;
    }
  }

  /// Start accessing the stored bookmarked directory
  /// Must be called before reading/writing files in the authorized directory
  Future<bool> startAccessingDirectory() async {
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return true;
    if (_isAccessing) return true;

    try {
      final result = await _channel.invokeMethod<bool>(
        'startAccessingDirectory',
      );
      _isAccessing = result ?? false;
      return _isAccessing;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error starting access', e);
      return false;
    }
  }

  /// Stop accessing the security-scoped resource
  /// Should be called when done with file operations
  Future<void> stopAccessingDirectory() async {
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return;
    if (!_isAccessing) return;

    try {
      await _channel.invokeMethod<bool>('stopAccessingDirectory');
      _isAccessing = false;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error stopping access', e);
    }
  }

  /// Clear the stored bookmark
  Future<void> clearBookmark() async {
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return;

    try {
      await _channel.invokeMethod<bool>('clearBookmark');
      _authorizedPath = null;
      _isAccessing = false;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error clearing bookmark', e);
    }
  }

  /// Find existing config directory in common locations
  /// Returns the first found directory path or null
  Future<String?> findConfigDirectory() async {
    // Personal build or non-macOS: check common paths directly
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) {
      // On non-macOS, check common paths directly
      final home = Platform.environment['HOME'] ??
          Platform.environment['USERPROFILE'] ??
          '';
      final paths = ['$home/.openclaw', '$home/.moltbot', '$home/.clawdbot'];

      for (final path in paths) {
        if (await Directory(path).exists()) {
          return path;
        }
      }
      return null;
    }

    try {
      final result = await _channel.invokeMethod<String?>(
        'findConfigDirectory',
      );
      return result;
    } on PlatformException catch (e) {
      appLogger.error('[Bookmark] Error finding config directory', e);
      return null;
    }
  }

  /// Initialize the service - check for stored bookmark and start accessing if available
  Future<bool> initialize() async {
    // Personal build or non-macOS: no bookmark needed
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) {
      appLogger.info('[Bookmark] 个人版构建,跳过 Bookmark 初始化');
      return true;
    }

    appLogger.info('[Bookmark] App Store 版,开始初始化 Bookmark 服务...');

    final hasBookmark = await hasStoredBookmark();
    appLogger.info('[Bookmark] 是否有存储的书签: $hasBookmark');

    if (hasBookmark) {
      _authorizedPath = await getBookmarkedPath();
      appLogger.info('[Bookmark] 获取到书签路径: $_authorizedPath');

      if (_authorizedPath != null) {
        final success = await startAccessingDirectory();
        appLogger.info('[Bookmark] 开始访问目录结果: $success');
        return success;
      }
    }

    appLogger.warning('[Bookmark] 初始化失败,没有有效的书签');
    return false;
  }

  /// Check if the config directory is accessible
  /// Returns true if we have valid authorization
  Future<bool> isConfigAccessible() async {
    // Personal build or non-macOS: always accessible
    if (!Platform.isMacOS || !BuildConfig.requiresDirectoryAuth) return true;

    final path = await getBookmarkedPath();
    return path != null;
  }

  /// Get the parent directory path (the .openclaw/.moltbot/.clawdbot folder)
  String? getConfigParentPath() {
    appLogger.debug('[Bookmark] 获取配置父路径,当前授权路径: $_authorizedPath');

    if (_authorizedPath == null) {
      appLogger.warning('[Bookmark] 配置父路径为空,未授权任何目录');
      return null;
    }

    // The authorized path could be the config dir itself or a parent
    // Check if it's one of the known config directories
    if (_authorizedPath!.endsWith('.openclaw') ||
        _authorizedPath!.endsWith('.moltbot') ||
        _authorizedPath!.endsWith('.clawdbot')) {
      appLogger.info('[Bookmark] 授权路径本身就是配置目录: $_authorizedPath');
      return _authorizedPath;
    }

    // Otherwise, check for config dirs within the authorized path
    final possibleConfigs = [
      '$_authorizedPath/.openclaw',
      '$_authorizedPath/.moltbot',
      '$_authorizedPath/.clawdbot',
    ];

    appLogger.debug('[Bookmark] 检查可能的配置目录: $possibleConfigs');

    for (final config in possibleConfigs) {
      final exists = Directory(config).existsSync();
      appLogger.debug('[Bookmark] 检查 $config: ${exists ? "存在" : "不存在"}');
      if (exists) {
        appLogger.info('[Bookmark] 找到配置目录: $config');
        return config;
      }
    }

    appLogger.info('[Bookmark] 未找到子配置目录,返回授权路径: $_authorizedPath');
    return _authorizedPath;
  }
}
