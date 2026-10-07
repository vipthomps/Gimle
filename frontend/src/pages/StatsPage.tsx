import { hostName } from "../functions/names";
import { createMemo, createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { apiGetStats } from "../functions/api";
import { HostUptime, Stats, TrendPoint } from "../functions/exports";
import { dayKeys, fmtPct, hourKeys, levelText, timeLabel, uptimeLevel } from "../functions/stats";
import LineChart, { Point } from "../components/Stats/LineChart";
import BarChart from "../components/Stats/BarChart";
import UptimeStrip from "../components/Stats/UptimeStrip";

const RANGES = [
  { days: 1, label: "Today" },
  { days: 7, label: "7 days" },
  { days: 30, label: "30 days" },
  { days: 90, label: "90 days" },
];

const readDays = () => {
  try { return parseInt(localStorage.getItem("statsDays") || "") || 7; } catch { return 7; }
};

const round1 = (v: number) => Math.round(v * 10) / 10;

function StatsPage() {

  const [days, setDays] = createSignal(readDays());
  const [stats, setStats] = createSignal<Stats>();
  const [loading, setLoading] = createSignal(false);
  const [sort, setSort] = createSignal("uptime");
  const [search, setSearch] = createSignal("");

  const load = async () => {
    setLoading(true);
    setStats(await apiGetStats(days()));
    setLoading(false);
  };

  onMount(() => {
    load();
    const timer = setInterval(load, 60000);
    onCleanup(() => clearInterval(timer));
  });

  const pickDays = (d: number) => {
    setDays(d);
    try { localStorage.setItem("statsDays", String(d)); } catch {}
    load();
  };

  // Every bucket in the range, so gaps without scans show as gaps
  const keys = createMemo(() => {
    const s = stats();
    if (!s) return [];
    return s.Bucket == "hour" ? hourKeys(s.From) : dayKeys(s.From);
  });
  const days_ = createMemo(() => stats() ? dayKeys(stats()!.From) : []);

  const series = (points: TrendPoint[], pick: (p: TrendPoint) => number): Point[] => {
    const byTime = new Map(points.map(p => [p.Time, p]));
    return keys().map(k => {
      const p = byTime.get(k);
      return { x: k, y: p ? round1(pick(p)) : null };
    });
  };

  const hosts = () => stats()?.Hosts ?? [];
  const onlineNow = () => hosts().filter(h => h.Host.Now == 1).length;
  const scanned = () => hosts().filter(h => h.Uptime >= 0);
  const avgUptime = () => scanned().length ? scanned().reduce((a, h) => a + h.Uptime, 0) / scanned().length : -1;
  const newCount = () => (stats()?.NewHosts ?? []).reduce((a, d) => a + d.Count, 0);
  const busiest = () => [...(stats()?.Subnets ?? [])].sort((a, b) => b.Now.Utilization - a.Now.Utilization)[0];

  const newPoints = () => {
    const byDay = new Map((stats()?.NewHosts ?? []).map(d => [d.Date, d.Count]));
    return days_().map(d => ({ x: d, y: byDay.get(d) ?? 0 }));
  };

  const hostLabel = (h: HostUptime) => hostName(h.Host, h.Host.IP);

  const rows = createMemo(() => {
    const q = search().trim().toLowerCase();
    const list = hosts().filter(h => q == "" ||
      [h.Host.Name, h.Host.IP, h.Host.Mac, h.Host.DNS, h.Host.Hw].some(v => v && v.toLowerCase().includes(q)));
    const by: Record<string, (a: HostUptime, b: HostUptime) => number> = {
      uptime: (a, b) => (a.Uptime < 0 ? 101 : a.Uptime) - (b.Uptime < 0 ? 101 : b.Uptime),
      name: (a, b) => hostLabel(a).localeCompare(hostLabel(b)),
      first: (a, b) => (b.FirstSeen || "").localeCompare(a.FirstSeen || ""),
    };
    return [...list].sort(by[sort()]);
  });

  return (
    <div class={loading() && stats() ? "stats-page loading" : "stats-page"}>
      <div class="d-flex flex-wrap gap-2 align-items-center mb-3">
        <div class="btn-group" role="group" aria-label="Time range">
          <For each={RANGES}>{r =>
            <button class={"btn btn-sm " + (days() == r.days ? "btn-primary" : "btn-outline-primary")}
              aria-pressed={days() == r.days} onClick={() => pickDays(r.days)}>{r.label}</button>
          }</For>
        </div>
        <Show when={stats()}>
          <small class="opacity-75">Since {timeLabel(stats()!.From)}, {stats()!.Bucket == "hour" ? "hourly" : "daily"} averages</small>
        </Show>
      </div>

      <Show when={stats()} fallback={<p class="opacity-75">Loading stats…</p>}>
        <div class="stat-tiles mb-3">
          <div class="stat-tile">
            <div class="stat-label">Online now</div>
            <div class="stat-value">{onlineNow()}<span class="stat-of"> of {hosts().length}</span></div>
          </div>
          <div class="stat-tile">
            <div class="stat-label">Average uptime</div>
            <div class="stat-value">{fmtPct(avgUptime())}</div>
            <div class="stat-note">{scanned().length} hosts scanned in this range</div>
          </div>
          <div class="stat-tile">
            <div class="stat-label">New devices</div>
            <div class="stat-value">{newCount()}</div>
            <div class="stat-note">first seen in this range</div>
          </div>
          <Show when={busiest()}>{b =>
            <div class="stat-tile">
              <div class="stat-label">Fullest subnet</div>
              <div class="stat-value">{b().Now.Utilization}%</div>
              <div class="stat-note text-truncate">{b().Subnet.Name} · next free {b().Now.NextFree || "none"}</div>
            </div>
          }</Show>
        </div>

        <div class="card border-primary mb-3">
          <div class="card-header">Hosts online</div>
          <div class="card-body">
            <Show when={stats()!.All.length > 0} fallback={<p class="opacity-75 mb-0">No scans recorded in this range yet. Stats start with the next scan.</p>}>
              <LineChart label="Hosts online over time" points={series(stats()!.All, p => p.Online)}
                max={Math.max(...stats()!.All.map(p => p.Used))}></LineChart>
            </Show>
          </div>
        </div>

        <Show when={stats()!.Subnets.length > 0}>
          <h6 class="stats-heading">Addresses online per subnet</h6>
          <div class="subnet-charts mb-3">
            <For each={stats()!.Subnets}>{st =>
              <div class="card">
                <div class="card-body">
                  <div class="d-flex justify-content-between gap-2">
                    <div class="text-truncate">
                      <a class="fw-semibold" href={"/subnet/" + st.Subnet.ID}>{st.Subnet.Name}</a>
                      <small class="opacity-75 ms-2">{st.Subnet.CIDR}</small>
                    </div>
                    <small class="text-nowrap">{st.Now.Utilization}% used</small>
                  </div>
                  <div class="small opacity-75 mb-1">
                    {st.Now.Online} online, {st.Now.Total - st.Now.Free} used of {st.Now.Total} · next free {st.Now.NextFree || "none"}
                  </div>
                  <Show when={st.Points.length > 0} fallback={<p class="small opacity-75 mb-0">No scans recorded in this range yet.</p>}>
                    <LineChart label={"Addresses online in " + st.Subnet.Name} height={130}
                      points={series(st.Points, p => p.Online)} max={Math.max(...st.Points.map(p => p.Used))}></LineChart>
                  </Show>
                </div>
              </div>
            }</For>
          </div>
        </Show>

        <Show when={days() > 1}>
          <div class="card border-primary mb-3">
            <div class="card-header">New devices per day</div>
            <div class="card-body">
              <BarChart label="New devices per day" unit="new" points={newPoints()}></BarChart>
            </div>
          </div>
        </Show>

        <div class="card border-primary">
          <div class="card-header d-flex flex-wrap gap-2 align-items-center">
            <span class="me-auto">Uptime by host</span>
            <input class="form-control form-control-sm" style="max-width: 12em;" placeholder="Find a host" aria-label="Find a host"
              value={search()} onInput={e => setSearch(e.target.value)}></input>
            <select class="form-select form-select-sm w-auto" aria-label="Sort hosts" value={sort()} onChange={e => setSort(e.target.value)}>
              <option value="uptime">Lowest uptime first</option>
              <option value="name">Name</option>
              <option value="first">Newest first</option>
            </select>
          </div>
          <div class="card-body pb-1">
            <div class="uptime-legend small mb-2">
              <For each={["good", "warn", "bad", "none"] as const}>{l =>
                <span><span class={"uptime-cell " + l}></span>{levelText[l]}</span>
              }</For>
            </div>
            <div class="table-responsive">
              <table class="table table-sm align-middle uptime-table">
                <thead><tr>
                  <th>Host</th><th>IP</th><th class="text-end">Uptime</th><th class="w-50">Each day</th><th>First seen</th>
                </tr></thead>
                <tbody>
                  <For each={rows()}>{h =>
                    <tr>
                      <td class="text-truncate" style="max-width: 12em;">
                        <span class={"status-dot me-2 " + (h.Host.Now == 1 ? "up" : "down")} title={h.Host.Now == 1 ? "Online" : "Offline"}></span>
                        <a href={"/host/" + h.Host.ID}>{hostLabel(h)}</a>
                      </td>
                      <td class="font-monospace small">{h.Host.IP}</td>
                      <td class={"text-end fw-semibold uptime-pct " + uptimeLevel(h.Uptime)}>{fmtPct(h.Uptime)}</td>
                      <td><UptimeStrip days={h.Days} keys={days_()}></UptimeStrip></td>
                      <td class="small text-nowrap">{h.FirstSeen ? (h.Before ? "by " : "") + timeLabel(h.FirstSeen.slice(0, 10)) : "–"}</td>
                    </tr>
                  }</For>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </Show>
    </div>
  )
}

export default StatsPage
