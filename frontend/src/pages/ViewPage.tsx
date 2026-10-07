import { useNavigate, useParams, useSearchParams } from "@solidjs/router";
import { createEffect, createMemo, createSignal, For, on, onCleanup, onMount, Show } from "solid-js";
import { apiApplyCategories, apiDelItem, apiDelView, apiGetBookmarks, apiGetTags, apiGetView, apiMoveItem, apiMoveView, apiSaveBookmark, apiSaveIcon, apiSaveView } from "../functions/api";
import { BookmarkInfo, MapHost, TagCount, View, ViewData, ViewItem } from "../functions/exports";
import { bounds, groupLayout, NODE_H, NODE_W } from "../functions/maplayout";
import HostPanel from "../components/Map/HostPanel";
import MapCanvas from "../components/Map/MapCanvas";
import ViewTabs, { reloadViews } from "../components/Views/ViewTabs";
import AppIcon from "../components/Views/AppIcon";
import IconInput from "../components/Views/IconInput";
import { markIconFailed, pickIcon } from "../functions/icons";

const clip = (s: string, n: number) => s.length > n ? s.slice(0, n - 1) + "…" : s;

const kindIcon = (kind: string) =>
  kind == "host" ? "bi-pc-display" : kind == "service" ? "bi-hdd-network" : "bi-bookmark";

const statusText = (s: string) => s == "up" ? "Up" : s == "down" ? "Down" : "Status unknown";

const splitTags = (s: string) => s.split(",").map(t => t.trim()).filter(t => t != "");

function ViewPage() {

  const params = useParams();
  const [search, setSearch] = useSearchParams();
  const navigate = useNavigate();

  const [data, setData] = createSignal<ViewData>();
  const [missing, setMissing] = createSignal(false);
  const [allTags, setAllTags] = createSignal<TagCount[]>([]);
  const [selected, setSelected] = createSignal<MapHost>();
  const auto = () => params.id == "auto"; // suggested categories, built from scratch on every load
  const editing = () => search.edit == "1" && !auto();

  // View settings being edited
  const [name, setName] = createSignal("");
  const [layoutKind, setLayoutKind] = createSignal("tiles");
  const [tags, setTags] = createSignal<string[]>([]);
  const [error, setError] = createSignal("");

  // New bookmark, or an existing one, for a group
  const [bookmarks, setBookmarks] = createSignal<BookmarkInfo[]>([]);
  const [linkName, setLinkName] = createSignal("");
  const [linkURL, setLinkURL] = createSignal("");
  const [linkTag, setLinkTag] = createSignal("");
  const [linkError, setLinkError] = createSignal("");

  // Item whose icon is being edited
  const [iconItem, setIconItem] = createSignal(0);

  const load = async () => {
    const res: ViewData | null = await apiGetView(params.id);
    setMissing(res == null);
    if (res == null) return;
    setData(res);
    const sel = selected();
    if (sel) {
      const items = res.Groups.flatMap(g => g.Items);
      setSelected(items.find(it => it.Host?.Mac == sel.Mac)?.Host ?? undefined);
    }
  };

  const resetForm = () => {
    const v = data()?.View;
    if (!v) return;
    setName(v.Name);
    setLayoutKind(v.Layout);
    setTags(splitTags(v.Tags));
    setError("");
  };

  createEffect(on(() => params.id, async () => {
    setData(undefined);
    setSelected(undefined);
    await load();
    resetForm();
  }));
  createEffect(on(editing, async (on) => {
    if (on) {
      resetForm();
      setAllTags(await apiGetTags());
      setBookmarks(await apiGetBookmarks());
    }
  }));

  onMount(() => {
    const timer = setInterval(() => { if (!editing()) load(); }, 30000);
    onCleanup(() => clearInterval(timer));
  });

  const view = (): View | undefined => data()?.View;

  const [applying, setApplying] = createSignal(false);
  const applyCategories = async () => {
    if (!confirm("Tag every untagged host and service with its suggested category, and add a Categories view? Existing tags stay as they are.")) return;
    setApplying(true);
    try {
      const v: View = await apiApplyCategories();
      await reloadViews();
      navigate("/view/" + v.ID);
    } finally {
      setApplying(false);
    }
  };
  const items = () => data()?.Groups.flatMap(g => g.Items) ?? [];
  const up = () => items().filter(it => it.Status == "up").length;

  // Opening an item: its link in a new tab, otherwise the host
  const open = (it: ViewItem, panel: boolean) => {
    if (it.Link) {
      window.open(it.Link, "_blank");
    } else if (it.Host) {
      panel ? setSelected(it.Host) : navigate("/host/" + it.Host.ID);
    }
  };

  const saveView = async (e: Event) => {
    e.preventDefault();
    try {
      await apiSaveView({ ID: view()!.ID, Name: name(), Layout: layoutKind(), Tags: tags().join(",") });
      await reloadViews();
      setSearch({ edit: undefined });
      await load();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const deleteView = async () => {
    if (confirm("Delete the view \"" + view()!.Name + "\"? Tags on hosts and services stay.")) {
      await apiDelView(view()!.ID);
      await reloadViews();
      navigate("/map");
    }
  };

  const moveView = async (dir: number) => {
    await apiMoveView(view()!.ID, dir);
    reloadViews();
  };

  const moveTag = (i: number, dir: number) => {
    const t = [...tags()];
    const j = i + dir;
    if (j < 0 || j >= t.length) return;
    [t[i], t[j]] = [t[j], t[i]];
    setTags(t);
  };

  const unusedTags = () => allTags().filter(t => !tags().includes(t.Tag));

  const moveItem = async (it: ViewItem, dir: number) => {
    await apiMoveItem(it.ID, dir);
    load();
  };

  const removeItem = async (it: ViewItem) => {
    const what = "the tag \"" + it.Tag + "\" from " + it.Title;
    if (confirm("Remove " + what + "?")) {
      await apiDelItem(it.ID);
      load();
    }
  };

  // Bookmarks keep their icon; hosts and services share theirs across tags
  const saveIcon = async (it: ViewItem, icon: string) => {
    try {
      if (it.Kind == "bookmark") {
        const b = (await apiGetBookmarks() as BookmarkInfo[]).find(b => b.ID == it.Bookmark);
        if (!b) return "This bookmark no longer exists";
        await apiSaveBookmark({ ...b, Icon: icon }, b.Tags);
      } else {
        await apiSaveIcon(it.Mac, it.Kind == "host" ? 0 : it.Port, icon);
      }
    } catch (err: any) {
      return err.message as string;
    }
    setIconItem(0);
    await load();
    return "";
  };

  const groupTag = () => linkTag().trim() || data()?.Groups[0]?.Tag || "";

  const addLink = async (e: Event) => {
    e.preventDefault();
    if (!groupTag()) {
      setLinkError("Give it a tag: that is the group it shows up in");
      return;
    }
    try {
      await apiSaveBookmark({ Name: linkName(), URL: linkURL() }, [groupTag()]);
      setLinkName(""); setLinkURL(""); setLinkError("");
      setAllTags(await apiGetTags());
      setBookmarks(await apiGetBookmarks());
      load();
    } catch (err: any) {
      setLinkError(err.message);
    }
  };

  // Tag a bookmark that already exists into the group
  const addExisting = async (id: number) => {
    const b = bookmarks().find(b => b.ID == id);
    if (!b) return;
    if (!groupTag()) {
      setLinkError("Choose the tag to add it to first");
      return;
    }
    try {
      await apiSaveBookmark(b, [...b.Tags, groupTag()]);
      setLinkError("");
      setAllTags(await apiGetTags());
      setBookmarks(await apiGetBookmarks());
      load();
    } catch (err: any) {
      setLinkError(err.message);
    }
  };

  const placed = createMemo(() => groupLayout(data()?.Groups ?? []));
  const box = () => items().length > 0 || placed().zones.length > 0 ? bounds(placed().zones, placed().nodes) : null;

  const emptyHint = () =>
    <div class="text-center p-4 opacity-75">
      Nothing here yet. Open a host on the <a href="/map">Network</a> map and give it, or one of its services, a tag.
      Each tag becomes a group in this view.
    </div>;

  return (
    <>
    <ViewTabs active={auto() ? -1 : parseInt(params.id)}></ViewTabs>
    <Show when={!missing()} fallback={
      <div class="card border-primary"><div class="card-body">This view no longer exists. <a href="/map">Back to the network map</a></div></div>
    }>
    <div class="card border-primary">
      <div class="card-header d-flex flex-wrap gap-2 align-items-center">
        <span class="me-auto">
          <b>{view()?.Name}</b>
          <Show when={items().length > 0}>
            <span class="opacity-75 ms-2">{up()} of {items().length} up</span>
          </Show>
        </span>
        <Show when={!auto()} fallback={
          <button class="btn btn-sm btn-outline-secondary" onClick={applyCategories} disabled={applying()}
            title="Tag every host and service that has no tag yet with its suggested category, and add a Categories view you can edit">
            <i class="bi bi-tags me-1"></i>Use as tags
          </button>
        }>
        <Show when={!editing()} fallback={
          <button class="btn btn-sm btn-outline-secondary" onClick={() => { resetForm(); setSearch({ edit: undefined }); }}>Done</button>
        }>
          <button class="btn btn-sm btn-outline-secondary" onClick={() => setSearch({ edit: "1" })}><i class="bi bi-pencil me-1"></i>Edit</button>
        </Show>
        </Show>
      </div>

      <Show when={auto()}>
        <div class="card-body border-bottom py-2 small opacity-75">
          Grouped by what Gimlé can tell about each host and service: page titles, container images, service names and vendors. Tag something to put it where you want.
        </div>
      </Show>
      <Show when={editing()}>
        <form class="card-body border-bottom view-settings" onSubmit={saveView}>
          <div class="row g-2 align-items-end">
            <div class="col-sm-5">
              <label class="form-label small mb-1" for="viewName">Name</label>
              <input class="form-control form-control-sm" id="viewName" value={name()} onInput={e => setName(e.target.value)}></input>
            </div>
            <div class="col-sm-3">
              <label class="form-label small mb-1" for="viewLayout">Layout</label>
              <select class="form-select form-select-sm" id="viewLayout" value={layoutKind()} onChange={e => setLayoutKind(e.target.value)}>
                <option value="tiles">Tiles</option>
                <option value="map">Map</option>
              </select>
            </div>
            <div class="col-sm-4 d-flex gap-1 justify-content-sm-end">
              <button class="btn btn-sm btn-outline-secondary" type="button" onClick={() => moveView(-1)} title="Move tab left"><i class="bi bi-arrow-left"></i></button>
              <button class="btn btn-sm btn-outline-secondary" type="button" onClick={() => moveView(1)} title="Move tab right"><i class="bi bi-arrow-right"></i></button>
              <button class="btn btn-sm btn-outline-danger" type="button" onClick={deleteView}>Delete</button>
              <button class="btn btn-sm btn-primary" type="submit">Save</button>
            </div>
          </div>
          <div class="mt-3">
            <div class="form-label small mb-1">Groups, in order</div>
            <div class="d-flex flex-wrap gap-1 align-items-center">
              <For each={tags()} fallback={<small class="opacity-75 me-2">Every tag, A to Z.</small>}>{(t, i) =>
                <span class="badge rounded-pill text-bg-light border tag-chip">
                  <button type="button" class="btn btn-link p-0 me-1" onClick={() => moveTag(i(), -1)} title="Earlier"><i class="bi bi-chevron-left"></i></button>
                  {t}
                  <button type="button" class="btn btn-link p-0 ms-1" onClick={() => moveTag(i(), 1)} title="Later"><i class="bi bi-chevron-right"></i></button>
                  <button type="button" class="btn-close ms-1" aria-label={"Remove group " + t} onClick={() => setTags(tags().filter(x => x != t))}></button>
                </span>
              }</For>
              <select class="form-select form-select-sm w-auto" value="" aria-label="Add a group"
                onChange={e => { if (e.target.value) setTags([...tags(), e.target.value]); e.target.value = ""; }}>
                <option value="">Add a group…</option>
                <For each={unusedTags()}>{t => <option value={t.Tag}>{t.Tag} ({t.Count})</option>}</For>
              </select>
            </div>
          </div>
          <Show when={error()}><div class="text-danger small mt-2">{error()}</div></Show>
        </form>
      </Show>

      <Show when={data()}>
        <Show when={view()!.Layout == "map"} fallback={
          <div class="card-body">
            <Show when={data()!.Groups.length > 0} fallback={emptyHint()}>
              <For each={data()!.Groups}>{g =>
                <section class="mb-4">
                  <h6 class="view-group-title">{g.Tag}
                    <span class="opacity-75 fw-normal ms-2">{g.Items.filter(it => it.Status == "up").length}/{g.Items.length}</span>
                  </h6>
                  <div class="view-grid">
                    <For each={g.Items} fallback={<small class="opacity-75">Nothing tagged {g.Tag} yet.</small>}>{(it, i) =>
                      <div class={"view-tile" + (editing() ? " editing" : "")} role="link" tabindex="0"
                        title={statusText(it.Status) + (it.Link ? "\n" + it.Link : "")}
                        onClick={() => { if (!editing()) open(it, false); }}
                        onKeyDown={e => { if (e.key == "Enter" && !editing()) open(it, false); }}>
                        <span class={"status-dot " + it.Status}></span>
                        <AppIcon icon={it.Icon} title={it.Title} guess={true} fallback={kindIcon(it.Kind)}></AppIcon>
                        <span class="view-tile-text">
                          <span class="view-tile-title">{it.Title}</span>
                          <span class="view-tile-sub">{it.Subtitle}</span>
                        </span>
                        <Show when={editing()}>
                          <span class="view-tile-tools">
                            <button class="btn btn-sm btn-link p-0 me-auto" onClick={() => setIconItem(iconItem() == it.ID ? 0 : it.ID)} title="Set icon"><i class="bi bi-image"></i> Icon</button>
                            <button class="btn btn-sm btn-link p-0" disabled={i() == 0} onClick={() => moveItem(it, -1)} title="Earlier"><i class="bi bi-arrow-up"></i></button>
                            <button class="btn btn-sm btn-link p-0" disabled={i() == g.Items.length - 1} onClick={() => moveItem(it, 1)} title="Later"><i class="bi bi-arrow-down"></i></button>
                            <button class="btn btn-sm btn-link p-0 text-danger" onClick={() => removeItem(it)} title="Remove from this group"><i class="bi bi-x-lg"></i></button>
                          </span>
                          <Show when={iconItem() == it.ID}>
                            <div class="w-100">
                              <IconInput value={it.Icon} onSave={icon => saveIcon(it, icon)} onCancel={() => setIconItem(0)}></IconInput>
                            </div>
                          </Show>
                        </Show>
                      </div>
                    }</For>
                  </div>
                </section>
              }</For>
            </Show>
          </div>
        }>
          <div class="map-wrap">
            <MapCanvas bounds={box()} onBackgroundClick={() => setSelected(undefined)}>
              <For each={placed().zones}>{z =>
                <g>
                  <rect class="map-zone" x={z.x} y={z.y} width={z.w} height={z.h} rx="14"></rect>
                  <text class="map-zone-title" x={z.x + 18} y={z.y + 24}>
                    {z.name}
                    <tspan class="map-zone-sub" dx="10">{z.online}/{z.total} up</tspan>
                  </text>
                </g>
              }</For>
              <For each={placed().nodes}>{p =>
                <g class={"map-node " + p.item.Status + (selected() && selected()!.Mac == p.item.Host?.Mac ? " sel" : "")}
                  transform={`translate(${p.x},${p.y})`}
                  onPointerDown={e => e.stopPropagation()}
                  onClick={() => open(p.item, true)}>
                  <title>{p.item.Title}{"\n"}{statusText(p.item.Status)}{p.item.Link ? "\n" + p.item.Link : ""}</title>
                  <rect class="map-node-box" width={NODE_W} height={NODE_H} rx="9"></rect>
                  <rect class="map-node-stripe" width="5" height={NODE_H - 16} x="7" y="8" rx="2.5"></rect>
                  <Show when={pickIcon(p.item.Icon, p.item.Title, true)}>{url =>
                    <image href={url()} x="17" y="19" width="20" height="20" onError={() => markIconFailed(url())}></image>
                  }</Show>
                  <text class="map-node-name" x={pickIcon(p.item.Icon, p.item.Title, true) ? 43 : 20} y="24">{clip(p.item.Title, pickIcon(p.item.Icon, p.item.Title, true) ? 15 : 19)}</text>
                  <text class="map-node-ip" x={pickIcon(p.item.Icon, p.item.Title, true) ? 43 : 20} y="44">{clip(p.item.Subtitle, pickIcon(p.item.Icon, p.item.Title, true) ? 17 : 21)}</text>
                </g>
              }</For>
            </MapCanvas>
            <Show when={data()!.Groups.length == 0}>
              <div class="map-empty">{emptyHint()}</div>
            </Show>
            <Show when={selected()}>{h =>
              <HostPanel host={h()} onClose={() => setSelected(undefined)} onChange={load}></HostPanel>
            }</Show>
          </div>
        </Show>
      </Show>

      <Show when={editing()}>
        <form class="card-footer" onSubmit={addLink}>
          <div class="small fw-semibold mb-1">Add a bookmark</div>
          <div class="row g-2">
            <div class="col-sm-3">
              <input class="form-control form-control-sm" placeholder="Name" aria-label="Bookmark name" value={linkName()} onInput={e => setLinkName(e.target.value)}></input>
            </div>
            <div class="col-sm-4">
              <input class="form-control form-control-sm" placeholder="https://app.example.lan" aria-label="Address" value={linkURL()} onInput={e => setLinkURL(e.target.value)}></input>
            </div>
            <div class="col-sm-3">
              <input class="form-control form-control-sm" placeholder={data()?.Groups[0]?.Tag || "Tag"} aria-label="Tag" list="viewTags"
                value={linkTag()} onInput={e => setLinkTag(e.target.value)}></input>
              <datalist id="viewTags">
                <For each={allTags()}>{t => <option value={t.Tag}></option>}</For>
              </datalist>
            </div>
            <div class="col-sm-2 d-grid">
              <button class="btn btn-sm btn-primary" type="submit">Add</button>
            </div>
          </div>
          <Show when={bookmarks().some(b => !b.Tags.includes(groupTag()))}>
            <div class="row g-2 mt-0">
              <div class="col-sm-10">
                <select class="form-select form-select-sm" aria-label="Add an existing bookmark" value=""
                  onChange={e => { const id = parseInt(e.target.value); e.target.value = ""; if (id) addExisting(id); }}>
                  <option value="">Or add an existing bookmark to {groupTag() || "a group"}…</option>
                  <For each={bookmarks().filter(b => !b.Tags.includes(groupTag()))}>{b =>
                    <option value={b.ID}>{b.Name || b.URL}{b.Name ? " · " + b.URL.replace(/^https?:\/\//, "") : ""}</option>
                  }</For>
                </select>
              </div>
            </div>
          </Show>
          <Show when={linkError()}><div class="text-danger small mt-1">{linkError()}</div></Show>
          <div class="small opacity-75 mt-2">Hosts and services join a group when you tag them on the Network map or their host page. Manage every bookmark, and link them to hosts, on the <a href="/bookmarks">Bookmarks</a> page.</div>
        </form>
      </Show>
    </div>
    </Show>
    </>
  )
}

export default ViewPage
