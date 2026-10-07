import { createEffect, createSignal, For, on, onCleanup, Show } from "solid-js";
import { apiGetPorts, apiScanPorts } from "../../functions/api";
import { Host, HostPorts } from "../../functions/exports";

const portLink = (ip: string, port: number, web: string) =>
  (web == "https" ? "https://" : "http://") + ip + ":" + port;

function Ports(props: { host: Host }) {

  const [data, setData] = createSignal<HostPorts>();
  const [custom, setCustom] = createSignal("");
  const [error, setError] = createSignal("");
  let timer: number | undefined;

  const load = async () => {
    if (!props.host.ID) return;
    const res: HostPorts = await apiGetPorts(props.host.ID);
    setData(res);
    clearTimeout(timer);
    if (res.Job.Running) {
      timer = setTimeout(load, 1500);
    }
  };
  createEffect(on(() => props.host.ID, load));
  onCleanup(() => clearTimeout(timer));

  const scan = async (list: string) => {
    const err = await apiScanPorts(props.host.ID, list);
    setError(err);
    load();
  };

  const running = () => data()?.Job.Running;
  const pct = () => data()?.Job.Total ? Math.floor(data()!.Job.Done * 100 / data()!.Job.Total) : 0;

  return (
    <div class="card border-primary">
      <div class="card-header d-flex justify-content-between">
        <span>Open ports</span>
        <small class="opacity-75">
          {data()?.LastScan.Date ? "Last scan " + data()!.LastScan.Date + " (" + data()!.LastScan.List + ")" : "Not scanned yet"}
        </small>
      </div>
      <div class="card-body">
        <Show when={running()} fallback={
          <div class="d-flex flex-wrap gap-2 mb-3">
            <button type="button" class="btn btn-primary btn-sm" onClick={() => scan("top")}>Scan common ports</button>
            <button type="button" class="btn btn-outline-primary btn-sm" onClick={() => scan("all")}>Scan all ports</button>
            <form class="input-group input-group-sm" style="max-width: 16em;" onSubmit={e => {e.preventDefault(); scan(custom());}}>
              <input class="form-control" placeholder="22,80,8000-8100" value={custom()} onInput={e => setCustom(e.target.value)}></input>
              <button type="submit" class="btn btn-outline-secondary">Scan</button>
            </form>
          </div>
        }>
          <div class="mb-3">
            <div class="progress" style="height: 0.9em;">
              <div class="progress-bar progress-bar-striped progress-bar-animated" style={{width: pct()+"%"}}></div>
            </div>
            <small class="opacity-75">Scanning {data()!.Job.Done} of {data()!.Job.Total} ports</small>
          </div>
        </Show>
        <Show when={error() != ""}>
          <div class="alert alert-danger">{error()}</div>
        </Show>
        <Show when={(data()?.Ports.length ?? 0) > 0} fallback={
          <p class="opacity-75 mb-0">{data()?.LastScan.Date ? "No open ports found." : "Run a scan to see which ports are open."}</p>
        }>
          <table class="table table-sm table-hover mb-0">
            <tbody>
            <For each={data()!.Ports}>{p =>
              <tr>
                <td style="width: 5em;">
                  <Show when={p.Web} fallback={p.Port}>
                    <a href={portLink(props.host.IP, p.Port, p.Web)} target="_blank">{p.Port}</a>
                  </Show>
                </td>
                <td>{p.Title || p.Container || p.Service}<Show when={(p.Title || p.Container) && p.Service}> <small class="opacity-75">({p.Service})</small></Show></td>
                <td class="text-end opacity-75"><small title={"First seen "+p.First}>seen {p.Last}</small></td>
              </tr>
            }</For>
            </tbody>
          </table>
        </Show>
      </div>
    </div>
  )
}

export default Ports
