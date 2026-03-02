class CatConfigMeta {
  final double stoppingStd;
  final int stoppingNumItems;
  final int minimumNumItems;

  CatConfigMeta({
    required this.stoppingStd,
    required this.stoppingNumItems,
    required this.minimumNumItems,
  });

  factory CatConfigMeta.fromJson(Map<String, dynamic> json) {
    return CatConfigMeta(
      stoppingStd: (json['stopping_std'] as num).toDouble(),
      stoppingNumItems: (json['stopping_num_items'] as num).toInt(),
      minimumNumItems: (json['minimum_num_items'] as num).toInt(),
    );
  }
}

class AssessmentMeta {
  final String name;
  final String description;
  final Map<String, String> scales;
  final CatConfigMeta catConfig;

  AssessmentMeta({
    required this.name,
    required this.description,
    required this.scales,
    required this.catConfig,
  });

  factory AssessmentMeta.fromJson(Map<String, dynamic> json) {
    final scalesJson = json['scales'] as Map<String, dynamic>;
    return AssessmentMeta(
      name: json['name'] as String,
      description: json['description'] as String,
      scales: scalesJson.map((k, v) => MapEntry(k, v as String)),
      catConfig:
          CatConfigMeta.fromJson(json['cat_config'] as Map<String, dynamic>),
    );
  }

  /// Total max items across all scales
  int get maxTotalItems => scales.length * catConfig.stoppingNumItems;

  /// Display name for a scale key, falling back to the key itself
  String scaleDisplayName(String key) => scales[key] ?? key;
}
