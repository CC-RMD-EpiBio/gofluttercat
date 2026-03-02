import 'package:flutter_test/flutter_test.dart';
import 'package:cat_app/models/assessment_meta.dart';
import 'package:cat_app/models/item.dart';
import 'package:cat_app/models/response.dart';
import 'package:cat_app/models/score.dart';
import 'package:cat_app/models/session.dart';
import 'package:cat_app/models/summary.dart';

void main() {
  group('Session', () {
    test('fromJson parses correctly', () {
      final json = {
        'session_id': 'catsession:abc-123',
        'start_time': '2026-03-01T12:00:00-05:00',
        'expiration_time': '2026-03-02T12:00:00-05:00',
      };
      final session = Session.fromJson(json);
      expect(session.sessionId, 'catsession:abc-123');
      expect(session.startTime.year, 2026);
    });

    test('fromJson strips timezone name suffix', () {
      final json = {
        'session_id': 'catsession:abc-123',
        'start_time': '2026-03-01T12:00:00-05:00 EST',
        'expiration_time': '2026-03-02T12:00:00-05:00 EST',
      };
      final session = Session.fromJson(json);
      expect(session.sessionId, 'catsession:abc-123');
    });

    test('fromJson parses Go time.String() format with monotonic clock', () {
      final json = {
        'session_id': 'catsession:abc-123',
        'start_time':
            '2026-03-01 22:34:09.418399281 -0500 EST m=+138.479555656',
        'expiration_time': '2026-03-02 22:34:09.418399341 -0500 EST',
      };
      final session = Session.fromJson(json);
      expect(session.sessionId, 'catsession:abc-123');
      expect(session.startTime.year, 2026);
      expect(session.startTime.month, 3);
    });

    test('isExpired returns true for past expiration', () {
      final json = {
        'session_id': 'catsession:abc-123',
        'start_time': '2020-01-01T00:00:00Z',
        'expiration_time': '2020-01-02T00:00:00Z',
      };
      final session = Session.fromJson(json);
      expect(session.isExpired, isTrue);
    });
  });

  group('AssessmentItem', () {
    late Map<String, dynamic> itemJson;

    setUp(() {
      itemJson = {
        'name': 'item_001',
        'question': 'How do you feel today?',
        'version': 1.0,
        'responses': {
          'a': {'text': 'Very bad', 'value': 1},
          'b': {'text': 'Bad', 'value': 2},
          'c': {'text': 'Neutral', 'value': 3},
          'd': {'text': 'Good', 'value': 4},
          'e': {'text': 'Very good', 'value': 5},
          'skip': {'text': 'Prefer not to answer', 'value': 0},
        },
      };
    });

    test('fromJson parses correctly', () {
      final item = AssessmentItem.fromJson(itemJson);
      expect(item.name, 'item_001');
      expect(item.question, 'How do you feel today?');
      expect(item.version, 1.0);
      expect(item.responses.length, 6);
    });

    test('sortedChoices puts skip last', () {
      final item = AssessmentItem.fromJson(itemJson);
      final sorted = item.sortedChoices;
      expect(sorted.last.value, 0);
      expect(sorted.first.value, 1);
    });

    test('likertChoices excludes skip', () {
      final item = AssessmentItem.fromJson(itemJson);
      expect(item.likertChoices.length, 5);
      expect(item.likertChoices.every((c) => c.value != 0), isTrue);
    });

    test('skipChoice returns the skip option', () {
      final item = AssessmentItem.fromJson(itemJson);
      expect(item.skipChoice, isNotNull);
      expect(item.skipChoice!.value, 0);
    });

    test('skipChoice returns null when no skip', () {
      itemJson['responses'] = {
        'a': {'text': 'Yes', 'value': 1},
        'b': {'text': 'No', 'value': 2},
      };
      final item = AssessmentItem.fromJson(itemJson);
      expect(item.skipChoice, isNull);
    });
  });

  group('ItemResponse', () {
    test('toJson serializes correctly', () {
      final response = ItemResponse(itemName: 'item_001', value: 3);
      final json = response.toJson();
      expect(json['item_name'], 'item_001');
      expect(json['value'], 3);
    });

    test('fromJson parses correctly', () {
      final json = {'item_name': 'item_002', 'value': 5};
      final response = ItemResponse.fromJson(json);
      expect(response.itemName, 'item_002');
      expect(response.value, 5);
    });
  });

  group('ScoreSummary', () {
    test('fromJson parses all fields', () {
      final json = {
        'mean': 0.5,
        'std': 0.8,
        'rb_mean': 0.4,
        'rb_std': 0.9,
        'deciles': [
          -1.5, -1.0, -0.5, 0.0, 0.5, 1.0, 1.5, 2.0, 2.5
        ],
        'rb_deciles': [
          -1.4, -0.9, -0.4, 0.1, 0.6, 1.1, 1.6, 2.1, 2.6
        ],
      };
      final score = ScoreSummary.fromJson(json);
      expect(score.mean, 0.5);
      expect(score.std, 0.8);
      expect(score.rbMean, 0.4);
      expect(score.rbStd, 0.9);
      expect(score.deciles.length, 9);
      expect(score.rbDeciles.length, 9);
    });

    test('median returns 5th decile', () {
      final json = {
        'mean': 0.5,
        'std': 0.8,
        'rb_mean': 0.4,
        'rb_std': 0.9,
        'deciles': [
          -1.5, -1.0, -0.5, 0.0, 0.5, 1.0, 1.5, 2.0, 2.5
        ],
        'rb_deciles': <double>[],
      };
      final score = ScoreSummary.fromJson(json);
      expect(score.median, 0.5);
    });

    test('median falls back to mean when deciles too short', () {
      final json = {
        'mean': 0.5,
        'std': 0.8,
        'rb_mean': 0.4,
        'rb_std': 0.9,
        'deciles': <double>[],
        'rb_deciles': <double>[],
      };
      final score = ScoreSummary.fromJson(json);
      expect(score.median, 0.5);
    });
  });

  group('Summary', () {
    test('fromJson parses scores and session', () {
      final json = {
        'scores': {
          'anxiety': {
            'mean': 0.5,
            'std': 0.8,
            'rb_mean': 0.4,
            'rb_std': 0.9,
            'deciles': [-1.0, 0.0, 0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5],
            'rb_deciles': [-0.9, 0.1, 0.6, 1.1, 1.6, 2.1, 2.6, 3.1, 3.6],
          },
        },
        'session': {
          'session_id': 'catsession:test-123',
          'start_time': '2026-03-01T12:00:00Z',
          'expiration_time': '2026-03-02T12:00:00Z',
          'responses': [
            {'item_name': 'item_001', 'value': 3},
          ],
        },
      };
      final summary = Summary.fromJson(json);
      expect(summary.scores.containsKey('anxiety'), isTrue);
      expect(summary.session.sessionId, 'catsession:test-123');
      expect(summary.session.responses.length, 1);
    });

    test('fromJson handles null responses', () {
      final json = {
        'scores': <String, dynamic>{},
        'session': {
          'session_id': 'catsession:test-123',
          'start_time': '2026-03-01T12:00:00Z',
          'expiration_time': '2026-03-02T12:00:00Z',
          'responses': null,
        },
      };
      final summary = Summary.fromJson(json);
      expect(summary.session.responses, isEmpty);
    });
  });

  group('AssessmentMeta', () {
    test('fromJson parses all fields', () {
      final json = {
        'name': 'Test Battery',
        'description': 'A test assessment.',
        'scales': {
          'A': 'Factor A',
          'B': 'Factor B',
        },
        'cat_config': {
          'stopping_std': 0.33,
          'stopping_num_items': 12,
          'minimum_num_items': 4,
        },
      };
      final meta = AssessmentMeta.fromJson(json);
      expect(meta.name, 'Test Battery');
      expect(meta.description, 'A test assessment.');
      expect(meta.scales.length, 2);
      expect(meta.scales['A'], 'Factor A');
      expect(meta.catConfig.stoppingStd, 0.33);
      expect(meta.catConfig.stoppingNumItems, 12);
      expect(meta.catConfig.minimumNumItems, 4);
    });

    test('maxTotalItems multiplies scales by stoppingNumItems', () {
      final meta = AssessmentMeta.fromJson({
        'name': 'Test',
        'description': 'Desc',
        'scales': {'A': 'A', 'B': 'B', 'C': 'C'},
        'cat_config': {
          'stopping_std': 0.5,
          'stopping_num_items': 10,
          'minimum_num_items': 3,
        },
      });
      expect(meta.maxTotalItems, 30);
    });

    test('scaleDisplayName falls back to key', () {
      final meta = AssessmentMeta.fromJson({
        'name': 'Test',
        'description': 'Desc',
        'scales': {'A': 'Factor A'},
        'cat_config': {
          'stopping_std': 0.5,
          'stopping_num_items': 10,
          'minimum_num_items': 3,
        },
      });
      expect(meta.scaleDisplayName('A'), 'Factor A');
      expect(meta.scaleDisplayName('unknown'), 'unknown');
    });
  });
}
