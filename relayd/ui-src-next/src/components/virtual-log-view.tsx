"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { logLineTone } from "@/lib/relay-utils";

const LINE_HEIGHT = 18; // px — fixed row height keeps the windowing math exact
const OVERSCAN = 30;

interface VirtualLogViewProps {
  lines: string[];
  className?: string;
  emptyText?: string;
}

// Windowed log renderer: only the visible slice (plus overscan) reaches the
// DOM, so a 4,000-line build log mounts ~80 nodes instead of 4,000 and stays
// smooth while streaming. Long lines scroll horizontally instead of wrapping
// so every row is exactly one line tall. Follows the tail until the user
// scrolls up, then stays put.
export function VirtualLogView({ lines, className, emptyText }: VirtualLogViewProps) {
  const ref = useRef<HTMLDivElement>(null);
  const followRef = useRef(true);
  const [range, setRange] = useState({ start: 0, end: OVERSCAN * 2 });

  const recompute = () => {
    const el = ref.current;
    if (!el) return;
    const start = Math.max(0, Math.floor(el.scrollTop / LINE_HEIGHT) - OVERSCAN);
    const end = Math.min(
      lines.length,
      Math.ceil((el.scrollTop + el.clientHeight) / LINE_HEIGHT) + OVERSCAN,
    );
    setRange((prev) => (prev.start === start && prev.end === end ? prev : { start, end }));
  };

  const onScroll = () => {
    const el = ref.current;
    if (!el) return;
    followRef.current = el.scrollTop + el.clientHeight >= el.scrollHeight - LINE_HEIGHT * 2;
    recompute();
  };

  useEffect(() => {
    const el = ref.current;
    if (el && followRef.current) {
      el.scrollTop = el.scrollHeight;
    }
    recompute();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lines.length, lines[lines.length - 1]]);

  const visible = lines.slice(range.start, range.end);

  return (
    <div ref={ref} onScroll={onScroll} className={cn("overflow-y-auto overflow-x-auto", className)}>
      {!lines.length && emptyText && <div className="text-white/30">{emptyText}</div>}
      <div style={{ height: lines.length * LINE_HEIGHT, position: "relative" }}>
        <div style={{ position: "absolute", top: range.start * LINE_HEIGHT, left: 0, right: 0 }}>
          {visible.map((line, i) => (
            <div
              key={range.start + i}
              className={cn("log-line", `log-line--${logLineTone(line)}`)}
              style={{ height: LINE_HEIGHT, lineHeight: `${LINE_HEIGHT}px`, whiteSpace: "pre" }}
            >
              {line}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
