import 'package:flutter/material.dart';

/// 应用字体工具类
/// 使用内置本地字体，支持中文显示
class AppFonts {
  static const String _interFamily = 'Inter';
  static const String _firaCodeFamily = 'FiraCode';
  static const String _notoSansSCFamily = 'NotoSansSC';
  static const String _robotoMonoFamily = 'RobotoMono';

  /// Inter 字体样式（与 GoogleFonts.inter 兼容的 API）
  static TextStyle inter({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? letterSpacing,
    double? height,
    TextDecoration? decoration,
    FontStyle? fontStyle,
  }) {
    return TextStyle(
      fontFamily: _interFamily,
      fontFamilyFallback: const [_notoSansSCFamily],
      fontSize: fontSize,
      fontWeight: fontWeight,
      color: color,
      letterSpacing: letterSpacing,
      height: height,
      decoration: decoration,
      fontStyle: fontStyle,
    );
  }

  /// Fira Code 字体样式（与 GoogleFonts.firaCode 兼容的 API）
  static TextStyle firaCode({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? letterSpacing,
    double? height,
    TextDecoration? decoration,
    FontStyle? fontStyle,
  }) {
    return TextStyle(
      fontFamily: _firaCodeFamily,
      fontFamilyFallback: const [_notoSansSCFamily],
      fontSize: fontSize,
      fontWeight: fontWeight,
      color: color,
      letterSpacing: letterSpacing,
      height: height,
      decoration: decoration,
      fontStyle: fontStyle,
    );
  }

  /// Roboto Mono 字体样式（与 GoogleFonts.robotoMono 兼容的 API）
  static TextStyle robotoMono({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? letterSpacing,
    double? height,
    TextDecoration? decoration,
    FontStyle? fontStyle,
  }) {
    return TextStyle(
      fontFamily: _robotoMonoFamily,
      fontFamilyFallback: const [_notoSansSCFamily],
      fontSize: fontSize,
      fontWeight: fontWeight,
      color: color,
      letterSpacing: letterSpacing,
      height: height,
      decoration: decoration,
      fontStyle: fontStyle,
    );
  }

  /// 获取文本样式（使用 Noto Sans SC，支持中文显示）
  static TextStyle getTextStyle({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? letterSpacing,
    double? height,
  }) {
    return TextStyle(
      fontFamily: _notoSansSCFamily,
      fontSize: fontSize,
      fontWeight: fontWeight,
      color: color,
      letterSpacing: letterSpacing,
      height: height,
    );
  }

  /// 获取普通文本样式
  static TextStyle regular({
    double fontSize = 14,
    Color color = Colors.white,
    double? letterSpacing,
  }) {
    return getTextStyle(
      fontSize: fontSize,
      fontWeight: FontWeight.w400,
      color: color,
      letterSpacing: letterSpacing,
    );
  }

  /// 获取中等粗细文本样式
  static TextStyle medium({
    double fontSize = 14,
    Color color = Colors.white,
    double? letterSpacing,
  }) {
    return getTextStyle(
      fontSize: fontSize,
      fontWeight: FontWeight.w500,
      color: color,
      letterSpacing: letterSpacing,
    );
  }

  /// 获取粗体文本样式
  static TextStyle bold({
    double fontSize = 14,
    Color color = Colors.white,
    double? letterSpacing,
  }) {
    return getTextStyle(
      fontSize: fontSize,
      fontWeight: FontWeight.w700,
      color: color,
      letterSpacing: letterSpacing,
    );
  }

  /// 获取代码文本样式（使用 Fira Code）
  static TextStyle code({double fontSize = 13, Color color = Colors.white70}) {
    return firaCode(fontSize: fontSize, color: color);
  }

  /// 获取 Inter TextTheme（与 GoogleFonts.interTextTheme 兼容的 API）
  static TextTheme interTextTheme(TextTheme base) {
    return base.apply(
      fontFamily: _interFamily,
      fontFamilyFallback: const [_notoSansSCFamily],
    );
  }

  /// 获取 TextTheme（用于 ThemeData，使用 NotoSansSC）
  static TextTheme getTextTheme() {
    return ThemeData.dark().textTheme.apply(fontFamily: _notoSansSCFamily);
  }
}
