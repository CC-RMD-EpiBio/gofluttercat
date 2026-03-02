import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:cat_app/providers/assessment_provider.dart';
import 'package:cat_app/providers/session_provider.dart';
import 'package:cat_app/services/api_client.dart';

ApiClient _mockClient(
    http.Response Function(http.Request) handler) {
  return ApiClient(
    client: MockClient(
        (request) async => handler(request)),
    baseUrl: 'http://test',
  );
}

final _sessionResponse = jsonEncode({
  'session_id': 'catsession:test-123',
  'start_time': '2026-03-01T12:00:00Z',
  'expiration_time': '2026-03-02T12:00:00Z',
});

final _itemResponse = jsonEncode({
  'name': 'item_001',
  'question': 'How do you feel?',
  'version': 1.0,
  'responses': {
    'a': {'text': 'Bad', 'value': 1},
    'b': {'text': 'Good', 'value': 2},
  },
});

void main() {
  group('SessionProvider', () {
    test('initial state is idle', () {
      final api = _mockClient((_) => http.Response('', 200));
      final provider = SessionProvider(apiClient: api);
      expect(provider.status, SessionStatus.idle);
      expect(provider.session, isNull);
      expect(provider.errorMessage, isNull);
    });

    test('createSession transitions to active on success', () async {
      final api = _mockClient((req) {
        if (req.method == 'POST' && req.url.path == '/session') {
          return http.Response(_sessionResponse, 200);
        }
        return http.Response('', 404);
      });

      final provider = SessionProvider(apiClient: api);
      await provider.createSession();
      expect(provider.status, SessionStatus.active);
      expect(provider.session, isNotNull);
      expect(provider.currentSessionId, 'catsession:test-123');
    });

    test('createSession transitions to error on failure', () async {
      final api = _mockClient((_) => http.Response('server error', 500));
      final provider = SessionProvider(apiClient: api);
      await provider.createSession();
      expect(provider.status, SessionStatus.error);
      expect(provider.errorMessage, isNotNull);
    });

    test('reset clears state', () async {
      final api = _mockClient(
          (_) => http.Response(_sessionResponse, 200));
      final provider = SessionProvider(apiClient: api);
      await provider.createSession();
      expect(provider.status, SessionStatus.active);

      provider.reset();
      expect(provider.status, SessionStatus.idle);
      expect(provider.session, isNull);
    });

    test('markExpired sets status', () {
      final api = _mockClient((_) => http.Response('', 200));
      final provider = SessionProvider(apiClient: api);
      provider.markExpired();
      expect(provider.status, SessionStatus.expired);
    });
  });

  group('AssessmentProvider', () {
    test('initial state is idle', () {
      final api = _mockClient((_) => http.Response('', 200));
      final provider = AssessmentProvider(apiClient: api);
      expect(provider.status, AssessmentStatus.idle);
      expect(provider.currentItem, isNull);
      expect(provider.questionsAnswered, 0);
    });

    test('fetchNextItem transitions to presenting', () async {
      final api = _mockClient((req) {
        if (req.url.path.endsWith('/item')) {
          return http.Response(_itemResponse, 200);
        }
        return http.Response('', 404);
      });

      final provider = AssessmentProvider(apiClient: api);
      await provider.fetchNextItem('catsession:test-123');
      expect(provider.status, AssessmentStatus.presenting);
      expect(provider.currentItem, isNotNull);
      expect(provider.currentItem!.name, 'item_001');
    });

    test('fetchNextItem transitions to complete on 204', () async {
      final api = _mockClient((_) => http.Response('', 204));

      final provider = AssessmentProvider(apiClient: api);
      await provider.fetchNextItem('catsession:test-123');
      expect(provider.status, AssessmentStatus.complete);
      expect(provider.currentItem, isNull);
    });

    test('submitResponse increments count and fetches next', () async {
      int callCount = 0;
      final api = _mockClient((req) {
        if (req.method == 'POST' && req.url.path.endsWith('/response')) {
          return http.Response('', 200);
        }
        if (req.method == 'GET' && req.url.path.endsWith('/item')) {
          callCount++;
          if (callCount <= 2) {
            return http.Response(_itemResponse, 200);
          }
          return http.Response('', 204);
        }
        return http.Response('', 404);
      });

      final provider = AssessmentProvider(apiClient: api);
      // First, get an item
      await provider.fetchNextItem('catsession:test-123');
      expect(provider.questionsAnswered, 0);

      // Submit response — should fetch next item
      await provider.submitResponse('catsession:test-123', 2);
      expect(provider.questionsAnswered, 1);
      expect(provider.status, AssessmentStatus.presenting);

      // Submit again — this time server returns 204 (complete)
      await provider.submitResponse('catsession:test-123', 1);
      expect(provider.questionsAnswered, 2);
      expect(provider.status, AssessmentStatus.complete);
    });

    test('fetchNextItem transitions to error on network failure', () async {
      final api = _mockClient((_) => http.Response('error', 500));

      final provider = AssessmentProvider(apiClient: api);
      await provider.fetchNextItem('catsession:test-123');
      expect(provider.status, AssessmentStatus.error);
      expect(provider.errorMessage, isNotNull);
    });

    test('reset clears all state', () async {
      final api = _mockClient((req) {
        if (req.url.path.endsWith('/item')) {
          return http.Response(_itemResponse, 200);
        }
        return http.Response('', 200);
      });

      final provider = AssessmentProvider(apiClient: api);
      await provider.fetchNextItem('catsession:test-123');
      await provider.submitResponse('catsession:test-123', 2);

      provider.reset();
      expect(provider.status, AssessmentStatus.idle);
      expect(provider.currentItem, isNull);
      expect(provider.questionsAnswered, 0);
    });
  });
}
