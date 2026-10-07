import { createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { niceMax, tickLabel, timeLabel } from "../../functions/stats";

export interface Point { x: string, y: number | null };

const PAD = { top: 10, right: 12, bottom: 24, left: 36 };

// A single-series line over time with a crosshair tooltip. Points with a null
// value (no scans in that bucket) break the line instead of drawing zero.
function LineChart(props: {
  points: Point[],
  label: string,
  max?: number,
  height?: number,
  format?: (v: number) => string,
}) {

  let wrap: HTMLDivElement | undefined;
  const [width, setWidth] = createSignal(600);
  const [hover, setHover] = createSignal<number>();
  const height = () => props.height ?? 180;
  const fmt = (v: number) => props.format ? props.format(v) : String(Math.round(v * 10) / 10);

  onMount(() => {
    const ro = new ResizeObserver(() => setWidth(wrap!.clientWidth || 600));
    ro.observe(wrap!);
    onCleanup(() => ro.disconnect());
  });

  const top = () => niceMax(props.max ?? Math.max(0, ...props.points.map(p => p.y ?? 0)));
  const plotW = () => Math.max(10, width() - PAD.left - PAD.right);
  const plotH = () => height() - PAD.top - PAD.bottom;
  const xAt = (i: number) => PAD.left + (props.points.length <= 1 ? plotW() / 2 : i * plotW() / (props.points.length - 1));
  const yAt = (v: number) => PAD.top + plotH() - (v / top()) * plotH();

  // Line and area paths, split into runs where data exists
  const paths = createMemo(() => {
    let line = "", area = "";
    let run: [number, number][] = [];
    const flush = () => {
      if (run.length == 0) return;
      line += run.map(([x, y], i) => (i ? "L" : "M") + x.toFixed(1) + "," + y.toFixed(1)).join("");
      const base = yAt(0).toFixed(1);
      area += `M${run[0][0].toFixed(1)},${base}` + run.map(([x, y]) => `L${x.toFixed(1)},${y.toFixed(1)}`).join("") +
        `L${run[run.length - 1][0].toFixed(1)},${base}Z`;
      run = [];
    };
    props.points.forEach((p, i) => p.y == null ? flush() : run.push([xAt(i), yAt(p.y)]));
    flush();
    return { line, area };
  });

  // Lone points between gaps get a dot so they are visible
  const lone = createMemo(() => props.points.map((p, i) => ({ p, i })).filter(({ p, i }) =>
    p.y != null && (props.points[i - 1]?.y ?? null) == null && (props.points[i + 1]?.y ?? null) == null));

  const yTicks = () => [0, top() / 2, top()];
  const xTicks = createMemo(() => {
    const n = props.points.length;
    if (n == 0) return [];
    const want = Math.max(2, Math.min(6, Math.floor(plotW() / 110)));
    // Hourly points over several days: tick at midnights so each label is a day
    const midnights = props.points.map((p, i) => p.x.endsWith(" 00") ? i : -1).filter(i => i >= 0);
    if (midnights.length >= 2) {
      const every = Math.max(1, Math.ceil(midnights.length / want));
      return midnights.filter((_, k) => k % every == 0);
    }
    const step = Math.max(1, Math.round((n - 1) / (want - 1)));
    const ticks: number[] = [];
    for (let i = 0; i < n; i += step) ticks.push(i);
    if (n - 1 - ticks[ticks.length - 1] >= step / 2) ticks.push(n - 1);
    return ticks;
  });

  const onMove = (e: PointerEvent) => {
    const r = (e.currentTarget as SVGElement).getBoundingClientRect();
    const n = props.points.length;
    if (n == 0) return;
    const i = Math.round(((e.clientX - r.left - PAD.left) / plotW()) * (n - 1));
    setHover(Math.max(0, Math.min(n - 1, i)));
  };

  const onKey = (e: KeyboardEvent) => {
    const n = props.points.length;
    if (n == 0) return;
    const i = hover() ?? n - 1;
    if (e.key == "ArrowLeft") setHover(Math.max(0, i - 1));
    else if (e.key == "ArrowRight") setHover(Math.min(n - 1, i + 1));
    else return;
    e.preventDefault();
  };

  const tipLeft = () => {
    const x = xAt(hover()!);
    return x > width() - 150 ? x - 150 : x + 10;
  };

  return (
    <div class="chart" ref={wrap}>
      <svg width={width()} height={height()} role="img" aria-label={props.label} tabindex="0"
        onPointerMove={onMove} onPointerLeave={() => setHover(undefined)}
        onFocus={() => setHover(props.points.length - 1)} onBlur={() => setHover(undefined)} onKeyDown={onKey}>
        <For each={yTicks()}>{t =>
          <g>
            <line class="chart-grid" x1={PAD.left} x2={PAD.left + plotW()} y1={yAt(t)} y2={yAt(t)}></line>
            <text class="chart-tick" x={PAD.left - 6} y={yAt(t) + 4} text-anchor="end">{fmt(t)}</text>
          </g>
        }</For>
        <For each={xTicks()}>{i =>
          <text class="chart-tick" x={xAt(i)} y={height() - 6}
            text-anchor={i == 0 ? "start" : i == props.points.length - 1 ? "end" : "middle"}>{tickLabel(props.points[i].x)}</text>
        }</For>
        <path class="chart-area" d={paths().area}></path>
        <path class="chart-line" d={paths().line}></path>
        <For each={lone()}>{({ p, i }) =>
          <circle class="chart-dot" cx={xAt(i)} cy={yAt(p.y!)} r="3"></circle>
        }</For>
        <Show when={hover() != undefined}>
          <line class="chart-cross" x1={xAt(hover()!)} x2={xAt(hover()!)} y1={PAD.top} y2={PAD.top + plotH()}></line>
          <Show when={props.points[hover()!].y != null}>
            <circle class="chart-dot sel" cx={xAt(hover()!)} cy={yAt(props.points[hover()!].y!)} r="4"></circle>
          </Show>
        </Show>
      </svg>
      <Show when={hover() != undefined}>
        <div class="chart-tip" style={{ left: tipLeft() + "px", top: PAD.top + "px" }}>
          <b>{props.points[hover()!].y == null ? "No scans" : fmt(props.points[hover()!].y!)}</b>
          <span>{timeLabel(props.points[hover()!].x)}</span>
        </div>
      </Show>
    </div>
  )
}

export default LineChart
