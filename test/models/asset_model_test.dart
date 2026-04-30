import 'package:bot_sec_manager/models/asset_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Asset', () {
    test('serializes to and from JSON round-trip', () {
      final original = Asset(
        id: 'openclaw:abc123',
        sourcePlugin: 'openclaw',
        name: 'OpenClaw Gateway',
        type: 'gateway',
        version: '2.1.0',
        ports: [8080, 8443],
        serviceName: 'openclaw-gateway',
        processPaths: ['/usr/bin/openclaw', '/opt/openclaw/bin/gateway'],
        metadata: {'env': 'production', 'region': 'cn-east'},
        displaySections: [
          DisplaySection(
            title: 'Network',
            icon: 'network',
            items: [
              DisplayItem(label: 'Bind Address', value: '127.0.0.1:8080', status: 'safe'),
              DisplayItem(label: 'TLS', value: 'enabled', status: 'safe'),
            ],
          ),
        ],
      );

      final json = original.toJson();
      final restored = Asset.fromJson(json);

      expect(restored.id, 'openclaw:abc123');
      expect(restored.sourcePlugin, 'openclaw');
      expect(restored.name, 'OpenClaw Gateway');
      expect(restored.type, 'gateway');
      expect(restored.version, '2.1.0');
      expect(restored.ports, [8080, 8443]);
      expect(restored.serviceName, 'openclaw-gateway');
      expect(restored.processPaths.length, 2);
      expect(restored.metadata['env'], 'production');
      expect(restored.displaySections.length, 1);
      expect(restored.displaySections[0].items.length, 2);
    });

    test('defaults to empty collections on missing fields', () {
      final restored = Asset.fromJson({});
      expect(restored.id, '');
      expect(restored.sourcePlugin, '');
      expect(restored.name, '');
      expect(restored.ports, isEmpty);
      expect(restored.processPaths, isEmpty);
      expect(restored.metadata, isEmpty);
      expect(restored.displaySections, isEmpty);
    });
  });

  group('DisplaySection', () {
    test('serializes to and from JSON round-trip', () {
      final original = DisplaySection(
        title: 'Config',
        icon: 'settings',
        items: [DisplayItem(label: 'Auth', value: 'token', status: 'warning')],
      );

      final json = original.toJson();
      final restored = DisplaySection.fromJson(json);

      expect(restored.title, 'Config');
      expect(restored.icon, 'settings');
      expect(restored.items.length, 1);
      expect(restored.items[0].label, 'Auth');
    });

    test('defaults to empty items when missing', () {
      final restored = DisplaySection.fromJson({'title': 't', 'icon': 'i'});
      expect(restored.items, isEmpty);
    });
  });

  group('DisplayItem', () {
    test('serializes to and from JSON round-trip', () {
      final original = DisplayItem(label: 'Port', value: '8080', status: 'safe');
      final json = original.toJson();
      final restored = DisplayItem.fromJson(json);
      expect(restored.label, 'Port');
      expect(restored.value, '8080');
      expect(restored.status, 'safe');
    });

    test('defaults status to neutral', () {
      final restored = DisplayItem.fromJson({'label': 'l', 'value': 'v'});
      expect(restored.status, 'neutral');
    });
  });
}
