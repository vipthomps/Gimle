# Gimle guide

Everything Gimle does, page by page. The [README](../README.md) has the short version.

## Install and update

Run the installer on your Proxmox host. It creates an unprivileged Debian 12 container with 512 MB of memory, installs `arp-scan`, downloads the latest release binary and runs it as a systemd service on port 8840:

```
bash <(curl -fsSL https://raw.githubusercontent.com/vipthomps/Gimle/main/proxmox/gimle-lxc.sh) --bridge vmbr0
```

Run the script with `--help` for every option. To discover hosts on another VLAN, give the container a NIC on it with `--nic vmbr0,20`; each NIC is added to the scanned interfaces. `--version v1.0.0` installs a particular release and `--binary ./gimle` installs one you built yourself.

To update, run `update` inside the container, from the Proxmox host with `pct exec CTID -- update`. It downloads the latest release, checks it against the release's checksums and restarts the service. `update v1.0.0` installs a particular release. Nothing is compiled in the container, so it needs no extra memory.

Outside Proxmox, any Debian or Ubuntu machine works: install `arp-scan` and `curl`, then download `proxmox/update.sh`, save it as `/usr/bin/update` and run it as root.

## The map

The homepage is a map of your network: one zone per subnet, one card per host, green for online and grey for offline. Hosts with a web service get a **web** badge that opens it; click a host for its details and every web service it runs. Drag hosts to arrange them (positions are saved), drag the background to pan, and scroll or use the zoom buttons to zoom. Reset puts every host back in its automatic spot. The old host table is under **Hosts**.

Switch the map to **Topology** for an automatic layout instead: each subnet with its gateway (the host on `.1` or `.254`) at the top, the other hosts below in one column per suggested category, and Docker hosts drawn taller with their running containers and published ports inside. With a Proxmox connector, each Proxmox node gets its own box holding its VMs and LXCs in VMID order; guests that were never seen on the network show as dashed cards. Topology is redrawn from scratch every time, so dragging is off there.

## Host names

New hosts are looked up in reverse DNS, using the system resolver or the DNS server set in Config → Host names (your router, Pi-hole or Technitium, as `192.168.1.1` or `192.168.1.1:5353`). Hosts that still have no name are asked directly over mDNS (Apple devices, Linux with Avahi, printers) and NetBIOS (Windows, Samba), and retried every hour. The map shows the short name (`nas` for `nas.example.lan`); the full DNS name stays on the host page. arp-scan's `(Unknown)` vendor text is dropped, and randomized phone and laptop addresses show as **Private MAC**.

Ports that may serve a web page are checked for its `<title>` after each port scan, so a service on port 2283 shows as **Immich** rather than a number.

## Connectors

Config → Connectors reads from other systems on your network. Each one only reads, and syncs on its own interval (15 minutes by default) or when you press sync. Tokens are stored in Gimle's database and never sent back to the browser.

| Kind | URL | Token | What it adds |
|---|---|---|---|
| Docker API | `http://host:2375`, ideally [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy) with `CONTAINERS=1` | optional (sent as a Bearer token) | Containers, their state and published ports |
| Dockhand | `http://host:3000` | an API token (`dh_...`), only when Dockhand has login turned on | Containers in every Docker environment Dockhand manages, matched to hosts by the environment's address or public IP |
| Scanopy | `http://host:60072` | a user API key (`scp_u_...`) | Host names, detected services and containers |
| UniFi console | `https://192.168.1.1` | an API key from UniFi Network → Settings → Control Plane → Integrations | Client aliases, DHCP host names and fixed IPs |
| Technitium DNS | `http://host:5380` | an API token | DHCP lease host names and the A records of local zones |
| Proxmox VE | `https://host:8006` | an API token as `USER@REALM!TOKENID=SECRET` with the PVEAuditor role on `/` | Every VM and LXC with its node, VMID, status, MAC and IP |
| Caddy | `http://host:2020`, a read-only proxy of its admin API (below) | optional | A bookmark for every site it reverse proxies, linked to the host and service behind it |

Reported hosts are matched to scanned ones by MAC address, then IP. A connector only fills in names that are still empty, never overwrites one you typed. Published container ports become services you can tag, named after the container (or the page title, once read). The host panel and host page list a host's containers, running first.

### Caddy

The Caddy connector reads Caddy's running config and adds a bookmark for every site name that ends in a `reverse_proxy`, such as `https://photos.example.lan`. Each one is linked to the host and port it forwards to, so tiles for that service open the proxy name instead of the IP. Upstreams on `localhost` count as the machine running Caddy (set **Caddy host IP** if the connector URL doesn't point at it), and an upstream named after a Docker container, like `immich-server:2283`, is matched to that container's host and published port when a Docker connector knows it.

Caddy owns each bookmark's address and link and updates them on every sync; the name, icon, note and tags are yours. A bookmark whose site leaves Caddy is removed. Delete one you don't want and it stays deleted. An address you already bookmarked by hand is left alone.

Caddy's admin API listens on `localhost:2019`, has no login, and can change Caddy's config, so don't open it to the network. Instead add a site that passes only `GET /config/` through, and point the connector at it:

```
:2020 {
	@read method GET
	handle @read {
		rewrite * /config/
		reverse_proxy localhost:2019 {
			header_up Host localhost:2019
		}
	}
	respond 403
}
```

Anyone who can reach port 2020 can read your Caddy config, which can include secrets such as DNS provider tokens, so limit it to Gimle's address with a `remote_ip` matcher or a firewall rule if that matters on your network.

## Suggested categories

Gimle suggests a category for every host and service, such as Network, Media, Downloads, Home automation or Monitoring, from page titles, container images, service names, host names and the MAC vendor. The **Categories** tab groups everything that way without any tags. **Use as tags** turns the suggestions into real tags for everything not tagged yet and adds an editable Categories view. The tag editor also offers each host's suggestion as a one-click tag. When a suggestion is wrong, pick the right category in the host panel on the map; it moves the host to that Topology column and Categories group, and "Suggested" puts it back.

## Tags and views

Tag hosts and the services they run (from the map's side panel or the host page) to group them, for example `Networking` or `Home Automation`. Bookmarks (below) take tags too.

Views are tabs next to **Network**. Each view shows a set of tags, in the order you pick, as groups; with no tags picked it shows every tag. A view's layout is **Tiles** (sections of cards, like a classic homepage) or **Map** (one zone per tag). Every card has a status dot: hosts use the last scan, and services and links are checked with a quick TCP connect when the view loads. Click **New view** to add one and **Edit** to rename it, change its layout and groups, reorder or remove cards, or add bookmarks.

The house button at the right of the tabs makes the current tab the start page, the one **Home** and `/` open. The network map is always at `/map`.

## Bookmarks

Bookmarks are addresses you add by hand, such as the names your reverse proxy (Caddy, Nginx Proxy Manager, Traefik) gives each app: `https://photos.example.lan` instead of `http://10.0.0.5:2283`. Add them on the **Bookmarks** page, or from a view's **Edit** mode. Each one has a name, an address, an optional icon and note, and any number of tags, so the same bookmark can sit in several groups. Typing an address without `https://` adds it.

A bookmark can be linked to a discovered host, or to one of its services. It then shows on that host's panel and page, and every tile and button for that service opens the bookmark's address instead of the IP and port. Its status dot checks the bookmark's own address. Links added to views before bookmarks existed become bookmarks on upgrade, keeping their tags.

Cards can have icons. Set one with **Icon** on a card in edit mode, or for a host in the map's side panel. An icon is a name from [dashboard-icons](https://dashboardicons.com) such as `immich` (add `.png` or `.webp` for those formats), or the URL of any image if you'd rather host your own. A service without its own icon uses its host's. Cards without an icon try the dashboard-icons name that matches their title (`Home Assistant` becomes `home-assistant`) and fall back to a plain symbol; set the icon to `none` to turn that off. Named icons load from the jsDelivr CDN, like the theme, so on a network without internet access use image URLs instead.

## Docs

**Docs** shows Markdown documentation kept in a GitHub repository, so notes on your hosts and runbooks sit next to the network they describe. The pages stay in the repository: Gimle reads them through the GitHub API each time (the file list is cached for 30 seconds), so a push from your editor shows up straight away. Set the repository, branch and folder under Settings → Docs.

- When the folder above the docs holds an `mkdocs.yml`, its `nav` sets the menu, so a site built for MkDocs moves over as is. Pages the nav leaves out are listed after it; without a nav, pages are grouped by folder.
- Links between pages, links to headings and images in the repository all work. MkDocs `!!! note` blocks show as quotes; raw HTML in a page is shown as text, never run.
- A public repository needs no token. For a private one, or to edit, use a [fine-grained token](https://github.com/settings/personal-access-tokens/new) limited to that one repository, with **Contents: read** to view or **Contents: read and write** to edit. The token is never sent to the browser. Set it with the `DOCS_TOKEN` environment variable to keep it out of the config file.
- With **Edit from Gimle** on, each page has an **Edit** button: write in Markdown, preview, and **Commit to GitHub** commits that one file to the branch. If the page changed on GitHub since you opened it, the commit is refused rather than overwriting it. **New page** adds a file.
- Gimle has no login of its own, so with editing on, anyone who can open it can commit to the repository. Keep it on a trusted network or behind your reverse proxy's authentication.

## Stats

**Stats** shows how your network has behaved over today, 7, 30 or 90 days: hosts online over time, addresses online in each subnet with how full it is and the next free IP, new devices per day, and every host's uptime with a cell per day. Each host page also shows its uptime for the last 30 days.

Every scan adds to a per-day counter for each host and a per-hour sum for each subnet, so stats outlive the trimmed history and stay small (`stats_days` sets how long they are kept). Uptime is the share of scans that saw a host online. Hosts already known when stats were added count as first seen on their oldest record, not as new devices.

## Subnets

The Subnets page lists each network with how many addresses are online, seen but offline, reserved and free, plus the next free address. Click one for a grid of every address.

- On first start, Gimle adds the IPv4 subnets on its own interfaces (only those in `ifaces`, if set). After that, add, edit or delete them on the Subnets page or through the API (`/api/subnets`, see `/swagger/index.html`).
- **Scan method** `arp` runs `arp-scan` against the subnet's CIDR on its interface (picked automatically when left empty). `none` tracks addresses without scanning.
- **Reserved addresses** are comma separated IPs or ranges (`192.168.1.1, 192.168.1.100-192.168.1.200`), for the gateway, the DHCP pool or statics you've set aside.
- Subnets from /16 to /32 are supported; the per-address grid is shown up to 4096 addresses.
- When no subnet uses `arp`, scanning falls back to the `ifaces` setting as in WatchYourLAN.

## Port scanning

Each host page has an **Open ports** card. Scan the common ports (about 80, weighted toward home lab services; Config → Port scanning lists them), all 65535, or your own list like `22,80,8000-8100`. The scan runs on the server, so you can leave the page. Open ports are stored with a service guess, and web ports link straight to the service.

Scheduled scans are off by default. Turn them on in Config → Port scanning to scan every online host on an interval and get a notification (through the Shoutrrr URL) when a host's ports open or close. A subnet can be left out of scheduled scans with its **Leave out of scheduled port scans** switch. Only ports in the scanned list are updated, so a quick scan never drops ports a full scan found.

Only scan networks you own or have permission to scan.

## Configuration

Settings live in `/var/lib/gimle/config_v2.yaml` and can be changed in the web UI. Any setting can also be set with an environment variable of the same name in upper case (`IFACES`, `TIMEOUT`, `PORT`, ...).

| Setting | Default | Meaning |
|---|---|---|
| `ifaces` | empty | Interfaces to scan, space separated |
| `timeout` | 120 | Seconds between scans |
| `port` | 8840 | Web UI port |
| `arp_strs` | empty | Extra `arp-scan` argument strings, e.g. a specific range |
| `port_scan` | false | Scheduled port scans of online hosts |
| `port_interval` | 360 | Minutes between scheduled port scans |
| `port_list` | top | Ports to scan on schedule: `top`, `all`, or a list like `22,80,8000-8100` |
| `port_workers` | 64 | Ports checked at once per host |
| `port_timeout` | 700 | Milliseconds to wait for each port |
| `stats_days` | 90 | Days of uptime and utilization stats to keep (0 keeps them forever) |
| `dns_server` | empty | DNS server for reverse lookups, `ip` or `ip:port`; empty uses the system resolver |
| `name_mdns` | true | Ask hosts without a DNS name for their name over mDNS |
| `name_netbios` | true | Ask hosts without a DNS name for their name over NetBIOS |
| `docs_repo` | empty | GitHub repository with the docs, `owner/name` |
| `docs_branch` | empty | Branch to read and commit to; empty uses the default branch |
| `docs_dir` | empty | Folder of Markdown pages in the repository; empty is the whole repository |
| `docs_token` | empty | GitHub token, needed for a private repository or editing |
| `docs_edit` | false | Allow editing pages and committing them from Gimle |
| `docs_api` | `https://api.github.com` | GitHub API address, only for GitHub Enterprise (`https://HOST/api/v3`) |

## Backups

Gimle has no login, so anyone who can open it can change its categories, views, bookmarks and settings. To make a bad change easy to undo, it keeps copies of its database and settings in `/var/lib/gimle/backups`:

| Kind | When | Kept |
|---|---|---|
| Every 4 hours | every 4 hours | 6 (one day) |
| Nightly | once a day after 03:00 local time | 7 (one week) |
| Manual | **Back up now** in Settings | 10 |
| Before restore | just before each restore | 5 |

Settings → Backups lists them. **Restore** puts back the hosts, tags, views, bookmarks, categories and settings from that copy, keeping the address and port Gimle is running on. The state just before the restore is saved first, so a restore can itself be undone. Backups hold connector and docs tokens, so they are never offered for download; copy them out of the container if you want them elsewhere.

## Development

```sh
cd frontend && npm ci && make all     # rebuilds the embedded UI
cd ../backend && go run ./cmd/gimle -d ./data
```

`make swag` in `backend` regenerates the API docs served at `/swagger/index.html`. Releases are built by GoReleaser when a `v*` tag is pushed; the tag must match `backend/internal/web/public/version`.
