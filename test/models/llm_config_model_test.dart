import 'package:bot_sec_manager/models/llm_config_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SecurityModelConfig', () {
    test('serializes to and from JSON round-trip', () {
      final original = SecurityModelConfig(
        provider: 'openai',
        endpoint: 'https://api.openai.com',
        apiKey: 'sk-test-123',
        model: 'gpt-4',
        secretKey: 'secret-abc',
      );

      final json = original.toJson();
      final restored = SecurityModelConfig.fromJson(json);

      expect(restored.provider, 'openai');
      expect(restored.endpoint, 'https://api.openai.com');
      expect(restored.apiKey, 'sk-test-123');
      expect(restored.model, 'gpt-4');
      expect(restored.secretKey, 'secret-abc');
    });

    test('secretKey not serialized when empty', () {
      final config = SecurityModelConfig(
        provider: 'openai', endpoint: '', apiKey: '', model: 'gpt-4',
      );
      final json = config.toJson();
      expect(json.containsKey('secret_key'), false);
    });

    test('isValid returns true when provider and model are non-empty', () {
      expect(
        SecurityModelConfig(provider: 'openai', endpoint: '', apiKey: '', model: 'gpt-4').isValid,
        true,
      );
      expect(
        SecurityModelConfig(provider: '', endpoint: '', apiKey: '', model: 'gpt-4').isValid,
        false,
      );
      expect(
        SecurityModelConfig(provider: 'openai', endpoint: '', apiKey: '', model: '').isValid,
        false,
      );
    });

    test('copyWith creates new instance with overridden fields', () {
      final original = SecurityModelConfig(
        provider: 'openai', endpoint: 'ep', apiKey: 'key', model: 'gpt-4',
      );
      final copy = original.copyWith(model: 'gpt-4o');
      expect(copy.model, 'gpt-4o');
      expect(copy.provider, 'openai');
      expect(identical(copy, original), false);
    });

    test('equality compares all fields', () {
      final a = SecurityModelConfig(
        provider: 'openai', endpoint: 'ep', apiKey: 'key', model: 'gpt-4',
      );
      final b = SecurityModelConfig(
        provider: 'openai', endpoint: 'ep', apiKey: 'key', model: 'gpt-4',
      );
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });

    test('fromJson handles legacy "type" field as provider', () {
      final json = {'type': 'anthropic', 'model': 'claude-3'};
      final config = SecurityModelConfig.fromJson(json);
      expect(config.provider, 'anthropic');
    });
  });

  group('BotModelConfig', () {
    test('serializes to and from JSON round-trip', () {
      final original = BotModelConfig(
        assetName: 'openclaw',
        assetID: 'openclaw:abc',
        provider: 'openai',
        baseUrl: 'https://api.openai.com',
        apiKey: 'sk-test',
        model: 'gpt-4',
        secretKey: 'secret',
      );

      final json = original.toJson();
      final restored = BotModelConfig.fromJson(json);

      expect(restored.assetName, 'openclaw');
      expect(restored.assetID, 'openclaw:abc');
      expect(restored.provider, 'openai');
      expect(restored.baseUrl, 'https://api.openai.com');
      expect(restored.apiKey, 'sk-test');
      expect(restored.model, 'gpt-4');
      expect(restored.secretKey, 'secret');
    });

    test('isValid checks assetName, provider, baseUrl', () {
      expect(
        BotModelConfig(assetName: 'a', provider: 'p', baseUrl: 'b', apiKey: '', model: '').isValid,
        true,
      );
      expect(
        BotModelConfig(assetName: '', provider: 'p', baseUrl: 'b', apiKey: '', model: '').isValid,
        false,
      );
      expect(
        BotModelConfig(assetName: 'a', provider: '', baseUrl: 'b', apiKey: '', model: '').isValid,
        false,
      );
      expect(
        BotModelConfig(assetName: 'a', provider: 'p', baseUrl: '', apiKey: '', model: '').isValid,
        false,
      );
    });

    test('fromJson handles legacy "endpoint" as base_url fallback', () {
      final json = {'asset_name': 'a', 'provider': 'p', 'endpoint': 'https://ep'};
      final config = BotModelConfig.fromJson(json);
      expect(config.baseUrl, 'https://ep');
    });

    test('fromJson handles legacy "type" as provider fallback', () {
      final json = {'asset_name': 'a', 'type': 'anthropic', 'base_url': 'b'};
      final config = BotModelConfig.fromJson(json);
      expect(config.provider, 'anthropic');
    });

    test('copyWith creates new instance with overridden fields', () {
      final original = BotModelConfig(
        assetName: 'a', provider: 'p', baseUrl: 'b', apiKey: 'k', model: 'm',
      );
      final copy = original.copyWith(model: 'm2', assetID: 'id1');
      expect(copy.model, 'm2');
      expect(copy.assetID, 'id1');
      expect(copy.assetName, 'a');
    });
  });

  group('ProtectionRuntimeConfig', () {
    test('serializes to and from JSON round-trip', () {
      final original = ProtectionRuntimeConfig(
        auditOnly: true,
        singleSessionTokenLimit: 10000,
        dailyTokenLimit: 100000,
        initialDailyTokenUsage: 5000,
      );

      final json = original.toJson();
      final restored = ProtectionRuntimeConfig.fromJson(json);

      expect(restored.auditOnly, true);
      expect(restored.singleSessionTokenLimit, 10000);
      expect(restored.dailyTokenLimit, 100000);
      expect(restored.initialDailyTokenUsage, 5000);
    });

    test('defaults to safe values', () {
      final config = ProtectionRuntimeConfig.fromJson({});
      expect(config.auditOnly, false);
      expect(config.singleSessionTokenLimit, 0);
      expect(config.dailyTokenLimit, 0);
      expect(config.initialDailyTokenUsage, 0);
    });

    test('copyWith creates new instance', () {
      final original = ProtectionRuntimeConfig(auditOnly: false, singleSessionTokenLimit: 100);
      final copy = original.copyWith(auditOnly: true);
      expect(copy.auditOnly, true);
      expect(copy.singleSessionTokenLimit, 100);
    });
  });

  group('ProtectionBaselineStatistics', () {
    test('serializes to and from JSON round-trip', () {
      final original = ProtectionBaselineStatistics(
        analysisCount: 50,
        blockedCount: 5,
        warningCount: 10,
        totalTokens: 100000,
        totalPromptTokens: 70000,
        totalCompletionTokens: 30000,
        totalToolCalls: 200,
        requestCount: 150,
        auditTokens: 20000,
        auditPromptTokens: 15000,
        auditCompletionTokens: 5000,
      );

      final json = original.toJson();
      final restored = ProtectionBaselineStatistics.fromJson(json);

      expect(restored.analysisCount, 50);
      expect(restored.blockedCount, 5);
      expect(restored.warningCount, 10);
      expect(restored.totalTokens, 100000);
      expect(restored.totalPromptTokens, 70000);
      expect(restored.totalCompletionTokens, 30000);
      expect(restored.totalToolCalls, 200);
      expect(restored.requestCount, 150);
      expect(restored.auditTokens, 20000);
      expect(restored.auditPromptTokens, 15000);
      expect(restored.auditCompletionTokens, 5000);
    });

    test('defaults to all zeros', () {
      final stats = ProtectionBaselineStatistics.fromJson({});
      expect(stats.analysisCount, 0);
      expect(stats.blockedCount, 0);
      expect(stats.totalTokens, 0);
      expect(stats.requestCount, 0);
    });
  });
}
