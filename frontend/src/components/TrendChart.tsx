'use client';

import { useEffect, useMemo, useRef, useState } from 'react';

export interface TrendPoint {
  month: string;
  scheduled: number;
  ongoing: number;
  completed: number;
  failed: number;
}

const LINE_COLOR = '#3b82f6';
const GRADIENT_ID = 'trend-gradient-fill';
const CHART_HEIGHT = 208;
const PAD = { top: 16, right: 14, bottom: 30, left: 42 };

interface Point {
  x: number;
  y: number;
}

const formatMonth = (month: string) => {
  const [year, m] = month.split('-');
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
  return `${months[Number(m) - 1]} ${year.slice(2)}`;
};

// Konversi titik ke kurva halus (Catmull-Rom spline → cubic Bézier)
const buildSmoothPath = (points: Point[]) => {
  if (points.length === 0) return '';
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`;

  let d = `M ${points[0].x} ${points[0].y}`;
  for (let i = 0; i < points.length - 1; i++) {
    const p0 = points[Math.max(0, i - 1)];
    const p1 = points[i];
    const p2 = points[i + 1];
    const p3 = points[Math.min(points.length - 1, i + 2)];

    const cp1x = p1.x + (p2.x - p0.x) / 6;
    const cp1y = p1.y + (p2.y - p0.y) / 6;
    const cp2x = p2.x - (p3.x - p1.x) / 6;
    const cp2y = p2.y - (p3.y - p1.y) / 6;

    d += ` C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${p2.x} ${p2.y}`;
  }
  return d;
};

export default function TrendChart({ data }: { data: TrendPoint[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const observer = new ResizeObserver((entries) => {
      const next = entries[0]?.contentRect.width ?? 0;
      setWidth(next);
    });
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  const chart = useMemo(() => {
    if (width <= 0 || data.length === 0) return null;

    const innerWidth = width - PAD.left - PAD.right;
    const innerHeight = CHART_HEIGHT - PAD.top - PAD.bottom;
    const totals = data.map((point) => point.scheduled + point.ongoing + point.completed + point.failed);
    const maxValue = Math.max(...totals, 0);
    const yMax = Math.max(4, Math.ceil(maxValue / 4) * 4);
    const stepX = data.length === 1 ? innerWidth / 2 : innerWidth / (data.length - 1);
    const xAt = (index: number) => PAD.left + index * stepX;
    const yAt = (value: number) => PAD.top + innerHeight - (value / yMax) * innerHeight;

    const points = totals.map((total, index) => ({ x: xAt(index), y: yAt(total) }));
    const linePath = buildSmoothPath(points);
    const areaPath = `${linePath} L ${points[points.length - 1].x} ${PAD.top + innerHeight} L ${points[0].x} ${PAD.top + innerHeight} Z`;
    const yTicks = Array.from({ length: 5 }, (_, index) => (yMax / 4) * index);

    return { innerWidth, innerHeight, totals, yMax, points, linePath, areaPath, yTicks };
  }, [width, data]);

  return (
    <div ref={containerRef} className="relative w-full" style={{ height: CHART_HEIGHT }}>
      {chart && (
        <svg
          width={width}
          height={CHART_HEIGHT}
          role="img"
          aria-label="Grafik area tren pemeliharaan per bulan"
          className="block"
        >
          <defs>
            <linearGradient id={GRADIENT_ID} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={LINE_COLOR} stopOpacity="0.4" />
              <stop offset="55%" stopColor={LINE_COLOR} stopOpacity="0.12" />
              <stop offset="100%" stopColor={LINE_COLOR} stopOpacity="0.02" />
            </linearGradient>
          </defs>

          {/* Grid horizontal */}
          {chart.yTicks.map((tick) => {
            const y = PAD.top + chart.innerHeight - (tick / chart.yMax) * chart.innerHeight;
            return (
              <line
                key={tick}
                x1={PAD.left}
                x2={width - PAD.right}
                y1={y}
                y2={y}
                className="stroke-border"
                strokeWidth={1}
              />
            );
          })}

          {/* Label sumbu Y */}
          {chart.yTicks.map((tick) => {
            const y = PAD.top + chart.innerHeight - (tick / chart.yMax) * chart.innerHeight;
            return (
              <text
                key={tick}
                x={PAD.left - 8}
                y={y + 3}
                textAnchor="end"
                fontSize={10}
                className="fill-muted"
              >
                {Math.round(tick)}
              </text>
            );
          })}

          {/* Area gradasi biru di bawah garis */}
          <path d={chart.areaPath} fill={`url(#${GRADIENT_ID})`} />

          {/* Garis tren melengkung halus */}
          <path
            d={chart.linePath}
            fill="none"
            stroke={LINE_COLOR}
            strokeWidth={2.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {/* Titik data */}
          {chart.points.map((point, index) => (
            <circle
              key={index}
              cx={point.x}
              cy={point.y}
              r={4}
              fill={LINE_COLOR}
              stroke="#ffffff"
              strokeWidth={2}
            />
          ))}

          {/* Label sumbu X */}
          {data.map((point, index) => (
            <text
              key={point.month}
              x={chart.points[index].x}
              y={CHART_HEIGHT - 8}
              textAnchor="middle"
              fontSize={10}
              className="fill-muted"
            >
              {formatMonth(point.month)}
            </text>
          ))}

          {/* Garis penanda saat hover */}
          {hoveredIndex !== null && chart.points[hoveredIndex] && (
            <>
              <line
                x1={chart.points[hoveredIndex].x}
                x2={chart.points[hoveredIndex].x}
                y1={PAD.top}
                y2={PAD.top + chart.innerHeight}
                stroke={LINE_COLOR}
                strokeWidth={1.5}
                strokeDasharray="4 4"
                opacity={0.55}
              />
              <circle
                cx={chart.points[hoveredIndex].x}
                cy={chart.points[hoveredIndex].y}
                r={6.5}
                fill={LINE_COLOR}
                stroke="#ffffff"
                strokeWidth={2}
              />
            </>
          )}

          {/* Area hit hover (lebih lebar dari titik visual) */}
          {chart.points.map((point, index) => (
            <circle
              key={index}
              cx={point.x}
              cy={point.y}
              r={16}
              fill="transparent"
              className="cursor-pointer"
              onMouseEnter={() => setHoveredIndex(index)}
              onMouseLeave={() => setHoveredIndex(null)}
            >
              <title>{`${formatMonth(data[index].month)}: ${chart.totals[index]} pekerjaan`}</title>
            </circle>
          ))}
        </svg>
      )}

      {/* Tooltip */}
      {chart && hoveredIndex !== null && chart.points[hoveredIndex] && (
        <div
          className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-xl border border-border bg-card px-3 py-2 text-xs shadow-lg"
          style={{ left: chart.points[hoveredIndex].x, top: chart.points[hoveredIndex].y - 10 }}
        >
          <p className="font-bold">{formatMonth(data[hoveredIndex].month)}</p>
          <p className="mt-0.5 text-muted">Total {chart.totals[hoveredIndex]} pekerjaan</p>
          <p className="mt-1 space-y-0.5 leading-relaxed text-muted">
            Terjadwal {data[hoveredIndex].scheduled} · Dalam proses {data[hoveredIndex].ongoing}
            <br />
            Selesai {data[hoveredIndex].completed} · Perlu Tindak Lanjut {data[hoveredIndex].failed}
          </p>
        </div>
      )}
    </div>
  );
}
