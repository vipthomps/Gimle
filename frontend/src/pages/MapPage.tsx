import { hostName } from "../functions/names";
import { createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { apiGetCategories, apiGetMap, apiResetMap, apiSaveMapPos } from "../functions/api";
import { MapData, MapHost } from "../functions/exports";
import { bounds, CT_ROW, GHOST_H, layout, NODE_H, NODE_W, Placed, topoBounds, topology, webLink } from "../functions/maplayout";
import HostPanel from "../components/Map/HostPanel";
import MapCanvas, { CanvasApi } from "../components/Map/MapCanvas";
import ViewTabs from "../components/Views/ViewTabs";
import { guestKind } from "../components/Containers/GuestList";
import { containerIcon, markIconFailed, pickIcon } from "../functions/icons";

const readBool = (key: string, def: boolean) => {
  const v = localStorage.getItem(key);
  return v == null ? def : v == "true";
};

const clip = (s: string, n: number) => s.length > n ? s.slice(0, n - 1) + "…" : s;

function MapPage() {

  const [data, setData] = createSignal<MapData>({ Subnets: [], Hosts: [] });
  const [showOffline, setShowOffline] = createSignal(readBool("mapOffline", true));
  const [topo, setTopo] = createSignal(readBool("mapTopology", false));
  const [categories, setCategories] = createSignal<string[]>([]);
  const [search, setSearch] = createSignal("");
  const [selected, setSelected] = createSignal<MapHost>();
  // Position of the node being dragged, by MAC
  const [dragPos, setDragPos] = createSignal<{ mac: string, x: number, y: number }>();

  let canvas: CanvasApi | undefined;
  let busy = false;

  const load = async () => {
    if (busy) return;
    const res: MapData = await apiGetMap();
    setData(res);
    const sel = selected();
    if (sel) setSelected(res.Hosts.find(h => h.Mac == sel.Mac));
  };

  onMount(() => {
    load();
    apiGetCategories().then(setCategories);
    const timer = setInterval(load, 30000);
    onCleanup(() => clearInterval(timer));
  });

  const placed = createMemo(() => layout(data(), showOffline()));

  const nodePos = (p: Placed) => {
    const d = dragPos();
    return d && d.mac == p.host.Mac ? d : p;
  };

  const matches = (h: MapHost) => {
    const q = search().trim().toLowerCase();
    if (q == "") return true;
    return [h.Name, h.IP, h.Mac, h.Hw, h.DNS].some(v => v && v.toLowerCase().includes(q));
  };

  const topoPlaced = createMemo(() => topology(data(), showOffline(), categories()));

  const box = () => data().Hosts.length == 0 ? null :
    topo() ? topoBounds(topoPlaced()) : bounds(placed().zones, placed().nodes);

  // Drag a node to move it; a click without movement opens the side panel
  const onNodeDown = (e: PointerEvent, p: Placed) => {
    if (e.button != 0) return;
    e.stopPropagation();
    busy = true;
    const start = canvas!.toMap(e);
    const origin = nodePos(p);
    let moved = false;

    const move = (ev: PointerEvent) => {
      const m = canvas!.toMap(ev);
      const dx = m.x - start.x, dy = m.y - start.y;
      if (!moved && Math.abs(dx) + Math.abs(dy) < 4 / canvas!.scale()) return;
      moved = true;
      setDragPos({ mac: p.host.Mac, x: Math.round(origin.x + dx), y: Math.round(origin.y + dy) });
    };
    const up = async () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      const d = dragPos();
      if (moved && d) {
        await apiSaveMapPos(d.mac, d.x, d.y);
        busy = false;
        await load();
        setDragPos(undefined);
      } else {
        busy = false;
        setSelected(p.host);
      }
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  const handleReset = async () => {
    if (confirm("Put every host back in its automatic spot?")) {
      await apiResetMap();
      await load();
      canvas?.fit();
    }
  };

  const toggleTopo = (v: boolean) => {
    setTopo(v);
    try { localStorage.setItem("mapTopology", String(v)); } catch {}
    setTimeout(() => canvas?.fit());
  };

  const toggleOffline = (v: boolean) => {
    setShowOffline(v);
    try { localStorage.setItem("mapOffline", String(v)); } catch {}
  };

  const online = () => data().Hosts.filter(h => h.Now == 1).length;

  return (
    <>
    <ViewTabs active={0}></ViewTabs>
    <div class="card border-primary">
      <div class="card-header d-flex flex-wrap gap-2 align-items-center">
        <span class="me-auto">
          <b>{online()}</b> of {data().Hosts.length} hosts online
        </span>
        <input class="form-control form-control-sm" style="max-width: 14em;" placeholder="Find a host"
          value={search()} onInput={e => setSearch(e.target.value)}></input>
        <div class="form-check form-switch mb-0">
          <input class="form-check-input" type="checkbox" id="mapOffline" checked={showOffline()}
            onChange={e => toggleOffline(e.target.checked)}></input>
          <label class="form-check-label" for="mapOffline">Offline</label>
        </div>
        <div class="btn-group btn-group-sm" role="group" aria-label="Map layout">
          <button class={"btn " + (topo() ? "btn-outline-secondary" : "btn-secondary")} onClick={() => toggleTopo(false)}
            title="Arrange hosts yourself by dragging them"><i class="bi bi-arrows-move me-1"></i>Free</button>
          <button class={"btn " + (topo() ? "btn-secondary" : "btn-outline-secondary")} onClick={() => toggleTopo(true)}
            title="Automatic layout: gateway, then hosts by category, with their containers"><i class="bi bi-diagram-2 me-1"></i>Topology</button>
        </div>
        <Show when={!topo()}>
          <button class="btn btn-sm btn-outline-secondary" onClick={handleReset} title="Reset layout"><i class="bi bi-arrow-counterclockwise"></i></button>
        </Show>
      </div>
      <div class="map-wrap">
        <MapCanvas bounds={box()} api={a => canvas = a} onBackgroundClick={() => setSelected(undefined)}>
          <Show when={topo()} fallback={<>
          <For each={placed().zones}>{z =>
            <g>
              <rect class="map-zone" x={z.x} y={z.y} width={z.w} height={z.h} rx="14"></rect>
              <text class="map-zone-title" x={z.x + 18} y={z.y + 24}>
                {z.name}
                <tspan class="map-zone-sub" dx="10">{z.cidr ? z.cidr + " · " : ""}{z.online}/{z.total} online</tspan>
              </text>
            </g>
          }</For>
          <For each={placed().nodes}>{p =>
            <g class={"map-node" + (p.host.Now == 1 ? " on" : " off") + (selected()?.Mac == p.host.Mac ? " sel" : "") + (matches(p.host) ? "" : " dim")}
              transform={`translate(${nodePos(p).x},${nodePos(p).y})`}
              onPointerDown={e => onNodeDown(e, p)}>
              <title>{hostName(p.host, p.host.IP)}{"\n"}{p.host.IP} · {p.host.Mac}</title>
              <rect class="map-node-box" width={NODE_W} height={NODE_H} rx="9"></rect>
              <rect class="map-node-stripe" width="5" height={NODE_H - 16} x="7" y="8" rx="2.5"></rect>
              <Show when={pickIcon(p.host.Icon, "", false)}>{url =>
                <image href={url()} x="17" y="19" width="20" height="20" onError={() => markIconFailed(url())}></image>
              }</Show>
              <text class="map-node-name" x={pickIcon(p.host.Icon, "", false) ? 43 : 20} y="24">{clip(hostName(p.host), pickIcon(p.host.Icon, "", false) ? 15 : 19)}</text>
              <text class="map-node-ip" x={pickIcon(p.host.Icon, "", false) ? 43 : 20} y="44">{p.host.IP}</text>
              <Show when={p.host.Web.length > 0}>
                <g class="map-node-web-link"
                  onPointerDown={e => e.stopPropagation()}
                  onClick={() => window.open(webLink(p.host.IP, p.host.Web[0].Port, p.host.Web[0].Web), "_blank")}>
                  <title>Open {p.host.Web[0].Service || "web service"}{p.host.Web.length > 1 ? " (" + (p.host.Web.length - 1) + " more in details)" : ""}</title>
                  <rect class="map-node-web" x={NODE_W - 46} y="32" width="38" height="18" rx="9"></rect>
                  <text class="map-node-web-text" x={NODE_W - 27} y="45" text-anchor="middle">{p.host.Web.length > 1 ? "web " + p.host.Web.length : "web"}</text>
                </g>
              </Show>
            </g>
          }</For>
          </>}>
          <For each={topoPlaced().zones}>{z =>
            <g>
              <rect class="map-zone" x={z.x} y={z.y} width={z.w} height={z.h} rx="14"></rect>
              <text class="map-zone-title" x={z.x + 18} y={z.y + 24}>
                {z.name}
                <tspan class="map-zone-sub" dx="10">{z.cidr ? z.cidr + " · " : ""}{z.online}/{z.total} online</tspan>
              </text>
            </g>
          }</For>
          <For each={topoPlaced().edges}>{e => <path class="topo-edge" d={e.d}></path>}</For>
          <For each={topoPlaced().blocks}>{b =>
            <g>
              <rect class="topo-block" x={b.x} y={b.y} width={b.w} height={b.h} rx="11"></rect>
              <text class="topo-block-title" x={b.x + 12} y={b.y + 18}>
                {b.title}
                <tspan class="map-zone-sub" dx="8">{b.sub}</tspan>
              </text>
            </g>
          }</For>
          <For each={topoPlaced().ghosts}>{gh =>
            <g class="topo-ghost" transform={`translate(${gh.x},${gh.y})`}>
              <title>{guestKind(gh.guest)} {gh.guest.VMID} on {gh.guest.Node}, {gh.guest.Status}{"\n"}Not seen on the network{gh.guest.Mac ? " (" + gh.guest.Mac + ")" : ""}</title>
              <rect class="topo-ghost-box" width={NODE_W} height={GHOST_H} rx="8"></rect>
              <text class="topo-ghost-name" x="12" y="17">{clip(gh.guest.VMID + " " + gh.guest.Name, 21)}</text>
              <text class="topo-ghost-sub" x="12" y="31">{guestKind(gh.guest)} · {gh.guest.Status}</text>
            </g>
          }</For>
          <For each={topoPlaced().columns}>{c =>
            <text class="topo-col" x={c.x + c.w / 2} y={c.y + 18} text-anchor="middle">{c.name}</text>
          }</For>
          <For each={topoPlaced().nodes}>{n =>
            <g class={"map-node" + (n.host.Now == 1 ? " on" : " off") + (selected()?.Mac == n.host.Mac ? " sel" : "") + (matches(n.host) ? "" : " dim") + (n.gateway ? " gateway" : "")}
              transform={`translate(${n.x},${n.y})`}
              onClick={() => setSelected(n.host)}>
              <title>{hostName(n.host, n.host.IP)}{"\n"}{n.host.IP} · {n.host.Mac}{n.host.Category ? "\n" + n.host.Category : ""}</title>
              <rect class="map-node-box" width={NODE_W} height={n.h} rx="9"></rect>
              <rect class="map-node-stripe" width="5" height={NODE_H - 16} x="7" y="8" rx="2.5"></rect>
              <Show when={pickIcon(n.host.Icon, "", false)}>{url =>
                <image href={url()} x="17" y="19" width="20" height="20" onError={() => markIconFailed(url())}></image>
              }</Show>
              <text class="map-node-name" x={pickIcon(n.host.Icon, "", false) ? 43 : 20} y="24">{clip(hostName(n.host), pickIcon(n.host.Icon, "", false) ? 15 : 19)}</text>
              <text class="map-node-ip" x={pickIcon(n.host.Icon, "", false) ? 43 : 20} y="44">{n.host.IP}{n.host.GuestOf ? " · " + guestKind(n.host.GuestOf) + " " + n.host.GuestOf.VMID : ""}{n.host.Containers?.length && !n.host.GuestOf ? " · " + n.host.Containers.filter(c => c.State == "running").length + " ctr" : ""}</text>
              <For each={n.shown}>{(c, i) =>
                <g class="topo-ct" transform={`translate(10,${NODE_H + 2 + i() * CT_ROW})`}>
                  <title>{c.Name}{"\n"}{c.Image}{c.Ports.filter(p => p.Public > 0).map(p => " :" + p.Public).join("")}</title>
                  <rect class="topo-ct-box" width={NODE_W - 20} height={CT_ROW - 3} rx="4"></rect>
                  <Show when={pickIcon("", containerIcon(c.Image, c.Name), true)}>{url =>
                    <image href={url()} x="3" y="2" width="13" height="13" onError={() => markIconFailed(url())}></image>
                  }</Show>
                  <text class="topo-ct-text" x="20" y="12.5">{clip(c.Name, 15)}</text>
                  <Show when={c.Ports.find(p => p.Public > 0)}>{p =>
                    <text class="topo-ct-port" x={NODE_W - 24} y="12.5" text-anchor="end">:{p().Public}</text>
                  }</Show>
                </g>
              }</For>
              <Show when={n.more > 0}>
                <text class="topo-ct-more" x="14" y={NODE_H + 14 + n.shown.length * CT_ROW}>+{n.more} more</text>
              </Show>
            </g>
          }</For>
          </Show>
        </MapCanvas>
        <Show when={data().Hosts.length == 0}>
          <div class="map-empty">No hosts yet. Gimlé adds them as it scans your <a href="/subnets">subnets</a>.</div>
        </Show>
        <Show when={selected()}>{h =>
          <HostPanel host={h()} onClose={() => setSelected(undefined)} onChange={load}
            onOpen={mac => setSelected(data().Hosts.find(x => x.Mac == mac) ?? h())}></HostPanel>
        }</Show>
      </div>
    </div>
    </>
  )
}

export default MapPage
