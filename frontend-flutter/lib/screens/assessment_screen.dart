import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/assessment_provider.dart';
import '../providers/session_provider.dart';
import '../widgets/error_banner.dart';
import '../widgets/likert_scale.dart';
import '../widgets/loading_overlay.dart';
import '../widgets/progress_indicator.dart';
import 'home_screen.dart';
import 'results_screen.dart';

class AssessmentScreen extends StatelessWidget {
  const AssessmentScreen({super.key});

  void _onChoiceSelected(BuildContext context, int value) {
    final sessionId = context.read<SessionProvider>().currentSessionId;
    if (sessionId == null) return;
    context.read<AssessmentProvider>().submitResponse(sessionId, value);
  }

  Future<void> _confirmQuit(BuildContext context) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Quit Assessment?'),
        content: const Text(
          'Your progress will be lost. Are you sure you want to quit?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Continue'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Quit'),
          ),
        ],
      ),
    );
    if (confirmed == true && context.mounted) {
      context.read<SessionProvider>().endSession();
      context.read<AssessmentProvider>().reset();
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const HomeScreen()),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Assessment'),
        centerTitle: true,
        leading: IconButton(
          icon: const Icon(Icons.close),
          tooltip: 'Quit assessment',
          onPressed: () => _confirmQuit(context),
        ),
      ),
      body: Consumer<AssessmentProvider>(
        builder: (context, provider, _) {
          // Navigate to results when complete
          if (provider.status == AssessmentStatus.complete) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              Navigator.of(context).pushReplacement(
                MaterialPageRoute(builder: (_) => const ResultsScreen()),
              );
            });
            return const Center(child: CircularProgressIndicator());
          }

          return Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 700),
              child: LoadingOverlay(
                isLoading: provider.status == AssessmentStatus.submitting,
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      AssessmentProgressIndicator(
                        questionsAnswered: provider.questionsAnswered,
                      ),
                      const SizedBox(height: 24),
                      if (provider.status == AssessmentStatus.error) ...[
                        ErrorBanner(
                          message:
                              provider.errorMessage ?? 'Something went wrong',
                          onRetry: () {
                            final sessionId = context
                                .read<SessionProvider>()
                                .currentSessionId;
                            if (sessionId != null) {
                              provider.fetchNextItem(sessionId);
                            }
                          },
                        ),
                        const SizedBox(height: 16),
                      ],
                      if (provider.status == AssessmentStatus.loading)
                        const Center(
                          child: Padding(
                            padding: EdgeInsets.all(48),
                            child: CircularProgressIndicator(),
                          ),
                        ),
                      if (provider.currentItem != null &&
                          (provider.status == AssessmentStatus.presenting ||
                              provider.status ==
                                  AssessmentStatus.submitting))
                        AnimatedSwitcher(
                          duration: const Duration(milliseconds: 300),
                          switchInCurve: Curves.easeOut,
                          switchOutCurve: Curves.easeIn,
                          transitionBuilder: (child, animation) {
                            return FadeTransition(
                              opacity: animation,
                              child: SlideTransition(
                                position: Tween<Offset>(
                                  begin: const Offset(0.05, 0),
                                  end: Offset.zero,
                                ).animate(animation),
                                child: child,
                              ),
                            );
                          },
                          child: Column(
                            key: ValueKey(provider.currentItem!.name),
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              Card(
                                child: Padding(
                                  padding: const EdgeInsets.all(20),
                                  child: Text(
                                    provider.currentItem!.question,
                                    style: theme.textTheme.titleLarge,
                                  ),
                                ),
                              ),
                              const SizedBox(height: 16),
                              LikertScale(
                                item: provider.currentItem!,
                                enabled: provider.status !=
                                    AssessmentStatus.submitting,
                                onSelected: (value) =>
                                    _onChoiceSelected(context, value),
                              ),
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}
