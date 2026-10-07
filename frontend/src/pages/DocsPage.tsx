import { createEffect, createMemo, createResource, createSignal, For, Match, on, onCleanup, Show, Switch } from "solid-js";
import { useBeforeLeave, useNavigate, useParams } from "@solidjs/router";
import { apiGetDocPage, apiGetDocs, apiPreviewDoc, apiSaveDocPage } from "../functions/api";
import { DocNav, DocPage, DocsIndex } from "../functions/exports";

// pages - every page in the menu, in menu order, with the section it sits in
function pages(nav: DocNav[], section = ""): { title: string, path: string, section: string }[] {
  return nav.flatMap(n => n.Path
    ? [{ title: n.Title, path: n.Path, section }]
    : pages(n.Children ?? [], section ? section + " / " + n.Title : n.Title));
}

function NavList(props: { items: DocNav[], here: string, filter: string, depth?: number }) {
  const match = (n: DocNav): boolean => n.Path
    ? n.Title.toLowerCase().includes(props.filter) || n.Path.toLowerCase().includes(props.filter)
    : (n.Children ?? []).some(match);
  return (
    <ul class="gm-docnav-list">
      <For each={props.items.filter(match)}>{n =>
        <li>
          <Show when={n.Path} fallback={
            <>
              <div class={props.depth ? "gm-docnav-sub" : "gm-navtitle mt-3"}>{props.depth ? n.Title : n.Title.toUpperCase()}</div>
              <NavList items={n.Children ?? []} here={props.here} filter={props.filter} depth={(props.depth ?? 0) + 1}></NavList>
            </>
          }>
            <a class={"gm-navlink" + (props.here == n.Path ? " active" : "")} href={"/docs/" + n.Path}>{n.Title}</a>
          </Show>
        </li>
      }</For>
    </ul>
  )
}

function DocsPage() {

  const params = useParams();
  const navigate = useNavigate();

  const [index, { refetch: reloadIndex }] = createResource<DocsIndex>(apiGetDocs);
  const all = createMemo(() => pages(index()?.Nav ?? []));

  // the page in the address, or the docs' home page
  const here = createMemo(() => {
    const p = decodeURIComponent(params.path ?? "").replace(/^\/+/, "");
    if (p) {
      return p;
    }
    const list = all();
    return list.find(x => x.path == "index.md")?.path ?? list[0]?.path ?? "";
  });

  const [pageErr, setPageErr] = createSignal("");
  const [page, { mutate: setPage, refetch: reloadPage }] = createResource(
    () => index()?.Repo && here() ? here() : undefined,
    async (p: string): Promise<DocPage | undefined> => {
      setPageErr("");
      try {
        return await apiGetDocPage(p);
      } catch (e: any) {
        setPageErr(e.message);
        return undefined;
      }
    });

  const [filter, setFilter] = createSignal("");

  // editing
  const [mode, setMode] = createSignal<"read" | "edit" | "new">("read");
  const [tab, setTab] = createSignal<"write" | "preview">("write");
  const [text, setText] = createSignal("");
  const [newPath, setNewPath] = createSignal("");
  const [message, setMessage] = createSignal("");
  const [preview, setPreview] = createSignal("");
  const [saveErr, setSaveErr] = createSignal("");
  const [conflict, setConflict] = createSignal(false);
  const [saving, setSaving] = createSignal(false);

  const dirty = () => mode() != "read" && (mode() == "new" ? text().trim() != "" : text() != page()?.Markdown);

  const guard = (e: BeforeUnloadEvent) => {
    if (dirty()) {
      e.preventDefault();
    }
  };
  window.addEventListener("beforeunload", guard);
  onCleanup(() => window.removeEventListener("beforeunload", guard));
  useBeforeLeave(e => {
    if (dirty() && !e.defaultPrevented && !confirm("Leave without saving?")) {
      e.preventDefault();
    }
  });

  // a different page leaves the editor
  createEffect(on(here, () => setMode("read"), { defer: true }));

  // jump to the heading in the address once the page is drawn
  createEffect(on(page, () => {
    if (location.hash) {
      setTimeout(() => document.getElementById(decodeURIComponent(location.hash.slice(1)))?.scrollIntoView(), 0);
    }
  }));

  const startEdit = () => {
    setText(page()?.Markdown ?? "");
    setMessage("");
    setSaveErr("");
    setConflict(false);
    setTab("write");
    setMode("edit");
  };

  const startNew = () => {
    const dir = here().includes("/") ? here().slice(0, here().lastIndexOf("/") + 1) : "";
    setNewPath(dir);
    setText("# New page\n\n");
    setMessage("");
    setSaveErr("");
    setConflict(false);
    setTab("write");
    setMode("new");
  };

  const cancel = () => {
    if (!dirty() || confirm("Throw away your changes?")) {
      setMode("read");
    }
  };

  const showPreview = async () => {
    setTab("preview");
    try {
      setPreview((await apiPreviewDoc(mode() == "new" ? newPath() : here(), text())).HTML);
    } catch (e: any) {
      setPreview("<p>" + e.message + "</p>");
    }
  };

  const save = async () => {
    let path = mode() == "new" ? newPath().trim().replace(/^\/+/, "") : here();
    if (mode() == "new") {
      if (!path || path.endsWith("/")) {
        setSaveErr("Give the page a file name, e.g. runbooks/backups.md");
        return;
      }
      if (!path.endsWith(".md")) {
        path += ".md";
      }
      if (all().some(p => p.path == path)) {
        setSaveErr(path + " already exists; open it and edit it instead");
        return;
      }
    }
    setSaving(true);
    setSaveErr("");
    setConflict(false);
    try {
      const saved: DocPage = await apiSaveDocPage(path, text(), mode() == "new" ? "" : page()?.Sha ?? "", message());
      const wasNew = mode() == "new";
      setMode("read");
      reloadIndex();
      if (wasNew) {
        navigate("/docs/" + path);
      } else {
        setPage({ ...saved, GitHub: page()?.GitHub ?? "", Title: saved.Title || page()?.Title || "" });
      }
    } catch (e: any) {
      setSaveErr(e.message);
      setConflict(e.message.includes("changed on GitHub"));
    }
    setSaving(false);
  };

  const loadLatest = async () => {
    if (confirm("Load the version on GitHub? Your changes here will be lost; copy them first if you need them.")) {
      await reloadPage();
      startEdit();
    }
  };

  const crumbs = () => here().split("/").slice(0, -1);

  return (
    <Switch>
      <Match when={index.loading && !index()}>
        <p class="text-muted">Loading docs…</p>
      </Match>

      <Match when={index() && !index()!.Repo}>
        <div class="card" style="max-width: 640px">
          <div class="card-body">
            <h1 class="h4"><i class="bi bi-journal-text me-2" style="color: var(--gm-accent)"></i>Docs</h1>
            <p style="color: var(--gm-muted)">
              Keep your lab's documentation as Markdown in a GitHub repository and read it here, next to
              the hosts it describes. Point Gimlé at the repository and folder under Settings; add a token
              to read a private repository, and to edit and commit pages from here.
            </p>
            <a class="btn btn-primary" href="/config">Set up docs</a>
          </div>
        </div>
      </Match>

      <Match when={index()}>
        <div class="gm-docs">
          <nav class="gm-docnav d-none d-lg-block" aria-label="Docs">
            <input class="form-control mb-2" placeholder="Find a page" value={filter()} onInput={e => setFilter(e.currentTarget.value.toLowerCase())}></input>
            <NavList items={index()!.Nav ?? []} here={here()} filter={filter()}></NavList>
            <Show when={index()!.CanEdit}>
              <button type="button" class="btn btn-outline-secondary btn-sm w-100 mt-3" onClick={startNew}>
                <i class="bi bi-plus-lg me-1"></i>New page
              </button>
            </Show>
          </nav>

          <div class="gm-docmain">
            <select class="form-select mb-3 d-lg-none" aria-label="Page" value={here()}
              onChange={e => navigate("/docs/" + e.currentTarget.value)}>
              <For each={all()}>{p =>
                <option value={p.path}>{p.section ? p.section + " / " : ""}{p.title}</option>
              }</For>
            </select>

            <Show when={index()!.Error}>
              <div class="alert alert-danger">The docs could not be read from {index()!.Repo}: {index()!.Error}. <a href="/config">Check Settings</a>.</div>
            </Show>

            <div class="d-flex flex-wrap align-items-center gap-2 mb-3">
              <div class="font-monospace small me-auto" style="color: var(--gm-muted)">
                {index()!.Repo}{index()!.Dir ? "/" + index()!.Dir : ""}
                <For each={crumbs()}>{c => <> / {c}</>}</For>
              </div>
              <Show when={mode() == "read"}>
                <Show when={page()?.GitHub}>
                  <a class="btn btn-outline-secondary btn-sm" href={page()!.GitHub} target="_blank"><i class="bi bi-github me-1"></i>GitHub</a>
                </Show>
                <Show when={index()!.CanEdit}>
                  <button type="button" class="btn btn-outline-secondary btn-sm d-lg-none" onClick={startNew}><i class="bi bi-plus-lg me-1"></i>New</button>
                  <Show when={page()}>
                    <button type="button" class="btn btn-primary btn-sm" onClick={startEdit}><i class="bi bi-pencil me-1"></i>Edit</button>
                  </Show>
                </Show>
              </Show>
            </div>

            <Switch>
              <Match when={mode() != "read"}>
                <div class="card">
                  <div class="card-header d-flex flex-wrap gap-2 align-items-center">
                    <Show when={mode() == "new"} fallback={<span class="font-monospace small">{here()}</span>}>
                      <input class="form-control form-control-sm font-monospace" style="max-width: 360px" aria-label="File name"
                        placeholder="folder/page-name.md" value={newPath()} onInput={e => setNewPath(e.currentTarget.value)}></input>
                    </Show>
                    <ul class="nav nav-pills ms-auto">
                      <li class="nav-item"><button type="button" class={"nav-link py-1" + (tab() == "write" ? " active" : "")} onClick={() => setTab("write")}>Write</button></li>
                      <li class="nav-item"><button type="button" class={"nav-link py-1" + (tab() == "preview" ? " active" : "")} onClick={showPreview}>Preview</button></li>
                    </ul>
                  </div>
                  <div class="card-body">
                    <Show when={tab() == "write"} fallback={<article class="gm-doc" innerHTML={preview()}></article>}>
                      <textarea class="form-control font-monospace gm-doc-editor" aria-label="Markdown" spellcheck={true}
                        value={text()} onInput={e => setText(e.currentTarget.value)}></textarea>
                    </Show>
                    <div class="d-flex flex-wrap gap-2 mt-3 align-items-center">
                      <input class="form-control" style="flex: 1 1 280px" aria-label="Commit message"
                        placeholder={(mode() == "new" ? "Add " : "Update ") + (mode() == "new" ? (newPath() || "page") : here())}
                        value={message()} onInput={e => setMessage(e.currentTarget.value)}></input>
                      <button type="button" class="btn btn-primary" disabled={saving()} onClick={save}>
                        {saving() ? "Committing…" : "Commit to GitHub"}
                      </button>
                      <button type="button" class="btn btn-outline-secondary" onClick={cancel}>Cancel</button>
                    </div>
                    <Show when={saveErr()}>
                      <div class="alert alert-danger mt-3 mb-0">
                        {saveErr()}
                        <Show when={conflict()}>
                          {" "}<button type="button" class="btn btn-sm btn-outline-secondary ms-2" onClick={loadLatest}>Load the latest version</button>
                        </Show>
                      </div>
                    </Show>
                  </div>
                </div>
              </Match>

              <Match when={pageErr()}>
                <div class="alert alert-secondary">{pageErr()}</div>
              </Match>

              <Match when={page()}>
                <article class="gm-doc" innerHTML={page()!.HTML}></article>
              </Match>

              <Match when={page.loading}>
                <p class="text-muted">Loading…</p>
              </Match>
            </Switch>
          </div>
        </div>
      </Match>
    </Switch>
  )
}

export default DocsPage
