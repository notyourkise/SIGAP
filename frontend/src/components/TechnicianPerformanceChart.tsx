'use client';

import { useEffect, useRef, useState } from 'react';

export interface TechnicianPerformanceChartRow {
  user_id: string;
  full_name: string;
  completion_rate: number;
}

const COMMAND_GREEN = '#467235';
const SIGNAL_YELLOW = '#c88f00';
const ERROR_RED = '#b54432';

function rateColor(rate: number) {
  if (rate >= 80) return COMMAND_GREEN;
  if (rate >= 50) return SIGNAL_YELLOW;
  return ERROR_RED;
}

function clampRate(rate: number) {
  return Math.min(100, Math.max(0, Number.isFinite(rate) ? rate : 0));
}

function shortName(name: string) {
  return name.length > 22 ? `${name.slice(0, 20)}...` : name;
}

export default function TechnicianPerformanceChart({
  data,
}: {
  data: TechnicianPerformanceChartRow[];
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;

    const observer = new ResizeObserver((entries) => {
      setWidth(entries[0]?.contentRect.width ?? 0);
    });
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  const rowHeight = 40;
  const chartHeight = Math.max(96, data.length * rowHeight + 24);
  const labelWidth = width < 480 ? 112 : 166;
  const valueWidth = 48;
  const barWidth = Math.max(80, width - labelWidth - valueWidth - 18);

  return (
    <div
      ref={containerRef}
      className="w-full"
      style={{ minHeight: chartHeight }}
    >
      {width > 0 && data.length > 0 ? (
        <svg
          width={width}
          height={chartHeight}
          role="img"
          aria-label="Grafik batang completion rate per teknisi"
          className="block"
        >
          {data.map((technician, index) => {
            const rate = clampRate(technician.completion_rate);
            const y = 12 + index * rowHeight;
            const color = rateColor(rate);

            return (
              <g key={technician.user_id}>
                <title>{`${technician.full_name}: ${rate.toFixed(1)}% completion rate`}</title>
                <text
                  x={0}
                  y={y + 15}
                  fontSize={12}
                  fontWeight={600}
                  className="fill-foreground"
                >
                  {shortName(technician.full_name)}
                </text>
                <rect
                  x={labelWidth}
                  y={y + 6}
                  width={barWidth}
                  height={12}
                  rx={6}
                  className="fill-background"
                />
                <rect
                  x={labelWidth}
                  y={y + 6}
                  width={(rate / 100) * barWidth}
                  height={12}
                  rx={6}
                  fill={color}
                />
                <text
                  x={labelWidth + barWidth + 10}
                  y={y + 16}
                  fontSize={12}
                  fontWeight={700}
                  textAnchor="start"
                  fill={color}
                >
                  {rate.toFixed(1)}%
                </text>
              </g>
            );
          })}
        </svg>
      ) : (
        <div className="flex min-h-24 items-center justify-center text-sm text-muted" role="status">
          Menyiapkan grafik...
        </div>
      )}
    </div>
  );
}
