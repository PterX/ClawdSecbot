import 'dart:io';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import '../utils/app_logger.dart';

/// 数据库服务：仅负责计算和提供数据库文件路径
///
/// 所有实际的数据库操作已迁移到Go层，通过FFI调用。
/// 此服务仅保留 dbPath 的计算逻辑，供 NativeLibraryService 初始化Go DB时使用。
class DatabaseService {
  static final DatabaseService _instance = DatabaseService._internal();
  String? _dbPath;

  factory DatabaseService() {
    return _instance;
  }

  DatabaseService._internal();

  /// 数据库文件路径
  String? get dbPath => _dbPath;

  /// 初始化：计算数据库路径并确保目录存在
  Future<void> init() async {
    if (_dbPath != null) return;

    final dir = await getApplicationSupportDirectory();
    final dbPath = p.join(dir.path, 'bot_sec_manager.db');
    _dbPath = dbPath;
    appLogger.info('[Database] Database path: $dbPath');

    // 确保目录存在
    final dbFile = File(dbPath);
    if (!await dbFile.parent.exists()) {
      await dbFile.parent.create(recursive: true);
    }
  }
}
