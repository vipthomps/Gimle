import { createSignal, For, Show } from "solid-js";
import { Container } from "../../functions/exports";
import { webLink } from "../../functions/maplayout";
import AppIcon from "../Views/AppIcon";
import { containerIcon } from "../../functions/icons";

const stateClass = (s: string) => s == "running" ? "up" : s == "" ? "unknown" : "down";

// Containers on one host, running first, grouped by compose project
function ContainerList(props: { containers: Container[], ip: string, compact?: boolean }) {

  const [showStopped, setShowStopped] = createSignal(false);

  const running = () => props.containers.filter(c => c.State == "running" || c.State == "");
  const stopped = () => props.containers.filter(c => c.State != "running" && c.State != "");
  const shown = () => showStopped() ? [...running(), ...stopped()] : running();

  const published = (c: Container) => c.Ports.filter(p => p.Public > 0 && p.Proto == "tcp");

  return (
    <div class={"container-list" + (props.compact ? " compact" : "")}>
      <For each={shown()} fallback={<p class="small opacity-75 mb-0">No running containers.</p>}>{c =>
        <div class="container-row" title={c.Image + (c.Status ? "\n" + c.Status : "")}>
          <span class={"status-dot " + stateClass(c.State)}></span>
          <AppIcon icon="" title={containerIcon(c.Image, c.Name)} guess={true} fallback="bi-box"></AppIcon>
          <span class="container-text">
            <span class="container-name">{c.Name}</span>
            <Show when={!props.compact || published(c).length == 0}>
              <span class="container-sub">{c.Project ? c.Project + " · " : ""}{c.Image}</span>
            </Show>
          </span>
          <span class="container-ports">
            <For each={published(c)}>{p =>
              <a class="badge text-bg-light border" href={webLink(props.ip, p.Public, p.Public == 443 || p.Public == 8443 ? "https" : "http")} target="_blank"
                title={p.Public == p.Private ? "port " + p.Public : p.Public + " on the host, " + p.Private + " in the container"}>:{p.Public}</a>
            }</For>
          </span>
        </div>
      }</For>
      <Show when={stopped().length > 0}>
        <button class="btn btn-sm btn-link p-0 mt-1" onClick={() => setShowStopped(!showStopped())}>
          {showStopped() ? "Hide stopped" : "Show " + stopped().length + " stopped"}
        </button>
      </Show>
    </div>
  )
}

export default ContainerList
