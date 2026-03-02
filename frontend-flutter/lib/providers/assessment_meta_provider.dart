import 'package:flutter/foundation.dart';

import '../models/assessment_meta.dart';
import '../services/api_client.dart';
import '../services/api_exceptions.dart';

enum MetaStatus { idle, loading, loaded, error }

class AssessmentMetaProvider extends ChangeNotifier {
  final ApiClient _apiClient;

  MetaStatus _status = MetaStatus.idle;
  AssessmentMeta? _meta;
  String? _errorMessage;

  AssessmentMetaProvider({required ApiClient apiClient})
      : _apiClient = apiClient;

  MetaStatus get status => _status;
  AssessmentMeta? get meta => _meta;
  String? get errorMessage => _errorMessage;

  Future<void> fetch({String? instrument}) async {
    if (_status == MetaStatus.loading) return;

    _status = MetaStatus.loading;
    _errorMessage = null;
    notifyListeners();

    try {
      _meta = await _apiClient.getAssessmentMeta(instrument: instrument);
      _status = MetaStatus.loaded;
    } on NetworkException catch (e) {
      _errorMessage = e.message;
      _status = MetaStatus.error;
    } catch (e) {
      _errorMessage = e.toString();
      _status = MetaStatus.error;
    }
    notifyListeners();
  }
}
