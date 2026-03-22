import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:window_manager/window_manager.dart';

/// Intent 用于 Command+W 快捷键隐藏窗口
class HideWindowIntent extends Intent {
  const HideWindowIntent();
}

/// 为窗口添加 macOS Command+W 快捷键支持的 Widget
///
/// 使用示例:
/// ```dart
/// HideWindowShortcut(
///   child: Scaffold(
///     // your content
///   ),
/// )
/// ```
class HideWindowShortcut extends StatelessWidget {
  final Widget child;

  const HideWindowShortcut({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    // 使用 FocusScope 确保快捷键在整个窗口范围内生效
    return FocusScope(
      autofocus: true,
      child: Shortcuts(
        shortcuts: {
          LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.keyW):
              const HideWindowIntent(),
        },
        child: Actions(
          actions: {
            HideWindowIntent: CallbackAction<HideWindowIntent>(
              onInvoke: (_) {
                windowManager.hide();
                return null;
              },
            ),
          },
          child: child,
        ),
      ),
    );
  }
}
