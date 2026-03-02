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

  /// Choices sorted: 1-9 ascending, then skip (0) last
  List<Choice> get sortedChoices {
    final choices = responses.values.toList();
    choices.sort((a, b) {
      if (a.value == 0) return 1;
      if (b.value == 0) return -1;
      return a.value.compareTo(b.value);
    });
    return choices;
  }

  /// Likert choices only (1-9), no skip
  List<Choice> get likertChoices {
    return sortedChoices.where((c) => c.value != 0).toList();
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
