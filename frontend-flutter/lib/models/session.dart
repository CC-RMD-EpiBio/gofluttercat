/// Parse Go's time.Time string format, stripping timezone name suffix
/// e.g. "2026-03-01T12:00:00.123456789-05:00 EST" → valid ISO 8601
DateTime parseGoTime(String s) {
  final cleaned = s.replaceAll(RegExp(r'\s+[A-Z]{1,5}$'), '');
  return DateTime.parse(cleaned);
}

class Session {
  final String sessionId;
  final DateTime startTime;
  final DateTime expirationTime;

  Session({
    required this.sessionId,
    required this.startTime,
    required this.expirationTime,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    return Session(
      sessionId: json['session_id'] as String,
      startTime: parseGoTime(json['start_time'] as String),
      expirationTime: parseGoTime(json['expiration_time'] as String),
    );
  }

  bool get isExpired => DateTime.now().isAfter(expirationTime);
}
