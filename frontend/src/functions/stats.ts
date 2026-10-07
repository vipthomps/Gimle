const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

const pad = (n: number) => String(n).padStart(2, "0");

const dayKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

const parseDay = (key: string) => {
  const [y, m, d] = key.slice(0, 10).split("-").map(Number);
  return new Date(y, m - 1, d);
};

// Every day from a day to today, as "2006-01-02"
export function dayKeys(from: string): string[] {
  const keys: string[] = [];
  const today = dayKey(new Date());
  for (let d = parseDay(from); dayKey(d) <= today; d.setDate(d.getDate() + 1)) {
    keys.push(dayKey(d));
  }
  return keys;
}

// Every hour from midnight of a day to the current hour, as "2006-01-02 15"
export function hourKeys(from: string): string[] {
  const keys: string[] = [];
  const now = new Date();
  const last = `${dayKey(now)} ${pad(now.getHours())}`;
  for (let d = parseDay(from); ; d.setHours(d.getHours() + 1)) {
    const k = `${dayKey(d)} ${pad(d.getHours())}`;
    if (k > last) break;
    keys.push(k);
  }
  return keys;
}

// "2026-10-06" -> "Oct 6", "2026-10-06 15" -> "Oct 6, 15:00"
export function timeLabel(key: string): string {
  const d = parseDay(key);
  const label = MONTHS[d.getMonth()] + " " + d.getDate();
  return key.length > 10 ? label + ", " + key.slice(11, 13) + ":00" : label;
}

// Short axis label: the day, or the hour when it isn't midnight
export function tickLabel(key: string): string {
  if (key.length > 10 && key.slice(11, 13) != "00") return key.slice(11, 13) + ":00";
  return timeLabel(key.slice(0, 10));
}

// A round number at or above v, for axis tops
export function niceMax(v: number): number {
  if (v <= 0) return 1;
  const p = Math.pow(10, Math.floor(Math.log10(v)));
  for (const m of [1, 2, 2.5, 5, 10]) {
    if (m * p >= v) return m * p;
  }
  return 10 * p;
}

export type UptimeLevel = "good" | "warn" | "bad" | "none";

export function uptimeLevel(pct: number): UptimeLevel {
  if (pct < 0) return "none";
  if (pct >= 99) return "good";
  if (pct >= 90) return "warn";
  return "bad";
}

export const levelText: Record<UptimeLevel, string> = {
  good: "99% or more",
  warn: "90 to 99%",
  bad: "under 90%",
  none: "not scanned",
};

export const fmtPct = (pct: number) => pct < 0 ? "–" : (pct >= 99.95 || pct == 0 ? pct.toFixed(0) : pct.toFixed(1)) + "%";
