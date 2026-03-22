import 'package:flutter/material.dart';
import '../utils/locale_utils.dart';

class LocaleProvider extends ChangeNotifier {
  Locale _locale = Locale(LocaleUtils.getSystemLanguageCode());

  Locale get locale => _locale;

  LocaleProvider();

  Future<void> setLocale(Locale locale) async {
    final resolvedLanguage = LocaleUtils.resolveLanguageCode(
      explicitLanguage: locale.languageCode,
    );
    final nextLocale = Locale(resolvedLanguage);
    if (_locale == nextLocale) return;
    _locale = nextLocale;
    notifyListeners();
  }

  /// 清理语言状态并回退到系统语言。
  void clearLocale() {
    _locale = Locale(LocaleUtils.getSystemLanguageCode());
    notifyListeners();
  }
}
