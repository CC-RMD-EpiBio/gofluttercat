import 'package:flutter/material.dart';

import '../models/score.dart';
import 'decile_chart.dart';
import 'score_gauge.dart';

class ScoreCard extends StatelessWidget {
  final String scaleName;
  final ScoreSummary score;

  const ScoreCard({
    super.key,
    required this.scaleName,
    required this.score,
  });

  String _formatLabel(String name) {
    // Convert snake_case or camelCase to Title Case
    return name
        .replaceAllMapped(RegExp(r'[_-]'), (_) => ' ')
        .replaceAllMapped(
          RegExp(r'([a-z])([A-Z])'),
          (m) => '${m[1]} ${m[2]}',
        )
        .split(' ')
        .map((w) => w.isEmpty ? w : '${w[0].toUpperCase()}${w.substring(1)}')
        .join(' ');
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.symmetric(vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              _formatLabel(scaleName),
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 12),
            ScoreGauge(mean: score.mean, std: score.std),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: [
                Chip(
                  avatar: Icon(Icons.center_focus_strong,
                      size: 16, color: theme.colorScheme.primary),
                  label: Text('Score: ${score.mean.toStringAsFixed(2)}'),
                ),
                Chip(
                  avatar: Icon(Icons.unfold_more,
                      size: 16, color: theme.colorScheme.secondary),
                  label:
                      Text('Uncertainty: \u00b1${score.std.toStringAsFixed(2)}'),
                ),
                Chip(
                  avatar: Icon(Icons.linear_scale,
                      size: 16, color: theme.colorScheme.tertiary),
                  label: Text('Median: ${score.median.toStringAsFixed(2)}'),
                ),
              ],
            ),
            if (score.deciles.isNotEmpty) ...[
              const SizedBox(height: 12),
              Text(
                'Posterior Distribution',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 4),
              DecileChart(deciles: score.deciles),
            ],
          ],
        ),
      ),
    );
  }
}
