import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../providers/assessment_provider.dart';
import '../providers/session_provider.dart';
import '../widgets/error_banner.dart';
import '../widgets/likert_scale.dart';
import '../widgets/loading_overlay.dart';
import '../widgets/progress_indicator.dart';
import 'home_screen.dart';
import 'results_screen.dart';

class AssessmentScreen extends StatefulWidget {
  const AssessmentScreen({super.key});

  @override
  State<AssessmentScreen> createState() => _AssessmentScreenState();
}

class _AssessmentScreenState extends State<AssessmentScreen> {
  final FocusNode _focusNode = FocusNode();

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  void _onChoiceSelected(BuildContext context, int value) {
    final sessionId = context.read<SessionProvider>().currentSessionId;
    if (sessionId == null) return;
    context.read<AssessmentProvider>().submitResponse(sessionId, value);
  }

  KeyEventResult _handleKeyEvent(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    final provider = context.read<AssessmentProvider>();
    if (provider.status != AssessmentStatus.presenting) {
      return KeyEventResult.ignored;
    }
    final item = provider.currentItem;
    if (item == null) return KeyEventResult.ignored;

    // Map digit keys: 1-9 select displayed choices, 0 skips
    final key = event.logicalKey;
    int? digit;
    if (key == LogicalKeyboardKey.digit0) digit = 0;
    else if (key == LogicalKeyboardKey.digit1) digit = 1;
    else if (key == LogicalKeyboardKey.digit2) digit = 2;
    else if (key == LogicalKeyboardKey.digit3) digit = 3;
    else if (key == LogicalKeyboardKey.digit4) digit = 4;
    else if (key == LogicalKeyboardKey.digit5) digit = 5;
    else if (key == LogicalKeyboardKey.digit6) digit = 6;
    else if (key == LogicalKeyboardKey.digit7) digit = 7;
    else if (key == LogicalKeyboardKey.digit8) digit = 8;
    else if (key == LogicalKeyboardKey.digit9) digit = 9;
    if (digit == null) return KeyEventResult.ignored;

    if (digit == 0) {
      // Skip
      _onChoiceSelected(context, item.skipValue);
    } else {
      // Digit N selects the Nth displayed choice (1-based)
      final choices = item.likertChoices;
      final index = digit - 1;
      if (index >= choices.length) return KeyEventResult.ignored;
      _onChoiceSelected(context, choices[index].value);
    }
    return KeyEventResult.handled;
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

  Future<void> _confirmFinishEarly(BuildContext context) async {
    final provider = context.read<AssessmentProvider>();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Finish Early?'),
        content: Text(
          'You have answered ${provider.questionsAnswered} question${provider.questionsAnswered == 1 ? '' : 's'}. '
          'Results will be based on responses so far. Continue to results?',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Keep Going'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('View Results'),
          ),
        ],
      ),
    );
    if (confirmed == true && context.mounted) {
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const ResultsScreen()),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Focus(
      focusNode: _focusNode,
      autofocus: true,
      onKeyEvent: _handleKeyEvent,
      child: Scaffold(
      appBar: AppBar(
        title: const Text('Assessment'),
        centerTitle: true,
        leading: IconButton(
          icon: const Icon(Icons.close),
          tooltip: 'Quit assessment',
          onPressed: () => _confirmQuit(context),
        ),
        actions: [
          Consumer<AssessmentProvider>(
            builder: (context, provider, _) {
              if (provider.questionsAnswered < 1) {
                return const SizedBox.shrink();
              }
              return TextButton.icon(
                onPressed: () => _confirmFinishEarly(context),
                icon: const Icon(Icons.done_all, size: 18),
                label: const Text('Finish Early'),
              );
            },
          ),
        ],
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
    ),
    );
  }
}
