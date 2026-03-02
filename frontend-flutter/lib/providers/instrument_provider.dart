import 'package:flutter/foundation.dart';

import '../models/instrument.dart';
import '../services/api_client.dart';
import '../services/api_exceptions.dart';

enum InstrumentStatus { idle, loading, loaded, error }

class InstrumentProvider extends ChangeNotifier {
  final ApiClient _apiClient;

  InstrumentStatus _status = InstrumentStatus.idle;
  List<InstrumentInfo> _instruments = [];
  String? _selectedId;
  String? _errorMessage;

  InstrumentProvider({required ApiClient apiClient}) : _apiClient = apiClient;

  InstrumentStatus get status => _status;
  List<InstrumentInfo> get instruments => _instruments;
  String? get selectedId => _selectedId;
  String? get errorMessage => _errorMessage;

  InstrumentInfo? get selectedInstrument {
    if (_selectedId == null) return null;
    try {
      return _instruments.firstWhere((i) => i.id == _selectedId);
    } catch (_) {
      return null;
    }
  }

  void select(String id) {
    _selectedId = id;
    notifyListeners();
  }

  Future<void> fetch() async {
    if (_status == InstrumentStatus.loading) return;

    _status = InstrumentStatus.loading;
    _errorMessage = null;
    notifyListeners();

    try {
      _instruments = await _apiClient.getInstruments();
      _status = InstrumentStatus.loaded;
      // Auto-select first instrument if nothing selected
      if (_selectedId == null && _instruments.isNotEmpty) {
        _selectedId = _instruments.first.id;
      }
    } on NetworkException catch (e) {
      _errorMessage = e.message;
      _status = InstrumentStatus.error;
    } catch (e) {
      _errorMessage = e.toString();
      _status = InstrumentStatus.error;
    }
    notifyListeners();
  }
}
