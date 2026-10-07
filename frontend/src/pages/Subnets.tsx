import { createSignal, For, onMount, Show } from "solid-js";
import { apiDelSubnet, apiDetectSubnets, apiGetSubnets, apiSaveSubnet } from "../functions/api";
import { emptySubnet, Subnet, SubnetStat } from "../functions/exports";
import UsageBar from "../components/Subnets/UsageBar";

function Subnets() {

  const [stats, setStats] = createSignal<SubnetStat[]>([]);
  const [form, setForm] = createSignal<Subnet>({...emptySubnet});
  const [error, setError] = createSignal("");
  const [detected, setDetected] = createSignal<Subnet[]>([]);

  const load = async () => {
    setStats(await apiGetSubnets());
  };
  onMount(load);

  const setField = (field: keyof Subnet, value: string | boolean) => {
    setForm({...form(), [field]: value});
  };

  const handleSave = async (e: Event) => {
    e.preventDefault();
    const err = await apiSaveSubnet(form());
    setError(err);
    if (err == "") {
      setForm({...emptySubnet});
      load();
    }
  };

  const handleDelete = async (s: Subnet) => {
    if (confirm("Delete subnet " + s.Name + "? Hosts stay in the database.")) {
      await apiDelSubnet(s.ID);
      load();
    }
  };

  const handleDetect = async () => {
    const known = stats().map(st => st.Subnet.CIDR);
    const found: Subnet[] = await apiDetectSubnets();
    setDetected(found.filter(s => !known.includes(s.CIDR)));
  };

  return (
    <div class="row">
      <div class="col-lg-7">
        <div class="card border-primary">
          <div class="card-header">Subnets</div>
          <div class="card-body table-responsive">
            <Show when={stats().length > 0} fallback={<p class="opacity-75">No subnets yet. Add one, or use Detect.</p>}>
            <table class="table table-hover align-middle">
              <thead><tr>
                <th>Name</th><th>Usage</th><th>Next free</th><th></th>
              </tr></thead>
              <tbody>
              <For each={stats()}>{st =>
                <tr>
                  <td>
                    <a href={"/subnet/"+st.Subnet.ID}>{st.Subnet.Name}</a><br/>
                    <small class="opacity-75">{st.Subnet.CIDR}{st.Subnet.Iface ? " on "+st.Subnet.Iface : ""} · {st.Subnet.Method == "arp" ? "ARP scan" : "not scanned"}{st.Subnet.SkipPorts ? " · no port scans" : ""}</small>
                  </td>
                  <td style="min-width: 10em;"><UsageBar stat={st}></UsageBar></td>
                  <td>{st.NextFree ? st.NextFree : "none"}</td>
                  <td class="text-end text-nowrap">
                    <i class="bi bi-pencil-square my-btn me-2" title="Edit" onClick={() => {setForm({...st.Subnet}); setError("");}}></i>
                    <i class="bi bi-trash my-btn" title="Delete" onClick={() => handleDelete(st.Subnet)}></i>
                  </td>
                </tr>
              }</For>
              </tbody>
            </table>
            </Show>
          </div>
        </div>
      </div>
      <div class="col-lg-5 mt-4 mt-lg-0">
        <div class="card border-primary">
          <div class="card-header">{form().ID ? "Edit subnet" : "Add subnet"}</div>
          <div class="card-body">
            <form onSubmit={handleSave}>
              <label class="form-label">CIDR</label>
              <input class="form-control mb-2" placeholder="192.168.1.0/24" value={form().CIDR}
                onInput={e => setField("CIDR", e.target.value)}></input>
              <label class="form-label">Name</label>
              <input class="form-control mb-2" placeholder="defaults to the CIDR" value={form().Name}
                onInput={e => setField("Name", e.target.value)}></input>
              <label class="form-label">Interface</label>
              <input class="form-control mb-2" placeholder="auto" value={form().Iface}
                onInput={e => setField("Iface", e.target.value)}></input>
              <label class="form-label">Scan method</label>
              <select class="form-select mb-2" value={form().Method} onChange={e => setField("Method", e.target.value)}>
                <option value="arp">ARP scan (subnet is on one of this machine's interfaces)</option>
                <option value="none">Don't scan (track addresses only)</option>
              </select>
              <label class="form-label">Reserved addresses</label>
              <input class="form-control mb-1" placeholder="192.168.1.1, 192.168.1.100-192.168.1.200" value={form().Reserved}
                onInput={e => setField("Reserved", e.target.value)}></input>
              <small class="opacity-75">Gateway, DHCP pool, statics you've set aside. Comma separated; ranges with a dash.</small>
              <div class="form-check form-switch mt-2">
                <input class="form-check-input" type="checkbox" id="skipPorts" checked={form().SkipPorts}
                  onChange={e => setField("SkipPorts", e.target.checked)}></input>
                <label class="form-check-label" for="skipPorts">Leave out of scheduled port scans</label>
              </div>
              <Show when={error() != ""}>
                <div class="alert alert-danger mt-2 mb-0">{error()}</div>
              </Show>
              <div class="mt-3">
                <button type="submit" class="btn btn-primary me-2">Save</button>
                <Show when={form().ID != 0}>
                  <button type="button" class="btn btn-outline-secondary" onClick={() => {setForm({...emptySubnet}); setError("");}}>Cancel</button>
                </Show>
              </div>
            </form>
          </div>
        </div>
        <div class="card border-primary mt-4">
          <div class="card-header d-flex justify-content-between align-items-center">
            Detect from this machine
            <button class="btn btn-sm btn-outline-primary" onClick={handleDetect}>Detect</button>
          </div>
          <Show when={detected().length > 0}>
          <div class="card-body">
            <For each={detected()}>{s =>
              <div class="d-flex justify-content-between align-items-center mb-2">
                <span>{s.CIDR} <small class="opacity-75">on {s.Iface}</small></span>
                <button class="btn btn-sm btn-outline-secondary" onClick={() => {setForm({...s}); setError("");}}>Use</button>
              </div>
            }</For>
          </div>
          </Show>
        </div>
      </div>
    </div>
  )
}

export default Subnets
