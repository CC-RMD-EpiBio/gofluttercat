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

    // Map digit keys 0-9 to response values
    final key = event.logicalKey;
    int? value;
    if (key == LogicalKeyboardKey.digit0) value = 0;
    else if (key == LogicalKeyboardKey.digit1) value = 1;
    else if (key == LogicalKeyboardKey.digit2) value = 2;
    else if (key == LogicalKeyboardKey.digit3) value = 3;
    else if (key == LogicalKeyboardKey.digit4) value = 4;
    else if (key == LogicalKeyboardKey.digit5) value = 5;
    else if (key == LogicalKeyboardKey.digit6) value = 6;
    else if (key == LogicalKeyboardKey.digit7) value = 7;
    else if (key == LogicalKeyboardKey.digit8) value = 8;
    else if (key == LogicalKeyboardKey.digit9) value = 9;
    if (value == null) return KeyEventResult.ignored;

    // Check that this value is a valid choice for the current item
    final validValues =
        item.responses.entries.map((e) => int.tryParse(e.key)).toSet();
    if (!validValues.contains(value)) return KeyEventResult.ignored;

    _onChoiceSelected(context, value);
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
