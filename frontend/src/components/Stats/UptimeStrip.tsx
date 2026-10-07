import { For } from "solid-js";
import { DayUptime } from "../../functions/exports";
import { fmtPct, levelText, timeLabel, uptimeLevel } from "../../functions/stats";

// One cell per day, colored by that day's uptime
function UptimeStrip(props: { days: DayUptime[], keys: string[] }) {
  const byDate = () => new Map(props.days.map(d => [d.Date, d]));
  return (
    <div class="uptime-strip" role="list">
      <For each={props.keys}>{k => {
        const d = byDate().get(k);
        const pct = d && d.Scans > 0 ? d.Online * 100 / d.Scans : -1;
        const level = uptimeLevel(pct);
        return <span role="listitem" class={"uptime-cell " + level}
          title={timeLabel(k) + ": " + (pct < 0 ? levelText.none : fmtPct(pct) + " online")}></span>;
      }}</For>
    </div>
  )
}

export default UptimeStrip
