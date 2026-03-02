import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Draws a posterior density curve over an ability grid.
/// When both observed and RB densities are provided, draws both curves.
class PosteriorChart extends StatelessWidget {
  final List<double> grid;
  final List<double> density;
  final List<double> rbDensity;
  final double mean;
  final double? rbMean;

  const PosteriorChart({
    super.key,
    required this.grid,
    required this.density,
    this.rbDensity = const [],
    required this.mean,
    this.rbMean,
  });

  @override
  Widget build(BuildContext context) {
    if (grid.isEmpty || density.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Posterior Density',
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 4),
        if (rbDensity.isNotEmpty)
          Row(
            children: [
              _LegendDot(color: theme.colorScheme.primary),
              const SizedBox(width: 4),
              Text('Marginalized',
                  style: TextStyle(
                      fontSize: 10,
                      color: theme.colorScheme.onSurfaceVariant)),
              const SizedBox(width: 12),
              _LegendDot(color: theme.colorScheme.tertiary),
              const SizedBox(width: 4),
              Text('Ignoring Missingness',
                  style: TextStyle(
                      fontSize: 10,
                      color: theme.colorScheme.onSurfaceVariant)),
            ],
          ),
        if (rbDensity.isNotEmpty) const SizedBox(height: 4),
        SizedBox(
          height: 120,
          child: CustomPaint(
            size: Size.infinite,
            painter: _PosteriorChartPainter(
              grid: grid,
              density: density,
              rbDensity: rbDensity,
              mean: mean,
              rbMean: rbMean,
              observedColor: theme.colorScheme.tertiary,
              rbColor: theme.colorScheme.primary,
              axisColor: theme.colorScheme.outlineVariant,
              labelColor: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      ],
    );
  }
}

class _LegendDot extends StatelessWidget {
  final Color color;
  const _LegendDot({required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 8,
      height: 8,
      decoration: BoxDecoration(color: color, shape: BoxShape.circle),
    );
  }
}

class _PosteriorChartPainter extends CustomPainter {
  final List<double> grid;
  final List<double> density;
  final List<double> rbDensity;
  final double mean;
  final double? rbMean;
  final Color observedColor;
  final Color rbColor;
  final Color axisColor;
  final Color labelColor;

  _PosteriorChartPainter({
    required this.grid,
    required this.density,
    required this.rbDensity,
    required this.mean,
    this.rbMean,
    required this.observedColor,
    required this.rbColor,
    required this.axisColor,
    required this.labelColor,
  });

  @override
  void paint(Canvas canvas, Size size) {
    const leftPad = 8.0;
    const rightPad = 8.0;
    const topPad = 4.0;
    const bottomPad = 20.0;
    final plotWidth = size.width - leftPad - rightPad;
    final plotHeight = size.height - topPad - bottomPad;

    final gridMin = grid.first;
    final gridMax = grid.last;

    // Find the max density for scaling
    double maxDensity = density.reduce(math.max);
    if (rbDensity.isNotEmpty) {
      maxDensity = math.max(maxDensity, rbDensity.reduce(math.max));
    }
    if (maxDensity <= 0) return;

    double toX(double gridVal) {
      final frac = (gridVal - gridMin) / (gridMax - gridMin);
      return leftPad + frac.clamp(0.0, 1.0) * plotWidth;
    }

    double toY(double densityVal) {
      final frac = densityVal / maxDensity;
      return topPad + plotHeight - frac.clamp(0.0, 1.0) * plotHeight;
    }

    // Draw x-axis
    final axisPaint = Paint()
      ..color = axisColor
      ..strokeWidth = 1;
    final baselineY = topPad + plotHeight;
    canvas.drawLine(
      Offset(leftPad, baselineY),
      Offset(size.width - rightPad, baselineY),
      axisPaint,
    );

    // Tick marks and labels
    final rangeMin = gridMin.ceilToDouble();
    final rangeMax = gridMax.floorToDouble();
    for (int i = rangeMin.toInt(); i <= rangeMax.toInt(); i++) {
      if (i % 2 != 0) continue;
      final x = toX(i.toDouble());
      canvas.drawLine(
        Offset(x, baselineY),
        Offset(x, baselineY + 4),
        axisPaint,
      );
      final tp = TextPainter(
        text: TextSpan(
          text: '$i',
          style: TextStyle(fontSize: 9, color: labelColor),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      tp.paint(canvas, Offset(x - tp.width / 2, baselineY + 5));
    }

    // Draw density curves
    void drawCurve(List<double> d, Color color, {bool fill = false}) {
      if (d.length != grid.length) return;
      final path = Path();
      path.moveTo(toX(grid[0]), toY(d[0]));
      for (int i = 1; i < grid.length; i++) {
        path.lineTo(toX(grid[i]), toY(d[i]));
      }

      if (fill) {
        final fillPath = Path.from(path);
        fillPath.lineTo(toX(grid.last), baselineY);
        fillPath.lineTo(toX(grid.first), baselineY);
        fillPath.close();
        canvas.drawPath(
          fillPath,
          Paint()
            ..color = color.withAlpha(30)
            ..style = PaintingStyle.fill,
        );
      }

      canvas.drawPath(
        path,
        Paint()
          ..color = color
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2
          ..strokeJoin = StrokeJoin.round,
      );
    }

    // Draw observed density first (behind)
    drawCurve(density, observedColor, fill: true);
    // Draw RB density on top if available
    if (rbDensity.isNotEmpty) {
      drawCurve(rbDensity, rbColor, fill: true);
    }

    // Draw mean markers as vertical dashed lines
    void drawMeanLine(double meanVal, Color color) {
      final x = toX(meanVal.clamp(gridMin, gridMax));
      final dashPaint = Paint()
        ..color = color.withAlpha(180)
        ..strokeWidth = 1.5;
      // Simple dashed line
      double y = topPad;
      while (y < baselineY) {
        final end = math.min(y + 4, baselineY);
        canvas.drawLine(Offset(x, y), Offset(x, end), dashPaint);
        y += 7;
      }
    }

    drawMeanLine(mean, observedColor);
    if (rbMean != null && rbDensity.isNotEmpty) {
      drawMeanLine(rbMean!, rbColor);
    }
  }

  @override
  bool shouldRepaint(covariant _PosteriorChartPainter oldDelegate) {
    return density != oldDelegate.density ||
        rbDensity != oldDelegate.rbDensity;
  }
}
