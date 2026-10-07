import { useSearchParams } from "@solidjs/router";
import { createMemo, createSignal, For, onMount, Show } from "solid-js";
import { apiDelBookmark, apiGetBookmarks, apiGetMap, apiGetTags, apiSaveBookmark } from "../functions/api";
import { BookmarkInfo, MapHost, TagCount } from "../functions/exports";
import { hostName, serviceName } from "../functions/names";
import { iconURL, iconFailed, markIconFailed } from "../functions/icons";
import AppIcon from "../components/Views/AppIcon";

const splitTags = (s: string) => s.split(",").map(t => t.trim()).filter(t => t != "");

const address = (url: string) => url.replace(/^https?:\/\//, "").replace(/\/$/, "");

// Addresses added by hand, such as names behind a reverse proxy, with tags
// that put them into views and an optional link to a discovered host
function BookmarksPage() {

  const [search, setSearch] = useSearchParams();
  const [list, setList] = createSignal<BookmarkInfo[]>([]);
  const [hosts, setHosts] = createSignal<MapHost[]>([]);
  const [allTags, setAllTags] = createSignal<TagCount[]>([]);
  const [filter, setFilter] = createSignal("");

  // Form
  const [editID, setEditID] = createSignal(0);
  const [name, setName] = createSignal("");
  const [url, setURL] = createSignal("");
  const [note, setNote] = createSignal("");
  const [icon, setIcon] = createSignal("");
  const [tags, setTags] = createSignal("");
  const [mac, setMac] = createSignal("");
  const [port, setPort] = createSignal(0);
  const [synced, setSynced] = createSignal(false); // kept in sync by a connector such as Caddy
  const [error, setError] = createSignal("");

  const load = async () => {
    const [b, m, t] = await Promise.all([apiGetBookmarks(), apiGetMap(), apiGetTags()]);
    setList(b);
    setHosts([...m.Hosts].sort((a: MapHost, b: MapHost) => label(a).localeCompare(label(b))));
    setAllTags(t);
  };

  onMount(async () => {
    await load();
    // "Add a bookmark" from a host links it straight away
    if (search.mac) {
      reset();
      setMac(String(search.mac));
      setPort(parseInt(String(search.port ?? "0")) || 0);
      setSearch({ mac: undefined, port: undefined });
      document.getElementById("bmName")?.focus();
    }
  });

  const label = (h: MapHost) => hostName(h, h.IP) + (h.Name || h.DNS ? " (" + h.IP + ")" : "");
  const host = () => hosts().find(h => h.Mac == mac());

  const reset = () => {
    setEditID(0); setName(""); setURL(""); setNote(""); setIcon(""); setTags(""); setMac(""); setPort(0); setSynced(false); setError("");
  };

  const edit = (b: BookmarkInfo) => {
    setEditID(b.ID); setName(b.Name); setURL(b.URL); setNote(b.Note); setIcon(b.Icon);
    setTags(b.Tags.join(", ")); setMac(b.Mac); setPort(b.Port); setSynced(!!b.Source); setError("");
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const save = async (e: Event) => {
    e.preventDefault();
    try {
      await apiSaveBookmark({ ID: editID(), Name: name(), URL: url(), Note: note(), Icon: icon().trim(), Mac: mac(), Port: mac() ? port() : 0 }, splitTags(tags()));
      reset();
      await load();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const remove = async (b: BookmarkInfo) => {
    const more = b.Source ? " Your reverse proxy still serves it, but it won't be added again." : "";
    if (!confirm("Delete the bookmark \"" + (b.Name || address(b.URL)) + "\"? It also leaves every view it was in." + more)) return;
    await apiDelBookmark(b.ID);
    if (editID() == b.ID) reset();
    load();
  };

  // Add a tag from the list to the comma separated field
  const addTag = (t: string) => {
    const now = splitTags(tags());
    if (!now.includes(t)) setTags([...now, t].join(", "));
  };

  const shown = createMemo(() => {
    const f = filter().toLowerCase();
    if (!f) return list();
    return list().filter(b => [b.Name, b.URL, b.Note, b.HostName, ...b.Tags].some(s => s.toLowerCase().includes(f)));
  });

  const linkText = (b: BookmarkInfo) => {
    if (!b.Mac) return "";
    const h = hosts().find(h => h.Mac == b.Mac);
    if (!h) return "removed host";
    const svc = h.Web.find(p => p.Port == b.Port);
    return b.Port ? (svc ? serviceName(svc) + " on " : "") + b.HostName + " :" + b.Port : b.HostName;
  };

  const iconPreview = () => iconURL(icon().trim());

  return (
    <>
    <div class="card border-primary mb-4">
      <div class="card-header"><b>{editID() ? "Edit bookmark" : "Add a bookmark"}</b></div>
      <form class="card-body" onSubmit={save}>
        <div class="row g-2">
          <div class="col-md-4">
            <label class="form-label small mb-1" for="bmName">Name</label>
            <input class="form-control form-control-sm" id="bmName" placeholder="Immich" value={name()} onInput={e => setName(e.target.value)}></input>
          </div>
          <div class="col-md-5">
            <label class="form-label small mb-1" for="bmURL">Address</label>
            <input class="form-control form-control-sm" id="bmURL" placeholder="https://photos.example.lan" disabled={synced()} value={url()} onInput={e => setURL(e.target.value)}></input>
          </div>
          <div class="col-md-3">
            <label class="form-label small mb-1" for="bmIcon">Icon</label>
            <div class="d-flex gap-1 align-items-center">
              <span class="icon-preview">
                <Show when={iconPreview() && !iconFailed(iconPreview())} fallback={<i class="bi bi-image opacity-50"></i>}>
                  <img src={iconPreview()} alt="" onError={() => markIconFailed(iconPreview())}></img>
                </Show>
              </span>
              <input class="form-control form-control-sm" id="bmIcon" placeholder="immich" value={icon()} onInput={e => setIcon(e.target.value)}></input>
            </div>
          </div>
          <div class="col-md-4">
            <label class="form-label small mb-1" for="bmTags">Tags <span class="opacity-75">(each is a group in views)</span></label>
            <input class="form-control form-control-sm" id="bmTags" placeholder="Media, Family" value={tags()} onInput={e => setTags(e.target.value)}></input>
            <div class="d-flex flex-wrap gap-1 mt-1">
              <For each={allTags().filter(t => !splitTags(tags()).includes(t.Tag)).slice(0, 12)}>{t =>
                <button type="button" class="btn btn-sm btn-link p-0 me-2 small" onClick={() => addTag(t.Tag)}>+ {t.Tag}</button>
              }</For>
            </div>
          </div>
          <div class="col-md-5">
            <label class="form-label small mb-1" for="bmHost">Linked to <span class="opacity-75">(optional)</span></label>
            <div class="d-flex gap-1">
              <select class="form-select form-select-sm" id="bmHost" disabled={synced()} value={mac()} onChange={e => { setMac(e.target.value); setPort(0); }}>
                <option value="">Nothing</option>
                <For each={hosts()}>{h => <option value={h.Mac}>{label(h)}</option>}</For>
              </select>
              <Show when={host()}>{h =>
                <select class="form-select form-select-sm w-auto" aria-label="Service" disabled={synced()} value={String(port())} onChange={e => setPort(parseInt(e.target.value))}>
                  <option value="0">The host</option>
                  <For each={h().Web}>{p => <option value={String(p.Port)}>{serviceName(p)} :{p.Port}</option>}</For>
                  <Show when={port() && !h().Web.some(p => p.Port == port())}>
                    <option value={String(port())}>:{port()}</option>
                  </Show>
                </select>
              }</Show>
            </div>
          </div>
          <div class="col-md-3">
            <label class="form-label small mb-1" for="bmNote">Note <span class="opacity-75">(optional)</span></label>
            <input class="form-control form-control-sm" id="bmNote" placeholder="shown instead of the address" value={note()} onInput={e => setNote(e.target.value)}></input>
          </div>
        </div>
        <div class="d-flex gap-2 align-items-center mt-3">
          <button class="btn btn-sm btn-primary" type="submit">{editID() ? "Save" : "Add bookmark"}</button>
          <Show when={editID()}>
            <button class="btn btn-sm btn-outline-secondary" type="button" onClick={reset}>Cancel</button>
          </Show>
          <Show when={error()}><span class="text-danger small">{error()}</span></Show>
        </div>
        <Show when={synced()}>
          <div class="form-text"><i class="bi bi-arrow-repeat me-1"></i>Added from your reverse proxy, which keeps its address and link up to date. The name, icon, note and tags are yours to change.</div>
        </Show>
        <div class="form-text">
          Linking a bookmark to a discovered host puts it on that host's panel, and a link to one of its services makes that service's tiles open the bookmark's address.
        </div>
      </form>
    </div>

    <div class="card border-primary">
      <div class="card-header d-flex flex-wrap gap-2 align-items-center">
        <b class="me-auto">Bookmarks <span class="opacity-75 fw-normal">{list().length}</span></b>
        <input class="form-control form-control-sm w-auto" placeholder="Find a bookmark" aria-label="Find a bookmark" value={filter()} onInput={e => setFilter(e.target.value)}></input>
      </div>
      <div class="card-body table-responsive">
        <Show when={list().length > 0} fallback={
          <div class="text-center p-3 opacity-75">No bookmarks yet. Add the addresses you use every day, like the names you give apps in your reverse proxy, and tag them to show them in a view.</div>
        }>
          <table class="table table-sm table-hover align-middle mb-0">
            <thead><tr><th></th><th>Name</th><th>Tags</th><th>Linked to</th><th></th></tr></thead>
            <tbody>
              <For each={shown()}>{b =>
                <tr>
                  <td class="bookmark-icon"><AppIcon icon={b.Icon} title={b.Name} guess={true} fallback="bi-bookmark"></AppIcon></td>
                  <td>
                    <a href={b.URL} target="_blank">{b.Name || address(b.URL)}</a>
                    <Show when={b.Source}><i class="bi bi-arrow-repeat ms-1 opacity-50" title="Kept in sync by your reverse proxy"></i></Show>
                    <div class="small opacity-75">{b.Note || address(b.URL)}</div>
                  </td>
                  <td>
                    <For each={b.Tags} fallback={<small class="opacity-50">none</small>}>{t =>
                      <span class="badge rounded-pill text-bg-light border me-1">{t}</span>
                    }</For>
                  </td>
                  <td class="small">
                    <Show when={b.Mac} fallback={<span class="opacity-50">nothing</span>}>
                      <Show when={b.HostID} fallback={linkText(b)}>
                        <a href={"/host/" + b.HostID}>{linkText(b)}</a>
                      </Show>
                    </Show>
                  </td>
                  <td class="text-end text-nowrap">
                    <button class="btn btn-sm btn-link p-0 me-2" onClick={() => edit(b)} title="Edit"><i class="bi bi-pencil"></i></button>
                    <button class="btn btn-sm btn-link p-0 text-danger" onClick={() => remove(b)} title="Delete"><i class="bi bi-trash"></i></button>
                  </td>
                </tr>
              }</For>
            </tbody>
          </table>
        </Show>
      </div>
    </div>
    </>
  )
}

export default BookmarksPage
