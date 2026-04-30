import 'package:bot_sec_manager/models/risk_model.dart';
import 'package:bot_sec_manager/models/asset_model.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('RiskInfo', () {
    test('serializes to and from JSON round-trip', () {
      final original = RiskInfo(
        id: 'gateway_bind_unsafe',
        args: {'bind': '0.0.0.0:8080'},
        assetID: 'openclaw:abc123',
        title: '网关监听地址不安全',
        titleEn: 'Unsafe gateway bind address',
        description: '网关绑定到所有接口',
        descriptionEn: 'Gateway bound to all interfaces',
        level: RiskLevel.high,
        icon: Icons.warning,
        mitigation: Mitigation(
          type: 'form',
          formSchema: [
            FormItem(key: 'bind_address', label: 'Bind Address', type: 'text', required: true),
          ],
          title: '收紧网关监听地址',
          titleEn: 'Tighten gateway bind address',
        ),
        sourcePlugin: 'openclaw',
      );

      final json = original.toJson();
      final restored = RiskInfo.fromJson(json);

      expect(restored.id, original.id);
      expect(restored.args?['bind'], '0.0.0.0:8080');
      expect(restored.assetID, original.assetID);
      expect(restored.title, original.title);
      expect(restored.titleEn, original.titleEn);
      expect(restored.description, original.description);
      expect(restored.descriptionEn, original.descriptionEn);
      expect(restored.level, RiskLevel.high);
      expect(restored.sourcePlugin, original.sourcePlugin);
      expect(restored.mitigation?.type, 'form');
      expect(restored.mitigation?.formSchema.length, 1);
    });

    test('parses risk level from string', () {
      final json = {
        'id': 'test', 'title': 'Test', 'description': 'Desc',
        'level': 'critical', 'icon_code_point': Icons.warning.codePoint,
      };
      expect(RiskInfo.fromJson(json).level, RiskLevel.critical);
    });

    test('parses risk level from int', () {
      final json = {
        'id': 'test', 'title': 'Test', 'description': 'Desc',
        'level': 3, 'icon_code_point': Icons.warning.codePoint,
      };
      expect(RiskInfo.fromJson(json).level, RiskLevel.critical);
    });

    test('defaults to low for unknown level string', () {
      final json = {
        'id': 'test', 'title': 'Test', 'description': 'Desc',
        'level': 'unknown', 'icon_code_point': Icons.warning.codePoint,
      };
      expect(RiskInfo.fromJson(json).level, RiskLevel.low);
    });

    test('displayTitle returns titleEn for English locale', () {
      final risk = RiskInfo(
        id: 'test', title: '中文标题', titleEn: 'English Title',
        description: 'desc', level: RiskLevel.low, icon: Icons.warning,
      );
      expect(risk.displayTitle('en'), 'English Title');
      expect(risk.displayTitle('zh'), '中文标题');
    });

    test('displayTitle falls back to title when titleEn is empty', () {
      final risk = RiskInfo(
        id: 'test', title: '中文标题', titleEn: '  ',
        description: 'desc', level: RiskLevel.low, icon: Icons.warning,
      );
      expect(risk.displayTitle('en'), '中文标题');
    });

    test('displayDescription works for English locale', () {
      final risk = RiskInfo(
        id: 'test', title: 't', description: '中文描述',
        descriptionEn: 'English Description', level: RiskLevel.low, icon: Icons.warning,
      );
      expect(risk.displayDescription('en_US'), 'English Description');
      expect(risk.displayDescription('zh'), '中文描述');
    });

    test('color maps to correct Material color per level', () {
      final levels = {
        RiskLevel.low: const Color(0xFF22C55E),
        RiskLevel.medium: const Color(0xFFF59E0B),
        RiskLevel.high: const Color(0xFFEF4444),
        RiskLevel.critical: const Color(0xFFDC2626),
      };
      for (final entry in levels.entries) {
        final risk = RiskInfo(
          id: 'test', title: 't', description: 'd',
          level: entry.key, icon: Icons.warning,
        );
        expect(risk.color, entry.value);
      }
    });

    test('_parseAssetID extracts from args when top-level is absent', () {
      final json = {
        'id': 'test', 'title': 't', 'description': 'd', 'level': 'low',
        'icon_code_point': Icons.warning.codePoint,
        'args': {'asset_id': 'from_args'},
      };
      expect(RiskInfo.fromJson(json).assetID, 'from_args');
    });

    test('handles null fields gracefully', () {
      final json = {
        'id': 'test', 'title': null, 'description': null,
        'level': null, 'icon_code_point': null,
      };
      final risk = RiskInfo.fromJson(json);
      expect(risk.title, '');
      expect(risk.description, '');
      expect(risk.level, RiskLevel.low);
    });
  });

  group('Mitigation', () {
    test('serializes to and from JSON round-trip', () {
      final original = Mitigation(
        type: 'form',
        formSchema: [
          FormItem(key: 'k1', label: 'Label1', type: 'text', required: true),
          FormItem(key: 'k2', label: 'Label2', type: 'select', options: ['a', 'b']),
        ],
        title: '修复标题', titleEn: 'Fix title',
        description: '修复描述', descriptionEn: 'Fix description',
        suggestions: [
          SuggestionGroup(
            priority: 'P0', category: 'Immediate actions',
            categoryEn: 'Immediate actions',
            items: [
              SuggestionItem(
                action: 'Upgrade', actionEn: 'Upgrade now',
                detail: 'Please upgrade', detailEn: 'Please upgrade now',
                command: 'upgrade --force',
              ),
            ],
          ),
        ],
      );

      final json = original.toJson();
      final restored = Mitigation.fromJson(json);

      expect(restored.type, 'form');
      expect(restored.formSchema.length, 2);
      expect(restored.formSchema[0].key, 'k1');
      expect(restored.formSchema[1].options, ['a', 'b']);
      expect(restored.title, '修复标题');
      expect(restored.suggestions?.length, 1);
      expect(restored.suggestions?[0].items[0].command, 'upgrade --force');
    });

    test('displayTitle returns titleEn for English locale', () {
      final m = Mitigation(type: 'suggestion', formSchema: [], title: '中文', titleEn: 'English');
      expect(m.displayTitle('en'), 'English');
      expect(m.displayTitle('zh'), '中文');
    });

    test('displayDescription falls back when titleEn is null', () {
      final m = Mitigation(type: 'suggestion', formSchema: [], title: '中文', description: '描述');
      expect(m.displayDescription('en'), '描述');
    });
  });

  group('SuggestionGroup', () {
    test('serializes to and from JSON round-trip', () {
      final original = SuggestionGroup(
        priority: 'P1', category: 'Short term hardening',
        categoryEn: 'Short term hardening',
        items: [
          SuggestionItem(action: 'Add CSP', detail: 'Add security headers'),
          SuggestionItem(action: 'Rotate keys', detail: 'Rotate weak keys'),
        ],
      );
      final json = original.toJson();
      final restored = SuggestionGroup.fromJson(json);
      expect(restored.priority, 'P1');
      expect(restored.items.length, 2);
      expect(restored.items[1].action, 'Rotate keys');
    });

    test('displayCategory returns categoryEn for English locale', () {
      final g = SuggestionGroup(
        priority: 'P0', category: '立即措施',
        categoryEn: 'Immediate actions', items: [],
      );
      expect(g.displayCategory('en'), 'Immediate actions');
      expect(g.displayCategory('zh'), '立即措施');
    });
  });

  group('SuggestionItem', () {
    test('displayAction and displayDetail respect locale', () {
      final item = SuggestionItem(
        action: '升级版本', actionEn: 'Upgrade version',
        detail: '执行升级命令', detailEn: 'Run the upgrade command',
        command: 'upgrade --latest',
      );
      expect(item.displayAction('en'), 'Upgrade version');
      expect(item.displayAction('zh_CN'), '升级版本');
      expect(item.displayDetail('en'), 'Run the upgrade command');
      expect(item.command, 'upgrade --latest');
    });
  });

  group('FormItem', () {
    test('serializes to and from JSON round-trip', () {
      final original = FormItem(
        key: 'auth_mode', label: 'Authentication Mode',
        type: 'select', defaultValue: 'token',
        options: ['token', 'password'], required: true,
        minLength: 1, regex: r'^[a-z]+$', regexMsg: 'Only lowercase letters',
      );
      final json = original.toJson();
      final restored = FormItem.fromJson(json);
      expect(restored.key, 'auth_mode');
      expect(restored.label, 'Authentication Mode');
      expect(restored.type, 'select');
      expect(restored.defaultValue, 'token');
      expect(restored.options, ['token', 'password']);
      expect(restored.required, true);
      expect(restored.minLength, 1);
      expect(restored.regex, r'^[a-z]+$');
      expect(restored.regexMsg, 'Only lowercase letters');
    });

    test('defaults are applied when fields are absent', () {
      final json = {'key': 'k', 'label': 'L', 'type': 'text'};
      final item = FormItem.fromJson(json);
      expect(item.required, false);
      expect(item.minLength, 0);
      expect(item.options, isNull);
    });
  });

  group('ScanResult', () {
    test('serializes to and from JSON round-trip', () {
      final original = ScanResult(
        config: {'key': 'value'},
        riskInfo: [
          RiskInfo(id: 'r1', title: 'Risk 1', description: 'desc',
              level: RiskLevel.medium, icon: Icons.warning),
        ],
        skillResult: [
          RiskInfo(id: 's1', title: 'Skill Risk', description: 'skill desc',
              level: RiskLevel.low, icon: Icons.warning),
        ],
        configFound: true,
        configPath: '/tmp/config.json',
        assets: [
          Asset(
            id: 'a1', sourcePlugin: 'openclaw', name: 'test',
            type: 'gateway', version: '1.0', ports: [8080],
            serviceName: 'svc', processPaths: ['/usr/bin/test'],
            metadata: {'env': 'prod'}, displaySections: [],
          ),
        ],
        scannedAt: DateTime(2026, 1, 1),
      );

      final json = original.toJson();
      final restored = ScanResult.fromJson(json);

      expect(restored.configFound, true);
      expect(restored.configPath, '/tmp/config.json');
      expect(restored.riskInfo.length, 1);
      expect(restored.riskInfo[0].id, 'r1');
      expect(restored.skillResult.length, 1);
      expect(restored.skillResult[0].id, 's1');
      expect(restored.risks.length, 2);
      expect(restored.assets.length, 1);
      expect(restored.assets[0].name, 'test');
      expect(restored.scannedAt, DateTime(2026, 1, 1));
    });

    test('risks getter merges riskInfo and skillResult', () {
      final result = ScanResult(
        riskInfo: [
          RiskInfo(id: 'r1', title: 'R1', description: 'd',
              level: RiskLevel.low, icon: Icons.warning),
        ],
        skillResult: [
          RiskInfo(id: 's1', title: 'S1', description: 'd',
              level: RiskLevel.low, icon: Icons.warning),
        ],
        configFound: false,
      );
      expect(result.risks.length, 2);
      expect(result.risks.map((r) => r.id), ['r1', 's1']);
    });

    test('handles legacy "risks" key in JSON', () {
      final json = {
        'config_found': false,
        'assets': [],
        'risks': [
          {
            'id': 'legacy', 'title': 'Legacy Risk',
            'description': 'desc', 'level': 'low',
            'icon_code_point': Icons.warning.codePoint,
          },
        ],
      };
      final result = ScanResult.fromJson(json);
      expect(result.riskInfo.length, 1);
      expect(result.riskInfo[0].id, 'legacy');
    });
  });
}

