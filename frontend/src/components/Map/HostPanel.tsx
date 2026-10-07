import { hostName, serviceName } from "../../functions/names";
import { For, Show, createResource } from "solid-js";
import { MapHost } from "../../functions/exports";
import { webLink } from "../../functions/maplayout";
import TagEditor from "../Views/TagEditor";
import ContainerList from "../Containers/ContainerList";
import GuestList, { guestKind } from "../Containers/GuestList";
import IconInput from "../Views/IconInput";
import { apiGetCategories, apiSaveHostCategory, apiSaveIcon } from "../../functions/api";

function HostPanel(props: { host: MapHost, onClose: () => void, onChange?: () => void, onOpen?: (mac: string) => void }) {

  const saveIcon = async (icon: string) => {
    try {
      await apiSaveIcon(props.host.Mac, 0, icon);
    } catch (err: any) {
      return err.message as string;
    }
    props.onChange?.();
    return "";
  };

  const [categories] = createResource(apiGetCategories);

  // Bookmarks linked to one of the web services replace its button; the rest get their own
  const markFor = (port: number) => (props.host.Bookmarks ?? []).find(b => b.Port == port && port != 0);
  const hostMarks = () => (props.host.Bookmarks ?? []).filter(b => !props.host.Web.some(p => p.Port == b.Port));
  const bareURL = (u: string) => u.replace(/^https?:\/\//, "").replace(/\/$/, "");
  const saveCategory = async (cat: string) => {
    try {
      await apiSaveHostCategory(props.host.Mac, cat);
    } catch (err: any) {
      alert(err.message);
    }
    props.onChange?.();
  };

  return (
    <div class="card map-panel shadow">
      <div class="card-header d-flex justify-content-between align-items-center">
        <b class="text-truncate">{hostName(props.host, props.host.IP)}</b>
        <button type="button" class="btn-close" aria-label="Close" onClick={props.onClose}></button>
      </div>
      <div class="card-body">
        <p class="mb-2">
          <span class={"badge " + (props.host.Now == 1 ? "bg-success" : "bg-secondary")}>
            {props.host.Now == 1 ? "online" : "offline"}
          </span>
          <Show when={props.host.Known == 0}>
            <span class="badge bg-warning text-dark ms-1">unknown</span>
          </Show>
        </p>
        <table class="table table-sm mb-3"><tbody>
          <tr><td class="opacity-75">IP</td><td>{props.host.IP}</td></tr>
          <tr><td class="opacity-75">MAC</td><td><small>{props.host.Mac}</small></td></tr>
          <Show when={props.host.Hw}><tr><td class="opacity-75">Vendor</td><td><small>{props.host.Hw}</small></td></tr></Show>
          <Show when={props.host.DNS}><tr><td class="opacity-75">DNS</td><td><small>{props.host.DNS}</small></td></tr></Show>
          <Show when={props.host.GuestOf}>{g =>
            <tr><td class="opacity-75">Runs on</td><td><small>
              <a href="#" onClick={e => { e.preventDefault(); if (g().NodeMac) props.onOpen?.(g().NodeMac); }}>{g().Node}</a> as {guestKind(g())} {g().VMID}
            </small></td></tr>
          }</Show>
          <tr><td class="opacity-75 align-middle">Category</td><td>
            <select class="form-select form-select-sm" aria-label="Category" title="Where the host goes on the map and in Categories"
              onChange={e => saveCategory(e.currentTarget.value)}>
              <option value="" selected={props.host.Suggested}>
                {props.host.Suggested ? "Suggested: " + (props.host.Category || "Other") : "Suggested"}
              </option>
              <For each={categories() ?? []}>{c =>
                <option value={c} selected={!props.host.Suggested && props.host.Category == c}>{c}</option>
              }</For>
            </select>
          </td></tr>
          <tr><td class="opacity-75">Seen</td><td><small>{props.host.Date}</small></td></tr>
        </tbody></table>
        <Show when={props.host.Guests?.length > 0}>
          <h6 class="small text-uppercase opacity-75 mb-1">VMs and LXCs <span class="text-lowercase">({props.host.Guests.filter(g => g.Status == "running").length} running)</span></h6>
          <div class="mb-3">
            <GuestList guests={props.host.Guests} onOpen={props.onOpen}></GuestList>
          </div>
        </Show>
        <Show when={props.host.Containers?.length > 0}>
          <h6 class="small text-uppercase opacity-75 mb-1">Containers</h6>
          <div class="mb-3">
            <ContainerList containers={props.host.Containers} ip={props.host.IP} compact={true}></ContainerList>
          </div>
        </Show>
        <Show when={props.host.Web.length > 0 || hostMarks().length > 0} fallback={
          <p class="small opacity-75">No web services found yet. Scan its ports on the host page.</p>
        }>
          <div class="d-grid gap-2 mb-1">
            <For each={hostMarks()}>{b =>
              <a class="btn btn-sm btn-outline-primary text-start text-truncate" href={b.URL} target="_blank" title={b.URL}>
                <i class="bi bi-bookmark me-2"></i>{b.Name || bareURL(b.URL)} <span class="opacity-75">{bareURL(b.URL)}</span>
              </a>
            }</For>
            <For each={props.host.Web}>{p =>
              <a class="btn btn-sm btn-outline-primary text-start text-truncate" href={markFor(p.Port)?.URL ?? webLink(props.host.IP, p.Port, p.Web)} target="_blank">
                <i class={"bi me-2 " + (markFor(p.Port) ? "bi-bookmark" : "bi-box-arrow-up-right")}></i>{markFor(p.Port)?.Name || serviceName(p)} <span class="opacity-75">{markFor(p.Port) ? bareURL(markFor(p.Port)!.URL) : ":" + p.Port}</span>
              </a>
            }</For>
          </div>
        </Show>
        <div class="mb-3 small">
          <a href={"/bookmarks?mac=" + encodeURIComponent(props.host.Mac)}><i class="bi bi-bookmark-plus me-1"></i>Add a bookmark for this host</a>
        </div>
        <div class="mb-3">
          <div class="small fw-semibold mb-1">Icon</div>
          <IconInput value={props.host.Icon} onSave={saveIcon}></IconInput>
        </div>
        <div class="mb-3">
          <div class="small fw-semibold mb-1">Tags</div>
          <TagEditor hostID={props.host.ID} mac={props.host.Mac}></TagEditor>
        </div>
        <a class="btn btn-sm btn-primary w-100" href={"/host/" + props.host.ID}>Host details and port scan</a>
      </div>
    </div>
  )
}

export default HostPanel
