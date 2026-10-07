import { createSignal } from "solid-js";

// Icons by name come from dashboard-icons (https://dashboardicons.com),
// the set Dashy and Homarr use. A full URL works too, for self-hosted images.
export const ICON_CDN = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons";

// Image URL for an icon value, "" for none
export const iconURL = (icon: string): string => {
  icon = (icon || "").trim();
  if (icon == "" || icon == "none") return "";
  if (/^https?:\/\//.test(icon) || (icon.startsWith("/") && !icon.startsWith("//"))) return icon;
  const m = icon.match(/\.(svg|png|webp)$/);
  return m ? `${ICON_CDN}/${m[1]}/${icon}` : `${ICON_CDN}/svg/${icon}.svg`;
};

// Name dashboard-icons would likely use for a title: "Home Assistant" -> "home-assistant"
export const guessIcon = (title: string) =>
  (title || "").toLowerCase().trim().replace(/[\s_]+/g, "-").replace(/[^a-z0-9.-]/g, "");

// Images that failed to load, so they are not retried on every refresh
const [failed, setFailed] = createSignal<string[]>([]);
export const iconFailed = (url: string) => failed().includes(url);
export const markIconFailed = (url: string) => {
  if (!iconFailed(url)) setFailed([...failed(), url]);
};

// The icon to show: the one set, else a guess from the title (when guessing), else ""
export const pickIcon = (icon: string, title: string, guess: boolean) => {
  let url = iconURL(icon);
  if (!icon && guess && guessIcon(title)) url = iconURL(guessIcon(title));
  return url && !iconFailed(url) ? url : "";
};

// Icon name guess for a container from its image:
// "ghcr.io/immich-app/immich-server:release" -> "immich", "lscr.io/linuxserver/sonarr" -> "sonarr"
export const containerIcon = (image: string, name: string) => {
  let base = (image || "").split("@")[0];
  base = base.slice(base.lastIndexOf("/") + 1).split(":")[0];
  base = base.replace(/[-_](server|app|web|ce|ee|oss|community|docker)$/, "");
  return guessIcon(base || name);
};
