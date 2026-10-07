import { useParams } from "@solidjs/router";
import { createSignal, For, onMount, Show } from "solid-js";
import { apiGetSubnetIPAM } from "../functions/api";
import { Address, IPAM } from "../functions/exports";
import UsageBar from "../components/Subnets/UsageBar";

const stateClass: Record<string, string> = {
  online:    "ip-cell ip-online",
  offline:   "ip-cell ip-offline",
  reserved:  "ip-cell ip-reserved",
  free:      "ip-cell ip-free",
  network:   "ip-cell ip-edge",
  broadcast: "ip-cell ip-edge",
};

const lastOctet = (ip: string) => ip.split(".")[3];

const cellTitle = (a: Address) => {
  let t = a.IP + " · " + a.State;
  if (a.Name) t += " · " + a.Name;
  if (a.Mac) t += " · " + a.Mac;
  if (a.Reserved && a.State != "reserved") t += " · reserved";
  return t;
};

function SubnetPage() {

  const [ipam, setIpam] = createSignal<IPAM>();
  const [error, setError] = createSignal("");

  onMount(async () => {
    const params = useParams();
    const res = await apiGetSubnetIPAM(params.id);
    res.error ? setError(res.error) : setIpam(res);
  });

  return (
    <>
    <Show when={error() != ""}>
      <div class="alert alert-danger">{error()}</div>
    </Show>
    <Show when={ipam()}>{data =>
      <div class="card border-primary">
        <div class="card-header">
          <div class="d-flex flex-wrap justify-content-between gap-2">
            <span><b>{data().Stat.Subnet.Name}</b> <span class="opacity-75">{data().Stat.Subnet.CIDR}</span></span>
            <span>Next free: <b>{data().Stat.NextFree ? data().Stat.NextFree : "none"}</b></span>
          </div>
        </div>
        <div class="card-body">
          <div class="mb-3" style="max-width: 30em;">
            <UsageBar stat={data().Stat}></UsageBar>
          </div>
          <div class="d-flex flex-wrap gap-3 mb-3 small">
            <span><span class="ip-key ip-online"></span> online ({data().Stat.Online})</span>
            <span><span class="ip-key ip-offline"></span> seen, offline ({data().Stat.Offline})</span>
            <span><span class="ip-key ip-reserved"></span> reserved ({data().Stat.Reserved})</span>
            <span><span class="ip-key ip-free"></span> free ({data().Stat.Free})</span>
          </div>
          <Show when={data().Addresses} fallback={
            <p class="opacity-75">This subnet is too large to show address by address.</p>
          }>
            <div class="ip-grid">
              <For each={data().Addresses}>{a =>
                <Show when={a.HostID} fallback={
                  <div class={stateClass[a.State]} title={cellTitle(a)}>{lastOctet(a.IP)}</div>
                }>
                  <a class={stateClass[a.State]} title={cellTitle(a)} href={"/host/"+a.HostID}>{lastOctet(a.IP)}</a>
                </Show>
              }</For>
            </div>
          </Show>
        </div>
      </div>
    }</Show>
    </>
  )
}

export default SubnetPage
