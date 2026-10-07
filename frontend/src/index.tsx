/* @refresh reload */
import { render } from 'solid-js/web'
import App from './App.tsx'

// After an update, a page opened before it can ask for page code that no
// longer exists. Reload once to pick up the new version instead of showing
// an empty page.
const reloadForUpdate = () => {
  try {
    const last = Number(sessionStorage.getItem('gimle-reloaded') ?? 0)
    if (Date.now() - last < 10000) return
    sessionStorage.setItem('gimle-reloaded', String(Date.now()))
  } catch { /* storage blocked: reload anyway */ }
  location.reload()
}
window.addEventListener('vite:preloadError', e => { e.preventDefault(); reloadForUpdate() })
window.addEventListener('unhandledrejection', e => {
  const msg = String(e.reason?.message ?? e.reason ?? '')
  if (/dynamically imported module|Importing a module script failed|does not provide an export/i.test(msg)) reloadForUpdate()
})

const root = document.getElementById('root')

render(() => <App />, root!)
