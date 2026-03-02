import 'package:flutter_test/flutter_test.dart';
import 'package:provider/provider.dart';

import 'package:cat_app/app.dart';
import 'package:cat_app/providers/assessment_provider.dart';
import 'package:cat_app/providers/session_provider.dart';
import 'package:cat_app/services/api_client.dart';

void main() {
  testWidgets('App renders home screen', (WidgetTester tester) async {
    final apiClient = ApiClient();

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          Provider<ApiClient>.value(value: apiClient),
          ChangeNotifierProvider(
            create: (_) => SessionProvider(apiClient: apiClient),
          ),
          ChangeNotifierProvider(
            create: (_) => AssessmentProvider(apiClient: apiClient),
          ),
        ],
        child: const CatApp(),
      ),
    );

    expect(find.text('Computer Adaptive Testing'), findsOneWidget);
    expect(find.text('Start Assessment'), findsOneWidget);
  });
}
