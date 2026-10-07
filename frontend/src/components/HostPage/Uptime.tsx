import { createEffect, createSignal, on, Show } from "solid-js";
import { apiGetHostStats } from "../../functions/api";
import { HostUptime } from "../../functions/exports";
import { dayKeys, fmtPct, timeLabel, uptimeLevel } from "../../functions/stats";
import UptimeStrip from "../Stats/UptimeStrip";

const DAYS = 30;

function Uptime(props: { mac: string }) {

  const [data, setData] = createSignal<HostUptime>();
  const [from, setFrom] = createSignal("");

  createEffect(on(() => props.mac, async mac => {
    if (!mac) return;
    setData(await apiGetHostStats(mac, DAYS) ?? undefined);
    const d = new Date();
    d.setDate(d.getDate() - DAYS + 1);
    setFrom(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`);
  }));

  return (
    <div class="card border-primary">
      <div class="card-header d-flex justify-content-between">
        <span>Uptime, last {DAYS} days</span>
        <a class="small" href="/stats">All stats</a>
      </div>
      <div class="card-body">
        <Show when={data()} fallback={<small class="opacity-75">Loading…</small>}>{u =>
          <>
            <div class="d-flex align-items-baseline gap-2 mb-2">
              <span class={"stat-value uptime-pct " + uptimeLevel(u().Uptime)}>{fmtPct(u().Uptime)}</span>
              <small class="opacity-75">
                {u().Scans > 0 ? `online in ${u().Online} of ${u().Scans} scans` : "no scans in this range yet"}
              </small>
            </div>
            <UptimeStrip days={u().Days} keys={dayKeys(from())}></UptimeStrip>
            <Show when={u().FirstSeen}>
              <div class="small opacity-75 mt-2">First seen {u().Before ? "by " : ""}{timeLabel(u().FirstSeen.slice(0, 10))}</div>
            </Show>
          </>
        }</Show>
      </div>
    </div>
  )
}

export default Uptime
