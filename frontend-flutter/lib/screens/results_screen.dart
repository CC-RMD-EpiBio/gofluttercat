import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../models/summary.dart';
import '../providers/assessment_meta_provider.dart';
import '../providers/assessment_provider.dart';
import '../providers/session_provider.dart';
import '../services/api_client.dart';
import '../widgets/error_banner.dart';
import '../widgets/score_card.dart';
import 'home_screen.dart';

class ResultsScreen extends StatefulWidget {
  const ResultsScreen({super.key});

  @override
  State<ResultsScreen> createState() => _ResultsScreenState();
}

class _ResultsScreenState extends State<ResultsScreen> {
  late Future<Summary> _summaryFuture;

  @override
  void initState() {
    super.initState();
    _loadSummary();
  }

  void _loadSummary() {
    final sessionId = context.read<SessionProvider>().currentSessionId;
    if (sessionId != null) {
      _summaryFuture = context.read<ApiClient>().getSummary(sessionId);
    }
  }

  void _startOver() {
    context.read<SessionProvider>().endSession();
    context.read<AssessmentProvider>().reset();
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(builder: (_) => const HomeScreen()),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Results'),
        centerTitle: true,
        automaticallyImplyLeading: false,
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 700),
          child: FutureBuilder<Summary>(
            future: _summaryFuture,
            builder: (context, snapshot) {
              if (snapshot.connectionState == ConnectionState.waiting) {
                return const Center(child: CircularProgressIndicator());
              }

              if (snapshot.hasError) {
                return Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      ErrorBanner(
                        message: 'Failed to load results: ${snapshot.error}',
                        onRetry: () {
                          setState(() {
                            _loadSummary();
                          });
                        },
                      ),
                      const SizedBox(height: 24),
                      OutlinedButton.icon(
                        onPressed: _startOver,
                        icon: const Icon(Icons.home),
                        label: const Text('Return Home'),
                      ),
                    ],
                  ),
                );
              }

              final summary = snapshot.data!;
              final scores = summary.scores;

              return SingleChildScrollView(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Icon(
                      Icons.check_circle,
                      size: 56,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'Assessment Complete',
                      style: theme.textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '${summary.session.responses.length} questions answered',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 24),
                    ...scores.entries.map(
                      (entry) {
                        final meta = context
                            .read<AssessmentMetaProvider>()
                            .meta;
                        final displayName =
                            meta?.scaleDisplayName(entry.key) ??
                                entry.key;
                        return ScoreCard(
                          scaleName: displayName,
                          score: entry.value,
                        );
                      },
                    ),
                    const SizedBox(height: 24),
                    Center(
                      child: FilledButton.icon(
                        onPressed: _startOver,
                        icon: const Icon(Icons.replay),
                        label: const Text('Start New Assessment'),
                      ),
                    ),
                    const SizedBox(height: 24),
                  ],
                ),
              );
            },
          ),
        ),
      ),
    );
  }
}
