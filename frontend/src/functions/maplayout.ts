import { Container, Guest, MapData, MapHost, Subnet, ViewGroup, ViewItem } from "./exports";

export const NODE_W = 168;
export const NODE_H = 58;
const GAP = 14;
const PAD = 18;
const HEAD = 36;
const ZONE_GAP = 44;
const ROW_WIDTH = 1500;

export interface Zone {
  id:     number;  // subnet ID, 0 for hosts outside every subnet
  name:   string;
  cidr:   string;
  x:      number;
  y:      number;
  w:      number;
  h:      number;
  online: number;
  total:  number;
};

export interface Placed {
  host: MapHost;
  x:    number;
  y:    number;
};

const ipNum = (ip: string) =>
  ip.split(".").reduce((n, part) => n * 256 + (parseInt(part) || 0), 0);

// Lay out hosts in one zone per subnet. Hosts with a saved position keep it;
// the rest fill a grid inside their zone, sorted by IP.
export function layout(data: MapData, showOffline: boolean): { zones: Zone[], nodes: Placed[] } {

  const groups: { subnet: Subnet | null, hosts: MapHost[] }[] =
    data.Subnets.map(s => ({ subnet: s, hosts: [] as MapHost[] }));
  const other: MapHost[] = [];

  for (const h of data.Hosts) {
    if (!showOffline && h.Now != 1) continue;
    const g = groups.find(g => g.subnet!.ID == h.SubnetID);
    g ? g.hosts.push(h) : other.push(h);
  }
  if (other.length > 0) {
    groups.push({ subnet: null, hosts: other });
  }

  const zones: Zone[] = [];
  const nodes: Placed[] = [];
  let cx = 0, cy = 0, rowH = 0;

  for (const g of groups) {
    g.hosts.sort((a, b) => ipNum(a.IP) - ipNum(b.IP));
    const auto = g.hosts.filter(h => !h.HasPos);

    const n = Math.max(auto.length, 1);
    const cols = Math.min(6, Math.max(2, Math.ceil(Math.sqrt(n * 1.5))));
    const rows = Math.ceil(n / cols);
    const gridW = cols * NODE_W + (cols - 1) * GAP;
    const gridH = rows * NODE_H + (rows - 1) * GAP;
    const w = gridW + 2 * PAD;
    const h = gridH + HEAD + PAD;

    if (cx > 0 && cx + w > ROW_WIDTH) {
      cx = 0;
      cy += rowH + ZONE_GAP;
      rowH = 0;
    }

    const zone: Zone = {
      id: g.subnet ? g.subnet.ID : 0,
      name: g.subnet ? g.subnet.Name : "Other hosts",
      cidr: g.subnet ? g.subnet.CIDR : "",
      x: cx, y: cy, w, h,
      online: g.hosts.filter(h => h.Now == 1).length,
      total: g.hosts.length,
    };

    auto.forEach((host, i) => {
      nodes.push({
        host,
        x: cx + PAD + (i % cols) * (NODE_W + GAP),
        y: cy + HEAD + Math.floor(i / cols) * (NODE_H + GAP),
      });
    });

    // Grow the zone around hosts that were dragged elsewhere
    for (const host of g.hosts.filter(h => h.HasPos)) {
      nodes.push({ host, x: host.X, y: host.Y });
      const x0 = Math.min(zone.x, host.X - PAD);
      const y0 = Math.min(zone.y, host.Y - HEAD);
      const x1 = Math.max(zone.x + zone.w, host.X + NODE_W + PAD);
      const y1 = Math.max(zone.y + zone.h, host.Y + NODE_H + PAD);
      zone.x = x0; zone.y = y0; zone.w = x1 - x0; zone.h = y1 - y0;
    }

    zones.push(zone);
    cx += w + ZONE_GAP;
    rowH = Math.max(rowH, h);
  }

  return { zones, nodes };
}

export interface PlacedItem {
  item: ViewItem;
  x:    number;
  y:    number;
};

// Lay out a view's groups as one zone per tag, items in a grid in tag order
export function groupLayout(groups: ViewGroup[]): { zones: Zone[], nodes: PlacedItem[] } {
  const zones: Zone[] = [];
  const nodes: PlacedItem[] = [];
  let cx = 0, cy = 0, rowH = 0;

  groups.forEach((g, gi) => {
    const n = Math.max(g.Items.length, 1);
    const cols = Math.min(6, Math.max(2, Math.ceil(Math.sqrt(n * 1.5))));
    const rows = Math.ceil(n / cols);
    const w = cols * NODE_W + (cols - 1) * GAP + 2 * PAD;
    const h = rows * NODE_H + (rows - 1) * GAP + HEAD + PAD;

    if (cx > 0 && cx + w > ROW_WIDTH) {
      cx = 0;
      cy += rowH + ZONE_GAP;
      rowH = 0;
    }
    zones.push({
      id: gi, name: g.Tag, cidr: "", x: cx, y: cy, w, h,
      online: g.Items.filter(it => it.Status == "up").length,
      total: g.Items.length,
    });
    g.Items.forEach((item, i) => {
      nodes.push({
        item,
        x: cx + PAD + (i % cols) * (NODE_W + GAP),
        y: cy + HEAD + Math.floor(i / cols) * (NODE_H + GAP),
      });
    });
    cx += w + ZONE_GAP;
    rowH = Math.max(rowH, h);
  });

  return { zones, nodes };
}

// Bounding box of everything on the map
export function bounds(zones: { x: number, y: number, w: number, h: number }[], nodes: { x: number, y: number }[]) {
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  for (const z of zones) {
    x0 = Math.min(x0, z.x); y0 = Math.min(y0, z.y);
    x1 = Math.max(x1, z.x + z.w); y1 = Math.max(y1, z.y + z.h);
  }
  for (const n of nodes) {
    x0 = Math.min(x0, n.x); y0 = Math.min(y0, n.y);
    x1 = Math.max(x1, n.x + NODE_W); y1 = Math.max(y1, n.y + NODE_H);
  }
  if (x0 == Infinity) return { x: 0, y: 0, w: 800, h: 400 };
  return { x: x0, y: y0, w: x1 - x0, h: y1 - y0 };
}

export const webLink = (ip: string, port: number, web: string) =>
  (web == "https" ? "https://" : "http://") + ip + (
    (web == "https" && port == 443) || (web == "http" && port == 80) ? "" : ":" + port
  );

// Topology: one zone per subnet with its gateway on top, hosts below in one
// column per suggested category, Docker hosts drawn taller with their
// containers inside, and each hypervisor as a block holding its VMs and LXCs.
// Laid out from scratch every time; positions are not saved.

export const CT_ROW = 20;       // height of one container line in a host box
export const CT_MAX = 8;        // container lines shown before "+N more"
export const GHOST_H = 40;      // a VM or LXC that was not seen on the network
const COL_HEAD = 30;
const GW_GAP = 46;
const BLOCK_PAD = 12;
const BLOCK_HEAD = 26;
const TOPO_ROW_WIDTH = 2600;

export interface TopoNode {
  host:  MapHost;
  x:     number;
  y:     number;
  h:     number;
  shown: Container[];   // containers drawn in the box
  more:  number;        // containers not drawn
  gateway: boolean;
};

export interface TopoGhost { guest: Guest; x: number; y: number; };
export interface TopoBlock { title: string; sub: string; x: number; y: number; w: number; h: number; };
export interface TopoColumn { name: string; x: number; y: number; w: number; };
export interface TopoEdge { d: string; };

export interface Topology {
  zones:   Zone[];
  blocks:  TopoBlock[];
  columns: TopoColumn[];
  nodes:   TopoNode[];
  ghosts:  TopoGhost[];
  edges:   TopoEdge[];
};

const isGateway = (h: MapHost, subnet: Subnet | null) => {
  if (!subnet) return false;
  const last = parseInt(h.IP.split(".")[3] ?? "");
  return last == 1 || last == 254;
};

const nodeFor = (host: MapHost, gateway: boolean): TopoNode => {
  const running = (host.Containers ?? []).filter(c => c.State == "running" || c.State == "");
  const shown = running.slice(0, CT_MAX);
  const more = running.length - shown.length;
  const extra = shown.length > 0 ? 8 + shown.length * CT_ROW + (more > 0 ? CT_ROW : 0) : 0;
  return { host, x: 0, y: 0, h: NODE_H + extra, shown, more, gateway };
};

// A column of the subnet zone: measured first, then placed
interface Col {
  name:  string;
  w:     number;
  h:     number;
  place: (x: number, y: number, out: Topology) => void;
};

const stackCol = (name: string, hosts: MapHost[]): Col => {
  const nodes = hosts.map(h => nodeFor(h, false));
  return {
    name, w: NODE_W,
    h: nodes.reduce((s, n) => s + n.h + GAP, 0) - GAP,
    place: (x, y, out) => {
      for (const n of nodes) {
        n.x = x; n.y = y;
        y += n.h + GAP;
        out.nodes.push(n);
      }
    },
  };
};

// A hypervisor: its own card, then its guests in VMID order, in a grid
const hyperCol = (node: MapHost, guests: MapHost[]): Col => {
  const items: ({ node: TopoNode } | { ghost: Guest })[] = [{ node: nodeFor(node, false) }];
  const seen = new Map(guests.map(g => [g.Mac, g]));
  for (const g of [...node.Guests].sort((a, b) => a.VMID - b.VMID)) {
    const h = g.HostMac ? seen.get(g.HostMac) : undefined;
    if (h) items.push({ node: nodeFor(h, false) });
    else if (!g.HostMac) items.push({ ghost: g });
    // a guest seen in another subnet stays in that subnet
  }
  const cols = Math.min(5, Math.max(2, Math.ceil(Math.sqrt(items.length * 1.3))));
  const height = (it: typeof items[number]) => "node" in it ? it.node.h : GHOST_H;
  const rows: number[] = [];
  items.forEach((it, i) => {
    const r = Math.floor(i / cols);
    rows[r] = Math.max(rows[r] ?? 0, height(it));
  });
  const w = cols * (NODE_W + GAP) - GAP + 2 * BLOCK_PAD;
  const h = BLOCK_HEAD + rows.reduce((s, r) => s + r + GAP, 0) - GAP + BLOCK_PAD;
  const running = node.Guests.filter(g => g.Status == "running").length;

  return {
    name: "", w, h,
    place: (x, y, out) => {
      out.blocks.push({
        title: node.Name || node.DNS.split(".")[0] || node.IP,
        sub: `${running}/${node.Guests.length} guests running`,
        x, y, w, h,
      });
      let ry = y + BLOCK_HEAD;
      rows.forEach((rh, r) => {
        items.slice(r * cols, (r + 1) * cols).forEach((it, c) => {
          const ix = x + BLOCK_PAD + c * (NODE_W + GAP);
          if ("node" in it) {
            it.node.x = ix; it.node.y = ry;
            out.nodes.push(it.node);
          } else {
            out.ghosts.push({ guest: it.ghost, x: ix, y: ry });
          }
        });
        ry += rh + GAP;
      });
    },
  };
};

export function topology(data: MapData, showOffline: boolean, categoryOrder: string[]): Topology {
  const groups: { subnet: Subnet | null, hosts: MapHost[] }[] =
    data.Subnets.map(s => ({ subnet: s, hosts: [] as MapHost[] }));
  const other: MapHost[] = [];
  for (const h of data.Hosts) {
    if (!showOffline && h.Now != 1) continue;
    const g = groups.find(g => g.subnet!.ID == h.SubnetID);
    g ? g.hosts.push(h) : other.push(h);
  }
  if (other.length > 0) groups.push({ subnet: null, hosts: other });

  const rank = (cat: string) => {
    const i = categoryOrder.indexOf(cat);
    return i < 0 ? categoryOrder.length : i;
  };

  const out: Topology = { zones: [], blocks: [], columns: [], nodes: [], ghosts: [], edges: [] };
  let cx = 0, cy = 0, rowH = 0;

  for (const g of groups) {
    if (g.hosts.length == 0) continue;
    g.hosts.sort((a, b) => ipNum(a.IP) - ipNum(b.IP));

    const gw = g.hosts.find(h => isGateway(h, g.subnet) && !(h.Guests?.length));
    const hypers = g.hosts.filter(h => h != gw && h.Guests?.length > 0);
    const inBlock = new Set<MapHost>();
    const blockCols = hypers.map(hv => {
      const guests = g.hosts.filter(h => h.GuestOf && h.GuestOf.NodeMac == hv.Mac && h != gw && !hypers.includes(h));
      guests.forEach(h => inBlock.add(h));
      return hyperCol(hv, guests);
    });
    const rest = g.hosts.filter(h => h != gw && !hypers.includes(h) && !inBlock.has(h));

    // Columns by category
    const byCat = new Map<string, MapHost[]>();
    for (const h of rest) {
      const cat = h.Category || "Other";
      if (!byCat.has(cat)) byCat.set(cat, []);
      byCat.get(cat)!.push(h);
    }
    const names = [...byCat.keys()].sort((a, b) => rank(a) - rank(b) || a.localeCompare(b));
    const cols: Col[] = [...blockCols, ...names.map(n => stackCol(n, byCat.get(n)!))];

    // Measure
    const width = Math.max(cols.reduce((s, c) => s + c.w + GAP, 0) - GAP, NODE_W) + 2 * PAD;
    const gwNode = gw ? nodeFor(gw, true) : null;
    const headH = HEAD + (gwNode ? gwNode.h + GW_GAP : 0);
    const height = headH + COL_HEAD + Math.max(0, ...cols.map(c => c.h)) + PAD;

    if (cx > 0 && cx + width > TOPO_ROW_WIDTH) {
      cx = 0;
      cy += rowH + ZONE_GAP;
      rowH = 0;
    }
    const zy = cy;
    const colTop = zy + headH;

    out.zones.push({
      id: g.subnet ? g.subnet.ID : 0,
      name: g.subnet ? g.subnet.Name : "Other hosts",
      cidr: g.subnet ? g.subnet.CIDR : "",
      x: cx, y: zy, w: width, h: height,
      online: g.hosts.filter(h => h.Now == 1).length,
      total: g.hosts.length,
    });

    let gwBottom = { x: 0, y: 0 };
    if (gwNode) {
      gwNode.x = cx + (width - NODE_W) / 2;
      gwNode.y = zy + HEAD;
      out.nodes.push(gwNode);
      gwBottom = { x: gwNode.x + NODE_W / 2, y: gwNode.y + gwNode.h };
    }

    let x = cx + PAD;
    for (const col of cols) {
      const top = col.name ? colTop : colTop + COL_HEAD - BLOCK_HEAD + 4; // blocks carry their own title
      if (col.name) out.columns.push({ name: col.name, x, y: colTop, w: col.w });
      if (gwNode) {
        const mid = gwBottom.y + GW_GAP / 2;
        out.edges.push({ d: `M${gwBottom.x},${gwBottom.y} V${mid} H${x + col.w / 2} V${col.name ? colTop + 2 : top}` });
      }
      col.place(x, col.name ? colTop + COL_HEAD : top, out);
      x += col.w + GAP;
    }

    cx += width + ZONE_GAP;
    rowH = Math.max(rowH, height);
  }
  return out;
}

export function topoBounds(t: Topology) {
  const boxes = [...t.zones, ...t.blocks];
  return bounds(boxes, t.nodes.map(n => ({ x: n.x, y: n.y + n.h - NODE_H })).concat(t.nodes));
}
