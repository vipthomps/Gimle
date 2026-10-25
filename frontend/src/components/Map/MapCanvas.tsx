import { createEffect, createSignal, JSX, onCleanup, onMount } from "solid-js";

export interface Box { x: number, y: number, w: number, h: number };

export interface CanvasApi {
  toMap: (e: PointerEvent | WheelEvent) => { x: number, y: number };
  scale: () => number;
  fit: () => void;
  // true while two fingers are on the map, and until the last one lifts:
  // drags and taps that began then are not moves or clicks
  pinching: () => boolean;
};

// Pan and zoom surface shared by the network map and map-layout views.
// It fits the content once it appears and on resize, until the user pans or zooms.
function MapCanvas(props: {
  bounds: Box | null,
  api?: (api: CanvasApi) => void,
  onBackgroundClick?: () => void,
  children: JSX.Element,
}) {

  const [view, setView] = createSignal({ x: 0, y: 0, k: 1 });
  let svg: SVGSVGElement | undefined;
  let fitted = false;
  let userMoved = false;

  // Touch points on the map, for pinch zoom with two fingers
  const touches = new Map<number, { x: number, y: number }>();
  let pinch: { d: number, mx: number, my: number, v: { x: number, y: number, k: number } } | null = null;
  let pinched = false;

  const fit = () => {
    const b = props.bounds;
    if (!svg || !b) return;
    const r = svg.getBoundingClientRect();
    const k = Math.min(1.2, Math.max(0.2, Math.min((r.width - 40) / b.w, (r.height - 40) / b.h)));
    setView({ k, x: (r.width - b.w * k) / 2 - b.x * k, y: 20 - b.y * k });
  };

  const toMap = (e: PointerEvent | WheelEvent) => {
    const r = svg!.getBoundingClientRect();
    const v = view();
    return { x: (e.clientX - r.left - v.x) / v.k, y: (e.clientY - r.top - v.y) / v.k };
  };

  // On a phone the whole network is too small to read, so the first view
  // starts at the top left at a readable size; Fit still shows everything
  const firstView = () => {
    const b = props.bounds;
    if (!svg || !b) return;
    fit();
    const r = svg.getBoundingClientRect();
    if (r.width < 600 && view().k < 0.6) {
      const k = 0.6;
      setView({ k, x: 12 - b.x * k, y: 12 - b.y * k });
    }
  };

  createEffect(() => {
    if (!fitted && props.bounds) {
      fitted = true;
      firstView();
    }
  });

  onMount(() => {
    const ro = new ResizeObserver(() => { if (fitted && !userMoved) firstView(); });
    ro.observe(svg!);
    onCleanup(() => ro.disconnect());
    props.api?.({ toMap, scale: () => view().k, fit: () => { userMoved = false; firstView(); }, pinching: () => pinched });

    const move = (e: PointerEvent) => {
      if (!touches.has(e.pointerId)) return;
      touches.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (!pinch || touches.size < 2) return;
      const [a, b] = [...touches.values()];
      const r = svg!.getBoundingClientRect();
      const d = Math.hypot(a.x - b.x, a.y - b.y) || 1;
      const mx = (a.x + b.x) / 2 - r.left, my = (a.y + b.y) / 2 - r.top;
      const v = pinch.v;
      const k = Math.min(3, Math.max(0.15, v.k * d / pinch.d));
      // keep the point between the fingers under them, and follow them as they move
      setView({ k, x: mx - (pinch.mx - v.x) * k / v.k, y: my - (pinch.my - v.y) * k / v.k });
    };
    const up = (e: PointerEvent) => {
      touches.delete(e.pointerId);
      if (touches.size < 2) pinch = null;
    };
    // capture, so fingers that land on a host count too
    svg!.addEventListener("pointerdown", onTouchDown, { capture: true });
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", up);
    onCleanup(() => {
      svg?.removeEventListener("pointerdown", onTouchDown, { capture: true });
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", up);
    });
  });

  // Every finger that lands on the map, before hosts see it
  const onTouchDown = (e: PointerEvent) => {
    if (e.pointerType == "mouse") return;
    if (touches.size == 0) pinched = false;
    touches.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (touches.size == 2) {
      const [a, b] = [...touches.values()];
      const r = svg!.getBoundingClientRect();
      pinch = {
        d: Math.hypot(a.x - b.x, a.y - b.y) || 1,
        mx: (a.x + b.x) / 2 - r.left, my: (a.y + b.y) / 2 - r.top,
        v: view(),
      };
      pinched = userMoved = true;
    }
  };

  // Zoom by factor around a point on screen (default: the centre)
  const zoom = (factor: number, px?: number, py?: number) => {
    userMoved = true;
    const v = view();
    const r = svg!.getBoundingClientRect();
    px = px ?? r.width / 2;
    py = py ?? r.height / 2;
    const k = Math.min(3, Math.max(0.15, v.k * factor));
    setView({ k, x: px - (px - v.x) * k / v.k, y: py - (py - v.y) * k / v.k });
  };

  const onWheel = (e: WheelEvent) => {
    e.preventDefault();
    const r = svg!.getBoundingClientRect();
    zoom(e.deltaY < 0 ? 1.12 : 1 / 1.12, e.clientX - r.left, e.clientY - r.top);
  };

  // Drag the background to pan
  const onBackgroundDown = (e: PointerEvent) => {
    if (e.button != 0) return;
    const start = { cx: e.clientX, cy: e.clientY, ...view() };
    let moved = false;
    const move = (ev: PointerEvent) => {
      if (pinched) return;
      const dx = ev.clientX - start.cx, dy = ev.clientY - start.cy;
      if (Math.abs(dx) + Math.abs(dy) > 3) moved = userMoved = true;
      setView({ k: start.k, x: start.x + dx, y: start.y + dy });
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      if (!moved && !pinched) props.onBackgroundClick?.();
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  return (
    <>
    <svg ref={svg} class="map-svg" onWheel={onWheel} onPointerDown={onBackgroundDown}>
      <g transform={`translate(${view().x},${view().y}) scale(${view().k})`}>
        {props.children}
      </g>
    </svg>
    <div class="btn-group-vertical map-zoom shadow-sm">
      <button class="btn btn-sm btn-light border" onClick={() => zoom(1.25)} title="Zoom in"><i class="bi bi-plus-lg"></i></button>
      <button class="btn btn-sm btn-light border" onClick={() => { userMoved = false; fit(); }} title="Fit to screen"><i class="bi bi-arrows-fullscreen"></i></button>
      <button class="btn btn-sm btn-light border" onClick={() => zoom(1 / 1.25)} title="Zoom out"><i class="bi bi-dash-lg"></i></button>
    </div>
    </>
  )
}

export default MapCanvas
