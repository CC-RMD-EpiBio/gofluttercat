class NetworkException implements Exception {
  final String message;
  NetworkException([this.message = 'Network error']);

  @override
  String toString() => 'NetworkException: $message';
}

class SessionExpiredException implements Exception {
  final String message;
  SessionExpiredException([this.message = 'Session expired or not found']);

  @override
  String toString() => 'SessionExpiredException: $message';
}

class NoItemsRemainingException implements Exception {
  final String message;
  NoItemsRemainingException([this.message = 'No items remaining']);

  @override
  String toString() => 'NoItemsRemainingException: $message';
}

class ApiException implements Exception {
  final int statusCode;
  final String message;
  ApiException(this.statusCode, [this.message = 'API error']);

  @override
  String toString() => 'ApiException($statusCode): $message';
}
