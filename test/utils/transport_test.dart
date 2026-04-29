import 'package:bot_sec_manager/core_transport/botsec_transport.dart';
import 'package:flutter_test/flutter_test.dart';
import 'dart:convert';

/// Stub transport for testing BotsecTransport._decodeEnvelope logic
class _StubTransport extends BotsecTransport {
  final Map<String, String> _responses;

  _StubTransport(this._responses);

  @override
  bool get isReady => true;

  @override
  String callRawNoArg(String method) => _responses[method] ?? '{}';

  @override
  String callRawOneArg(String method, String arg) => _responses[method] ?? '{}';

  @override
  String callRawTwoArgs(String method, String arg1, String arg2) =>
      _responses[method] ?? '{}';

  @override
  String callRawOneInt(String method, int arg) => _responses[method] ?? '{}';

  @override
  String callRawOneArgOneInt(String method, String arg, int value) =>
      _responses[method] ?? '{}';

  @override
  String callRawThreeInts(String method, int arg1, int arg2, int arg3) =>
      _responses[method] ?? '{}';
}

void main() {
  group('BotsecTransport envelope decoding', () {
    test('callNoArg decodes valid envelope', () {
      final transport = _StubTransport({
        'GetStatus': '{"success": true, "data": {"running": true}}',
      });
      final result = transport.callNoArg('GetStatus');
      expect(result['success'], true);
      expect(result['data']['running'], true);
    });

    test('callOneArg decodes valid envelope', () {
      final transport = _StubTransport({
        'GetConfig': '{"success": true, "data": {"key": "value"}}',
      });
      final result = transport.callOneArg('GetConfig', 'openclaw');
      expect(result['success'], true);
      expect(result['data']['key'], 'value');
    });

    test('returns error for non-object JSON', () {
      final transport = _StubTransport({
        'ListItems': '"just a string"',
      });
      final result = transport.callNoArg('ListItems');
      expect(result['success'], false);
      expect(result['error'], contains('non-object JSON'));
    });

    test('returns error for invalid JSON', () {
      final transport = _StubTransport({
        'Broken': 'not json at all{',
      });
      final result = transport.callNoArg('Broken');
      expect(result['success'], false);
      expect(result['error'], contains('invalid JSON'));
    });

    test('callTwoArgs decodes response', () {
      final transport = _StubTransport({
        'SetConfig': '{"success": true, "data": null}',
      });
      final result = transport.callTwoArgs('SetConfig', 'key', 'value');
      expect(result['success'], true);
    });

    test('callOneInt decodes response', () {
      final transport = _StubTransport({
        'SetPort': '{"success": true}',
      });
      final result = transport.callOneInt('SetPort', 8080);
      expect(result['success'], true);
    });

    test('callOneArgOneInt decodes response', () {
      final transport = _StubTransport({
        'Configure': '{"success": true}',
      });
      final result = transport.callOneArgOneInt('Configure', 'mode', 1);
      expect(result['success'], true);
    });

    test('callThreeInts decodes response', () {
      final transport = _StubTransport({
        'SetWindow': '{"success": true}',
      });
      final result = transport.callThreeInts('SetWindow', 100, 200, 300);
      expect(result['success'], true);
    });

    test('callRawOneArgAsync falls back to sync by default', () async {
      final transport = _StubTransport({
        'TestAsync': '{"success": true, "data": "async_result"}',
      });
      final raw = await transport.callRawOneArgAsync('TestAsync', 'arg');
      final decoded = jsonDecode(raw);
      expect(decoded['data'], 'async_result');
    });

    test('callOneArgAsync decodes envelope', () async {
      final transport = _StubTransport({
        'TestAsync': '{"success": true, "data": {"status": "ok"}}',
      });
      final result = await transport.callOneArgAsync('TestAsync', 'arg');
      expect(result['success'], true);
      expect(result['data']['status'], 'ok');
    });
  });
}
