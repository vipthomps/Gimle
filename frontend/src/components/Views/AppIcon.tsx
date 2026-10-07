import { Show } from "solid-js";
import { markIconFailed, pickIcon } from "../../functions/icons";

// An item's icon as HTML: the image set (or guessed), else a Bootstrap icon
function AppIcon(props: { icon: string, title: string, guess: boolean, fallback: string }) {
  const url = () => pickIcon(props.icon, props.title, props.guess);
  return (
    <Show when={url()} fallback={<i class={"bi " + props.fallback + " view-tile-icon"}></i>}>
      <img class="view-tile-img" src={url()} alt="" onError={() => markIconFailed(url())}></img>
    </Show>
  )
}

export default AppIcon
