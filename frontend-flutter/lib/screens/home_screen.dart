import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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
  final _stoppingStdController = TextEditingController(text: '0.33');
  final _stoppingNumItemsController = TextEditingController(text: '0');
  bool _showSettings = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<InstrumentProvider>().fetch();
    });
  }

  @override
  void dispose() {
    _stoppingStdController.dispose();
    _stoppingNumItemsController.dispose();
    super.dispose();
  }

  void _onInstrumentChanged(String id) {
    context.read<InstrumentProvider>().select(id);
    context.read<AssessmentMetaProvider>().fetch(instrument: id);
  }

  Future<void> _startAssessment(BuildContext context) async {
    final sessionProvider = context.read<SessionProvider>();
    final assessmentProvider = context.read<AssessmentProvider>();
    final instrument = context.read<InstrumentProvider>().selectedId;

    final stoppingStd = double.tryParse(_stoppingStdController.text);
    final stoppingNumItems = int.tryParse(_stoppingNumItemsController.text);

    await sessionProvider.createSession(
      instrument: instrument,
      stoppingStd: stoppingStd,
      stoppingNumItems: stoppingNumItems,
    );

    if (!context.mounted) return;
    if (sessionProvider.status != SessionStatus.active) return;

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
        child: SingleChildScrollView(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 600),
            child: Padding(
              padding: const EdgeInsets.all(32),
              child:
                  Consumer3<
                    SessionProvider,
                    InstrumentProvider,
                    AssessmentMetaProvider
                  >(
                    builder:
                        (
                          context,
                          sessionProvider,
                          instrumentProvider,
                          metaProvider,
                          _,
                        ) {
                          return Column(
                            mainAxisSize: MainAxisSize.min,
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
                              if (instrumentProvider.status ==
                                  InstrumentStatus.loading)
                                const Padding(
                                  padding: EdgeInsets.all(16),
                                  child: CircularProgressIndicator(),
                                ),
                              if (instrumentProvider.status ==
                                  InstrumentStatus.error) ...[
                                ErrorBanner(
                                  message:
                                      instrumentProvider.errorMessage ??
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
                                  children: metaProvider.meta!.scales.entries
                                      .map((e) {
                                        return Chip(
                                          label: Text(e.value),
                                          visualDensity: VisualDensity.compact,
                                        );
                                      })
                                      .toList(),
                                ),
                              ],
                              const SizedBox(height: 16),
                              _catSettingsSection(theme),
                              const SizedBox(height: 24),
                              if (sessionProvider.status ==
                                  SessionStatus.error) ...[
                                ErrorBanner(
                                  message:
                                      sessionProvider.errorMessage ??
                                      'Failed to start session',
                                  onRetry: () => _startAssessment(context),
                                ),
                                const SizedBox(height: 16),
                              ],
                              FilledButton.icon(
                                onPressed:
                                    sessionProvider.status ==
                                            SessionStatus.creating ||
                                        instrumentProvider.selectedId == null
                                    ? null
                                    : () => _startAssessment(context),
                                icon:
                                    sessionProvider.status ==
                                        SessionStatus.creating
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
                                  sessionProvider.status ==
                                          SessionStatus.creating
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
      ),
    );
  }

  Widget _catSettingsSection(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        InkWell(
          onTap: () => setState(() => _showSettings = !_showSettings),
          borderRadius: BorderRadius.circular(8),
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(
                  _showSettings ? Icons.expand_less : Icons.expand_more,
                  size: 18,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 4),
                Text(
                  'CAT Settings',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
        if (_showSettings) ...[
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TextField(
                    controller: _stoppingStdController,
                    decoration: const InputDecoration(
                      labelText: 'Stopping threshold (posterior SD)',
                      helperText: 'Stop scale when posterior SD drops below this',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    keyboardType: const TextInputType.numberWithOptions(decimal: true),
                    inputFormatters: [
                      FilteringTextInputFormatter.allow(RegExp(r'[\d.]')),
                    ],
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _stoppingNumItemsController,
                    decoration: const InputDecoration(
                      labelText: 'Max items per scale',
                      helperText: '0 = unlimited',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    keyboardType: TextInputType.number,
                    inputFormatters: [
                      FilteringTextInputFormatter.digitsOnly,
                    ],
                  ),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }
}
