import { useParams } from "@solidjs/router";
import { createSignal, For, onMount, Show } from "solid-js";

import { apiGetContainers, apiGetHost, apiGetMap } from "../functions/api";
import ContainerList from "../components/Containers/ContainerList";
import GuestList, { guestKind } from "../components/Containers/GuestList";

import HostCard from "../components/HostPage/HostCard";
import Ports from "../components/HostPage/Ports";
import HistCard from "../components/HostPage/HistCard";
import TagEditor from "../components/Views/TagEditor";
import Uptime from "../components/HostPage/Uptime";
import { Container, emptyHost, Host, MapData, MapHost } from "../functions/exports";

function HostPage() {

  const [currentHost, setCurrentHost] = createSignal<Host>(emptyHost);
  const [containers, setContainers] = createSignal<Container[]>([]);
  const [mapData, setMapData] = createSignal<MapData>();
  const mapHost = (): MapHost | undefined => mapData()?.Hosts.find(h => h.Mac == currentHost().Mac);
  const hostLink = (mac: string) => {
    const h = mapData()?.Hosts.find(h => h.Mac == mac);
    if (h) window.location.href = "/host/" + h.ID;
  };

  onMount(async () => {
    const params = useParams();
    const host = await apiGetHost(params.id);

    setCurrentHost(host);
    setContainers(await apiGetContainers(host.Mac));
    setMapData(await apiGetMap());
  });

  return (
    <>
    <div class="row">
      <div class="col-md">
        <HostCard host={currentHost()}></HostCard>
        <div class="card border-primary mt-4">
          <div class="card-header">Tags</div>
          <div class="card-body">
            <TagEditor hostID={currentHost().ID} mac={currentHost().Mac}></TagEditor>
          </div>
        </div>
      </div>
      <div class="col-md">
        <Show when={mapHost()?.GuestOf}>{g =>
          <div class="alert alert-secondary py-2 small">
            <i class={"bi " + (g().Type == "lxc" ? "bi-box" : "bi-pc-display") + " me-1"}></i>
            {guestKind(g())} {g().VMID} on <a href="#" onClick={e => { e.preventDefault(); hostLink(g().NodeMac); }}>{g().Node}</a>
          </div>
        }</Show>
        <Ports host={currentHost()}></Ports>
        <div class="card border-primary mt-4">
          <div class="card-header d-flex align-items-center">
            <span class="me-auto">Bookmarks</span>
            <a class="small" href={"/bookmarks?mac=" + encodeURIComponent(currentHost().Mac)}><i class="bi bi-plus-lg me-1"></i>Add</a>
          </div>
          <div class="card-body">
            <For each={mapHost()?.Bookmarks ?? []} fallback={
              <small class="opacity-75">No bookmarks linked. Add the names you reach this host by, like the ones in your reverse proxy.</small>
            }>{b =>
              <div class="container-row">
                <i class="bi bi-bookmark me-2 opacity-75"></i>
                <a href={b.URL} target="_blank">{b.Name || b.URL}</a>
                <span class="small opacity-75 ms-2">{b.URL.replace(/^https?:\/\//, "")}{b.Port ? " → :" + b.Port : ""}</span>
              </div>
            }</For>
          </div>
        </div>
        <Show when={(mapHost()?.Guests.length ?? 0) > 0}>
          <div class="card border-primary mt-4">
            <div class="card-header">VMs and LXCs <span class="opacity-75 small ms-1">{mapHost()!.Guests.filter(g => g.Status == "running").length} running</span></div>
            <div class="card-body">
              <GuestList guests={mapHost()!.Guests} onOpen={hostLink}></GuestList>
            </div>
          </div>
        </Show>
        <Show when={containers().length > 0}>
          <div class="card border-primary mt-4">
            <div class="card-header">Containers <span class="opacity-75 small ms-1">{containers().filter(c => c.State == "running").length} running</span></div>
            <div class="card-body">
              <ContainerList containers={containers()} ip={currentHost().IP}></ContainerList>
            </div>
          </div>
        </Show>
        <div class="mt-4">
          <Uptime mac={currentHost().Mac}></Uptime>
        </div>
      </div>
    </div>
    <div class="row mt-4">
      <div class="col-md">
        <HistCard mac={currentHost().Mac}></HistCard>
      </div>
    </div>
    </>
  )
}

export default HostPage