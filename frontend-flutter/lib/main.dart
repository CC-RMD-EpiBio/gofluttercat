import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'providers/assessment_meta_provider.dart';
import 'providers/assessment_provider.dart';
import 'providers/session_provider.dart';
import 'services/api_client.dart';

void main() {
  final apiClient = ApiClient();

  runApp(
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
}
