import { createResource, createSignal, For, Show } from "solid-js";
import { apiCreateBackup, apiGetBackups, apiRestoreBackup } from "../../functions/api";

const kinds: Record<string, string> = {
  auto: "Every 4 hours",
  nightly: "Nightly",
  manual: "Manual",
  prerestore: "Before a restore",
};

function size(n: number) {
  return n < 1048576 ? Math.max(1, Math.round(n / 1024)) + " KB" : (n / 1048576).toFixed(1) + " MB";
}

function Backups() {

  const [list, { refetch }] = createResource(apiGetBackups);
  const [busy, setBusy] = createSignal("");
  const [msg, setMsg] = createSignal<{ ok: boolean, text: string } | null>(null);

  const backupNow = async () => {
    setBusy("now");
    setMsg(null);
    try {
      await apiCreateBackup();
      setMsg({ ok: true, text: "Backed up." });
      refetch();
    } catch (e: any) {
      setMsg({ ok: false, text: e.message });
    }
    setBusy("");
  };

  const restore = async (name: string, when: string) => {
    if (!confirm(`Restore the backup from ${when}? Hosts, categories, views, bookmarks and settings go back to how they were then. What you have now is backed up first, so you can undo this.`)) {
      return;
    }
    setBusy(name);
    setMsg(null);
    try {
      await apiRestoreBackup(name);
      location.reload();
    } catch (e: any) {
      setMsg({ ok: false, text: e.message });
      setBusy("");
      refetch();
    }
  };

  const when = (t: string) => new Date(t).toLocaleString([], { dateStyle: "medium", timeStyle: "short" });

  return (
    <div class="card border-primary">
      <div class="card-header d-flex justify-content-between align-items-center">
        <span>Backups</span>
        <Show when={list()?.Enabled}>
          <button type="button" class="btn btn-outline-secondary btn-sm" disabled={busy() != ""} onClick={backupNow}>
            <i class="bi bi-cloud-arrow-down me-1"></i>Back up now
          </button>
        </Show>
      </div>
      <div class="card-body">
        <Show when={list()} fallback={<p class="text-muted mb-0">Loading…</p>}>
          <Show when={list()!.Enabled} fallback={<p class="text-muted mb-0">Backups cover the built-in SQLite database. With PostgreSQL, back it up with its own tools.</p>}>
            <Show when={msg()}>
              <div class={"small mb-2 " + (msg()!.ok ? "text-success" : "text-danger")}>{msg()!.text}</div>
            </Show>
            <Show when={(list()!.Backups ?? []).length > 0} fallback={<p class="text-muted">The first backup is made within a few minutes of starting.</p>}>
              <div class="table-responsive" style="max-height: 340px">
                <table class="table table-sm align-middle mb-2">
                  <tbody>
                    <For each={list()!.Backups}>{b =>
                      <tr>
                        <td>{when(b.Time)}</td>
                        <td class="text-muted small">{kinds[b.Kind] ?? b.Kind}</td>
                        <td class="text-muted small text-end">{size(b.Size)}</td>
                        <td class="text-end">
                          <button type="button" class="btn btn-outline-secondary btn-sm" disabled={busy() != ""} onClick={() => restore(b.Name, when(b.Time))}>
                            {busy() == b.Name ? "Restoring…" : "Restore"}
                          </button>
                        </td>
                      </tr>
                    }</For>
                  </tbody>
                </table>
              </div>
            </Show>
            <p class="text-muted small mb-0">
              Hosts, categories, tags, views, bookmarks, connectors and settings are saved every 4 hours (the last
              day is kept) and each night (the last week is kept). A restore keeps the address and port in use. Backups stay
              in <code>{list()!.Dir}</code> and are never offered for download, because they hold your connector and docs tokens.
            </p>
          </Show>
        </Show>
      </div>
    </div>
  )
}

export default Backups
