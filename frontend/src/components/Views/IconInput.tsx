import { createEffect, createSignal, Show } from "solid-js";
import { iconFailed, iconURL, markIconFailed } from "../../functions/icons";

// Set an icon: a dashboard-icons name like "immich", an image URL, or "none".
// onSave returns an error message, or "" when saved.
function IconInput(props: { value: string, onSave: (icon: string) => Promise<string>, onCancel?: () => void }) {

  const [icon, setIcon] = createSignal(props.value);
  const [error, setError] = createSignal("");
  createEffect(() => setIcon(props.value));

  const url = () => iconURL(icon());

  const save = async (e: Event) => {
    e.preventDefault();
    e.stopPropagation();
    setError(await props.onSave(icon().trim()));
  };

  return (
    <form class="icon-input" onSubmit={save} onClick={e => e.stopPropagation()}>
      <div class="d-flex gap-1 align-items-center">
        <span class="icon-preview">
          <Show when={url() && !iconFailed(url())} fallback={<i class="bi bi-image opacity-50"></i>}>
            <img src={url()} alt="" onError={() => markIconFailed(url())}></img>
          </Show>
        </span>
        <input class="form-control form-control-sm" placeholder="immich, or an image URL" aria-label="Icon"
          value={icon()} onInput={e => setIcon(e.target.value)}></input>
        <button class="btn btn-sm btn-primary" type="submit" title="Save icon"><i class="bi bi-check-lg"></i></button>
        <Show when={props.onCancel}>
          <button class="btn btn-sm btn-outline-secondary" type="button" title="Cancel" onClick={() => props.onCancel!()}><i class="bi bi-x-lg"></i></button>
        </Show>
      </div>
      <Show when={error()} fallback={
        <div class="form-text">Names from <a href="https://dashboardicons.com" target="_blank">dashboardicons.com</a>. Empty uses the default, "none" hides it.</div>
      }>
        <div class="text-danger small mt-1">{error()}</div>
      </Show>
    </form>
  )
}

export default IconInput
