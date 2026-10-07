// shortName - the host part of a DNS name ("nas.example.lan" -> "nas"); IPs and plain text are unchanged
export function shortName(dns: string): string {
  const first = (dns || "").trim().split(/\s+/)[0].replace(/\.$/, "");
  if (/^\d+\.\d+\.\d+\.\d+$/.test(first) || first.includes(":")) {
    return first;
  }
  const i = first.indexOf(".");
  return i > 0 ? first.slice(0, i) : first;
}

// hostName - what to call a host: its name, else its short DNS name, else its vendor
export function hostName(h: { Name: string; DNS: string; Hw: string }, fallback = "Unnamed"): string {
  return h.Name || shortName(h.DNS) || h.Hw || fallback;
}

// serviceName - a port's page title, else the service usually on that port
export function serviceName(p: { Title?: string; Container?: string; Service: string }, fallback = "Web"): string {
  return p.Title || p.Container || p.Service || fallback;
}
