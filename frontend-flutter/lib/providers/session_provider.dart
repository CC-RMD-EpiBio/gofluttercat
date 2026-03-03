import 'package:flutter/foundation.dart';

import '../models/session.dart';
import '../services/api_client.dart';
import '../services/api_exceptions.dart';

enum SessionStatus { idle, creating, active, expired, error }

class SessionProvider extends ChangeNotifier {
  final ApiClient _apiClient;

  SessionStatus _status = SessionStatus.idle;
  Session? _session;
  String? _errorMessage;

  SessionProvider({required ApiClient apiClient}) : _apiClient = apiClient;

  SessionStatus get status => _status;
  Session? get session => _session;
  String? get currentSessionId => _session?.sessionId;
  String? get errorMessage => _errorMessage;

  Future<void> createSession({
    String? instrument,
    int? stoppingNumItems,
    double? stoppingStd,
  }) async {
    _status = SessionStatus.creating;
    _errorMessage = null;
    notifyListeners();

    try {
      _session = await _apiClient.createSession(
        instrument: instrument,
        stoppingNumItems: stoppingNumItems,
        stoppingStd: stoppingStd,
      );
      _status = SessionStatus.active;
    } on NetworkException catch (e) {
      _errorMessage = e.message;
      _status = SessionStatus.error;
    } catch (e) {
      _errorMessage = e.toString();
      _status = SessionStatus.error;
    }
    notifyListeners();
  }

  void markExpired() {
    _status = SessionStatus.expired;
    notifyListeners();
  }

  Future<void> endSession() async {
    if (_session != null) {
      try {
        await _apiClient.deleteSession(_session!.sessionId);
      } on NetworkException {
        // Best-effort cleanup; ignore failures
      } catch (_) {
        // ignore
      }
    }
    reset();
  }

  void reset() {
    _status = SessionStatus.idle;
    _session = null;
    _errorMessage = null;
    notifyListeners();
  }
}
