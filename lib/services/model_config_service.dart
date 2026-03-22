import 'dart:convert';
import 'dart:ffi' as ffi;
import 'package:ffi/ffi.dart';
import 'package:flutter/foundation.dart';
import '../models/llm_config_model.dart';
import '../utils/app_logger.dart';
import 'model_config_database_service.dart';
import 'native_library_service.dart';

typedef TestModelConnectionFFIC =
    ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8> configJSON);
typedef TestModelConnectionFFIDart =
    ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8> configJSON);

typedef FreeStringC = ffi.Void Function(ffi.Pointer<Utf8>);
typedef FreeStringDart = void Function(ffi.Pointer<Utf8>);

/// Payload for background isolate test connection.
class _TestConnectionPayload {
  final Map<String, String> config;
  final String? libraryPath;

  _TestConnectionPayload({required this.config, this.libraryPath});
}

/// Tests model connection in a background isolate.
/// 后台 Isolate 必须通过 DynamicLibrary.open() 重新打开动态库
Future<Map<String, dynamic>> _testConnectionInIsolate(
  _TestConnectionPayload payload,
) async {
  try {
    // 后台 Isolate 中重新打开动态库
    final libPath = payload.libraryPath;
    if (libPath == null || libPath.isEmpty) {
      return {'success': false, 'error': 'Library path not available'};
    }

    final dylib = ffi.DynamicLibrary.open(libPath);
    final testModelConnection = dylib
        .lookupFunction<TestModelConnectionFFIC, TestModelConnectionFFIDart>(
          'TestModelConnectionFFI',
        );
    final freeString = dylib.lookupFunction<FreeStringC, FreeStringDart>(
      'FreeString',
    );

    final testConfig = {
      'provider': payload.config['type'] ?? payload.config['provider'] ?? '',
      'endpoint': payload.config['endpoint'] ?? '',
      'api_key': payload.config['apiKey'] ?? '',
      'model': payload.config['model'] ?? '',
      if (payload.config['secretKey']?.isNotEmpty == true)
        'secret_key': payload.config['secretKey']!,
    };

    final configJSON = jsonEncode(testConfig);
    final configPtr = configJSON.toNativeUtf8();
    final resultPtr = testModelConnection(configPtr);
    malloc.free(configPtr);

    final resultStr = resultPtr.toDartString();
    freeString(resultPtr);

    return jsonDecode(resultStr) as Map<String, dynamic>;
  } catch (e) {
    return {'success': false, 'error': e.toString()};
  }
}

/// 安全模型配置服务
/// 用于 ShepherdGate 风险检测的模型配置管理
class SecurityModelConfigService {
  static final SecurityModelConfigService _instance =
      SecurityModelConfigService._internal();

  factory SecurityModelConfigService() => _instance;

  SecurityModelConfigService._internal();

  /// 加载安全模型配置
  Future<SecurityModelConfig> loadConfig() async {
    try {
      final dbService = ModelConfigDatabaseService();
      final config = await dbService.getSecurityModelConfig();

      if (config != null) {
        return config;
      }
    } catch (e) {
      appLogger.error('[SecurityModelConfig] Failed to load config', e);
    }
    // 返回默认配置
    return SecurityModelConfig(
      provider: 'ollama',
      endpoint: 'http://localhost:11434',
      apiKey: '',
      model: 'llama3',
    );
  }

  /// 保存安全模型配置
  Future<bool> saveConfig(SecurityModelConfig config) async {
    try {
      final dbService = ModelConfigDatabaseService();
      return await dbService.saveSecurityModelConfig(config);
    } catch (e) {
      appLogger.error('[SecurityModelConfig] Failed to save config', e);
      return false;
    }
  }

  /// 测试安全模型连接
  Future<Map<String, dynamic>> testConnection(
    SecurityModelConfig config,
  ) async {
    final configMap = {
      'provider': config.provider,
      'endpoint': config.endpoint,
      'apiKey': config.apiKey,
      'model': config.model,
      'secretKey': config.secretKey,
    };

    final libPath = NativeLibraryService().libraryPath;
    if (libPath == null || libPath.isEmpty) {
      return {'success': false, 'error': 'Native library not initialized'};
    }

    return await compute(
      _testConnectionInIsolate,
      _TestConnectionPayload(config: configMap, libraryPath: libPath),
    );
  }

  /// 检查是否存在有效配置
  Future<bool> hasValidConfig() async {
    final dbService = ModelConfigDatabaseService();
    return await dbService.hasValidSecurityModelConfig();
  }
}

/// Bot 模型配置服务
/// 仅用于写入 openclaw.json,记录被代理的 LLM 信息
class BotModelConfigService {
  BotModelConfigService({required this.assetName, this.assetID = ''});

  final String assetName;
  final String assetID;

  /// 加载 Bot 模型配置
  Future<BotModelConfig?> loadConfig() async {
    try {
      final dbService = ModelConfigDatabaseService();
      return await dbService.getBotModelConfig(assetName, assetID);
    } catch (e) {
      appLogger.error('[BotModelConfig] Failed to load config', e);
      return null;
    }
  }

  /// 保存 Bot 模型配置
  Future<bool> saveConfig(BotModelConfig config) async {
    try {
      final dbService = ModelConfigDatabaseService();
      return await dbService.saveBotModelConfig(config);
    } catch (e) {
      appLogger.error('[BotModelConfig] Failed to save config', e);
      return false;
    }
  }

  /// 删除 Bot 模型配置
  Future<bool> deleteConfig() async {
    try {
      final dbService = ModelConfigDatabaseService();
      return await dbService.deleteBotModelConfig(assetName, assetID);
    } catch (e) {
      appLogger.error('[BotModelConfig] Failed to delete config', e);
      return false;
    }
  }

  /// 测试 Bot 模型连接
  Future<Map<String, dynamic>> testConnection(BotModelConfig config) async {
    final configMap = {
      'provider': config.provider,
      'endpoint': config.baseUrl,
      'apiKey': config.apiKey,
      'model': config.model,
      'secretKey': config.secretKey,
    };

    final libPath = NativeLibraryService().libraryPath;
    if (libPath == null || libPath.isEmpty) {
      return {'success': false, 'error': 'Native library not initialized'};
    }

    return await compute(
      _testConnectionInIsolate,
      _TestConnectionPayload(config: configMap, libraryPath: libPath),
    );
  }

  /// 检查是否存在有效配置
  Future<bool> hasValidConfig() async {
    final dbService = ModelConfigDatabaseService();
    return await dbService.hasValidBotModelConfig(assetName, assetID);
  }
}
