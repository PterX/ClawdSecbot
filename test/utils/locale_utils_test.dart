import 'package:bot_sec_manager/utils/locale_utils.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('LocaleUtils.normalizeLanguageCode', () {
    test('normalizes "zh" to "zh"', () {
      expect(LocaleUtils.normalizeLanguageCode('zh'), 'zh');
    });

    test('normalizes "zh-CN" to "zh"', () {
      expect(LocaleUtils.normalizeLanguageCode('zh-CN'), 'zh');
    });

    test('normalizes "zh_Hant" to "zh"', () {
      expect(LocaleUtils.normalizeLanguageCode('zh_Hant'), 'zh');
    });

    test('normalizes "zh-tw" to "zh"', () {
      expect(LocaleUtils.normalizeLanguageCode('zh-tw'), 'zh');
    });

    test('normalizes "en" to "en"', () {
      expect(LocaleUtils.normalizeLanguageCode('en'), 'en');
    });

    test('normalizes "en-US" to "en"', () {
      expect(LocaleUtils.normalizeLanguageCode('en-US'), 'en');
    });

    test('normalizes "fr" to "en" (unsupported)', () {
      expect(LocaleUtils.normalizeLanguageCode('fr'), 'en');
    });

    test('normalizes null to "en"', () {
      expect(LocaleUtils.normalizeLanguageCode(null), 'en');
    });

    test('normalizes empty string to "en"', () {
      expect(LocaleUtils.normalizeLanguageCode(''), 'en');
    });

    test('normalizes whitespace-only string to "en"', () {
      expect(LocaleUtils.normalizeLanguageCode('  '), 'en');
    });

    test('trims whitespace before normalizing', () {
      expect(LocaleUtils.normalizeLanguageCode('  zh-CN  '), 'zh');
      expect(LocaleUtils.normalizeLanguageCode('  en  '), 'en');
    });

    test('is case-insensitive', () {
      expect(LocaleUtils.normalizeLanguageCode('ZH'), 'zh');
      expect(LocaleUtils.normalizeLanguageCode('EN'), 'en');
      expect(LocaleUtils.normalizeLanguageCode('ZH-cn'), 'zh');
    });
  });

  group('LocaleUtils.resolveLanguageCode', () {
    test('explicitLanguage takes highest priority', () {
      expect(
        LocaleUtils.resolveLanguageCode(
          explicitLanguage: 'en',
          savedLanguage: 'zh',
        ),
        'en',
      );
    });

    test('savedLanguage takes second priority', () {
      expect(
        LocaleUtils.resolveLanguageCode(
          explicitLanguage: null,
          savedLanguage: 'zh',
        ),
        'zh',
      );
    });

    test('falls back to system language when both are null', () {
      final result = LocaleUtils.resolveLanguageCode();
      expect(result, anyOf('zh', 'en'));
    });

    test('empty explicitLanguage falls through to savedLanguage', () {
      expect(
        LocaleUtils.resolveLanguageCode(
          explicitLanguage: '  ',
          savedLanguage: 'zh',
        ),
        'zh',
      );
    });

    test('empty savedLanguage falls through to system language', () {
      final result = LocaleUtils.resolveLanguageCode(
        savedLanguage: '  ',
      );
      expect(result, anyOf('zh', 'en'));
    });

    test('explicitLanguage is normalized', () {
      expect(
        LocaleUtils.resolveLanguageCode(explicitLanguage: 'zh-TW'),
        'zh',
      );
    });
  });

  group('LocaleUtils.supportedLanguages', () {
    test('contains zh and en', () {
      expect(LocaleUtils.supportedLanguages, contains('zh'));
      expect(LocaleUtils.supportedLanguages, contains('en'));
      expect(LocaleUtils.supportedLanguages.length, 2);
    });
  });
}
