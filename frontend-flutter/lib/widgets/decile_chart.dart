import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Displays a decile distribution as a horizontal dot chart.
/// Each dot represents a decile (10th, 20th, ... 90th percentile).
class DecileChart extends StatelessWidget {
  final List<double> deciles;
  final double rangeMin;
  final double rangeMax;

  const DecileChart({
    super.key,
    required this.deciles,
    this.rangeMin = -4.0,
    this.rangeMax = 4.0,
  });

  @override
  Widget build(BuildContext context) {
    if (deciles.isEmpty) return const SizedBox.shrink();

    return SizedBox(
      height: 56,
      child: CustomPaint(
        size: Size.infinite,
        painter: _DecileChartPainter(
          deciles: deciles,
          rangeMin: rangeMin,
          rangeMax: rangeMax,
          dotColor: Theme.of(context).colorScheme.tertiary,
          medianColor: Theme.of(context).colorScheme.primary,
          lineColor: Theme.of(context).colorScheme.outlineVariant,
          labelColor: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

class _DecileChartPainter extends CustomPainter {
  final List<double> deciles;
  final double rangeMin;
  final double rangeMax;
  final Color dotColor;
  final Color medianColor;
  final Color lineColor;
  final Color labelColor;

  _DecileChartPainter({
    required this.deciles,
    required this.rangeMin,
    required this.rangeMax,
    required this.dotColor,
    required this.medianColor,
    required this.lineColor,
    required this.labelColor,
  });

  double _toX(double value, double width, double padding) {
    final usable = width - 2 * padding;
    final fraction = (value - rangeMin) / (rangeMax - rangeMin);
    return padding + fraction.clamp(0.0, 1.0) * usable;
  }

  @override
  void paint(Canvas canvas, Size size) {
    const padding = 24.0;
    final centerY = size.height * 0.4;

    // Draw baseline
    final linePaint = Paint()
      ..color = lineColor
      ..strokeWidth = 1;
    canvas.drawLine(
      Offset(padding, centerY),
      Offset(size.width - padding, centerY),
      linePaint,
    );

    // Draw connecting line between first and last decile (IQR-style)
    if (deciles.length >= 2) {
      final iqrPaint = Paint()
        ..color = dotColor.withAlpha(80)
        ..strokeWidth = 4
        ..strokeCap = StrokeCap.round;
      final firstX = _toX(deciles.first, size.width, padding);
      final lastX = _toX(deciles.last, size.width, padding);
      canvas.drawLine(Offset(firstX, centerY), Offset(lastX, centerY), iqrPaint);
    }

    // Draw each decile as a dot
    final dotPaint = Paint()
      ..color = dotColor
      ..style = PaintingStyle.fill;
    final medianPaint = Paint()
      ..color = medianColor
      ..style = PaintingStyle.fill;

    for (int i = 0; i < deciles.length; i++) {
      final x = _toX(deciles[i], size.width, padding);
      final isMedian = i == 4; // 5th decile = 50th percentile
      final radius = isMedian ? 5.0 : 3.5;
      canvas.drawCircle(
        Offset(x, centerY),
        radius,
        isMedian ? medianPaint : dotPaint,
      );
    }

    // Labels for key percentiles
    final labels = <int, String>{};
    if (deciles.isNotEmpty) labels[0] = '10th';
    if (deciles.length >= 5) labels[4] = '50th';
    if (deciles.length >= 9) labels[8] = '90th';

    for (final entry in labels.entries) {
      final i = entry.key;
      final label = entry.value;
      final x = _toX(deciles[i], size.width, padding);
      final tp = TextPainter(
        text: TextSpan(
          text: '$label\n${deciles[i].toStringAsFixed(1)}',
          style: TextStyle(fontSize: 9, color: labelColor, height: 1.2),
        ),
        textDirection: TextDirection.ltr,
        textAlign: TextAlign.center,
      )..layout();
      // Clamp label position to stay within bounds
      final labelX = math.max(
        padding,
        math.min(x - tp.width / 2, size.width - padding - tp.width),
      );
      tp.paint(canvas, Offset(labelX, centerY + 8));
    }
  }

  @override
  bool shouldRepaint(covariant _DecileChartPainter oldDelegate) {
    return deciles != oldDelegate.deciles;
  }
}
