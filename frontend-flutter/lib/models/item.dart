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

  /// Whether a choice is the explicit skip option (by text, not value).
  static bool _isSkip(Choice c) => c.text.toLowerCase() == 'skip';

  /// Scorable choices sorted by presentation order (map key).
  /// Excludes any explicit "skip" choice.
  List<Choice> get likertChoices {
    final entries = responses.entries
        .where((e) => !_isSkip(e.value))
        .toList();
    entries.sort((a, b) {
      final aKey = int.tryParse(a.key) ?? 0;
      final bKey = int.tryParse(b.key) ?? 0;
      return aKey.compareTo(bKey);
    });
    return entries.map((e) => e.value).toList();
  }

  /// The skip value to send to the backend.
  /// Uses the explicit skip choice's value if present, otherwise -1
  /// (which the backend's findIndex will skip).
  int get skipValue {
    try {
      return responses.values.firstWhere(_isSkip).value;
    } catch (_) {
      return -1;
    }
  }
}
