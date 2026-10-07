import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { appConfig, setAppConfig } from "../functions/exports";
import { apiGetConfig } from "../functions/api";
import { views, reloadViews } from "./Views/ViewTabs";

// The hall of Gimlé: a gold roof with the gem at its peak
export function Mark(props: { size?: number }) {
  const s = () => props.size ?? 30;
  return (
    <svg width={s()} height={s()} viewBox="0 0 32 32" fill="none" stroke="currentColor"
      stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M3 15L16 4l13 11h-4.5L16 8.6 7.5 15z" fill="currentColor" stroke="none"></path>
      <path d="M7 15v12h18V15"></path>
      <path d="M13.5 27v-6.5h5V27"></path>
      <path d="M16 0.8l1.3 1.6L16 4l-1.3-1.6z" fill="currentColor" stroke="none"></path>
    </svg>
  )
}

const main = [
  { href: "/", icon: "bi-house", label: "Home" },
  { href: "/map", icon: "bi-diagram-3", label: "Network map" },
  { href: "/bookmarks", icon: "bi-bookmark", label: "Bookmarks" },
  { href: "/docs", icon: "bi-journal-text", label: "Docs" },
  { href: "/hosts", icon: "bi-hdd-stack", label: "Hosts" },
  { href: "/subnets", icon: "bi-grid", label: "Subnets" },
  { href: "/stats", icon: "bi-bar-chart", label: "Stats" },
];

const below = [
  { href: "/history", icon: "bi-clock-history", label: "History" },
  { href: "/config", icon: "bi-gear", label: "Settings" },
  { href: "/about", icon: "bi-info-circle", label: "About" },
];

function Header() {

  const [here, setHere] = createSignal(location.pathname);

  // Bootstrap goes at the top of <head>, ahead of the app's own stylesheet, so
  // theme.css can restyle its components instead of being overruled by them.
  const stylesheet = (id: string, href: string) => {
    let link = document.getElementById(id) as HTMLLinkElement | null;
    if (!link) {
      link = document.createElement("link");
      link.id = id;
      link.rel = "stylesheet";
      document.head.prepend(link);
    }
    link.href = href;
  }

  const setCurrentTheme = async () => {
    setAppConfig(await apiGetConfig());

    const color = appConfig().Color ? appConfig().Color : "dark";

    if (appConfig().NodePath == '') {
      stylesheet("gm-font", "https://fonts.googleapis.com/css2?family=Instrument+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&family=Cormorant+Garamond:wght@700&display=swap");
      stylesheet("gm-icons", "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.css");
      stylesheet("gm-bootstrap", "https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css");
    } else {
      stylesheet("gm-icons", appConfig().NodePath + "/node_modules/bootstrap-icons/font/bootstrap-icons.css");
      stylesheet("gm-bootstrap", appConfig().NodePath + "/node_modules/bootstrap/dist/css/bootstrap.min.css");
    }

    document.documentElement.setAttribute("data-bs-theme", color);
    document.documentElement.setAttribute("data-gm-accent", appConfig().Theme ? appConfig().Theme : "gold");
  }
  setCurrentTheme();

  // Highlight the page you are on, including after an in-app navigation
  onMount(() => {
    reloadViews();
    const tick = setInterval(() => setHere(location.pathname), 400);
    onCleanup(() => clearInterval(tick));
  });

  const on = (href: string) => href == "/" ? here() == "/" : here().startsWith(href);

  return (
    <nav class="gm-side" aria-label="Main">
      <a class="gm-brand" href="/">
        <span style="color: var(--gm-accent)"><Mark size={32}></Mark></span>
        <span>
          <span class="gm-wordmark d-block">Gimlé</span>
          <span class="gm-tagline">the hall under the gold roof</span>
        </span>
      </a>
      <div class="gm-navgroup">
        <For each={main}>{m =>
          <a class={"gm-navlink" + (on(m.href) ? " active" : "")} href={m.href}><i class={"bi " + m.icon}></i>{m.label}</a>
        }</For>
      </div>
      <Show when={views().length > 0}>
        <div class="gm-navgroup">
          <div class="gm-navtitle">VIEWS</div>
          <For each={views()}>{v =>
            <a class={"gm-navlink" + (here() == "/view/" + v.ID ? " active" : "")} href={"/view/" + v.ID}>
              <i class={"bi " + (v.Layout == "map" ? "bi-bounding-box" : "bi-grid-3x3-gap")}></i>{v.Name}
            </a>
          }</For>
        </div>
      </Show>
      <div class="gm-sidefoot">
        <div class="gm-navgroup">
          <For each={below}>{m =>
            <a class={"gm-navlink" + (on(m.href) ? " active" : "")} href={m.href}><i class={"bi " + m.icon}></i>{m.label}</a>
          }</For>
        </div>
        <p class="gm-verse">
          “A hall I see, fairer than the sun, roofed with gold, on Gimlé.”
          <small>After Völuspá 64: the hall that still stands after Ragnarök.</small>
        </p>
      </div>
    </nav>
  )
};

export default Header
