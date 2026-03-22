class Asset {
  final String id;
  final String sourcePlugin;
  final String name;
  final String type;
  final String version;
  final List<int> ports;
  final String serviceName;
  final List<String> processPaths;
  final Map<String, String> metadata;
  final List<DisplaySection> displaySections;

  Asset({
    required this.id,
    required this.sourcePlugin,
    required this.name,
    required this.type,
    required this.version,
    required this.ports,
    required this.serviceName,
    required this.processPaths,
    required this.metadata,
    required this.displaySections,
  });

  factory Asset.fromJson(Map<String, dynamic> json) {
    return Asset(
      id: json['id'] ?? '',
      sourcePlugin: json['source_plugin'] ?? '',
      name: json['name'] ?? '',
      type: json['type'] ?? '',
      version: json['version'] ?? '',
      ports: (json['ports'] as List?)?.cast<int>() ?? [],
      serviceName: json['service_name'] ?? '',
      processPaths: (json['process_paths'] as List?)?.cast<String>() ?? [],
      metadata: (json['metadata'] as Map?)?.cast<String, String>() ?? {},
      displaySections:
          (json['display_sections'] as List?)
              ?.map((s) => DisplaySection.fromJson(s as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'source_plugin': sourcePlugin,
      'name': name,
      'type': type,
      'version': version,
      'ports': ports,
      'service_name': serviceName,
      'process_paths': processPaths,
      'metadata': metadata,
      'display_sections': displaySections.map((s) => s.toJson()).toList(),
    };
  }
}

class DisplaySection {
  final String title;
  final String icon;
  final List<DisplayItem> items;

  DisplaySection({
    required this.title,
    required this.icon,
    required this.items,
  });

  factory DisplaySection.fromJson(Map<String, dynamic> json) {
    return DisplaySection(
      title: json['title'] ?? '',
      icon: json['icon'] ?? '',
      items:
          (json['items'] as List?)
              ?.map((i) => DisplayItem.fromJson(i as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'title': title,
      'icon': icon,
      'items': items.map((i) => i.toJson()).toList(),
    };
  }
}

class DisplayItem {
  final String label;
  final String value;
  final String status;

  DisplayItem({required this.label, required this.value, required this.status});

  factory DisplayItem.fromJson(Map<String, dynamic> json) {
    return DisplayItem(
      label: json['label'] ?? '',
      value: json['value'] ?? '',
      status: json['status'] ?? 'neutral',
    );
  }

  Map<String, dynamic> toJson() {
    return {'label': label, 'value': value, 'status': status};
  }
}
