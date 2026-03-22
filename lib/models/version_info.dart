class VersionInfo {
  final String version;
  final String downloadUrl;
  final String hash;
  final bool forceUpdate;
  final String changeLog;

  VersionInfo({
    required this.version,
    required this.downloadUrl,
    required this.hash,
    required this.forceUpdate,
    required this.changeLog,
  });

  factory VersionInfo.fromJson(Map<String, dynamic> json) {
    return VersionInfo(
      version: json['version'] ?? '',
      downloadUrl: json['download_url'] ?? '',
      hash: json['hash'] ?? '',
      forceUpdate: json['force_update'] ?? false,
      changeLog: json['change_log'] ?? '',
    );
  }
}
