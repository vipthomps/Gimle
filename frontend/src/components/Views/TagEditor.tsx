import { serviceName } from "../../functions/names";
import { createEffect, createSignal, For, on, Show } from "solid-js";
import { apiDelItem, apiGetHostTags, apiGetPorts, apiGetSuggestedTags, apiGetTags, apiSaveItem } from "../../functions/api";
import { HostPorts, Item, Port, TagCount } from "../../functions/exports";

// Tags on a host and its open ports. Each tag puts the host or service
// into the group of that name on every view that shows the tag.
function TagEditor(props: { hostID: number, mac: string }) {

  const [items, setItems] = createSignal<Item[]>([]);
  const [ports, setPorts] = createSignal<Port[]>([]);
  const [allTags, setAllTags] = createSignal<TagCount[]>([]);
  const [target, setTarget] = createSignal("0"); // port number, 0 for the host itself
  const [tag, setTag] = createSignal("");
  const [error, setError] = createSignal("");
  const [suggested, setSuggested] = createSignal<Record<string, string>>({});

  const load = async () => {
    if (!props.mac) return;
    setItems(await apiGetHostTags(props.mac));
    setAllTags(await apiGetTags());
    const res: HostPorts = await apiGetPorts(props.hostID);
    setPorts(res.Ports ?? []);
    setSuggested(await apiGetSuggestedTags(props.mac));
  };

  // Suggestions for whatever is picked in the list, unless it already has that tag
  const suggestion = () => {
    const port = parseInt(target());
    const cat = suggested()[String(port)];
    if (!cat) return "";
    const kind = port ? "service" : "host";
    return items().some(it => it.Tag == cat && it.Kind == kind && (kind == "host" || it.Port == port)) ? "" : cat;
  };
  createEffect(on(() => props.mac, load));

  const portName = (port: number) => {
    const p = ports().find(p => p.Port == port);
    return (p ? serviceName(p, "port") : "port") + " :" + port;
  };

  const add = async (e: Event) => {
    e.preventDefault();
    await addTag(tag());
  };

  const addTag = async (value: string) => {
    if (value.trim() == "") return;
    const port = parseInt(target());
    try {
      await apiSaveItem({ Tag: value, Kind: port ? "service" : "host", Mac: props.mac, Port: port });
      setTag("");
      setError("");
      load();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const remove = async (id: number) => {
    await apiDelItem(id);
    load();
  };

  const listID = () => "tags-" + props.hostID;

  return (
    <div>
      <div class="d-flex flex-wrap gap-1 mb-2">
        <For each={items()} fallback={
          <small class="opacity-75">No tags yet. Tag this host or one of its services to show it in a view group.</small>
        }>{it =>
          <span class="badge rounded-pill text-bg-light border tag-chip">
            <i class="bi bi-tag me-1"></i>{it.Tag}
            <Show when={it.Kind == "service"}><span class="opacity-75"> · {portName(it.Port)}</span></Show>
            <button type="button" class="btn-close ms-1" aria-label={"Remove tag " + it.Tag} onClick={() => remove(it.ID)}></button>
          </span>
        }</For>
      </div>
      <form class="d-flex gap-1" onSubmit={add}>
        <select class="form-select form-select-sm" style="max-width: 45%;" value={target()}
          onChange={e => setTarget(e.target.value)} aria-label="What to tag">
          <option value="0">This host</option>
          <For each={ports()}>{p =>
            <option value={p.Port}>{serviceName(p, "port") + " :" + p.Port}</option>
          }</For>
        </select>
        <input class="form-control form-control-sm" placeholder="Tag, e.g. Networking" list={listID()}
          value={tag()} onInput={e => setTag(e.target.value)}></input>
        <datalist id={listID()}>
          <For each={allTags()}>{t => <option value={t.Tag}></option>}</For>
        </datalist>
        <button class="btn btn-sm btn-outline-primary" type="submit" title="Add tag"><i class="bi bi-plus-lg"></i></button>
      </form>
      <Show when={suggestion()}>
        <div class="small mt-1">
          <span class="opacity-75">Suggested:</span>
          <button type="button" class="btn btn-sm btn-link p-0 ms-1 align-baseline" onClick={() => addTag(suggestion())}>
            <i class="bi bi-plus-circle me-1"></i>{suggestion()}
          </button>
        </div>
      </Show>
      <Show when={error()}><div class="text-danger small mt-1">{error()}</div></Show>
    </div>
  )
}

export default TagEditor
