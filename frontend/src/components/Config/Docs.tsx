import { createSignal, Show } from "solid-js";
import { apiGetConfig, apiSaveDocsSettings } from "../../functions/api";
import { appConfig, setAppConfig } from "../../functions/exports";

function Docs() {

  const [token, setToken] = createSignal("");
  const [clear, setClear] = createSignal(false);
  const [msg, setMsg] = createSignal<{ ok: boolean, text: string } | null>(null);
  const [busy, setBusy] = createSignal(false);

  const save = async (e: SubmitEvent) => {
    e.preventDefault();
    const f = new FormData(e.currentTarget as HTMLFormElement);
    setBusy(true);
    setMsg(null);
    try {
      const out = await apiSaveDocsSettings({
        Repo: f.get("repo"), Branch: f.get("branch"), Dir: f.get("dir"),
        Token: token(), ClearToken: clear(), Edit: f.get("edit") == "on",
      });
      setToken("");
      setClear(false);
      setAppConfig(await apiGetConfig());
      if (!out.Repo) {
        setMsg({ ok: true, text: "Saved. Docs are off until a repository is set." });
      } else if (out.Error) {
        setMsg({ ok: false, text: "Saved, but the docs could not be read: " + out.Error });
      } else {
        const n = JSON.stringify(out.Nav ?? []).split('"Path"').length - 1;
        setMsg({ ok: true, text: `Saved. Found ${n} page${n == 1 ? "" : "s"}.` });
      }
    } catch (err: any) {
      setMsg({ ok: false, text: err.message });
    }
    setBusy(false);
  };

  return (
    <div class="card border-primary">
      <div class="card-header d-flex justify-content-between align-items-center">
        <span>Docs</span>
        <Show when={appConfig().DocsRepo}><a href="/docs">Open docs</a></Show>
      </div>
      <div class="card-body table-responsive">
        <form onSubmit={save}>
          <table class="table table-borderless"><tbody>
            <tr>
              <td>Repository</td>
              <td><input name="repo" type="text" class="form-control" placeholder="owner/name or its GitHub address" value={appConfig().DocsRepo}></input></td>
            </tr>
            <tr>
              <td>Branch</td>
              <td><input name="branch" type="text" class="form-control" placeholder="the default branch" value={appConfig().DocsBranch}></input></td>
            </tr>
            <tr>
              <td>Folder</td>
              <td><input name="dir" type="text" class="form-control" placeholder="the whole repository, or e.g. docs" value={appConfig().DocsDir}></input></td>
            </tr>
            <tr>
              <td>GitHub token</td>
              <td>
                <Show when={!appConfig().DocsTokenEnv} fallback={<span class="text-muted small">Set by the <code>DOCS_TOKEN</code> environment variable.</span>}>
                  <input type="password" class="form-control" autocomplete="off"
                    placeholder={appConfig().DocsHasToken ? "saved; type a new one to replace it" : "only needed for a private repository or editing"}
                    value={token()} onInput={e => setToken(e.currentTarget.value)}></input>
                  <Show when={appConfig().DocsHasToken}>
                    <div class="form-check mt-1 small">
                      <input class="form-check-input" type="checkbox" id="docs-clear" checked={clear()} onChange={e => setClear(e.currentTarget.checked)}></input>
                      <label class="form-check-label" for="docs-clear">Forget the saved token</label>
                    </div>
                  </Show>
                </Show>
              </td>
            </tr>
            <tr>
              <td>Edit from Gimlé</td>
              <td>
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" name="edit" checked={appConfig().DocsEdit}></input>
                </div>
              </td>
            </tr>
            <tr>
              <td><button type="submit" class="btn btn-primary" disabled={busy()}>Save</button></td>
              <td class="text-muted small">
                <Show when={msg()}>
                  <div class={"mb-2 " + (msg()!.ok ? "text-success" : "text-danger")}>{msg()!.text}</div>
                </Show>
                Markdown pages stay in your repository and are read from GitHub; when the folder above them holds
                an <code>mkdocs.yml</code>, its <code>nav</code> sets the menu. Use a fine-grained token limited to
                this one repository: <b>Contents: read</b> to show a private repository, <b>read and write</b> to edit.
                Gimlé has no login, so with editing on anyone who can open it can commit to the repository.
              </td>
            </tr>
          </tbody></table>
        </form>
      </div>
    </div>
  )
}

export default Docs
