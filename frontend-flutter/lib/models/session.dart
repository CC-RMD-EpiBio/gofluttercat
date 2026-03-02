/// Parse Go's time.Time.String() format into a Dart DateTime.
///
/// Go produces: "2026-03-01 22:34:09.418399281 -0500 EST m=+138.479555656"
/// Steps:
///   1. Strip monotonic clock suffix (m=+...)
///   2. Strip timezone abbreviation (EST, UTC, etc.)
///   3. Insert 'T' between date and time
///   4. Insert colon into timezone offset (-0500 → -05:00)
DateTime parseGoTime(String s) {
  // Remove monotonic clock suffix
  var cleaned = s.replaceAll(RegExp(r'\s*m=[+-][\d.]+$'), '');
  // Remove timezone abbreviation
  cleaned = cleaned.replaceAll(RegExp(r'\s+[A-Z]{1,5}$'), '');
  // Trim and replace first space with T for ISO 8601
  cleaned = cleaned.trim();
  cleaned = cleaned.replaceFirst(' ', 'T');
  // Insert colon into bare offset: -0500 → -05:00, +0000 → +00:00
  cleaned = cleaned.replaceFirstMapped(
    RegExp(r'([+-])(\d{2})(\d{2})$'),
    (m) => '${m[1]}${m[2]}:${m[3]}',
  );
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
