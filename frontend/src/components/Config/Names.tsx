import { apiPath } from "../../functions/api"
import { appConfig } from "../../functions/exports"

function Names() {

  return (
    <div class="card border-primary">
      <div class="card-header">Host names</div>
      <div class="card-body table-responsive">
        <form action={apiPath + '/api/config_names/'} method="post">
          <table class="table table-borderless"><tbody>
            <tr>
              <td>DNS server</td>
              <td><input name="dnsserver" type="text" class="form-control" placeholder="system resolver" value={appConfig().DNSServer}></input></td>
            </tr>
            <tr>
              <td>Ask hosts over mDNS</td>
              <td>
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" name="mdns" checked={appConfig().NameMDNS}></input>
                </div>
              </td>
            </tr>
            <tr>
              <td>Ask hosts over NetBIOS</td>
              <td>
                <div class="form-check form-switch">
                  <input class="form-check-input" type="checkbox" name="netbios" checked={appConfig().NameNetBIOS}></input>
                </div>
              </td>
            </tr>
            <tr>
              <td><button type="submit" class="btn btn-primary">Save</button></td>
              <td class="text-muted small">Hosts without a name are looked up in DNS (your router or Pi-hole, as an IP like <code>192.168.1.1</code> or <code>192.168.1.1:5353</code>), then asked directly over mDNS (Apple, Linux, printers) and NetBIOS (Windows, Samba). Hosts that don't answer are retried every hour. Saving retries all of them on the next scan.</td>
            </tr>
          </tbody></table>
        </form>
      </div>
    </div>
  )
}

export default Names
