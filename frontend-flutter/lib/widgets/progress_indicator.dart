import 'package:flutter/material.dart';

import '../config.dart';

class AssessmentProgressIndicator extends StatelessWidget {
  final int questionsAnswered;

  const AssessmentProgressIndicator({
    super.key,
    required this.questionsAnswered,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Current question is questionsAnswered + 1
    final currentQuestion = questionsAnswered + 1;
    final progress = questionsAnswered / maxItems;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Question $currentQuestion of ~$maxItems',
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
