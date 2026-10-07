import { useNavigate } from "@solidjs/router";
import { createSignal, For, onMount } from "solid-js";
import { apiGetConfig, apiGetViews, apiSaveStartPage, apiSaveView } from "../../functions/api";
import { View } from "../../functions/exports";

export const [views, setViews] = createSignal<View[]>([]);
const [startPage, setStartPage] = createSignal<string>();

export const reloadViews = async () => {
  setViews(await apiGetViews());
};

// Tabs above the dashboard: the network map, suggested categories, then each custom view
function ViewTabs(props: { active: number }) {

  const navigate = useNavigate();

  onMount(async () => {
    reloadViews();
    if (startPage() === undefined) setStartPage((await apiGetConfig()).StartPage ?? "");
  });

  // The page this tab shows, as saved for the start page ("" is the network map)
  const here = () => props.active == 0 ? "" : props.active == -1 ? "/view/auto" : "/view/" + props.active;
  const isStart = () => startPage() == here();
  const makeStart = async () => {
    await apiSaveStartPage(here());
    setStartPage(here());
  };

  const addView = async () => {
    const v: View = await apiSaveView({ Name: "New view", Layout: "tiles", Tags: "" });
    await reloadViews();
    navigate("/view/" + v.ID + "?edit=1");
  };

  return (
    <ul class="nav nav-tabs view-tabs mb-0">
      <li class="nav-item">
        <a class={"nav-link" + (props.active == 0 ? " active" : "")} href="/map">
          <i class="bi bi-diagram-3 me-1"></i>Network
        </a>
      </li>
      <li class="nav-item">
        <a class={"nav-link" + (props.active == -1 ? " active" : "")} href="/view/auto" title="Hosts and services grouped by suggested category">
          <i class="bi bi-stars me-1"></i>Categories
        </a>
      </li>
      <For each={views()}>{v =>
        <li class="nav-item">
          <a class={"nav-link" + (props.active == v.ID ? " active" : "")} href={"/view/" + v.ID}>
            <i class={"bi me-1 " + (v.Layout == "map" ? "bi-bounding-box" : "bi-grid-3x3-gap")}></i>{v.Name}
          </a>
        </li>
      }</For>
      <li class="nav-item">
        <button class="nav-link" onClick={addView} title="Add a view"><i class="bi bi-plus-lg"></i><span class="ms-1">New view</span></button>
      </li>
      <li class="nav-item ms-auto">
        <button class={"nav-link start-toggle" + (isStart() ? " is-start" : "")} onClick={makeStart} disabled={isStart()}
          title={isStart() ? "This tab opens when you visit Gimlé" : "Open this tab when you visit Gimlé"}>
          <i class={"bi " + (isStart() ? "bi-house-fill" : "bi-house")}></i><span class="ms-1">{isStart() ? "Start page" : "Make start page"}</span>
        </button>
      </li>
    </ul>
  )
}

export default ViewTabs
