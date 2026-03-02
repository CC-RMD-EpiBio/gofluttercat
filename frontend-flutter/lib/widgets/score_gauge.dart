import 'dart:math' as math;

import 'package:flutter/material.dart';

class ScoreGauge extends StatelessWidget {
  final double mean;
  final double std;
  final double rangeMin;
  final double rangeMax;

  const ScoreGauge({
    super.key,
    required this.mean,
    required this.std,
    this.rangeMin = -4.0,
    this.rangeMax = 4.0,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 48,
      child: CustomPaint(
        size: Size.infinite,
        painter: _ScoreGaugePainter(
          mean: mean,
          std: std,
          rangeMin: rangeMin,
          rangeMax: rangeMax,
          trackColor: Theme.of(context).colorScheme.surfaceContainerHighest,
          bandColor: Theme.of(context).colorScheme.primary.withAlpha(60),
          markerColor: Theme.of(context).colorScheme.primary,
          tickColor: Theme.of(context).colorScheme.outline,
          labelColor: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

class _ScoreGaugePainter extends CustomPainter {
  final double mean;
  final double std;
  final double rangeMin;
  final double rangeMax;
  final Color trackColor;
  final Color bandColor;
  final Color markerColor;
  final Color tickColor;
  final Color labelColor;

  _ScoreGaugePainter({
    required this.mean,
    required this.std,
    required this.rangeMin,
    required this.rangeMax,
    required this.trackColor,
    required this.bandColor,
    required this.markerColor,
    required this.tickColor,
    required this.labelColor,
  });

  double _toX(double value, double width, double padding) {
    final usable = width - 2 * padding;
    final fraction = (value - rangeMin) / (rangeMax - rangeMin);
    return padding + fraction * usable;
  }

  @override
  void paint(Canvas canvas, Size size) {
    const padding = 24.0;
    final trackY = size.height / 2;
    const trackHeight = 6.0;

    // Draw track
    final trackPaint = Paint()
      ..color = trackColor
      ..style = PaintingStyle.fill;
    canvas.drawRRect(
      RRect.fromRectAndRadius(
        Rect.fromCenter(
          center: Offset(size.width / 2, trackY),
          width: size.width - 2 * padding,
          height: trackHeight,
        ),
        const Radius.circular(3),
      ),
      trackPaint,
    );

    // Draw uncertainty band (2*std width centered on mean)
    final bandLeft = _toX(mean - std, size.width, padding);
    final bandRight = _toX(mean + std, size.width, padding);
    final clampedLeft = math.max(bandLeft, padding);
    final clampedRight = math.min(bandRight, size.width - padding);
    if (clampedRight > clampedLeft) {
      final bandPaint = Paint()
        ..color = bandColor
        ..style = PaintingStyle.fill;
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTRB(clampedLeft, trackY - 10, clampedRight, trackY + 10),
          const Radius.circular(5),
        ),
        bandPaint,
      );
    }

    // Draw tick marks at integers
    final tickPaint = Paint()
      ..color = tickColor
      ..strokeWidth = 1;
    for (int i = rangeMin.ceil(); i <= rangeMax.floor(); i++) {
      final x = _toX(i.toDouble(), size.width, padding);
      canvas.drawLine(
        Offset(x, trackY + trackHeight / 2 + 2),
        Offset(x, trackY + trackHeight / 2 + 7),
        tickPaint,
      );

      // Label every 2 ticks and the endpoints
      if (i % 2 == 0 || i == rangeMin.ceil() || i == rangeMax.floor()) {
        final textPainter = TextPainter(
          text: TextSpan(
            text: '$i',
            style: TextStyle(fontSize: 9, color: labelColor),
          ),
          textDirection: TextDirection.ltr,
        )..layout();
        textPainter.paint(
          canvas,
          Offset(x - textPainter.width / 2, trackY + trackHeight / 2 + 9),
        );
      }
    }

    // Draw marker dot at mean
    final markerX = _toX(
      mean.clamp(rangeMin, rangeMax),
      size.width,
      padding,
    );
    final markerPaint = Paint()
      ..color = markerColor
      ..style = PaintingStyle.fill;
    canvas.drawCircle(Offset(markerX, trackY), 7, markerPaint);

    // White inner dot
    final innerPaint = Paint()
      ..color = Colors.white
      ..style = PaintingStyle.fill;
    canvas.drawCircle(Offset(markerX, trackY), 3, innerPaint);
  }

  @override
  bool shouldRepaint(covariant _ScoreGaugePainter oldDelegate) {
    return mean != oldDelegate.mean || std != oldDelegate.std;
  }
}
