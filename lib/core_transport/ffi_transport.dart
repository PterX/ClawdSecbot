import 'dart:ffi' as ffi;
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import '../services/native_library_service.dart';
import '../utils/app_logger.dart';
import 'botsec_transport.dart';

typedef _NoArgC = ffi.Pointer<Utf8> Function();
typedef _NoArgDart = ffi.Pointer<Utf8> Function();

typedef _OneArgC = ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>);
typedef _OneArgDart = ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>);

typedef _TwoArgC =
    ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>, ffi.Pointer<Utf8>);
typedef _TwoArgDart =
    ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>, ffi.Pointer<Utf8>);

typedef _OneIntC = ffi.Pointer<Utf8> Function(ffi.Int32);
typedef _OneIntDart = ffi.Pointer<Utf8> Function(int);

typedef _OneArgOneIntC = ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>, ffi.Int32);
typedef _OneArgOneIntDart = ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>, int);

typedef _ThreeIntC = ffi.Pointer<Utf8> Function(ffi.Int32, ffi.Int32, ffi.Int32);
typedef _ThreeIntDart = ffi.Pointer<Utf8> Function(int, int, int);

typedef _FreeStrC = ffi.Void Function(ffi.Pointer<Utf8>);
typedef _FreeStrDart = void Function(ffi.Pointer<Utf8>);

class FfiTransport extends BotsecTransport {
  FfiTransport._();

  static final FfiTransport instance = FfiTransport._();

  ffi.DynamicLibrary? get _dylib => NativeLibraryService().dylib;
  FreeStringDart? get _freeString => NativeLibraryService().freeString;

  @override
  bool get isReady => _dylib != null && _freeString != null;

  @override
  String callRawNoArg(String method) {
    if (!isReady) {
      return _notReadyJson();
    }

    try {
      final fn = _dylib!.lookupFunction<_NoArgC, _NoArgDart>(method);
      final resultPtr = fn();
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    }
  }

  @override
  String callRawOneArg(String method, String arg) {
    if (!isReady) {
      return _notReadyJson();
    }

    ffi.Pointer<Utf8>? argPtr;
    try {
      final fn = _dylib!.lookupFunction<_OneArgC, _OneArgDart>(method);
      argPtr = arg.toNativeUtf8();
      final resultPtr = fn(argPtr);
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    } finally {
      if (argPtr != null) {
        malloc.free(argPtr);
      }
    }
  }

  /// 在后台 isolate 执行同步 FFI 调用，保持 UI isolate 空闲。
  /// 适用于一次性的长耗时调用（如 LLM 连通性测试）。
  @override
  Future<String> callRawOneArgAsync(String method, String arg) async {
    if (!isReady) {
      return _notReadyJson();
    }
    final libPath = NativeLibraryService().libraryPath;
    if (libPath == null || libPath.isEmpty) {
      return callRawOneArg(method, arg);
    }
    final payload = _FfiOneArgPayload(libPath, method, arg);
    try {
      return await Isolate.run<String>(
        () => _invokeFfiOneArgInIsolate(payload),
        debugName: 'ffi-$method',
      );
    } catch (e) {
      appLogger.error('[Transport][FFI-Async] $method failed: $e');
      return _errorJson(method, e);
    }
  }

  @override
  String callRawTwoArgs(String method, String arg1, String arg2) {
    if (!isReady) {
      return _notReadyJson();
    }

    ffi.Pointer<Utf8>? arg1Ptr;
    ffi.Pointer<Utf8>? arg2Ptr;
    try {
      final fn = _dylib!.lookupFunction<_TwoArgC, _TwoArgDart>(method);
      arg1Ptr = arg1.toNativeUtf8();
      arg2Ptr = arg2.toNativeUtf8();
      final resultPtr = fn(arg1Ptr, arg2Ptr);
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    } finally {
      if (arg1Ptr != null) {
        malloc.free(arg1Ptr);
      }
      if (arg2Ptr != null) {
        malloc.free(arg2Ptr);
      }
    }
  }

  @override
  String callRawOneInt(String method, int arg) {
    if (!isReady) {
      return _notReadyJson();
    }

    try {
      final fn = _dylib!.lookupFunction<_OneIntC, _OneIntDart>(method);
      final resultPtr = fn(arg);
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    }
  }

  @override
  String callRawOneArgOneInt(String method, String arg, int value) {
    if (!isReady) {
      return _notReadyJson();
    }

    ffi.Pointer<Utf8>? argPtr;
    try {
      final fn = _dylib!.lookupFunction<_OneArgOneIntC, _OneArgOneIntDart>(
        method,
      );
      argPtr = arg.toNativeUtf8();
      final resultPtr = fn(argPtr, value);
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    } finally {
      if (argPtr != null) {
        malloc.free(argPtr);
      }
    }
  }

  @override
  String callRawThreeInts(String method, int arg1, int arg2, int arg3) {
    if (!isReady) {
      return _notReadyJson();
    }

    try {
      final fn = _dylib!.lookupFunction<_ThreeIntC, _ThreeIntDart>(method);
      final resultPtr = fn(arg1, arg2, arg3);
      final result = resultPtr.toDartString();
      _freeString!.call(resultPtr);
      return result;
    } catch (e) {
      appLogger.error('[Transport][FFI] $method failed: $e');
      return _errorJson(method, e);
    }
  }

  String _notReadyJson() {
    return '{"success":false,"error":"Native library not initialized"}';
  }

  String _errorJson(String method, Object e) {
    final escaped = e.toString().replaceAll('"', '\\"');
    return '{"success":false,"error":"$method failed: $escaped"}';
  }
}

/// Parameters passed into the background isolate for a one-arg FFI call.
/// Keeps the isolate entry self-contained (no capture of FfiTransport state).
class _FfiOneArgPayload {
  const _FfiOneArgPayload(this.libPath, this.method, this.arg);

  final String libPath;
  final String method;
  final String arg;
}

/// Top-level isolate entry: re-opens the dylib in the worker isolate and
/// performs a single FFI call. Allocating and freeing native memory happen in
/// the same isolate, so no cross-isolate pointer ownership is involved.
/// The underlying OS library is already loaded in the process; DynamicLibrary
/// .open just increments its reference count.
String _invokeFfiOneArgInIsolate(_FfiOneArgPayload p) {
  ffi.Pointer<Utf8>? argPtr;
  try {
    final lib = ffi.DynamicLibrary.open(p.libPath);
    final fn = lib.lookupFunction<_OneArgC, _OneArgDart>(p.method);
    final freeStr = lib.lookupFunction<_FreeStrC, _FreeStrDart>('FreeString');
    argPtr = p.arg.toNativeUtf8();
    final resultPtr = fn(argPtr);
    final result = resultPtr.toDartString();
    freeStr(resultPtr);
    return result;
  } finally {
    if (argPtr != null) {
      malloc.free(argPtr);
    }
  }
}
