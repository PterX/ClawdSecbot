import 'dart:async';
import '../utils/app_logger.dart';

/// Result type for operations that can fail
sealed class Result<T> {
  const Result();
  
  factory Result.success(T data) = Success<T>;
  factory Result.failure(AppException error) = Failure<T>;
  
  bool get isSuccess => this is Success<T>;
  bool get isFailure => this is Failure<T>;
  
  T? get dataOrNull {
    if (this case Success(:final data)) return data;
    return null;
  }
  
  AppException? get errorOrNull {
    if (this case Failure(:final error)) return error;
    return null;
  }
  
  R when<R>({
    required R Function(T data) success,
    required R Function(AppException error) failure,
  }) {
    return switch (this) {
      Success<T>(:final data) => success(data),
      Failure<T>(:final error) => failure(error),
    };
  }
}

class Success<T> extends Result<T> {
  final T data;
  const Success(this.data);
}

class Failure<T> extends Result<T> {
  final AppException error;
  const Failure(this.error);
}

/// Base exception class
class AppException implements Exception {
  final String message;
  final String? code;
  final dynamic originalError;
  final StackTrace? stackTrace;
  
  const AppException(this.message, {this.code, this.originalError, this.stackTrace});
  
  @override
  String toString() => 'AppException: $message${code != null ? ' ($code)' : ''}';
}

/// Exception types
class NetworkException extends AppException {
  const NetworkException(super.message, {super.code, super.originalError});
}

class DatabaseException extends AppException {
  const DatabaseException(super.message, {super.code, super.originalError});
}

class ConfigurationException extends AppException {
  const ConfigurationException(super.message, {super.code, super.originalError});
}

class SecurityException extends AppException {
  const SecurityException(super.message, {super.code, super.originalError});
}

/// Try-catch wrapper with Result
extension ResultExtensions<T> on Future<T> {
  Future<Result<T>> asResult() async {
    try {
      return Result.success(await this);
    } on AppException catch (e) {
      return Result.failure(e);
    } catch (e, st) {
      return Result.failure(AppException('Unknown error', originalError: e, stackTrace: st));
    }
  }
}

/// Async operation runner with error handling
class AsyncRunner {
  static Future<Result<T>> run<T>(
    Future<T> Function() operation, {
    String? operationName,
  }) async {
    try {
      final result = await operation();
      return Result.success(result);
    } on AppException catch (e) {
      appLogger.error('Operation failed: ${operationName ?? "unknown"}', e);
      return Result.failure(e);
    } catch (e, st) {
      appLogger.error('Operation error: ${operationName ?? "unknown"}', e);
      return Result.failure(AppException(
        'Operation failed',
        originalError: e,
        stackTrace: st,
      ));
    }
  }
}
