import { createSignal, For, Show } from "solid-js";
import { apiGetTopPorts, apiPath } from "../../functions/api"
import { appConfig } from "../../functions/exports"

function PortScan() {

  const [top, setTop] = createSignal<{ port: number; service: string }[]>([]);
  const loadTop = async (e: Event) => {
    if ((e.currentTarget as HTMLDetailsElement).open && top().length == 0) {
      setTop(await apiGetTopPorts());
    }
  };

  return (
    <div class="card border-primary">
      <div class="card-header">Port scanning</div>
      <div class="card-body table-responsive">
        <form action={apiPath + '/api/config_ports/'} method="post">
          <table class="table table-borderless"><tbody>
            <tr>
              <td>Scheduled scans</td>
              <td>
                <div class="form-check form-switch">
                  {appConfig().PortScan
                    ? <input class="form-check-input" type="checkbox" name="enable" checked></input>
                    : <input class="form-check-input" type="checkbox" name="enable"></input>
                  }
                </div>
              </td>
            </tr>
            <tr>
              <td>Every (minutes)</td>
              <td><input name="interval" type="number" min="1" class="form-control" value={appConfig().PortInterval}></input></td>
            </tr>
            <tr>
              <td>Ports</td>
              <td><input name="list" type="text" class="form-control" value={appConfig().PortList}></input></td>
            </tr>
            <tr>
              <td>Ports at once</td>
              <td><input name="workers" type="number" min="1" class="form-control" value={appConfig().PortWorkers}></input></td>
            </tr>
            <tr>
              <td>Timeout (ms)</td>
              <td><input name="timeout" type="number" min="50" class="form-control" value={appConfig().PortTimeout}></input></td>
            </tr>
            <tr>
              <td><button type="submit" class="btn btn-primary">Save</button></td>
              <td class="text-muted small">Scheduled scans cover online hosts, except in subnets set to skip port scans. Ports: <code>top</code>, <code>all</code>, or a list like <code>22,80,8000-8100</code>. Changes are sent as notifications.</td>
            </tr>
          </tbody></table>
        </form>
        <details onToggle={loadTop}>
          <summary class="small">What <code>top</code> scans</summary>
          <Show when={top().length > 0} fallback={<p class="small opacity-75 mt-2">Loading...</p>}>
            <p class="small opacity-75 mt-2 mb-1">{top().length} ports common on home networks and self-hosted apps. Open ports that may serve a web page are also checked for its title, which names the service.</p>
            <div class="top-ports small">
              <For each={top()}>{p =>
                <div><code>{p.port}</code> <span class="opacity-75">{p.service}</span></div>
              }</For>
            </div>
          </Show>
        </details>
      </div>
    </div>
  )
}

export default PortScan
