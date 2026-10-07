import { For } from "solid-js";
import { Guest } from "../../functions/exports";

const stateClass = (s: string) => s == "running" ? "up" : s == "" ? "unknown" : "down";

export const guestKind = (g: Guest) => g.Type == "lxc" ? "LXC" : "VM";

const memText = (b: number) => b >= 1 << 30 ? Math.round(b / (1 << 30)) + " GB" : Math.round(b / (1 << 20)) + " MB";

// VMs and LXCs on a hypervisor node, in VMID order. A guest Gimlé has
// discovered on the network opens its own details.
function GuestList(props: { guests: Guest[], onOpen?: (mac: string) => void }) {
  return (
    <div class="container-list compact">
      <For each={props.guests}>{g =>
        <div class={"container-row" + (g.HostMac && props.onOpen ? " clickable" : "")}
          title={guestKind(g) + " " + g.VMID + " on " + g.Node + (g.CPUs ? "\n" + g.CPUs + " CPU, " + memText(g.MaxMem) : "") + (g.Tags ? "\nTags: " + g.Tags.split(";").join(", ") : "")}
          onClick={() => { if (g.HostMac) props.onOpen?.(g.HostMac); }}>
          <span class={"status-dot " + stateClass(g.Status)}></span>
          <i class={"bi " + (g.Type == "lxc" ? "bi-box" : "bi-pc-display") + " view-tile-icon"}></i>
          <span class="container-text">
            <span class="container-name">{g.VMID} {g.Name}</span>
            <span class="container-sub">{guestKind(g)} · {g.Status}{g.IP ? " · " + g.IP : ""}{g.HostMac ? "" : " · not seen on the network"}</span>
          </span>
        </div>
      }</For>
    </div>
  )
}

export default GuestList
