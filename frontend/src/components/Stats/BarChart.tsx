import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { niceMax, tickLabel, timeLabel } from "../../functions/stats";

const PAD = { top: 10, right: 12, bottom: 24, left: 36 };

// A single-series bar chart over days; each bar carries its own tooltip
function BarChart(props: { points: { x: string, y: number }[], label: string, unit: string, height?: number }) {

  let wrap: HTMLDivElement | undefined;
  const [width, setWidth] = createSignal(600);
  const [hover, setHover] = createSignal<number>();
  const height = () => props.height ?? 140;

  onMount(() => {
    const ro = new ResizeObserver(() => setWidth(wrap!.clientWidth || 600));
    ro.observe(wrap!);
    onCleanup(() => ro.disconnect());
  });

  const top = () => niceMax(Math.max(1, ...props.points.map(p => p.y)));
  const plotW = () => Math.max(10, width() - PAD.left - PAD.right);
  const plotH = () => height() - PAD.top - PAD.bottom;
  const slot = () => plotW() / Math.max(1, props.points.length);
  const barW = () => Math.max(2, Math.min(28, slot() - 2));
  const xAt = (i: number) => PAD.left + i * slot() + (slot() - barW()) / 2;
  const yAt = (v: number) => PAD.top + plotH() - (v / top()) * plotH();

  // Rounded top, square base
  const bar = (i: number, v: number) => {
    const x = xAt(i), w = barW(), y = yAt(v), b = yAt(0);
    const r = Math.min(4, w / 2, b - y);
    return `M${x},${b}V${y + r}Q${x},${y} ${x + r},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${b}Z`;
  };

  const xTicks = () => {
    const n = props.points.length;
    if (n == 0) return [];
    const step = Math.max(1, Math.ceil(n / Math.max(2, Math.floor(plotW() / 90))));
    const ticks: number[] = [];
    for (let i = 0; i < n; i += step) ticks.push(i);
    return ticks;
  };

  return (
    <div class="chart" ref={wrap}>
      <svg width={width()} height={height()} role="img" aria-label={props.label}>
        <For each={[0, top()]}>{t =>
          <g>
            <line class="chart-grid" x1={PAD.left} x2={PAD.left + plotW()} y1={yAt(t)} y2={yAt(t)}></line>
            <text class="chart-tick" x={PAD.left - 6} y={yAt(t) + 4} text-anchor="end">{t}</text>
          </g>
        }</For>
        <For each={xTicks()}>{i =>
          <text class="chart-tick" x={xAt(i) + barW() / 2} y={height() - 6} text-anchor="middle">{tickLabel(props.points[i].x)}</text>
        }</For>
        <For each={props.points}>{(p, i) =>
          <g tabindex={p.y > 0 ? "0" : undefined} onPointerEnter={() => setHover(i())} onPointerLeave={() => setHover(undefined)}
            onFocus={() => setHover(i())} onBlur={() => setHover(undefined)}>
            <rect class="chart-hit" x={PAD.left + i() * slot()} y={PAD.top} width={slot()} height={plotH()}></rect>
            <Show when={p.y > 0}>
              <path class={"chart-bar" + (hover() == i() ? " sel" : "")} d={bar(i(), p.y)}></path>
            </Show>
          </g>
        }</For>
      </svg>
      <Show when={hover() != undefined}>
        <div class="chart-tip" style={{ left: Math.min(width() - 150, xAt(hover()!) + barW() + 6) + "px", top: PAD.top + "px" }}>
          <b>{props.points[hover()!].y} {props.unit}</b>
          <span>{timeLabel(props.points[hover()!].x)}</span>
        </div>
      </Show>
    </div>
  )
}

export default BarChart
