import 'dart:convert';
import 'dart:ffi' as ffi;
import 'package:ffi/ffi.dart';
import 'native_library_service.dart' hide FreeStringDart;

// C function signatures
typedef GetSupportedProvidersC = ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>);
typedef GetSupportedProvidersDart =
    ffi.Pointer<Utf8> Function(ffi.Pointer<Utf8>);

typedef FreeStringC = ffi.Void Function(ffi.Pointer<Utf8>);
typedef FreeStringDart = void Function(ffi.Pointer<Utf8>);

/// Provider scope for filtering.
enum ProviderScope {
  security('security'),
  bot('bot'),
  all('all');

  const ProviderScope(this.value);
  final String value;
}

/// Provider information from Go layer.
class ProviderInfo {
  final String name;
  final String displayName;
  final String icon;
  final String scope;
  final bool needsEndpoint;
  final bool needsAPIKey;
  final bool needsSecretKey;
  final String defaultBaseURL;
  final String defaultModel;
  final String apiKeyHint;
  final String modelHint;

  const ProviderInfo({
    required this.name,
    required this.displayName,
    required this.icon,
    required this.scope,
    required this.needsEndpoint,
    required this.needsAPIKey,
    required this.needsSecretKey,
    required this.defaultBaseURL,
    required this.defaultModel,
    required this.apiKeyHint,
    required this.modelHint,
  });

  factory ProviderInfo.fromJson(Map<String, dynamic> json) {
    return ProviderInfo(
      name: json['name'] ?? '',
      displayName: json['display_name'] ?? '',
      icon: json['icon'] ?? 'sparkles',
      scope: json['scope'] ?? 'all',
      needsEndpoint: json['needs_endpoint'] ?? false,
      needsAPIKey: json['needs_api_key'] ?? true,
      needsSecretKey: json['needs_secret_key'] ?? false,
      defaultBaseURL: json['default_base_url'] ?? '',
      defaultModel: json['default_model'] ?? '',
      apiKeyHint: json['api_key_hint'] ?? '',
      modelHint: json['model_hint'] ?? '',
    );
  }
}

/// Service to get supported providers from Go layer via FFI.
class ProviderService {
  static final ProviderService _instance = ProviderService._internal();

  factory ProviderService() => _instance;

  ProviderService._internal();

  List<ProviderInfo>? _cachedSecurityProviders;
  List<ProviderInfo>? _cachedBotProviders;

  ffi.DynamicLibrary _loadLibrary() {
    final dylib = NativeLibraryService().dylib;
    if (dylib == null) {
      throw Exception('Plugin library not loaded');
    }
    return dylib;
  }

  /// Get supported providers for a given scope.
  List<ProviderInfo> getProviders(ProviderScope scope) {
    // Check cache first
    if (scope == ProviderScope.security && _cachedSecurityProviders != null) {
      return _cachedSecurityProviders!;
    }
    if (scope == ProviderScope.bot && _cachedBotProviders != null) {
      return _cachedBotProviders!;
    }

    try {
      final dylib = _loadLibrary();

      final getSupportedProviders = dylib
          .lookupFunction<GetSupportedProvidersC, GetSupportedProvidersDart>(
            'GetSupportedProviders',
          );
      final freeString = dylib.lookupFunction<FreeStringC, FreeStringDart>(
        'FreeString',
      );

      final scopePtr = scope.value.toNativeUtf8();
      final resultPtr = getSupportedProviders(scopePtr);
      calloc.free(scopePtr);

      final jsonStr = resultPtr.toDartString();
      freeString(resultPtr);

      final List<dynamic> jsonList = jsonDecode(jsonStr);
      final providers = jsonList.map((e) => ProviderInfo.fromJson(e)).toList();

      // Cache the result
      if (scope == ProviderScope.security) {
        _cachedSecurityProviders = providers;
      } else if (scope == ProviderScope.bot) {
        _cachedBotProviders = providers;
      }

      return providers;
    } catch (e) {
      // Return empty list on error
      return [];
    }
  }

  /// Clear cached providers (call when plugin is reloaded).
  void clearCache() {
    _cachedSecurityProviders = null;
    _cachedBotProviders = null;
  }
}
