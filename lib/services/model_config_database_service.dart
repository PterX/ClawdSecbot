import '../models/llm_config_model.dart';
import '../utils/app_logger.dart';
import 'plugin_service.dart';

/// 模型配置 FFI 持久化门面：通过 FFI 委托 Go 层进行数据持久化，Flutter 不直接操作 DB。
/// 包括安全模型配置（全局唯一）和 Bot 模型配置（按资产名称关联）。
class ModelConfigDatabaseService {
  static final ModelConfigDatabaseService _instance =
      ModelConfigDatabaseService._internal();

  factory ModelConfigDatabaseService() => _instance;

  ModelConfigDatabaseService._internal();

  final PluginService _pluginService = PluginService();

  /// 保存安全模型配置（全局唯一），通过FFI调用Go层
  Future<bool> saveSecurityModelConfig(SecurityModelConfig config) async {
    try {
      final configMap = {
        'provider': config.provider,
        'endpoint': config.endpoint,
        'api_key': config.apiKey,
        'model': config.model,
        'secret_key': config.secretKey,
      };

      final result = await _pluginService.saveSecurityModelConfig(configMap);
      if (result['success'] == true) {
        appLogger.info('[ModelConfigDB] Security model config saved via FFI');
        return true;
      }
      appLogger.error(
        '[ModelConfigDB] Failed to save security model config: ${result['error']}',
      );
      return false;
    } catch (e) {
      appLogger.error(
        '[ModelConfigDB] Failed to save security model config',
        e,
      );
      return false;
    }
  }

  /// 获取安全模型配置，通过FFI调用Go层
  Future<SecurityModelConfig?> getSecurityModelConfig() async {
    try {
      final result = await _pluginService.getSecurityModelConfig();
      if (result['success'] != true) {
        appLogger.error(
          '[ModelConfigDB] Failed to get security model config: ${result['error']}',
        );
        return null;
      }

      final data = result['data'];
      if (data == null) return null;

      final configMap = Map<String, dynamic>.from(data as Map);
      return SecurityModelConfig.fromJson(configMap);
    } catch (e) {
      appLogger.error('[ModelConfigDB] Failed to get security model config', e);
      return null;
    }
  }

  /// 保存Bot模型配置（按资产名称关联），通过FFI调用Go层
  Future<bool> saveBotModelConfig(BotModelConfig config) async {
    try {
      final configMap = {
        'asset_name': config.assetName,
        'asset_id': config.assetID,
        'provider': config.provider,
        'base_url': config.baseUrl,
        'api_key': config.apiKey,
        'model': config.model,
        'secret_key': config.secretKey,
      };

      final result = await _pluginService.saveBotModelConfig(configMap);
      if (result['success'] == true) {
        appLogger.info(
          '[ModelConfigDB] Bot model config saved via FFI: asset=${config.assetName}/${config.assetID}',
        );
        return true;
      }
      appLogger.error(
        '[ModelConfigDB] Failed to save bot model config: ${result['error']}',
      );
      return false;
    } catch (e) {
      appLogger.error('[ModelConfigDB] Failed to save bot model config', e);
      return false;
    }
  }

  /// 获取指定资产的Bot模型配置，通过FFI调用Go层
  Future<BotModelConfig?> getBotModelConfig(
    String assetName, [
    String assetID = '',
  ]) async {
    try {
      final result = await _pluginService.getBotModelConfig(assetName, assetID);
      if (result['success'] != true) {
        appLogger.error(
          '[ModelConfigDB] Failed to get bot model config: ${result['error']}',
        );
        return null;
      }

      final data = result['data'];
      if (data == null) return null;

      final configMap = Map<String, dynamic>.from(data as Map);
      return BotModelConfig.fromJson(configMap);
    } catch (e) {
      appLogger.error('[ModelConfigDB] Failed to get bot model config', e);
      return null;
    }
  }

  /// 删除指定资产的Bot模型配置，通过FFI调用Go层
  Future<bool> deleteBotModelConfig(
    String assetName, [
    String assetID = '',
  ]) async {
    try {
      final result = await _pluginService.deleteBotModelConfig(
        assetName,
        assetID,
      );
      if (result['success'] == true) {
        appLogger.info(
          '[ModelConfigDB] Bot model config deleted via FFI: asset=$assetName/$assetID',
        );
        return true;
      }
      appLogger.error(
        '[ModelConfigDB] Failed to delete bot model config: ${result['error']}',
      );
      return false;
    } catch (e) {
      appLogger.error('[ModelConfigDB] Failed to delete bot model config', e);
      return false;
    }
  }

  /// 检查是否存在有效的安全模型配置
  Future<bool> hasValidSecurityModelConfig() async {
    final config = await getSecurityModelConfig();
    return config?.isValid ?? false;
  }

  /// 检查指定资产是否存在有效的Bot模型配置
  Future<bool> hasValidBotModelConfig(
    String assetName, [
    String assetID = '',
  ]) async {
    final config = await getBotModelConfig(assetName, assetID);
    return config?.isValid ?? false;
  }
}
