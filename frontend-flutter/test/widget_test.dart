import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';

import 'package:cat_app/app.dart';
import 'package:cat_app/providers/assessment_meta_provider.dart';
import 'package:cat_app/providers/assessment_provider.dart';
import 'package:cat_app/providers/session_provider.dart';
import 'package:cat_app/services/api_client.dart';

void main() {
  testWidgets('App renders home screen', (WidgetTester tester) async {
    // Use a mock client so the metadata fetch doesn't hit a real server
    final apiClient = ApiClient(
      client: MockClient((_) async => http.Response('', 404)),
      baseUrl: 'http://test',
    );

    await tester.pumpWidget(
      MultiProvider(
        providers: [
          Provider<ApiClient>.value(value: apiClient),
          ChangeNotifierProvider(
            create: (_) => AssessmentMetaProvider(apiClient: apiClient),
          ),
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

    // Wait for async metadata fetch to complete
    await tester.pumpAndSettle();

    // Fallback title shown when metadata fetch fails
    expect(find.text('Computer Adaptive Testing'), findsOneWidget);
    expect(find.text('Start Assessment'), findsOneWidget);
  });
}
