import { createSignal, For, onMount, Show } from "solid-js";
import { apiDelConnector, apiGetConnectorKinds, apiGetConnectors, apiSaveConnector, apiSyncConnector } from "../../functions/api";
import { Connector, ConnectorKind } from "../../functions/exports";

const help: Record<string, string> = {
  docker: "A Docker API, ideally read-only through tecnativa/docker-socket-proxy with CONTAINERS=1. Lists every container and its published ports. Host IP is only needed when the URL does not point at the Docker host itself.",
  dockhand: "Reads containers from every environment Dockhand manages. A Dockhand API token (dh_...) is only needed when Dockhand has authentication turned on. Environment limits it to one, by name or ID.",
  scanopy: "Imports host names, detected services and containers. Needs a Scanopy user API key (scp_u_...) from Platform > API Keys.",
  unifi: "Reads client names, DHCP host names, fixed IPs and addresses from a UniFi OS console such as a Dream Machine. Make an API key under UniFi Network > Settings > Control Plane > Integrations. Consoles use a self-signed certificate.",
  proxmox: "Reads every VM and LXC with its node, VMID, status and MAC address, so guests show inside their Proxmox host on the map. Make an API token under Datacenter > Permissions > API Tokens and give it the PVEAuditor role on /. Enter it as USER@REALM!TOKENID=SECRET, for example root@pam!gimle=1234-abcd. Node IP is only needed when the URL does not point at the node itself.",
  caddy: "Turns every site Caddy reverse proxies into a bookmark, linked to the host and service it forwards to, so tiles open the proxy name instead of the IP. Caddy's admin API listens on localhost:2019 and can change Caddy's config, so don't open it to the network: add a read-only proxy of GET /config/ to your Caddyfile (see the guide) and point the URL at that. Caddy host IP is only needed when the URL does not point at the machine running Caddy; it is where upstreams on localhost run. You can rename, tag or delete the bookmarks it adds; deleted ones stay away.",
  technitium: "Reads DHCP leases and the A records of local zones, so hosts get their DNS names. Make an API token under Administration > Sessions > Create Token.",
};

const blank = (): Connector => ({
  ID: 0, Kind: "docker", Name: "", URL: "", User: "", Site: "", HostIP: "",
  Insecure: false, Enabled: true, Interval: 15, LastSync: "", LastError: "", LastCount: 0, HasToken: false,
});

// Other systems Gimlé reads hosts, names, services and containers from
function Connectors() {

  const [list, setList] = createSignal<Connector[]>([]);
  const [kinds, setKinds] = createSignal<Record<string, ConnectorKind>>({});
  const [form, setForm] = createSignal<Connector | null>(null);
  const [token, setToken] = createSignal("");
  const [clearToken, setClearToken] = createSignal(false);
  const [error, setError] = createSignal("");
  const [busy, setBusy] = createSignal(0);

  const load = async () => setList(await apiGetConnectors());
  onMount(async () => {
    setKinds(await apiGetConnectorKinds());
    await load();
  });

  const set = (field: keyof Connector, value: any) => setForm({ ...form()!, [field]: value });
  const kind = () => kinds()[form()?.Kind ?? ""];

  const edit = (c: Connector | null) => {
    setForm(c ? { ...c } : blank());
    setToken("");
    setClearToken(false);
    setError("");
  };

  const save = async (e: Event) => {
    e.preventDefault();
    try {
      await apiSaveConnector({ ...form()!, Token: token(), ClearToken: clearToken() });
      setForm(null);
      await load();
      setTimeout(load, 4000); // first sync runs in the background
    } catch (err: any) {
      setError(err.message);
    }
  };

  const sync = async (c: Connector) => {
    setBusy(c.ID);
    await apiSyncConnector(c.ID);
    setBusy(0);
    load();
  };

  const remove = async (c: Connector) => {
    if (!confirm("Delete " + c.Name + "? The containers and ports it reported go with it.")) return;
    await apiDelConnector(c.ID);
    setForm(null);
    load();
  };

  return (
    <div class="card border-primary">
      <div class="card-header d-flex justify-content-between align-items-center">
        Connectors
        <Show when={!form()}>
          <button class="btn btn-sm btn-outline-primary" onClick={() => edit(null)}><i class="bi bi-plus-lg me-1"></i>Add</button>
        </Show>
      </div>
      <div class="card-body">
        <Show when={!form()}>
          <For each={list()} fallback={
            <p class="small opacity-75 mb-0">Read VMs and LXCs from Proxmox, containers from Docker or Dockhand, names and services from Scanopy, a UniFi console or Technitium DNS, and bookmarks from Caddy. Each connector only reads.</p>
          }>{c =>
            <div class="connector-row">
              <div class="me-auto">
                <b>{c.Name}</b> <span class="badge text-bg-light border">{kinds()[c.Kind]?.Label ?? c.Kind}</span>
                <Show when={!c.Enabled}><span class="badge text-bg-secondary ms-1">off</span></Show>
                <div class="small opacity-75 text-break">{c.URL}</div>
                <div class="small">
                  <Show when={c.LastSync} fallback={<span class="opacity-75">Not synced yet</span>}>
                    <Show when={c.LastError} fallback={
                      <span class="text-success"><i class="bi bi-check-circle me-1"></i>{c.LastCount} {c.Kind == "caddy" ? "sites linked to hosts" : "hosts matched"}, {c.LastSync}</span>
                    }>
                      <span class="text-danger"><i class="bi bi-exclamation-triangle me-1"></i>{c.LastError}</span>
                    </Show>
                  </Show>
                </div>
              </div>
              <div class="d-flex gap-1 align-items-start">
                <button class="btn btn-sm btn-outline-secondary" disabled={busy() == c.ID} onClick={() => sync(c)} title="Sync now">
                  <i class={"bi bi-arrow-repeat" + (busy() == c.ID ? " spin" : "")}></i>
                </button>
                <button class="btn btn-sm btn-outline-secondary" onClick={() => edit(c)} title="Edit"><i class="bi bi-pencil"></i></button>
              </div>
            </div>
          }</For>
        </Show>

        <Show when={form()}>
          <form onSubmit={save}>
            <div class="row g-2">
              <div class="col-sm-5">
                <label class="form-label small mb-1" for="connKind">Kind</label>
                <select class="form-select form-select-sm" id="connKind" value={form()!.Kind} disabled={form()!.ID != 0}
                  onChange={e => { set("Kind", e.target.value); if (e.target.value == "unifi" || e.target.value == "proxmox") set("Insecure", true); }}>
                  <For each={Object.entries(kinds())}>{([k, info]) => <option value={k}>{info.Label}</option>}</For>
                </select>
              </div>
              <div class="col-sm-7">
                <label class="form-label small mb-1" for="connName">Name</label>
                <input class="form-control form-control-sm" id="connName" placeholder={kind()?.Label} value={form()!.Name} onInput={e => set("Name", e.target.value)}></input>
              </div>
              <div class="col-12">
                <label class="form-label small mb-1" for="connURL">URL</label>
                <input class="form-control form-control-sm" id="connURL" placeholder={kind()?.Example} value={form()!.URL} onInput={e => set("URL", e.target.value)}></input>
              </div>
              <div class="col-12">
                <label class="form-label small mb-1" for="connToken">
                  {form()!.Kind == "unifi" ? "API key" : "API token"} <span class="opacity-75">({kind()?.Token})</span>
                </label>
                <input class="form-control form-control-sm" id="connToken" type="password" autocomplete="off"
                  placeholder={form()!.HasToken ? "saved; leave empty to keep it" : ""} value={token()} onInput={e => setToken(e.target.value)}></input>
                <Show when={form()!.HasToken}>
                  <div class="form-check small mt-1">
                    <input class="form-check-input" type="checkbox" id="connClear" checked={clearToken()} onChange={e => setClearToken(e.target.checked)}></input>
                    <label class="form-check-label" for="connClear">Remove the saved token</label>
                  </div>
                </Show>
              </div>
              <Show when={form()!.Kind == "docker" || form()!.Kind == "proxmox" || form()!.Kind == "caddy"}>
                <div class="col-sm-6">
                  <label class="form-label small mb-1" for="connHost">{{ docker: "Docker host IP", proxmox: "Node IP", caddy: "Caddy host IP" }[form()!.Kind]} <span class="opacity-75">(optional)</span></label>
                  <input class="form-control form-control-sm" id="connHost" value={form()!.HostIP} onInput={e => set("HostIP", e.target.value)}></input>
                </div>
              </Show>
              <Show when={form()!.Kind == "dockhand" || form()!.Kind == "unifi"}>
                <div class="col-sm-6">
                  <label class="form-label small mb-1" for="connSite">{form()!.Kind == "unifi" ? "Site" : "Environment"} <span class="opacity-75">(optional)</span></label>
                  <input class="form-control form-control-sm" id="connSite" placeholder={form()!.Kind == "unifi" ? "default" : "all"} value={form()!.Site} onInput={e => set("Site", e.target.value)}></input>
                </div>
              </Show>
              <div class="col-sm-6">
                <label class="form-label small mb-1" for="connInterval">Sync every (minutes)</label>
                <input class="form-control form-control-sm" id="connInterval" type="number" min="1" value={form()!.Interval} onInput={e => set("Interval", parseInt(e.target.value) || 15)}></input>
              </div>
              <div class="col-12 d-flex flex-wrap gap-3">
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" id="connEnabled" checked={form()!.Enabled} onChange={e => set("Enabled", e.target.checked)}></input>
                  <label class="form-check-label small" for="connEnabled">Enabled</label>
                </div>
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" id="connInsecure" checked={form()!.Insecure} onChange={e => set("Insecure", e.target.checked)}></input>
                  <label class="form-check-label small" for="connInsecure">Accept self-signed certificate</label>
                </div>
              </div>
              <div class="col-12 small opacity-75">{help[form()!.Kind]}</div>
              <Show when={error()}><div class="col-12 text-danger small">{error()}</div></Show>
              <div class="col-12 d-flex gap-2">
                <button type="submit" class="btn btn-sm btn-primary">Save</button>
                <button type="button" class="btn btn-sm btn-outline-secondary" onClick={() => setForm(null)}>Cancel</button>
                <Show when={form()!.ID != 0}>
                  <button type="button" class="btn btn-sm btn-outline-danger ms-auto" onClick={() => remove(form()!)}>Delete</button>
                </Show>
              </div>
            </div>
          </form>
        </Show>
      </div>
    </div>
  )
}

export default Connectors
