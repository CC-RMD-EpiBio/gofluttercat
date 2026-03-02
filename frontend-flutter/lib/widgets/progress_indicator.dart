import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../config.dart';
import '../providers/assessment_meta_provider.dart';

class AssessmentProgressIndicator extends StatelessWidget {
  final int questionsAnswered;

  const AssessmentProgressIndicator({
    super.key,
    required this.questionsAnswered,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final meta = context.watch<AssessmentMetaProvider>().meta;
    final estimatedMax = meta?.maxTotalItems ?? maxItems;
    final currentQuestion = questionsAnswered + 1;
    final progress = questionsAnswered / estimatedMax;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Question $currentQuestion of ~$estimatedMax',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
        const SizedBox(height: 4),
        LinearProgressIndicator(
          value: progress.clamp(0.0, 1.0),
          borderRadius: BorderRadius.circular(4),
        ),
      ],
    );
  }
}
