import 'response.dart';
import 'score.dart';
import 'session.dart';

class SessionSummary {
  final String sessionId;
  final DateTime startTime;
  final DateTime expirationTime;
  final List<ItemResponse> responses;

  SessionSummary({
    required this.sessionId,
    required this.startTime,
    required this.expirationTime,
    required this.responses,
  });

  factory SessionSummary.fromJson(Map<String, dynamic> json) {
    return SessionSummary(
      sessionId: json['session_id'] as String,
      startTime: parseGoTime(json['start_time'] as String),
      expirationTime: parseGoTime(json['expiration_time'] as String),
      responses: (json['responses'] as List<dynamic>?)
              ?.map((e) => ItemResponse.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

class Summary {
  final Map<String, ScoreSummary> scores;
  final SessionSummary session;

  Summary({required this.scores, required this.session});

  factory Summary.fromJson(Map<String, dynamic> json) {
    final scoresJson = json['scores'] as Map<String, dynamic>;
    final scores = scoresJson.map(
      (key, value) => MapEntry(
        key,
        ScoreSummary.fromJson(value as Map<String, dynamic>),
      ),
    );
    return Summary(
      scores: scores,
      session: SessionSummary.fromJson(json['session'] as Map<String, dynamic>),
    );
  }
}
