import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/assessment_meta_provider.dart';
import '../providers/assessment_provider.dart';
import '../providers/instrument_provider.dart';
import '../providers/session_provider.dart';
import '../widgets/error_banner.dart';
import 'assessment_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<InstrumentProvider>().fetch();
    });
  }

  void _onInstrumentChanged(String id) {
    context.read<InstrumentProvider>().select(id);
    context.read<AssessmentMetaProvider>().fetch(instrument: id);
  }

  Future<void> _startAssessment(BuildContext context) async {
    final sessionProvider = context.read<SessionProvider>();
    final assessmentProvider = context.read<AssessmentProvider>();
    final instrument = context.read<InstrumentProvider>().selectedId;

    await sessionProvider.createSession(instrument: instrument);

    if (!context.mounted) return;
    if (sessionProvider.status != SessionStatus.active) return;

    // Fetch metadata for the selected instrument so the results screen
    // can resolve scale display names.
    context.read<AssessmentMetaProvider>().fetch(instrument: instrument);

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
            child: Consumer3<SessionProvider, InstrumentProvider,
                AssessmentMetaProvider>(
              builder: (context, sessionProvider, instrumentProvider,
                  metaProvider, _) {
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
                      'Select an instrument and start your assessment. '
                      'Questions adapt to your responses for efficient measurement.',
                      style: theme.textTheme.bodyLarge?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 24),
                    if (instrumentProvider.status == InstrumentStatus.loading)
                      const Padding(
                        padding: EdgeInsets.all(16),
                        child: CircularProgressIndicator(),
                      ),
                    if (instrumentProvider.status ==
                        InstrumentStatus.error) ...[
                      ErrorBanner(
                        message: instrumentProvider.errorMessage ??
                            'Failed to load instruments',
                        onRetry: () => instrumentProvider.fetch(),
                      ),
                      const SizedBox(height: 16),
                    ],
                    if (instrumentProvider.instruments.isNotEmpty)
                      Card(
                        child: RadioGroup<String>(
                          groupValue: instrumentProvider.selectedId,
                          onChanged: (id) {
                            if (id != null) _onInstrumentChanged(id);
                          },
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: instrumentProvider.instruments
                                .map(
                                  (inst) => RadioListTile<String>(
                                    value: inst.id,
                                    title: Text(inst.name),
                                    subtitle: Text(
                                      inst.description,
                                      maxLines: 2,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                )
                                .toList(),
                          ),
                        ),
                      ),
                    if (metaProvider.meta != null) ...[
                      const SizedBox(height: 12),
                      Wrap(
                        spacing: 8,
                        runSpacing: 4,
                        alignment: WrapAlignment.center,
                        children: metaProvider.meta!.scales.entries.map((e) {
                          return Chip(
                            label: Text(e.value),
                            visualDensity: VisualDensity.compact,
                          );
                        }).toList(),
                      ),
                    ],
                    const SizedBox(height: 24),
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
                          sessionProvider.status == SessionStatus.creating ||
                                  instrumentProvider.selectedId == null
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
