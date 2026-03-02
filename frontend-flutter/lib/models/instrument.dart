class InstrumentInfo {
  final String id;
  final String name;
  final String description;
  final Map<String, String> scales;

  InstrumentInfo({
    required this.id,
    required this.name,
    required this.description,
    required this.scales,
  });

  factory InstrumentInfo.fromJson(Map<String, dynamic> json) {
    final scalesJson = json['scales'] as Map<String, dynamic>? ?? {};
    return InstrumentInfo(
      id: json['id'] as String,
      name: json['name'] as String,
      description: json['description'] as String,
      scales: scalesJson.map((k, v) => MapEntry(k, v as String)),
    );
  }
}
