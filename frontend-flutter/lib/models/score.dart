class ScoreSummary {
  final double mean;
  final double std;
  final double rbMean;
  final double rbStd;
  final List<double> deciles;
  final List<double> rbDeciles;

  ScoreSummary({
    required this.mean,
    required this.std,
    required this.rbMean,
    required this.rbStd,
    required this.deciles,
    required this.rbDeciles,
  });

  factory ScoreSummary.fromJson(Map<String, dynamic> json) {
    return ScoreSummary(
      mean: (json['mean'] as num).toDouble(),
      std: (json['std'] as num).toDouble(),
      rbMean: ((json['rb_mean'] ?? 0) as num).toDouble(),
      rbStd: ((json['rb_std'] ?? 0) as num).toDouble(),
      deciles: (json['deciles'] as List<dynamic>?)
              ?.map((e) => (e as num).toDouble())
              .toList() ??
          [],
      rbDeciles: (json['rb_deciles'] as List<dynamic>?)
              ?.map((e) => (e as num).toDouble())
              .toList() ??
          [],
    );
  }

  /// Median is the 5th decile (index 4, the 50th percentile)
  double get median => deciles.length > 4 ? deciles[4] : mean;

  /// Whether Rao-Blackwell marginalized scores are available
  bool get hasRb => rbDeciles.isNotEmpty && (rbMean != 0 || rbStd != 0);

  /// RB median is the 5th RB decile
  double get rbMedian => rbDeciles.length > 4 ? rbDeciles[4] : rbMean;
}
