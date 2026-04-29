import 'package:bot_sec_manager/models/skill_scan_result_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SkillSecurityIssue', () {
    test('serializes to and from JSON round-trip', () {
      final original = SkillSecurityIssue(
        type: 'command_injection',
        severity: 'high',
        file: 'SKILL.md',
        description: 'Executes arbitrary shell commands',
        evidence: 'os.system(user_input)',
      );

      final json = original.toJson();
      final restored = SkillSecurityIssue.fromJson(json);

      expect(restored.type, 'command_injection');
      expect(restored.severity, 'high');
      expect(restored.file, 'SKILL.md');
      expect(restored.description, 'Executes arbitrary shell commands');
      expect(restored.evidence, 'os.system(user_input)');
    });

    test('defaults to safe values on missing fields', () {
      final restored = SkillSecurityIssue.fromJson({});
      expect(restored.type, '');
      expect(restored.severity, 'medium');
      expect(restored.file, '');
      expect(restored.description, '');
      expect(restored.evidence, '');
    });
  });

  group('SkillAnalysisResult', () {
    test('serializes to and from JSON round-trip', () {
      final original = SkillAnalysisResult(
        safe: false,
        riskLevel: 'high',
        issues: [
          SkillSecurityIssue(
            type: 'path_traversal',
            severity: 'critical',
            file: 'main.py',
            description: 'Reads arbitrary files via path traversal',
            evidence: '../../../etc/passwd',
          ),
          SkillSecurityIssue(
            type: 'network_exfiltration',
            severity: 'high',
            file: 'helper.sh',
            description: 'Sends data to external endpoint',
          ),
        ],
        summary: 'Found 2 security issues',
        rawOutput: 'Detailed LLM analysis output',
      );

      final json = original.toJson();
      final restored = SkillAnalysisResult.fromJson(json);

      expect(restored.safe, false);
      expect(restored.riskLevel, 'high');
      expect(restored.issues.length, 2);
      expect(restored.issues[0].type, 'path_traversal');
      expect(restored.issues[1].type, 'network_exfiltration');
      expect(restored.summary, 'Found 2 security issues');
      expect(restored.rawOutput, 'Detailed LLM analysis output');
    });

    test('defaults to safe with empty issues', () {
      final restored = SkillAnalysisResult.fromJson({});
      expect(restored.safe, false);
      expect(restored.riskLevel, 'unknown');
      expect(restored.issues, isEmpty);
      expect(restored.summary, '');
      expect(restored.rawOutput, '');
    });

    test('rawOutput not serialized when empty', () {
      final result = SkillAnalysisResult(safe: true, riskLevel: 'low');
      final json = result.toJson();
      expect(json.containsKey('raw_output'), false);
    });

    test('rawOutput serialized when non-empty', () {
      final result = SkillAnalysisResult(safe: true, riskLevel: 'low', rawOutput: 'data');
      final json = result.toJson();
      expect(json['raw_output'], 'data');
    });
  });
}
