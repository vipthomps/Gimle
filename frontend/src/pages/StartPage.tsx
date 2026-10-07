import { useNavigate } from "@solidjs/router";
import { createSignal, lazy, onMount, Show } from "solid-js";
import { apiGetConfig } from "../functions/api";

const MapPage = lazy(() => import("./MapPage"));

// "/" shows the start page chosen with the house button on the tabs, the network map by default
function StartPage() {

  const navigate = useNavigate();
  const [showMap, setShowMap] = createSignal(false);

  onMount(async () => {
    try {
      const conf = await apiGetConfig();
      if (conf.StartPage) {
        navigate(conf.StartPage, { replace: true });
        return;
      }
    } catch (e) {
      // fall back to the map
    }
    setShowMap(true);
  });

  return <Show when={showMap()}><MapPage></MapPage></Show>
}

export default StartPage
