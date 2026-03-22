import 'dart:io';
import 'package:desktop_multi_window/desktop_multi_window.dart';
import 'package:window_manager/window_manager.dart';

/// 窗口动画辅助工具类
/// 提供窗口显示和隐藏时的动画效果
class WindowAnimationHelper {
  /// Linux 子窗口中 windowManager 的 show/hide/focus 等方法可能抛出
  /// MissingPluginException，降级使用 desktop_multi_window 的 WindowController。
  static Future<void> _linuxShow() async {
    try {
      await windowManager.show();
      await windowManager.focus();
    } catch (_) {
      final controller = await WindowController.fromCurrentEngine();
      await controller.show();
    }
  }

  static Future<void> _linuxHide() async {
    try {
      await windowManager.hide();
    } catch (_) {
      final controller = await WindowController.fromCurrentEngine();
      await controller.hide();
    }
  }

  /// 使用淡入淡出效果隐藏窗口
  ///
  /// [duration] 动画持续时间（毫秒），默认 200ms
  static Future<void> hideWithAnimation({int duration = 200}) async {
    // Linux 平台 window_manager 不完整支持，降级为直接隐藏
    if (Platform.isLinux) {
      await _linuxHide();
      return;
    }

    // 获取当前窗口透明度
    final currentOpacity = await windowManager.getOpacity();

    // 如果已经隐藏或完全透明，直接返回
    if (currentOpacity <= 0.0) {
      await windowManager.hide();
      return;
    }

    // 淡出动画：逐渐降低透明度
    const steps = 20;
    final stepDuration = duration ~/ steps;
    final opacityStep = currentOpacity / steps;

    for (int i = 1; i <= steps; i++) {
      final newOpacity = currentOpacity - (opacityStep * i);
      await windowManager.setOpacity(newOpacity.clamp(0.0, 1.0));
      await Future.delayed(Duration(milliseconds: stepDuration));
    }

    // 最后隐藏窗口
    await windowManager.hide();

    // 重置透明度为1.0，为下次显示做准备
    await windowManager.setOpacity(1.0);
  }

  /// 使用淡入淡出效果显示窗口
  ///
  /// [duration] 动画持续时间（毫秒），默认 200ms
  static Future<void> showWithAnimation({int duration = 200}) async {
    // Linux 平台 window_manager 不完整支持，降级为直接显示
    if (Platform.isLinux) {
      await _linuxShow();
      return;
    }

    // 先将透明度设置为0
    await windowManager.setOpacity(0.0);

    // 显示窗口（但是透明的）
    await windowManager.show();
    await windowManager.focus();

    // 淡入动画：逐渐提高透明度
    const steps = 20;
    final stepDuration = duration ~/ steps;
    final opacityStep = 1.0 / steps;

    for (int i = 1; i <= steps; i++) {
      final newOpacity = opacityStep * i;
      await windowManager.setOpacity(newOpacity.clamp(0.0, 1.0));
      await Future.delayed(Duration(milliseconds: stepDuration));
    }

    // 确保最终透明度为1.0
    await windowManager.setOpacity(1.0);
  }

  /// 使用缩放效果隐藏窗口（仅 macOS 支持）
  ///
  /// macOS 原生的最小化动画
  static Future<void> minimizeWithAnimation() async {
    await windowManager.minimize();
  }

  /// 快速隐藏（无动画）
  static Future<void> hideInstantly() async {
    if (Platform.isLinux) {
      await _linuxHide();
      return;
    }
    await windowManager.hide();
  }

  /// 快速显示（无动画）
  static Future<void> showInstantly() async {
    if (Platform.isLinux) {
      await _linuxShow();
      return;
    }
    await windowManager.show();
    await windowManager.focus();
  }
}
