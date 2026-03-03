class Choice {
  final String text;
  final int value;

  Choice({required this.text, required this.value});

  factory Choice.fromJson(Map<String, dynamic> json) {
    return Choice(
      text: json['text'] as String,
      value: (json['value'] as num).toInt(),
    );
  }
}

class AssessmentItem {
  final String name;
  final String question;
  final double version;
  final Map<String, Choice> responses;

  AssessmentItem({
    required this.name,
    required this.question,
    required this.version,
    required this.responses,
  });

  factory AssessmentItem.fromJson(Map<String, dynamic> json) {
    final responsesJson = json['responses'] as Map<String, dynamic>;
    final responses = responsesJson.map(
      (key, value) => MapEntry(
        key,
        Choice.fromJson(value as Map<String, dynamic>),
      ),
    );
    return AssessmentItem(
      name: json['name'] as String,
      question: json['question'] as String,
      version: (json['version'] as num).toDouble(),
      responses: responses,
    );
  }

  /// Choices sorted by presentation order (map key 1-9), skip (key "0") last.
  /// We sort by key rather than value because reverse-scored items have
  /// inverted values while keeping the same presentation order.
  List<Choice> get sortedChoices {
    final entries = responses.entries.toList();
    entries.sort((a, b) {
      final aKey = int.tryParse(a.key) ?? 0;
      final bKey = int.tryParse(b.key) ?? 0;
      if (aKey == 0) return 1;
      if (bKey == 0) return -1;
      return aKey.compareTo(bKey);
    });
    return entries.map((e) => e.value).toList();
  }

  /// Likert choices only (keys 1-9), no skip, sorted by presentation order.
  List<Choice> get likertChoices {
    final entries = responses.entries
        .where((e) => e.key != '0')
        .toList();
    entries.sort((a, b) {
      final aKey = int.tryParse(a.key) ?? 0;
      final bKey = int.tryParse(b.key) ?? 0;
      return aKey.compareTo(bKey);
    });
    return entries.map((e) => e.value).toList();
  }

  /// The skip choice (value == 0), if present
  Choice? get skipChoice {
    try {
      return responses.values.firstWhere((c) => c.value == 0);
    } catch (_) {
      return null;
    }
  }
}
