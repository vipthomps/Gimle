export const apiPath = 'http://0.0.0.0:8840';

export const apiGetAllHosts = async () => {
  const url = apiPath+'/api/all';
  const hosts = await (await fetch(url)).json();

  return hosts;
};

export const apiGetConfig = async () => {

  const url = apiPath+'/api/config';
  const res = await (await fetch(url)).json();

  return res;
};

export const apiGetVersion = async () => {

  const url = apiPath+'/api/version';
  const res = await (await fetch(url)).json();

  return res;
};

export const apiTestNotify = async () => {

  const url = apiPath+'/api/notify_test';
  await fetch(url);
};

export const apiEditHost = async (id:number, name:string, known:string) => {

  const url = apiPath+'/api/edit/'+id+'/'+name+'/'+known;
  const res = await (await fetch(url)).json();

  return res;
};

export const apiGetHost = async (id:string) => {

  const url = apiPath+'/api/host/'+id;
  const res = await (await fetch(url)).json();

  return res;
};

export const apiDelHost = async (id:number) => {

  const url = apiPath+'/api/host/del/'+id;
  const res = await (await fetch(url)).json();

  return res;
};

export const apiPortScan = async (ip:string, port:number) => {

  const url = apiPath+'/api/port/'+ip+'/'+port;
  const res = await (await fetch(url)).json();

  return res;
};

export const apiGetHistory = async (mac:string) => {
  const url = apiPath+'/api/history/'+mac+'/?num=210';
  const hosts = await (await fetch(url)).json();

  return hosts;
};

export const apiGetHistoryByDate = async (mac:string, date: string) => {
  const url = apiPath+'/api/history/'+mac+'/'+date;
  const hosts = await (await fetch(url)).json();

  return hosts;
};

export const apiWOL = async (mac:string) => {

  const url = apiPath+'/api/wol/'+mac;
  const res = await (await fetch(url)).json();

  return res;
};
export const apiGetSubnets = async () => {
  const url = apiPath+'/api/subnets';
  const res = await (await fetch(url)).json();

  return res;
};

export const apiGetSubnetIPAM = async (id:string) => {
  const url = apiPath+'/api/subnets/'+id+'/ipam';
  const res = await (await fetch(url)).json();

  return res;
};

export const apiDetectSubnets = async () => {
  const url = apiPath+'/api/subnets/detect';
  const res = await (await fetch(url)).json();

  return res;
};

// Returns an error message, or "" on success
export const apiSaveSubnet = async (subnet:any) => {
  const url = apiPath+'/api/subnets';
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(subnet),
  });
  if (!res.ok) {
    const body = await res.json();
    return body.error ? body.error : res.statusText;
  }
  return "";
};

export const apiDelSubnet = async (id:number) => {
  const url = apiPath+'/api/subnets/'+id;
  await fetch(url, { method: 'DELETE' });
};

export const apiGetPorts = async (id:number) => {
  const url = apiPath+'/api/ports/'+id;
  const res = await (await fetch(url)).json();

  return res;
};

// Returns an error message, or "" on success
export const apiScanPorts = async (id:number, list:string) => {
  const url = apiPath+'/api/ports/'+id+'/scan?list='+encodeURIComponent(list);
  const res = await fetch(url, { method: 'POST' });
  if (!res.ok) {
    const body = await res.json();
    return body.error ? body.error : res.statusText;
  }
  return "";
};

export const apiGetTopPorts = async () => {
  const url = apiPath+'/api/portlist/top';
  const res = await (await fetch(url)).json();

  return res as { port: number; service: string }[];
};

export const apiGetMap = async () => {
  const url = apiPath+'/api/map';
  const res = await (await fetch(url)).json();

  return res;
};

export const apiSaveMapPos = async (mac:string, x:number, y:number) => {
  const url = apiPath+'/api/map/pos';
  await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ Mac: mac, X: x, Y: y }),
  });
};

export const apiResetMap = async () => {
  const url = apiPath+'/api/map/pos';
  await fetch(url, { method: 'DELETE' });
};

const postJSON = async (url:string, body:any) => {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ? data.error : res.statusText);
  }
  return data;
};

export const apiGetViews = async () => {
  const url = apiPath+'/api/views';
  return await (await fetch(url)).json();
};

// Returns the view data, or null if the view is gone
export const apiGetView = async (id:string) => {
  const url = apiPath+'/api/views/'+id;
  const res = await fetch(url);
  return res.ok ? await res.json() : null;
};

// Returns the saved view; throws with the error message
export const apiSaveView = async (view:any) => {
  return await postJSON(apiPath+'/api/views', view);
};

export const apiDelView = async (id:number) => {
  await fetch(apiPath+'/api/views/'+id, { method: 'DELETE' });
};

export const apiMoveView = async (id:number, dir:number) => {
  await fetch(apiPath+'/api/views/'+id+'/move?dir='+dir, { method: 'POST' });
};

export const apiGetTags = async () => {
  const url = apiPath+'/api/tags';
  return await (await fetch(url)).json();
};

export const apiGetHostTags = async (mac:string) => {
  const url = apiPath+'/api/tags/host/'+encodeURIComponent(mac);
  return await (await fetch(url)).json();
};

// Returns the saved item; throws with the error message
export const apiSaveItem = async (item:any) => {
  return await postJSON(apiPath+'/api/items', item);
};

export const apiDelItem = async (id:number) => {
  await fetch(apiPath+'/api/items/'+id, { method: 'DELETE' });
};

export const apiMoveItem = async (id:number, dir:number) => {
  await fetch(apiPath+'/api/items/'+id+'/move?dir='+dir, { method: 'POST' });
};

// Port 0 is the host itself; an empty icon goes back to the default
export const apiSaveIcon = async (mac:string, port:number, icon:string) => {
  return await postJSON(apiPath+'/api/icons', { Mac: mac, Port: port, Icon: icon });
};

export const apiGetStats = async (days:number) => {
  const url = apiPath+'/api/stats?days='+days;
  return await (await fetch(url)).json();
};

export const apiGetHostStats = async (mac:string, days:number) => {
  const url = apiPath+'/api/stats/host/'+encodeURIComponent(mac)+'?days='+days;
  const res = await fetch(url);
  return res.ok ? await res.json() : null;
};

export const apiGetConnectors = async () => {
  return await (await fetch(apiPath+'/api/connectors')).json();
};

export const apiGetConnectorKinds = async () => {
  return await (await fetch(apiPath+'/api/connectors/kinds')).json();
};

// Returns the saved connector; throws with the error message
export const apiSaveConnector = async (c:any) => {
  return await postJSON(apiPath+'/api/connectors', c);
};

export const apiDelConnector = async (id:number) => {
  await fetch(apiPath+'/api/connectors/'+id, { method: 'DELETE' });
};

export const apiSyncConnector = async (id:number) => {
  return await (await fetch(apiPath+'/api/connectors/'+id+'/sync', { method: 'POST' })).json();
};

export const apiGetContainers = async (mac:string) => {
  return await (await fetch(apiPath+'/api/containers?mac='+encodeURIComponent(mac))).json();
};

// Tags untagged hosts and services with their suggested category; returns the new view
export const apiApplyCategories = async () => {
  return await postJSON(apiPath+'/api/views/auto/apply', {});
};

// Suggested category per port number ("0" is the host itself)
export const apiGetSuggestedTags = async (mac:string) => {
  return await (await fetch(apiPath+'/api/tags/suggest/'+encodeURIComponent(mac))).json() as Record<string, string>;
};

export const apiSaveHostCategory = async (mac:string, category:string) => {
  return await postJSON(apiPath+'/api/categories/host', { Mac: mac, Category: category });
};

export const apiGetCategories = async () => {
  return await (await fetch(apiPath+'/api/categories')).json() as string[];
};

export const apiGetBookmarks = async () => {
  return await (await fetch(apiPath+'/api/bookmarks')).json();
};

export const apiSaveBookmark = async (b:any, tags:string[]) => {
  return await postJSON(apiPath+'/api/bookmarks', { ...b, Tags: tags });
};

export const apiDelBookmark = async (id:number) => {
  await fetch(apiPath+'/api/bookmarks/'+id, { method: 'DELETE' });
};

export const apiGetDocs = async () => {
  const res = await fetch(apiPath+'/api/docs');
  return await res.json();
};

// getJSON - like postJSON, a failed request throws with the server's message
const getJSON = async (url:string) => {
  const res = await fetch(url);
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.error ? data.error : res.statusText);
  }
  return data;
};

export const apiGetDocPage = async (path:string) => {
  return await getJSON(apiPath+'/api/docs/page?path='+encodeURIComponent(path));
};

export const apiSaveDocPage = async (path:string, markdown:string, sha:string, message:string) => {
  return await postJSON(apiPath+'/api/docs/page', { Path: path, Markdown: markdown, Sha: sha, Message: message });
};

export const apiPreviewDoc = async (path:string, markdown:string) => {
  return await postJSON(apiPath+'/api/docs/preview', { Path: path, Markdown: markdown });
};

export const apiSaveDocsSettings = async (s:any) => {
  return await postJSON(apiPath+'/api/config/docs', s);
};

export const apiGetBackups = async () => {
  return await getJSON(apiPath+'/api/backups');
};

export const apiCreateBackup = async () => {
  return await postJSON(apiPath+'/api/backups', {});
};

export const apiRestoreBackup = async (name:string) => {
  return await postJSON(apiPath+'/api/backups/'+encodeURIComponent(name)+'/restore', {});
};

export const apiSaveStartPage = async (page:string) => {
  return await postJSON(apiPath+'/api/config/start', { Page: page });
};
