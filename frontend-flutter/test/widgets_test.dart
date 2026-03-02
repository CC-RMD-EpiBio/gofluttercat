import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:cat_app/models/item.dart';
import 'package:cat_app/models/score.dart';
import 'package:cat_app/widgets/error_banner.dart';
import 'package:cat_app/widgets/likert_scale.dart';
import 'package:cat_app/widgets/loading_overlay.dart';
import 'package:cat_app/widgets/progress_indicator.dart';
import 'package:cat_app/widgets/score_card.dart';

Widget _wrap(Widget child) {
  return MaterialApp(home: Scaffold(body: child));
}

AssessmentItem _testItem({bool withSkip = true}) {
  final responses = <String, dynamic>{
    'a': {'text': 'Strongly disagree', 'value': 1},
    'b': {'text': 'Disagree', 'value': 2},
    'c': {'text': 'Neutral', 'value': 3},
    'd': {'text': 'Agree', 'value': 4},
    'e': {'text': 'Strongly agree', 'value': 5},
  };
  if (withSkip) {
    responses['skip'] = {'text': 'Prefer not to answer', 'value': 0};
  }
  return AssessmentItem.fromJson({
    'name': 'test_item',
    'question': 'Test question?',
    'version': 1.0,
    'responses': responses,
  });
}

void main() {
  group('LikertScale', () {
    testWidgets('renders all likert choices', (tester) async {
      await tester.pumpWidget(_wrap(
        LikertScale(
          item: _testItem(),
          onSelected: (_) {},
        ),
      ));

      expect(find.text('Strongly disagree'), findsOneWidget);
      expect(find.text('Disagree'), findsOneWidget);
      expect(find.text('Neutral'), findsOneWidget);
      expect(find.text('Agree'), findsOneWidget);
      expect(find.text('Strongly agree'), findsOneWidget);
      expect(find.text('Prefer not to answer'), findsOneWidget);
    });

    testWidgets('tapping a choice calls onSelected with value',
        (tester) async {
      int? selectedValue;
      await tester.pumpWidget(_wrap(
        LikertScale(
          item: _testItem(),
          onSelected: (v) => selectedValue = v,
        ),
      ));

      await tester.tap(find.text('Neutral'));
      expect(selectedValue, 3);
    });

    testWidgets('tapping skip calls onSelected with 0', (tester) async {
      int? selectedValue;
      await tester.pumpWidget(_wrap(
        LikertScale(
          item: _testItem(),
          onSelected: (v) => selectedValue = v,
        ),
      ));

      await tester.tap(find.text('Prefer not to answer'));
      expect(selectedValue, 0);
    });

    testWidgets('disabled state prevents taps', (tester) async {
      int? selectedValue;
      await tester.pumpWidget(_wrap(
        LikertScale(
          item: _testItem(),
          enabled: false,
          onSelected: (v) => selectedValue = v,
        ),
      ));

      await tester.tap(find.text('Neutral'));
      expect(selectedValue, isNull);
    });

    testWidgets('no skip option when absent', (tester) async {
      await tester.pumpWidget(_wrap(
        LikertScale(
          item: _testItem(withSkip: false),
          onSelected: (_) {},
        ),
      ));

      expect(find.text('Prefer not to answer'), findsNothing);
    });
  });

  group('ErrorBanner', () {
    testWidgets('displays message', (tester) async {
      await tester.pumpWidget(_wrap(
        const ErrorBanner(message: 'Something broke'),
      ));

      expect(find.text('Something broke'), findsOneWidget);
    });

    testWidgets('shows retry button when onRetry provided', (tester) async {
      bool retried = false;
      await tester.pumpWidget(_wrap(
        ErrorBanner(
          message: 'Error',
          onRetry: () => retried = true,
        ),
      ));

      expect(find.text('Retry'), findsOneWidget);
      await tester.tap(find.text('Retry'));
      expect(retried, isTrue);
    });

    testWidgets('hides retry button when onRetry is null', (tester) async {
      await tester.pumpWidget(_wrap(
        const ErrorBanner(message: 'Error'),
      ));

      expect(find.text('Retry'), findsNothing);
    });
  });

  group('LoadingOverlay', () {
    testWidgets('shows spinner when loading', (tester) async {
      await tester.pumpWidget(_wrap(
        const LoadingOverlay(
          isLoading: true,
          child: Text('Content'),
        ),
      ));

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Content'), findsOneWidget);
    });

    testWidgets('hides spinner when not loading', (tester) async {
      await tester.pumpWidget(_wrap(
        const LoadingOverlay(
          isLoading: false,
          child: Text('Content'),
        ),
      ));

      expect(find.byType(CircularProgressIndicator), findsNothing);
      expect(find.text('Content'), findsOneWidget);
    });
  });

  group('AssessmentProgressIndicator', () {
    testWidgets('displays question count', (tester) async {
      await tester.pumpWidget(_wrap(
        const AssessmentProgressIndicator(questionsAnswered: 3),
      ));

      expect(find.text('Question 4 of ~12'), findsOneWidget);
      expect(find.byType(LinearProgressIndicator), findsOneWidget);
    });
  });

  group('ScoreCard', () {
    testWidgets('displays scale name and score chips', (tester) async {
      final score = ScoreSummary.fromJson({
        'mean': 1.23,
        'std': 0.45,
        'rb_mean': 1.2,
        'rb_std': 0.4,
        'deciles': [-0.5, 0.0, 0.5, 0.8, 1.2, 1.5, 1.8, 2.0, 2.5],
        'rb_deciles': <double>[],
      });

      await tester.pumpWidget(_wrap(
        ScoreCard(scaleName: 'physical_function', score: score),
      ));

      expect(find.text('Physical Function'), findsOneWidget);
      expect(find.text('Score: 1.23'), findsOneWidget);
      expect(find.textContaining('0.45'), findsOneWidget);
    });

    testWidgets('converts snake_case to Title Case', (tester) async {
      final score = ScoreSummary.fromJson({
        'mean': 0.0,
        'std': 1.0,
        'rb_mean': 0.0,
        'rb_std': 1.0,
        'deciles': <double>[],
        'rb_deciles': <double>[],
      });

      await tester.pumpWidget(_wrap(
        ScoreCard(scaleName: 'emotional_well_being', score: score),
      ));

      expect(find.text('Emotional Well Being'), findsOneWidget);
    });
  });
}
