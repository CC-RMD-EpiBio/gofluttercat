import 'package:flutter/material.dart';

import '../models/item.dart';

class LikertScale extends StatelessWidget {
  final AssessmentItem item;
  final ValueChanged<int> onSelected;
  final bool enabled;

  const LikertScale({
    super.key,
    required this.item,
    required this.onSelected,
    this.enabled = true,
  });

  @override
  Widget build(BuildContext context) {
    final likertChoices = item.likertChoices;
    final theme = Theme.of(context);

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ...likertChoices.asMap().entries.map((entry) {
          final displayNumber = entry.key + 1;
          final choice = entry.value;
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 3),
            child: OutlinedButton(
              onPressed: enabled ? () => onSelected(choice.value) : null,
              style: OutlinedButton.styleFrom(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                alignment: Alignment.centerLeft,
              ),
              child: Row(
                children: [
                  CircleAvatar(
                    radius: 14,
                    backgroundColor: theme.colorScheme.primaryContainer,
                    child: Text(
                      '$displayNumber',
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.bold,
                        color: theme.colorScheme.onPrimaryContainer,
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      choice.text,
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
                ],
              ),
            ),
          );
        }),
        const Divider(height: 24),
        Center(
          child: TextButton(
            onPressed: enabled ? () => onSelected(item.skipValue) : null,
            child: Text(
              'Skip',
              style: TextStyle(color: theme.colorScheme.outline),
            ),
          ),
        ),
      ],
    );
  }
}
