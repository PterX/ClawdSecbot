import 'package:bot_sec_manager/models/version_info.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('VersionInfo', () {
    test('parses from JSON with all fields', () {
      final json = {
        'version': '1.0.3',
        'download_url': 'https://example.com/download',
        'hash': 'abc123def456',
        'force_update': true,
        'change_log': 'Bug fixes and improvements',
      };
      final info = VersionInfo.fromJson(json);
      expect(info.version, '1.0.3');
      expect(info.downloadUrl, 'https://example.com/download');
      expect(info.hash, 'abc123def456');
      expect(info.forceUpdate, true);
      expect(info.changeLog, 'Bug fixes and improvements');
    });

    test('defaults to empty/false for missing fields', () {
      final info = VersionInfo.fromJson({});
      expect(info.version, '');
      expect(info.downloadUrl, '');
      expect(info.hash, '');
      expect(info.forceUpdate, false);
      expect(info.changeLog, '');
    });

    test('parses partial JSON correctly', () {
      final info = VersionInfo.fromJson({
        'version': '2.0.0',
        'force_update': false,
      });
      expect(info.version, '2.0.0');
      expect(info.downloadUrl, '');
      expect(info.forceUpdate, false);
    });
  });
}
