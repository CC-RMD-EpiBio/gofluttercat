import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../models/score.dart';

/// A forest plot showing all scales with point estimates and credible intervals.
/// Each row shows observed (and optionally RB) estimates with 80% CIs from deciles.
class ForestPlot extends StatelessWidget {
  final Map<String, ScoreSummary> scores;
  final String Function(String key)? labelFormatter;

  const ForestPlot({
    super.key,
    required this.scores,
    this.labelFormatter,
  });

  String _defaultFormat(String name) {
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
    if (scores.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final entries = scores.entries.toList();
    final hasRb = entries.any((e) => e.value.hasRb);
    final format = labelFormatter ?? _defaultFormat;

    return Card(
      margin: const EdgeInsets.symmetric(vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Score Comparison',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            if (hasRb)
              Row(
                children: [
                  _LegendItem(
                      color: theme.colorScheme.primary, label: 'RB Estimate'),
                  const SizedBox(width: 16),
                  _LegendItem(
                      color: theme.colorScheme.tertiary,
                      label: 'Observed Estimate'),
                ],
              ),
            const SizedBox(height: 12),
            SizedBox(
              height: entries.length * (hasRb ? 52.0 : 36.0) + 24,
              child: CustomPaint(
                size: Size.infinite,
                painter: _ForestPlotPainter(
                  entries: entries,
                  labels: entries.map((e) => format(e.key)).toList(),
                  primaryColor: theme.colorScheme.primary,
                  tertiaryColor: theme.colorScheme.tertiary,
                  axisColor: theme.colorScheme.outlineVariant,
                  labelColor: theme.colorScheme.onSurface,
                  sublabelColor: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LegendItem extends StatelessWidget {
  final Color color;
  final String label;
  const _LegendItem({required this.color, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(width: 12, height: 3, color: color),
        const SizedBox(width: 4),
        Container(
          width: 6,
          height: 6,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 4),
        Container(width: 12, height: 3, color: color),
        const SizedBox(width: 6),
        Text(label,
            style: TextStyle(
                fontSize: 10,
                color: Theme.of(context).colorScheme.onSurfaceVariant)),
      ],
    );
  }
}

class _ForestPlotPainter extends CustomPainter {
  final List<MapEntry<String, ScoreSummary>> entries;
  final List<String> labels;
  final Color primaryColor;
  final Color tertiaryColor;
  final Color axisColor;
  final Color labelColor;
  final Color sublabelColor;

  _ForestPlotPainter({
    required this.entries,
    required this.labels,
    required this.primaryColor,
    required this.tertiaryColor,
    required this.axisColor,
    required this.labelColor,
    required this.sublabelColor,
  });

  @override
  void paint(Canvas canvas, Size size) {
    final hasRb = entries.any((e) => e.value.hasRb);
    final rowHeight = hasRb ? 52.0 : 36.0;
    const labelWidth = 120.0;
    const rightPad = 16.0;
    const topPad = 0.0;
    final bottomPad = 24.0;
    final plotLeft = labelWidth;
    final plotRight = size.width - rightPad;
    final plotWidth = plotRight - plotLeft;

    // Determine range from all scores
    double rangeMin = -3.0;
    double rangeMax = 3.0;
    for (final e in entries) {
      final s = e.value;
      rangeMin = math.min(rangeMin, s.mean - 2 * s.std);
      if (s.hasRb) rangeMin = math.min(rangeMin, s.rbMean - 2 * s.rbStd);
      rangeMax = math.max(rangeMax, s.mean + 2 * s.std);
      if (s.hasRb) rangeMax = math.max(rangeMax, s.rbMean + 2 * s.rbStd);
    }
    // Round to nice numbers
    rangeMin = (rangeMin - 0.5).floorToDouble();
    rangeMax = (rangeMax + 0.5).ceilToDouble();

    double toX(double val) {
      final frac = (val - rangeMin) / (rangeMax - rangeMin);
      return plotLeft + frac.clamp(0.0, 1.0) * plotWidth;
    }

    // Draw zero line
    final zeroX = toX(0);
    final zeroPaint = Paint()
      ..color = axisColor
      ..strokeWidth = 1;
    canvas.drawLine(
      Offset(zeroX, topPad),
      Offset(zeroX, topPad + entries.length * rowHeight),
      zeroPaint,
    );

    // Draw x-axis at bottom
    final axisY = topPad + entries.length * rowHeight;
    canvas.drawLine(
      Offset(plotLeft, axisY),
      Offset(plotRight, axisY),
      zeroPaint,
    );

    // Tick labels
    for (int i = rangeMin.toInt(); i <= rangeMax.toInt(); i++) {
      final x = toX(i.toDouble());
      canvas.drawLine(Offset(x, axisY), Offset(x, axisY + 4), zeroPaint);
      final tp = TextPainter(
        text: TextSpan(
          text: '$i',
          style: TextStyle(fontSize: 9, color: sublabelColor),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      tp.paint(canvas, Offset(x - tp.width / 2, axisY + 5));
    }

    // Draw each row
    for (int i = 0; i < entries.length; i++) {
      final score = entries[i].value;
      final label = labels[i];
      final rowTop = topPad + i * rowHeight;

      // Row separator
      if (i > 0) {
        canvas.drawLine(
          Offset(0, rowTop),
          Offset(size.width, rowTop),
          Paint()
            ..color = axisColor.withAlpha(60)
            ..strokeWidth = 0.5,
        );
      }

      // Label
      final labelTp = TextPainter(
        text: TextSpan(
          text: label,
          style: TextStyle(fontSize: 11, color: labelColor),
        ),
        textDirection: TextDirection.ltr,
        maxLines: 2,
        ellipsis: '...',
      )..layout(maxWidth: labelWidth - 8);
      labelTp.paint(
        canvas,
        Offset(4, rowTop + (rowHeight - labelTp.height) / 2),
      );

      // Draw observed estimate
      final obsY = hasRb ? rowTop + rowHeight * 0.62 : rowTop + rowHeight / 2;
      _drawEstimate(
        canvas,
        mean: score.mean,
        low: score.deciles.isNotEmpty ? score.deciles.first : score.mean - score.std,
        high: score.deciles.length >= 9 ? score.deciles[8] : score.mean + score.std,
        y: obsY,
        color: tertiaryColor,
        toX: toX,
      );

      // Draw RB estimate if available
      if (score.hasRb) {
        final rbY = rowTop + rowHeight * 0.32;
        _drawEstimate(
          canvas,
          mean: score.rbMean,
          low: score.rbDeciles.isNotEmpty
              ? score.rbDeciles.first
              : score.rbMean - score.rbStd,
          high: score.rbDeciles.length >= 9
              ? score.rbDeciles[8]
              : score.rbMean + score.rbStd,
          y: rbY,
          color: primaryColor,
          toX: toX,
        );
      }
    }
  }

  void _drawEstimate(
    Canvas canvas, {
    required double mean,
    required double low,
    required double high,
    required double y,
    required Color color,
    required double Function(double) toX,
  }) {
    final meanX = toX(mean);
    final lowX = toX(low);
    final highX = toX(high);

    // CI line
    canvas.drawLine(
      Offset(lowX, y),
      Offset(highX, y),
      Paint()
        ..color = color
        ..strokeWidth = 2
        ..strokeCap = StrokeCap.round,
    );

    // CI whiskers
    canvas.drawLine(
      Offset(lowX, y - 4),
      Offset(lowX, y + 4),
      Paint()
        ..color = color
        ..strokeWidth = 1.5,
    );
    canvas.drawLine(
      Offset(highX, y - 4),
      Offset(highX, y + 4),
      Paint()
        ..color = color
        ..strokeWidth = 1.5,
    );

    // Point estimate diamond
    final diamondPath = Path()
      ..moveTo(meanX, y - 5)
      ..lineTo(meanX + 4, y)
      ..lineTo(meanX, y + 5)
      ..lineTo(meanX - 4, y)
      ..close();
    canvas.drawPath(
      diamondPath,
      Paint()
        ..color = color
        ..style = PaintingStyle.fill,
    );
  }

  @override
  bool shouldRepaint(covariant _ForestPlotPainter oldDelegate) {
    return entries != oldDelegate.entries;
  }
}
