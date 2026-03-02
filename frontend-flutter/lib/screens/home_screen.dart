import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/assessment_provider.dart';
import '../providers/session_provider.dart';
import '../widgets/error_banner.dart';
import 'assessment_screen.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  Future<void> _startAssessment(BuildContext context) async {
    final sessionProvider = context.read<SessionProvider>();
    final assessmentProvider = context.read<AssessmentProvider>();

    await sessionProvider.createSession();

    if (!context.mounted) return;
    if (sessionProvider.status != SessionStatus.active) return;

    final sessionId = sessionProvider.currentSessionId!;
    await assessmentProvider.fetchNextItem(sessionId);

    if (!context.mounted) return;
    if (assessmentProvider.status == AssessmentStatus.presenting) {
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const AssessmentScreen()),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 600),
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Consumer<SessionProvider>(
              builder: (context, sessionProvider, _) {
                return Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      Icons.psychology,
                      size: 80,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(height: 24),
                    Text(
                      'Computer Adaptive Testing',
                      style: theme.textTheme.headlineMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      'This assessment adapts to your responses, selecting '
                      'the most informative questions to measure your traits '
                      'efficiently. It typically takes 4\u201312 questions per scale.',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 32),
                    if (sessionProvider.status == SessionStatus.error) ...[
                      ErrorBanner(
                        message: sessionProvider.errorMessage ??
                            'Failed to start session',
                        onRetry: () => _startAssessment(context),
                      ),
                      const SizedBox(height: 16),
                    ],
                    FilledButton.icon(
                      onPressed:
                          sessionProvider.status == SessionStatus.creating
                              ? null
                              : () => _startAssessment(context),
                      icon: sessionProvider.status == SessionStatus.creating
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: Colors.white,
                              ),
                            )
                          : const Icon(Icons.play_arrow),
                      label: Text(
                        sessionProvider.status == SessionStatus.creating
                            ? 'Starting...'
                            : 'Start Assessment',
                      ),
                    ),
                  ],
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}
