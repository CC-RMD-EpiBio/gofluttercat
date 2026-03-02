import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config.dart';
import '../models/item.dart';
import '../models/response.dart';
import '../models/session.dart';
import '../models/summary.dart';
import 'api_exceptions.dart';

class ApiClient {
  final http.Client _client;
  final String _baseUrl;

  ApiClient({http.Client? client, String? baseUrl})
      : _client = client ?? http.Client(),
        _baseUrl = baseUrl ?? apiBaseUrl;

  Uri _uri(String path) => Uri.parse('$_baseUrl$path');

  /// POST /session — create a new CAT session
  Future<Session> createSession() async {
    final response = await _request(() => _client.post(_uri('/session')));
    return Session.fromJson(jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// GET /session — list active session IDs
  Future<List<String>> getSessions() async {
    final response = await _request(() => _client.get(_uri('/session')));
    final list = jsonDecode(response.body) as List<dynamic>;
    return list.cast<String>();
  }

  /// GET /{sid}/item — get next item, returns null on 204 (no items remaining)
  Future<AssessmentItem?> getNextItem(String sessionId) async {
    final response = await _request(
      () => _client.get(_uri('/$sessionId/item')),
      allow204: true,
    );
    if (response.statusCode == 204) return null;
    return AssessmentItem.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// GET /{sid}/{scale}/item — get next item for specific scale
  Future<AssessmentItem?> getNextScaleItem(
      String sessionId, String scale) async {
    final response = await _request(
      () => _client.get(_uri('/$sessionId/$scale/item')),
      allow204: true,
    );
    if (response.statusCode == 204) return null;
    return AssessmentItem.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// POST /{sid}/response — submit a response
  Future<void> submitResponse(String sessionId, ItemResponse item) async {
    await _request(
      () => _client.post(
        _uri('/$sessionId/response'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode(item.toJson()),
      ),
    );
  }

  /// GET /{sid} — get session summary with scores
  Future<Summary> getSummary(String sessionId) async {
    final response =
        await _request(() => _client.get(_uri('/$sessionId')));
    return Summary.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  /// DELETE /{sid} — deactivate session
  Future<void> deleteSession(String sessionId) async {
    await _request(() => _client.delete(_uri('/$sessionId')));
  }

  Future<http.Response> _request(
    Future<http.Response> Function() fn, {
    bool allow204 = false,
  }) async {
    final http.Response response;
    try {
      response = await fn();
    } on http.ClientException catch (e) {
      throw NetworkException(e.message);
    } catch (e) {
      if (e is NetworkException ||
          e is SessionExpiredException ||
          e is ApiException) {
        rethrow;
      }
      throw NetworkException(e.toString());
    }

    if (allow204 && response.statusCode == 204) return response;

    if (response.statusCode == 404) {
      throw SessionExpiredException();
    }

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(response.statusCode, response.body);
    }

    return response;
  }
}
