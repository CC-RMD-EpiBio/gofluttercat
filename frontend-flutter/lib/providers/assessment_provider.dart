import 'package:flutter/foundation.dart';

import '../models/item.dart';
import '../models/response.dart';
import '../services/api_client.dart';
import '../services/api_exceptions.dart';

enum AssessmentStatus { idle, loading, presenting, submitting, complete, error }

class AssessmentProvider extends ChangeNotifier {
  final ApiClient _apiClient;

  AssessmentStatus _status = AssessmentStatus.idle;
  AssessmentItem? _currentItem;
  String? _errorMessage;
  int _questionsAnswered = 0;

  AssessmentProvider({required ApiClient apiClient}) : _apiClient = apiClient;

  AssessmentStatus get status => _status;
  AssessmentItem? get currentItem => _currentItem;
  String? get errorMessage => _errorMessage;
  int get questionsAnswered => _questionsAnswered;

  Future<void> fetchNextItem(String sessionId) async {
    _status = AssessmentStatus.loading;
    _errorMessage = null;
    notifyListeners();

    try {
      final item = await _apiClient.getNextItem(sessionId);
      if (item == null) {
        _status = AssessmentStatus.complete;
        _currentItem = null;
      } else {
        _currentItem = item;
        _status = AssessmentStatus.presenting;
      }
    } on SessionExpiredException {
      _errorMessage = 'Session expired or not found';
      _status = AssessmentStatus.error;
    } on NetworkException catch (e) {
      _errorMessage = e.message;
      _status = AssessmentStatus.error;
    } catch (e) {
      _errorMessage = e.toString();
      _status = AssessmentStatus.error;
    }
    notifyListeners();
  }

  Future<void> submitResponse(String sessionId, int value) async {
    if (_currentItem == null) return;

    _status = AssessmentStatus.submitting;
    notifyListeners();

    try {
      final itemResponse = ItemResponse(
        itemName: _currentItem!.name,
        value: value,
      );
      await _apiClient.submitResponse(sessionId, itemResponse);
      _questionsAnswered++;

      // Fetch next item
      final nextItem = await _apiClient.getNextItem(sessionId);
      if (nextItem == null) {
        _status = AssessmentStatus.complete;
        _currentItem = null;
      } else {
        _currentItem = nextItem;
        _status = AssessmentStatus.presenting;
      }
    } on SessionExpiredException {
      _errorMessage = 'Session expired or not found';
      _status = AssessmentStatus.error;
    } on NetworkException catch (e) {
      _errorMessage = e.message;
      _status = AssessmentStatus.error;
    } catch (e) {
      _errorMessage = e.toString();
      _status = AssessmentStatus.error;
    }
    notifyListeners();
  }

  void reset() {
    _status = AssessmentStatus.idle;
    _currentItem = null;
    _errorMessage = null;
    _questionsAnswered = 0;
    notifyListeners();
  }
}
