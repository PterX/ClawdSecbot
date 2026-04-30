import 'package:bot_sec_manager/core/result.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Result', () {
    test('Success returns correct data and flags', () {
      final result = Result.success(42);
      expect(result.isSuccess, true);
      expect(result.isFailure, false);
      expect(result.dataOrNull, 42);
      expect(result.errorOrNull, isNull);
    });

    test('Failure returns correct error and flags', () {
      final error = AppException('test error', code: 'E001');
      final result = Result<int>.failure(error);
      expect(result.isSuccess, false);
      expect(result.isFailure, true);
      expect(result.dataOrNull, isNull);
      expect(result.errorOrNull, error);
    });

    test('when dispatches to correct branch', () {
      final success = Result.success('data');
      final failure = Result<String>.failure(AppException('err'));

      expect(
        success.when(success: (d) => 'ok:$d', failure: (e) => 'fail'),
        'ok:data',
      );
      expect(
        failure.when(success: (d) => 'ok', failure: (e) => 'fail:${e.message}'),
        'fail:err',
      );
    });
  });

  group('AppException', () {
    test('toString includes message and optional code', () {
      expect(
        AppException('connection failed').toString(),
        'AppException: connection failed',
      );
      expect(
        AppException('timeout', code: 'NET_TIMEOUT').toString(),
        'AppException: timeout (NET_TIMEOUT)',
      );
    });
  });

  group('Exception subclasses', () {
    test('NetworkException is AppException', () {
      final e = NetworkException('offline', code: 'NO_NET');
      expect(e, isA<AppException>());
      expect(e.message, 'offline');
      expect(e.code, 'NO_NET');
    });

    test('DatabaseException is AppException', () {
      final e = DatabaseException('locked');
      expect(e, isA<AppException>());
      expect(e.message, 'locked');
    });

    test('ConfigurationException is AppException', () {
      final e = ConfigurationException('invalid key');
      expect(e, isA<AppException>());
    });

    test('SecurityException is AppException', () {
      final e = SecurityException('unauthorized');
      expect(e, isA<AppException>());
    });
  });

  group('ResultExtensions.asResult', () {
    test('wraps successful future as Success', () async {
      final result = Future.value('hello').asResult();
      expect((await result).isSuccess, true);
      expect((await result).dataOrNull, 'hello');
    });

    test('wraps AppException as Failure', () async {
      final result = Future<String>.error(
        NetworkException('offline'),
      ).asResult();
      final r = await result;
      expect(r.isFailure, true);
      expect(r.errorOrNull, isA<NetworkException>());
    });

    test('wraps generic exception as Failure with AppException', () async {
      final result = Future<String>.error(
        FormatException('bad'),
      ).asResult();
      final r = await result;
      expect(r.isFailure, true);
      expect(r.errorOrNull?.message, 'Unknown error');
    });
  });

  group('AsyncRunner.run', () {
    test('returns Success for successful operation', () async {
      final result = await AsyncRunner.run(() async => 42);
      expect(result.isSuccess, true);
      expect(result.dataOrNull, 42);
    });

    test('returns Failure for AppException', () async {
      final result = await AsyncRunner.run<String>(() async {
        throw DatabaseException('db locked');
      });
      expect(result.isFailure, true);
      expect(result.errorOrNull, isA<DatabaseException>());
    });

    test('returns Failure for generic exception', () async {
      final result = await AsyncRunner.run<String>(() async {
        throw StateError('bad state');
      });
      expect(result.isFailure, true);
      expect(result.errorOrNull?.message, 'Operation failed');
    });
  });
}
