import 'dart:io';
import 'dart:ui';

/// 语言环境工具类，用于统一解析应用语言。
class LocaleUtils {
  /// 应用支持的语言代码列表。
  static const List<String> supportedLanguages = ['zh', 'en'];

  /// 归一化任意语言标识到应用支持范围。
  ///
  /// 规则：只要是中文（如 zh、zh-CN、zh_Hant）统一映射为 `zh`，
  /// 其余全部映射为 `en`。
  static String normalizeLanguageCode(String? languageCode) {
    final normalizedRaw = languageCode?.trim().toLowerCase();
    if (normalizedRaw == null || normalizedRaw.isEmpty) {
      return 'en';
    }

    final normalized = normalizedRaw.replaceAll('_', '-');
    if (normalized == 'zh' || normalized.startsWith('zh-')) {
      return 'zh';
    }
    return 'en';
  }

  /// 获取系统语言代码。
  /// 只要系统语言为中文就返回 `zh`，否则返回 `en`。
  static String getSystemLanguageCode() {
    try {
      final platformLocales = PlatformDispatcher.instance.locales;
      if (platformLocales.isNotEmpty) {
        final primaryLocale = platformLocales.first;
        final localeTag = primaryLocale.toLanguageTag();
        return normalizeLanguageCode(localeTag);
      }

      final localeName = Platform.localeName;
      return normalizeLanguageCode(localeName);
    } catch (_) {
      // 忽略异常，统一回退默认语言。
    }

    return 'en';
  }

  /// 解析最终语言代码。
  /// 优先级：显式语言 > 已保存语言 > 系统语言。
  ///
  /// 注意：显式语言与已保存语言都会按“中文/非中文”规则归一化。
  static String resolveLanguageCode({
    String? explicitLanguage,
    String? savedLanguage,
  }) {
    if (explicitLanguage != null && explicitLanguage.trim().isNotEmpty) {
      return normalizeLanguageCode(explicitLanguage);
    }

    if (savedLanguage != null && savedLanguage.trim().isNotEmpty) {
      return normalizeLanguageCode(savedLanguage);
    }

    return getSystemLanguageCode();
  }
}
