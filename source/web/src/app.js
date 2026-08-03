/* Bonghos SPA — dependency-free vanilla JS */
"use strict";

// ---------------------------------------------------------------------------
// tiny helpers
// ---------------------------------------------------------------------------
const $ = (sel, root = document) => root.querySelector(sel);
const el = (tag, attrs = {}, ...children) => {
  const n = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") n.className = v;
    else if (k.startsWith("on")) n.addEventListener(k.slice(2), v);
    else if (v !== null && v !== undefined) n.setAttribute(k, v);
  }
  for (const c of children.flat()) {
    if (c === null || c === undefined) continue;
    n.append(c.nodeType ? c : document.createTextNode(c));
  }
  return n;
};
const esc = (s) => String(s ?? "");
const fmtBytes = (b) => {
  if (b === null || b === undefined || isNaN(b)) return "—";
  const u = ["B", "KiB", "MiB", "GiB", "TiB"];
  let i = 0; b = Number(b);
  while (b >= 1024 && i < u.length - 1) { b /= 1024; i++; }
  return b.toFixed(i ? 1 : 0) + " " + u[i];
};
const fmtDur = (s) => {
  if (!s && s !== 0) return "—";
  s = Math.floor(s);
  const d = Math.floor(s / 86400), h = Math.floor(s % 86400 / 3600), m = Math.floor(s % 3600 / 60);
  if (d) return `${d}d ${h}h`;
  if (h) return `${h}h ${m}m`;
  return `${m}m ${s % 60}s`;
};
const fmtTime = (iso) => iso ? new Date(iso).toLocaleString() : "—";

function toast(msg, kind = "") {
  const t = el("div", { class: "toast " + kind }, msg);
  $("#toast-host").append(t);
  setTimeout(() => t.remove(), 6000);
}

function modal(title, bodyNodes, actions) {
  const host = $("#modal-host");
  host.innerHTML = "";
  const close = () => (host.innerHTML = "");
  const m = el("div", { class: "overlay", onclick: (e) => { if (e.target === m) close(); } },
    el("div", { class: "modal" },
      el("h2", {}, title),
      ...bodyNodes,
      el("div", { class: "actions" },
        ...actions.map(([label, cls, fn]) =>
          el("button", { class: "btn " + cls, onclick: () => fn(close) }, label)))));
  host.append(m);
  return close;
}

function confirmModal(title, message, confirmLabel, onConfirm, danger = true) {
  modal(title, [el("p", {}, message)], [
    ["Cancel", "ghost", (c) => c()],
    [confirmLabel, danger ? "danger" : "primary", async (c) => { c(); await onConfirm(); }],
  ]);
}

// ---------------------------------------------------------------------------
// API
// ---------------------------------------------------------------------------
let csrfToken = "";
async function api(path, opts = {}) {
  const headers = opts.headers || {};
  if (opts.method && opts.method !== "GET") headers["X-Bonghos-CSRF"] = csrfToken;
  if (opts.json !== undefined) {
    headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(opts.json);
  }
  const res = await fetch("/api" + path, { ...opts, headers, credentials: "same-origin" });
  if (res.status === 401) { showLogin(); throw new Error("Session expired — please sign in again."); }
  let data = null;
  try { data = await res.json(); } catch { /* empty */ }
  if (!res.ok) throw new Error((data && data.error) || res.statusText);
  return data;
}

// ---------------------------------------------------------------------------
// state
// ---------------------------------------------------------------------------
const S = {
  me: null,
  servers: [],
  activeId: 0,
  status: { state: "stopped" },
  ws: null,
  page: "overview",
  consoleLines: [],
  perf: [],
};
const can = (p) => S.me && S.me.permissions && S.me.permissions.includes(p);

// ---------------------------------------------------------------------------
// websocket
// ---------------------------------------------------------------------------
let wsRetry = 1000;
// Topics that stay subscribed for the whole session.
const BASE_TOPICS = ["overview", "servers", "backups"];
// Heavier per-page topics, subscribed only while that page is open so Bonghos
// does not broadcast console lines or metrics to a browser that is not showing
// them (specification section 28).
const PAGE_TOPICS = {
  console: "console",
  performance: "performance",
  players: "players",
  schedules: "schedules",
  activity: "activity",
};
let currentPageTopic = null;

function wsSend(obj) {
  if (S.ws && S.ws.readyState === WebSocket.OPEN) {
    try { S.ws.send(JSON.stringify(obj)); } catch {}
    return true;
  }
  return false;
}

// syncPageSubscription unsubscribes the previous page topic and subscribes the
// new one. Base topics are untouched.
function syncPageSubscription(page) {
  const next = PAGE_TOPICS[page] || null;
  if (next === currentPageTopic) return;
  if (currentPageTopic && !BASE_TOPICS.includes(currentPageTopic)) {
    wsSend({ action: "unsubscribe", topic: currentPageTopic });
  }
  currentPageTopic = next;
  if (next) wsSend({ action: "subscribe", topic: next });
}

function connectWS() {
  if (S.ws) try { S.ws.close(); } catch {}
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const ws = new WebSocket(`${proto}//${location.host}/api/ws`);
  S.ws = ws;
  ws.onopen = () => {
    wsRetry = 1000;
    // Always-on topics: status and long-running operations must keep arriving
    // whatever page is open, so an import or backup started elsewhere is seen.
    BASE_TOPICS.forEach((t) => wsSend({ action: "subscribe", topic: t }));
    // Server-side subscriptions were lost with the old connection, so forget
    // what we think is subscribed and re-send for the current page.
    currentPageTopic = null;
    syncPageSubscription(S.page);
  };
  ws.onmessage = (ev) => {
    let m; try { m = JSON.parse(ev.data); } catch { return; }
    handleEvent(m);
  };
  ws.onclose = () => {
    if (!S.me) return;
    setTimeout(connectWS, wsRetry);
    wsRetry = Math.min(wsRetry * 2, 15000);
  };
}

function handleEvent(m) {
  const { topic, type, data } = m;
  if (topic === "console" && type === "line") {
    S.consoleLines.push(data.line);
    if (S.consoleLines.length > 1000) S.consoleLines.splice(0, S.consoleLines.length - 1000);
    appendConsoleLine(data.line);
  } else if (type === "status") {
    S.status = data || S.status;
    renderStatusPill();
    if (S.page === "overview") renderPage();
  } else if (topic === "performance" && type === "sample") {
    S.perf.push(data);
    if (S.perf.length > 360) S.perf.shift();
    if (S.page === "performance" || S.page === "overview") updateLiveStats(data);
  } else if (topic === "players") {
    if (S.page === "players" || S.page === "overview") renderPage();
  } else if (topic === "servers" && (type === "operation" || type === "installed")) {
    updateOperation(data, type);
  } else if (topic === "backups") {
    if (type === "created") { toast("Backup completed: " + data.backup_id, "ok"); if (S.page === "backups") renderPage(); }
    if (type === "failed") toast("Backup failed: " + data.error, "err");
    if (type === "progress" && S.page === "backups") updateBackupProgress(data);
  } else if (topic === "activity" && S.page === "activity") {
    renderPage();
  }
}

// ---------------------------------------------------------------------------
// auth flow
// ---------------------------------------------------------------------------
function showLogin() {
  S.me = null;
  if (S.ws) try { S.ws.close(); } catch {}
  $("#app-view").classList.add("hidden");
  $("#login-view").classList.remove("hidden");
  loginStep(1);
}

// loginStep switches between the credential step and the authenticator step.
// Step two is always reached, whatever was typed in step one: the interface
// must not reveal whether an account exists any more than the API does.
function loginStep(n) {
  const s1 = $("#login-step-1"), s2 = $("#login-step-2");
  if (!s1 || !s2) return;
  s1.classList.toggle("hidden", n !== 1);
  s2.classList.toggle("hidden", n !== 2);
  $("#login-error").classList.add("hidden");
  if (n === 2) {
    $("#login-step-2-who").textContent = "Signing in as " + $("#login-user").value.trim();
    $("#login-code").value = "";
    setTimeout(() => $("#login-code").focus(), 30);
  } else {
    setTimeout(() => $("#login-user").focus(), 30);
  }
}

async function boot() {
  const c = await api("/auth/csrf");
  csrfToken = c.csrf;
  try {
    S.me = await api("/auth/me");
    enterApp();
  } catch { showLogin(); }
}

$("#login-back").addEventListener("click", () => loginStep(1));

$("#login-form").addEventListener("submit", async (e) => {
  e.preventDefault();

  // Step one never contacts the server. Nothing is checked here, so nothing
  // can be learned from how fast or how differently it responds.
  const onStepOne = $("#login-step-2").classList.contains("hidden");
  if (onStepOne) {
    const user = $("#login-user").value.trim();
    const pass = $("#login-pass").value;
    if (!user || !pass) {
      const eb = $("#login-error");
      eb.textContent = "Enter your username and password.";
      eb.classList.remove("hidden");
      return;
    }
    loginStep(2);
    return;
  }

  const btn = $("#login-btn"); btn.disabled = true;
  $("#login-error").classList.add("hidden");
  try {
    S.me = await api("/auth/login", { method: "POST", json: {
      username: $("#login-user").value.trim(),
      password: $("#login-pass").value,
      code: $("#login-code").value.trim(),
    }});
    const c = await api("/auth/csrf"); csrfToken = c.csrf;
    $("#login-pass").value = ""; $("#login-code").value = "";
    enterApp();
  } catch (err) {
    const eb = $("#login-error"); eb.textContent = err.message; eb.classList.remove("hidden");
    $("#login-code").value = "";
    $("#login-code").focus();
  } finally { btn.disabled = false; }
});

$("#logout-btn").addEventListener("click", async () => {
  try { await api("/auth/logout", { method: "POST", json: {} }); } catch {}
  showLogin();
});

function enterApp() {
  $("#login-view").classList.add("hidden");
  $("#app-view").classList.remove("hidden");
  $("#whoami").textContent = `${S.me.username} · ${S.me.role}`;
  buildNav();
  connectWS();
  refreshServers().then(() => navigate(S.page));
}

// ---------------------------------------------------------------------------
// navigation
// ---------------------------------------------------------------------------
const PAGES = [
  ["overview", "Overview", "server.view"],
  ["console", "Console", "server.console.view"],
  ["players", "Players", "server.players.view"],
  ["files", "Files", "server.files.manage"],
  ["configuration", "Configuration", "server.configuration.manage"],
  ["backups", "Backups", "server.backups.view"],
  ["schedules", "Schedules", "server.schedules.manage"],
  ["performance", "Performance", "server.view"],
  ["servers", "Servers", "server.view"],
  ["activity", "Activity", "server.configuration.manage"],
  ["users", "Users", "users.manage"],
  ["host", "Host", "server.configuration.manage"],
];

function buildNav() {
  const nav = $("#nav"); nav.innerHTML = "";
  for (const [id, label, perm] of PAGES) {
    if (perm && !can(perm)) continue;
    nav.append(el("div", { class: "nav-item", "data-page": id, onclick: () => navigate(id) }, label));
  }
}

function navigate(page) {
  S.page = page;
  syncPageSubscription(page);
  document.querySelectorAll(".nav-item").forEach((n) =>
    n.classList.toggle("active", n.dataset.page === page));
  renderPage();
}

async function refreshServers() {
  try {
    const d = await api("/servers");
    S.servers = d.servers || [];
    S.activeId = d.active_id || 0;
    renderServerPicker();
    const st = await api("/server/status");
    S.status = st;
  } catch (e) { /* server list may 403 for some roles */ }
}

function renderServerPicker() {
  const host = $("#server-picker"); host.innerHTML = "";
  const active = S.servers.find((s) => s.id === S.activeId);
  host.append(
    el("div", { class: "muted", style: "margin-bottom:6px" }, "Active project"),
    el("div", { style: "font-weight:600; font-size:0.9rem" }, active ? active.display_name : "None selected"),
    renderStatusPillNode());
}

function renderStatusPillNode() {
  const st = S.status.state || "stopped";
  return el("div", { class: "pill " + st, id: "status-pill", style: "margin-top:8px" },
    el("span", { class: "dot" }), st.charAt(0).toUpperCase() + st.slice(1));
}
function renderStatusPill() {
  const p = $("#status-pill");
  if (p) p.replaceWith(renderStatusPillNode());
}

// ---------------------------------------------------------------------------
// pages
// ---------------------------------------------------------------------------
async function renderPage() {
  const main = $("#main");
  try {
    switch (S.page) {
      case "overview": return await pageOverview(main);
      case "console": return await pageConsole(main);
      case "players": return await pagePlayers(main);
      case "files": return await pageFiles(main);
      case "configuration": return await pageConfiguration(main);
      case "backups": return await pageBackups(main);
      case "schedules": return await pageSchedules(main);
      case "performance": return await pagePerformance(main);
      case "servers": return await pageServers(main);
      case "activity": return await pageActivity(main);
      case "users": return await pageUsers(main);
      case "host": return await pageHost(main);
    }
  } catch (err) {
    main.innerHTML = "";
    main.append(el("h1", {}, "Something went wrong"), el("p", { class: "muted" }, err.message));
  }
}

// ----- overview -------------------------------------------------------------
async function pageOverview(main) {
  const d = await api("/overview");
  S.status = { state: d.state, detail: d.supervisor };
  renderStatusPill();
  const s = d.sample || {};
  const inst = d.instance;
  main.innerHTML = "";
  main.append(
    el("div", { class: "toolbar" },
      el("h1", {}, inst ? inst.display_name : "Overview"),
      renderStatusPillNode(),
      el("div", { class: "spacer" }),
      lifecycleButtons()),
    el("div", { class: "grid cols-4" },
      statCard("Uptime", fmtDur(s.uptime_seconds), ""),
      statCard("Players online", s.online_players ?? 0, ""),
      statCard("Process memory", fmtBytes(s.rss_bytes), "resident set (not Java heap)"),
      statCard("CPU", (s.cpu_percent ?? 0).toFixed(1) + "%", "of one core = 100%")),
    el("div", { class: "grid cols-3", style: "margin-top:16px" },
      statCard("Disk free", fmtBytes(s.disk_free), "of " + fmtBytes(s.disk_total)),
      statCard("Last backup", d.last_backup ? fmtTime(d.last_backup.created_at) : "None yet",
        d.last_backup ? d.last_backup.backup_type : ""),
      statCard("Next schedule", d.next_schedule_at ? fmtTime(d.next_schedule_at) : "None", "")),
    inst ? el("div", { class: "card", style: "margin-top:16px" },
      el("h3", {}, "Project"),
      el("dl", { class: "kv" },
        el("dt", {}, "MOTD"), el("dd", {}, d.motd || "—"),
        el("dt", {}, "Port"), el("dd", {}, d.port || "25565"),
        el("dt", {}, "Modloader"), el("dd", {}, inst.modloader || "unknown"),
        el("dt", {}, "Startup script"), el("dd", { class: "mono" }, inst.startup_script || "not selected"),
        el("dt", {}, "Restart policy"), el("dd", {}, inst.restart_policy || "never"))) : null);
}

function statCard(title, value, sub) {
  return el("div", { class: "card" },
    el("h3", {}, title),
    el("div", { class: "stat" }, String(value), sub ? el("small", {}, " " + sub) : null));
}

function lifecycleButtons() {
  const st = S.status.state || "stopped";
  const running = st === "running" || st === "starting";
  const wrap = el("div", { class: "row-actions" });
  const act = async (path, label) => {
    try { await api(path, { method: "POST", json: {} }); toast(label + " requested", "ok"); }
    catch (e) { toast(e.message, "err"); }
  };
  if (can("server.start") && !running)
    wrap.append(el("button", { class: "btn primary", onclick: () => act("/server/start", "Start") }, "Start"));
  if (can("server.stop") && running)
    wrap.append(el("button", { class: "btn", onclick: () => act("/server/stop", "Stop") }, "Stop"));
  if (can("server.restart") && running)
    wrap.append(el("button", { class: "btn", onclick: () => act("/server/restart", "Restart") }, "Restart"));
  if (can("server.force_stop") && st !== "stopped")
    wrap.append(el("button", { class: "btn danger", onclick: () =>
      confirmModal("Force stop", "Force stop kills the Java process immediately. Unsaved world data may be lost. Continue?",
        "Force stop", async () => {
          try { await api("/server/force-stop", { method: "POST", json: { confirm: true } }); toast("Force stop sent", "ok"); }
          catch (e) { toast(e.message, "err"); }
        }) }, "Force stop"));
  return wrap;
}

// ----- console --------------------------------------------------------------
async function pageConsole(main) {
  main.innerHTML = "";
  const box = el("div", { class: "console", id: "console-box" });
  for (const line of S.consoleLines) box.append(consoleLineNode(line));
  const input = el("input", { placeholder: can("server.console.use") ? "Type a command (e.g. say hello) and press Enter" : "Read-only console", spellcheck: "false" });
  if (!can("server.console.use")) input.disabled = true;
  input.addEventListener("keydown", async (e) => {
    if (e.key !== "Enter" || !input.value.trim()) return;
    const cmd = input.value.trim(); input.value = "";
    try { await api("/server/command", { method: "POST", json: { command: cmd } }); }
    catch (err) { toast(err.message, "err"); }
  });
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Console"), renderStatusPillNode(),
      el("div", { class: "spacer" }), lifecycleButtons()),
    box,
    el("div", { class: "console-input" }, input,
      el("button", { class: "btn ghost", onclick: () => { box.innerHTML = ""; S.consoleLines = []; } }, "Clear view")));
  box.scrollTop = box.scrollHeight;
}

function consoleLineNode(line) {
  let cls = "";
  if (/ERROR|SEVERE|FATAL/.test(line)) cls = "errline";
  else if (/WARN/.test(line)) cls = "warnline";
  return el("div", { class: cls }, line);
}

function appendConsoleLine(line) {
  const box = $("#console-box");
  if (!box) return;
  const atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
  box.append(consoleLineNode(line));
  while (box.childNodes.length > 1000) box.firstChild.remove();
  if (atBottom) box.scrollTop = box.scrollHeight;
}

// ----- players --------------------------------------------------------------
async function pagePlayers(main) {
  const d = await api("/players");
  const players = d.players || [];
  main.innerHTML = "";
  const rows = players.map((p) => el("tr", {},
    el("td", {}, el("span", { class: "pill " + (p.online ? "running" : "") },
      el("span", { class: "dot" }), p.username)),
    el("td", {}, p.online ? "Online" : "Offline"),
    el("td", {}, fmtTime(p.last_seen_at)),
    el("td", {}, fmtDur(p.observed_playtime_seconds)),
    el("td", { class: "row-actions" }, can("server.players.manage") ? playerActions(p) : "")));
  main.append(
    el("h1", {}, "Players"),
    el("p", { class: "page-sub" }, "Playtime reflects only sessions observed while Bonghos was running."),
    el("table", {},
      el("thead", {}, el("tr", {},
        el("th", {}, "Player"), el("th", {}, "Status"), el("th", {}, "Last seen"),
        el("th", {}, "Observed playtime"), el("th", {}, ""))),
      el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "5", class: "muted" }, "No players seen yet.")))));
}

function playerActions(p) {
  const act = (action, needsReason) => () => {
    const reason = el("input", { placeholder: "Reason (optional)" });
    modal(`${action} ${p.username}`,
      needsReason ? [el("div", { class: "field-row" }, reason)] : [el("p", {}, `Confirm ${action} for ${p.username}?`)],
      [["Cancel", "ghost", (c) => c()],
       [action, "danger", async (c) => {
         c();
         try {
           await api("/players/action", { method: "POST", json: { action: action.toLowerCase(), username: p.username, reason: reason.value || "" } });
           toast(`${action} sent for ${p.username}`, "ok");
         } catch (e) { toast(e.message, "err"); }
       }]]);
  };
  const wrap = el("div", { class: "row-actions" });
  if (p.online) wrap.append(el("button", { class: "btn ghost", onclick: act("Kick", true) }, "Kick"));
  wrap.append(
    el("button", { class: "btn ghost", onclick: act("Ban", true) }, "Ban"),
    el("button", { class: "btn ghost", onclick: act("Op", false) }, "Op"),
    el("button", { class: "btn ghost", onclick: act("Deop", false) }, "Deop"));
  return wrap;
}

// ----- files ----------------------------------------------------------------
let filePath = "";
async function pageFiles(main, path = filePath) {
  filePath = path;
  const entries = await api("/files?path=" + encodeURIComponent(path));
  main.innerHTML = "";
  const crumbs = el("div", { class: "breadcrumb" },
    el("span", { onclick: () => pageFiles(main, "") }, "root"));
  let acc = "";
  for (const part of path.split("/").filter(Boolean)) {
    acc += (acc ? "/" : "") + part;
    const p = acc;
    crumbs.append(" / ", el("span", { onclick: () => pageFiles(main, p) }, part));
  }
  const upInput = el("input", { type: "file", multiple: true, style: "display:none" });
  upInput.addEventListener("change", async () => {
    const fd = new FormData();
    for (const f of upInput.files) fd.append("file", f);
    try {
      await fetch("/api/files/upload?path=" + encodeURIComponent(path),
        { method: "POST", body: fd, headers: { "X-Bonghos-CSRF": csrfToken }, credentials: "same-origin" });
      toast("Uploaded", "ok"); pageFiles(main, path);
    } catch (e) { toast(e.message, "err"); }
  });
  const rows = (entries || []).map((e2) => el("tr", {},
    el("td", { class: "mono", style: "cursor:pointer", onclick: () => {
      if (e2.is_dir) pageFiles(main, (path ? path + "/" : "") + e2.name);
      else openFileEditor(main, (path ? path + "/" : "") + e2.name);
    } }, (e2.is_dir ? "📁 " : "📄 ") + e2.name),
    el("td", {}, e2.is_dir ? "—" : fmtBytes(e2.size)),
    el("td", {}, fmtTime(e2.mod_time)),
    el("td", { class: "row-actions" },
      !e2.is_dir ? el("a", { class: "btn ghost", href: "/api/files/download?path=" + encodeURIComponent((path ? path + "/" : "") + e2.name) }, "Download") : "",
      el("button", { class: "btn ghost", onclick: () => renameEntry(main, path, e2.name) }, "Rename"),
      el("button", { class: "btn danger", onclick: () => deleteEntry(main, path, e2.name) }, "Delete"))));
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Files"), el("div", { class: "spacer" }),
      el("button", { class: "btn", onclick: () => upInput.click() }, "Upload"),
      el("button", { class: "btn", onclick: () => mkdirPrompt(main, path) }, "New folder"),
      upInput),
    crumbs,
    el("div", { class: "file-list" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", {}, "Size"), el("th", {}, "Modified"), el("th", {}, ""))),
        el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "4", class: "muted" }, "Empty directory"))))));
}

async function openFileEditor(main, rel) {
  let data;
  try { data = await api("/files/content?path=" + encodeURIComponent(rel)); }
  catch (e) { toast(e.message, "err"); return; }
  main.innerHTML = "";
  const ta = el("textarea", { class: "editor", spellcheck: "false" });
  ta.value = data.content;
  main.append(
    el("div", { class: "toolbar" },
      el("h1", { class: "mono", style: "font-size:1rem" }, rel),
      el("div", { class: "spacer" }),
      el("button", { class: "btn ghost", onclick: () => pageFiles(main) }, "Back"),
      el("button", { class: "btn primary", onclick: async () => {
        try {
          await api("/files/content", { method: "POST", json: { path: rel, content: ta.value } });
          toast("Saved (a .bonghos-backup copy of important files is kept)", "ok");
        } catch (e) { toast(e.message, "err"); }
      } }, "Save")),
    ta);
}

function mkdirPrompt(main, path) {
  const inp = el("input", { placeholder: "folder-name" });
  modal("New folder", [el("div", { class: "field-row" }, inp)], [
    ["Cancel", "ghost", (c) => c()],
    ["Create", "primary", async (c) => {
      c();
      try { await api("/files/mkdir", { method: "POST", json: { path: (path ? path + "/" : "") + inp.value } }); pageFiles(main, path); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function renameEntry(main, path, name) {
  const inp = el("input", { value: name });
  modal("Rename", [el("div", { class: "field-row" }, inp)], [
    ["Cancel", "ghost", (c) => c()],
    ["Rename", "primary", async (c) => {
      c();
      const from = (path ? path + "/" : "") + name;
      const to = (path ? path + "/" : "") + inp.value;
      try { await api("/files/rename", { method: "POST", json: { from, to } }); pageFiles(main, path); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function deleteEntry(main, path, name) {
  const rel = (path ? path + "/" : "") + name;
  confirmModal("Delete", `Delete ${rel}? This cannot be undone.`, "Delete", async () => {
    try { await api("/files/delete", { method: "POST", json: { path: rel, confirm: true } }); pageFiles(main, path); }
    catch (e) { toast(e.message, "err"); }
  });
}

// ----- configuration --------------------------------------------------------
async function pageConfiguration(main) {
  const d = await api("/configuration");
  const inst = d.instance;
  main.innerHTML = "";

  const xms = el("input", { value: (d.jvm && d.jvm.xms) || inst.jvm_xms || "" , placeholder: "e.g. 2G" });
  const xmx = el("input", { value: (d.jvm && d.jvm.xmx) || inst.jvm_xmx || "" , placeholder: "e.g. 6G" });
  const scriptSel = el("select", {},
    ...(d.scripts || []).map((s) => el("option", { value: s.path, selected: s.path === inst.startup_script ? "" : null },
      `${s.path} (${s.modloader || "unknown"}, score ${s.score})`)));
  const javaSel = el("select", {},
    el("option", { value: "auto", selected: inst.java_selection === "auto" || !inst.java_selection ? "" : null }, "Automatic"),
    ...(d.java || []).map((j) => el("option", { value: j.path, selected: inst.java_selection === j.path ? "" : null },
      `${j.version} — ${j.path}`)));

  const props = d.properties || {};
  const commonProps = ["motd", "server-port", "max-players", "difficulty", "gamemode", "white-list", "pvp", "view-distance", "online-mode"];
  const propRows = commonProps.filter((k) => k in props).map((k) => {
    const v = el("input", { value: props[k] });
    return el("div", { class: "field-row" },
      el("label", {}, k, v),
      el("button", { class: "btn ghost", style: "align-self:flex-start", onclick: async () => {
        try {
          await api("/configuration/property", { method: "POST", json: { key: k, value: v.value } });
          toast(k + " saved — restart required to apply", "ok");
        } catch (e) { toast(e.message, "err"); }
      } }, "Save"));
  });

  main.append(
    el("h1", {}, "Configuration"),
    el("p", { class: "page-sub" }, "Changes to memory, properties and scripts take effect after a restart."),
    d.eula ? null : el("div", { class: "notice" },
      "The Minecraft EULA has not been accepted for this project. The server will not start until it is. ",
      el("button", { class: "btn", style: "margin-left:8px", onclick: () =>
        confirmModal("Accept Minecraft EULA",
          "By accepting you agree to the Minecraft End User License Agreement (https://aka.ms/MinecraftEULA). Bonghos never accepts it silently on your behalf.",
          "I accept the EULA", async () => {
            try { await api("/configuration/eula", { method: "POST", json: { accept: true } }); toast("EULA accepted", "ok"); renderPage(); }
            catch (e) { toast(e.message, "err"); }
          }, false) }, "Review & accept")),
    el("div", { class: "grid cols-2" },
      el("div", { class: "card" },
        el("h3", {}, "JVM memory"),
        d.jvm ? el("p", { class: "muted" }, `Detected in ${d.jvm.source_file} (${d.jvm.source_kind})${d.jvm.editable ? "" : " — not safely editable here"}`) : el("p", { class: "muted" }, "No JVM configuration detected."),
        el("div", { class: "field-row" }, el("label", {}, "Minimum (-Xms)", xms)),
        el("div", { class: "field-row" }, el("label", {}, "Maximum (-Xmx)", xmx)),
        el("button", { class: "btn primary", onclick: async () => {
          try {
            await api("/configuration/jvm", { method: "POST", json: { xms: xms.value, xmx: xmx.value } });
            toast("Memory saved — restart required to apply", "ok");
          } catch (e) { toast(e.message, "err"); }
        } }, "Save memory")),
      el("div", { class: "card" },
        el("h3", {}, "Startup"),
        el("div", { class: "field-row" }, el("label", {}, "Startup script", scriptSel)),
        el("button", { class: "btn", onclick: async () => {
          try { await api("/configuration/startup-script", { method: "POST", json: { script: scriptSel.value } }); toast("Startup script saved", "ok"); }
          catch (e) { toast(e.message, "err"); }
        } }, "Save script"),
        el("div", { class: "field-row", style: "margin-top:14px" }, el("label", {}, "Java installation", javaSel)),
        el("button", { class: "btn", onclick: async () => {
          try { await api(`/servers/${inst.id}`, { method: "PATCH", json: { java_selection: javaSel.value } }); toast("Java selection saved", "ok"); }
          catch (e) { toast(e.message, "err"); }
        } }, "Save Java"))),
    el("h2", {}, "server.properties"),
    el("div", { class: "card" }, propRows.length ? propRows : el("p", { class: "muted" }, "No server.properties found yet (it is created on first start).")),
    el("h2", {}, "Automation"),
    autoCard(inst));
}

function autoCard(inst) {
  const auto = el("input", { type: "checkbox" }); auto.checked = !!inst.autostart_enabled;
  const recover = el("input", { type: "checkbox" }); recover.checked = !!inst.recover_after_unclean_shutdown;
  const delay = el("input", { type: "number", min: "0", value: inst.boot_delay_seconds || 0, style: "width:110px" });
  const policy = el("select", {},
    ...["never", "on-failure", "always"].map((p) => el("option", { value: p, selected: p === (inst.restart_policy || "never") ? "" : null }, p)));
  return el("div", { class: "card" },
    el("div", { class: "field-row" }, el("label", { style: "flex-direction:row; align-items:center; gap:10px" }, auto, " Start this server when the machine boots")),
    el("div", { class: "field-row" }, el("label", { style: "flex-direction:row; align-items:center; gap:10px" }, recover, " Recover after unclean shutdown (power loss)")),
    el("div", { class: "field-row" }, el("label", {}, "Boot delay (seconds)", delay)),
    el("div", { class: "field-row" }, el("label", {}, "Crash restart policy", policy),
      el("span", { class: "hint" }, "Crash-loop protection pauses automatic restarts after repeated rapid crashes.")),
    el("button", { class: "btn primary", onclick: async () => {
      try {
        await api(`/servers/${inst.id}`, { method: "PATCH", json: {
          autostart_enabled: auto.checked, recover_after_unclean_shutdown: recover.checked,
          boot_delay_seconds: Number(delay.value), restart_policy: policy.value } });
        toast("Automation settings saved", "ok");
      } catch (e) { toast(e.message, "err"); }
    } }, "Save automation"));
}

// ----- backups --------------------------------------------------------------
async function pageBackups(main) {
  const list = await api("/backups");
  main.innerHTML = "";
  const mkBtn = (type, label) => el("button", { class: "btn", onclick: async () => {
    try { await api("/backups", { method: "POST", json: { type } }); toast(label + " backup started", "ok"); }
    catch (e) { toast(e.message, "err"); }
  } }, label);
  const rows = (list || []).map((b) => el("tr", {},
    el("td", { class: "mono" }, b.backup_id),
    el("td", {}, b.backup_type.replace(/_/g, " ")),
    el("td", {}, b.consistency_mode + " / " + b.trigger_type),
    el("td", {}, fmtBytes(b.compressed_size)),
    el("td", {}, b.verification_status || "—"),
    el("td", {}, fmtTime(b.created_at)),
    el("td", { class: "row-actions" },
      can("server.backups.restore") ? el("button", { class: "btn ghost", onclick: () => restoreBackup(b) }, "Restore") : "",
      el("button", { class: "btn ghost", onclick: async () => {
        try { await api(`/backups/${b.backup_id}/verify`, { method: "POST", json: {} }); toast("Verified", "ok"); renderPage(); }
        catch (e) { toast(e.message, "err"); }
      } }, "Verify"),
      el("button", { class: "btn ghost", onclick: async () => {
        try { await api(`/backups/${b.backup_id}/protect`, { method: "POST", json: { protected: !b.protected } }); renderPage(); }
        catch (e) { toast(e.message, "err"); }
      } }, b.protected ? "Unprotect" : "Protect"),
      el("button", { class: "btn danger", onclick: () =>
        confirmModal("Delete backup", `Delete backup ${b.backup_id}?` + (b.protected ? " It is PROTECTED." : ""), "Delete", async () => {
          try { await api(`/backups/${b.backup_id}`, { method: "DELETE" }); renderPage(); }
          catch (e) { toast(e.message, "err"); }
        }) }, "Delete"))));
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Backups"), el("div", { class: "spacer" }),
      can("server.backups.create") ? [mkBtn("world", "World backup"), mkBtn("full", "Full backup"), mkBtn("configuration", "Config backup")] : []),
    el("p", { class: "page-sub" }, "Online backups pause world saving briefly (save-off / save-all flush / save-on). Every backup is verified after creation."),
    el("div", { class: "progress hidden", id: "backup-progress" }, el("div", { style: "width:0%" })),
    el("table", {},
      el("thead", {}, el("tr", {}, el("th", {}, "ID"), el("th", {}, "Type"), el("th", {}, "Mode"), el("th", {}, "Size"), el("th", {}, "Verified"), el("th", {}, "Created"), el("th", {}, ""))),
      el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "7", class: "muted" }, "No backups yet.")))));
}

function updateBackupProgress(d) {
  const p = $("#backup-progress");
  if (!p) return;
  p.classList.remove("hidden");
  const bar = p.firstChild;
  if (d.total_bytes > 0) bar.style.width = Math.min(100, (d.bytes_processed / d.total_bytes) * 100) + "%";
  else p.classList.add("indeterminate");
}

function restoreBackup(b) {
  // Default to the narrowest scope the archive actually supports, so a
  // world-only backup never quietly triggers a full-server restore.
  const defaultScope =
    b.backup_type === "world_and_player_data" ? "world_only" :
    b.backup_type === "configuration_only" ? "configuration_only" : "full_server";

  const scopeSel = el("select", { class: "input" },
    el("option", { value: "full_server", selected: defaultScope === "full_server" }, "Full server — everything in the archive"),
    el("option", { value: "world_only", selected: defaultScope === "world_only" }, "World only — world, dimensions and player data"),
    el("option", { value: "configuration_only", selected: defaultScope === "configuration_only" }, "Configuration only — settings, mod config and scripts"));

  modal("Restore backup", [
    el("p", {}, `Restore ${b.backup_id} (${b.backup_type.replace(/_/g, " ")}) created ${fmtTime(b.created_at)}.`),
    el("label", { class: "field" }, el("span", {}, "Restore scope"), scopeSel),
    el("p", { class: "muted" },
      "The server must be stopped. Bonghos takes a verified emergency backup of the current state first — that backup is how you undo this, and it appears in the list below. Files in the selected scope are then replaced. If a world-only restore brings back a world under a different name, level-name is repointed at it so the server actually loads it."),
  ], [
    ["Cancel", "ghost", (c) => c()],
    ["Restore", "danger", async (c) => {
      c();
      toast("Creating emergency pre-restore backup…", "ok");
      try {
        const r = await api(`/backups/${b.backup_id}/restore`,
          { method: "POST", json: { scope: scopeSel.value, confirm: true } });
        let msg = "Restore complete (" + (r.scope || scopeSel.value).replace(/_/g, " ") + ")";
        if (r.level_name_updated) {
          msg += ` — level-name now points at “${r.world_name}” (was “${r.previous_level}”)`;
        }
        toast(msg, "ok");
        renderPage();
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

// ----- schedules ------------------------------------------------------------
async function pageSchedules(main) {
  const list = await api("/schedules");
  main.innerHTML = "";
  const rows = (list || []).map((s) => el("tr", {},
    el("td", {}, s.name, s.enabled ? "" : el("span", { class: "tag", style: "margin-left:8px" }, "disabled")),
    el("td", {}, s.action.replace(/_/g, " ")),
    el("td", { class: "mono" }, s.schedule_type + ": " + s.schedule_expression + " (" + (s.timezone || "UTC") + ")"),
    el("td", {}, fmtTime(s.next_run_at)),
    el("td", {}, s.last_result || "—"),
    el("td", { class: "row-actions" },
      can("server.schedules.manage") ? [
        el("button", { class: "btn ghost", onclick: async () => {
          try { await api(`/schedules/${s.id}/run`, { method: "POST", json: {} }); toast("Running now", "ok"); }
          catch (e) { toast(e.message, "err"); }
        } }, "Run now"),
        el("button", { class: "btn ghost", onclick: () => scheduleForm(s) }, "Edit"),
        el("button", { class: "btn danger", onclick: () =>
          confirmModal("Delete schedule", `Delete schedule "${s.name}"?`, "Delete", async () => {
            try { await api(`/schedules/${s.id}`, { method: "DELETE" }); renderPage(); } catch (e) { toast(e.message, "err"); }
          }) }, "Delete")] : "")));
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Schedules"), el("div", { class: "spacer" }),
      can("server.schedules.manage") ? el("button", { class: "btn primary", onclick: () => scheduleForm(null) }, "New schedule") : ""),
    el("p", { class: "page-sub" }, "Schedules run on the Linux host even when this browser is closed. Missed runs while the machine was off follow each schedule's missed-run policy."),
    el("table", {},
      el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", {}, "Action"), el("th", {}, "When"), el("th", {}, "Next run"), el("th", {}, "Last result"), el("th", {}, ""))),
      el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "6", class: "muted" }, "No schedules yet.")))));
}

function scheduleForm(s) {
  const name = el("input", { value: s ? s.name : "" });
  const type = el("select", {},
    ...[["daily", "Daily at a time"], ["weekly", "Weekly"], ["interval", "Every N minutes"], ["cron", "Cron expression"]]
      .map(([v, l]) => el("option", { value: v, selected: s && s.schedule_type === v ? "" : null }, l)));
  const expr = el("input", { value: s ? s.schedule_expression : "04:00", placeholder: "e.g. 04:00 · sun 04:00 · 30 · 0 4 * * *" });
  const tz = el("input", { value: s ? s.timezone : Intl.DateTimeFormat().resolvedOptions().timeZone });
  const action = el("select", {},
    ...[["restart_server", "Restart server"], ["backup", "Create backup"], ["stop_server", "Stop server"],
        ["start_server", "Start server"], ["send_console_command", "Console command"], ["broadcast_message", "Broadcast message"], ["save_all", "Save all"]]
      .map(([v, l]) => el("option", { value: v, selected: s && s.action === v ? "" : null }, l)));
  const payload = el("input", { value: s && s.action_payload ? (typeof s.action_payload === "string" ? s.action_payload : JSON.stringify(s.action_payload)) : "", placeholder: 'command/message or {"type":"world"}' });
  const offline = el("select", {},
    ...[["skip", "Skip if server offline"], ["start_first", "Start server first"], ["run_anyway", "Run anyway"]]
      .map(([v, l]) => el("option", { value: v, selected: s && s.offline_policy === v ? "" : null }, l)));
  const missed = el("select", {},
    ...[["skip", "Skip missed runs"], ["run_once_on_boot", "Run once after boot"]]
      .map(([v, l]) => el("option", { value: v, selected: s && s.missed_run_policy === v ? "" : null }, l)));
  const enabled = el("input", { type: "checkbox" }); enabled.checked = s ? !!s.enabled : true;

  modal(s ? "Edit schedule" : "New schedule", [
    el("div", { class: "field-row" }, el("label", {}, "Name", name)),
    el("div", { class: "field-row" }, el("label", {}, "Type", type)),
    el("div", { class: "field-row" }, el("label", {}, "Expression", expr),
      el("span", { class: "hint" }, "daily: HH:MM · weekly: day HH:MM · interval: minutes · cron: five fields")),
    el("div", { class: "field-row" }, el("label", {}, "Timezone", tz)),
    el("div", { class: "field-row" }, el("label", {}, "Action", action)),
    el("div", { class: "field-row" }, el("label", {}, "Payload", payload)),
    el("div", { class: "field-row" }, el("label", {}, "If server is offline", offline)),
    el("div", { class: "field-row" }, el("label", {}, "Missed while machine was off", missed)),
    el("div", { class: "field-row" }, el("label", { style: "flex-direction:row; gap:10px; align-items:center" }, enabled, " Enabled")),
  ], [
    ["Cancel", "ghost", (c) => c()],
    ["Save", "primary", async (c) => {
      let ap = payload.value.trim();
      if (action.value === "backup" && ap && !ap.startsWith("{")) ap = JSON.stringify({ type: ap });
      if (ap && !ap.startsWith("{") && !ap.startsWith("\"")) ap = JSON.stringify(ap);
      const body = {
        name: name.value, schedule_type: type.value, schedule_expression: expr.value,
        timezone: tz.value, action: action.value,
        action_payload: ap ? JSON.parse(ap) : null,
        offline_policy: offline.value, missed_run_policy: missed.value,
        conflict_policy: "wait", enabled: enabled.checked, description: "",
      };
      try {
        if (s) await api(`/schedules/${s.id}`, { method: "PATCH", json: body });
        else await api("/schedules", { method: "POST", json: body });
        c(); renderPage();
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

// ----- performance ----------------------------------------------------------
async function pagePerformance(main) {
  const hist = await api("/metrics?hours=1");
  S.perf = hist || [];
  main.innerHTML = "";
  main.append(
    el("h1", {}, "Performance"),
    el("p", { class: "page-sub" }, "Process memory shown is the Java process resident set, not the Java heap."),
    el("div", { class: "grid cols-2" },
      el("div", { class: "card" }, el("h3", {}, "CPU (% of one core)"), el("div", { class: "chart", id: "chart-cpu" })),
      el("div", { class: "card" }, el("h3", {}, "Process memory"), el("div", { class: "chart", id: "chart-mem" })),
      el("div", { class: "card" }, el("h3", {}, "Players online"), el("div", { class: "chart", id: "chart-players" })),
      el("div", { class: "card" }, el("h3", {}, "Disk free"), el("div", { class: "chart", id: "chart-disk" }))));
  drawCharts();
}

function updateLiveStats() { if (S.page === "performance") drawCharts(); }

function drawCharts() {
  sparkline("#chart-cpu", S.perf.map((s) => s.cpu_percent || 0));
  sparkline("#chart-mem", S.perf.map((s) => s.rss_bytes || 0), fmtBytes);
  sparkline("#chart-players", S.perf.map((s) => s.online_players || 0));
  sparkline("#chart-disk", S.perf.map((s) => s.disk_free || 0), fmtBytes);
}

function sparkline(sel, values, fmt = (v) => v.toFixed(1)) {
  const host = $(sel);
  if (!host) return;
  const W = 520, H = 160, pad = 6;
  if (!values.length) { host.innerHTML = '<div class="muted">No data yet.</div>'; return; }
  const max = Math.max(...values, 1), min = Math.min(...values, 0);
  const pts = values.map((v, i) => {
    const x = pad + (i / Math.max(values.length - 1, 1)) * (W - 2 * pad);
    const y = H - pad - ((v - min) / (max - min || 1)) * (H - 2 * pad);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
  host.innerHTML = `<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none">
    <polyline points="${pts}" fill="none" stroke="var(--accent-2)" stroke-width="2"/>
  </svg><div class="muted">now: ${fmt(values[values.length - 1])} · peak: ${fmt(max)}</div>`;
}

// ----- servers (projects + import) ------------------------------------------
async function pageServers(main) {
  await refreshServers();
  const ops = await api("/operations?active=true").catch(() => []);
  main.innerHTML = "";
  const cards = S.servers.map((s2) => el("div", { class: "card" },
    el("div", { class: "toolbar", style: "margin-bottom:8px" },
      el("strong", {}, s2.display_name),
      s2.id === S.activeId ? el("span", { class: "tag" }, "active") : "",
      s2.external_directory ? el("span", { class: "tag" }, "external link") : "",
      el("div", { class: "spacer" })),
    el("div", { class: "muted mono" }, s2.slug),
    el("div", { class: "muted" }, (s2.modloader || "unknown modloader") + " · imported via " + s2.source_type),
    el("div", { class: "row-actions", style: "margin-top:12px" },
      s2.id !== S.activeId && can("server.configuration.manage")
        ? el("button", { class: "btn", onclick: async () => {
            try { await api(`/servers/${s2.id}/select`, { method: "POST", json: {} }); toast("Active project changed", "ok"); refreshServers().then(renderPage); }
            catch (e) { toast(e.message, "err"); }
          } }, "Make active") : "",
      can("server.import.manage") ? el("button", { class: "btn danger", onclick: () => deleteProject(s2) }, "Delete") : "")));
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Servers"), el("div", { class: "spacer" }),
      can("server.import.manage") ? el("button", { class: "btn primary", onclick: importWizard }, "Import server") : ""),
    el("p", { class: "page-sub" }, "One server runs at a time; all imported projects stay ready to select."),
    el("div", { id: "ops-host" }, ...(ops || []).map(opCard)),
    el("div", { class: "grid cols-2" }, cards.length ? cards : el("p", { class: "muted" }, "No servers imported yet — use “Import server”.")));
}

function opCard(op) {
  const pct = op.total_bytes > 0 ? Math.min(100, (op.bytes_processed / op.total_bytes) * 100) : null;
  return el("div", { class: "card", "data-op": op.id, style: "margin-bottom:14px" },
    el("div", { class: "toolbar", style: "margin-bottom:8px" },
      el("strong", {}, op.kind), el("span", { class: "tag" }, op.stage.replace(/_/g, " ")),
      el("div", { class: "spacer" }),
      ["completed", "failed", "cancelled"].includes(op.stage) ? "" :
        el("button", { class: "btn ghost", onclick: async () => {
          try { await api(`/operations/${op.id}/cancel`, { method: "POST", json: {} }); } catch (e) { toast(e.message, "err"); }
        } }, "Cancel")),
    el("div", { class: "progress" + (pct === null ? " indeterminate" : "") },
      el("div", { style: `width:${pct === null ? 35 : pct}%` })),
    el("div", { class: "muted", style: "margin-top:6px" },
      op.message || "", pct !== null ? ` ${fmtBytes(op.bytes_processed)} / ${fmtBytes(op.total_bytes)}` : ` ${fmtBytes(op.bytes_processed || 0)}`),
    op.error ? el("div", { class: "error" }, op.error) : "");
}

function updateOperation(op, type) {
  if (type === "installed") { toast("Server installed: " + (op.display_name || op.slug), "ok"); refreshServers().then(() => { if (S.page === "servers") renderPage(); }); return; }
  const host = $("#ops-host");
  if (!host) return;
  const existing = host.querySelector(`[data-op="${op.id}"]`);
  const fresh = opCard(op);
  if (existing) existing.replaceWith(fresh); else host.prepend(fresh);
  if (["completed", "failed", "cancelled"].includes(op.stage)) setTimeout(() => fresh.remove(), 8000);
}

function deleteProject(s2) {
  const delFiles = el("input", { type: "checkbox" });
  modal("Delete project", [
    el("p", {}, `Delete "${s2.display_name}"? Backups are kept unless removed separately.`),
    el("label", { style: "flex-direction:row; gap:10px; align-items:center" }, delFiles, " Also delete the server files on disk"),
  ], [
    ["Cancel", "ghost", (c) => c()],
    ["Delete", "danger", async (c) => {
      c();
      try {
        await api(`/servers/${s2.id}?delete_files=${delFiles.checked}`, { method: "DELETE" });
        toast("Project deleted", "ok"); refreshServers().then(renderPage);
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

// uploadArchive streams the file with XMLHttpRequest rather than fetch,
// because only XHR reports upload progress and can be aborted mid-transfer.
// The browser streams the body; the whole archive is never held in memory.
function uploadArchive(file, displayName) {
  const host = $("#ops-host");
  const started = Date.now();
  let lastTime = started, lastLoaded = 0, instantRate = 0;

  const bar = el("div", { style: "width:0%" });
  const line = el("div", { class: "muted", style: "margin-top:6px" }, "Preparing upload…");
  const xhr = new XMLHttpRequest();
  const cancelBtn = el("button", { class: "btn ghost", onclick: () => xhr.abort() }, "Cancel");
  const card = el("div", { class: "card", style: "margin-bottom:14px" },
    el("div", { class: "toolbar", style: "margin-bottom:8px" },
      el("strong", {}, "Uploading " + file.name),
      el("span", { class: "tag" }, "browser → host"),
      el("div", { class: "spacer" }), cancelBtn),
    el("div", { class: "progress" }, bar), line);
  if (host) host.prepend(card); else toast("Upload started", "ok");

  xhr.upload.addEventListener("progress", (e) => {
    if (!e.lengthComputable) {
      line.textContent = `${fmtBytes(e.loaded)} sent`;
      return;
    }
    const now = Date.now();
    const dt = (now - lastTime) / 1000;
    if (dt >= 0.5) {
      instantRate = (e.loaded - lastLoaded) / dt;
      lastTime = now; lastLoaded = e.loaded;
    }
    const avg = e.loaded / Math.max(0.001, (now - started) / 1000);
    const pct = (e.loaded / e.total) * 100;
    bar.style.width = pct.toFixed(1) + "%";
    const remaining = instantRate > 0 ? (e.total - e.loaded) / instantRate : null;
    line.textContent =
      `${fmtBytes(e.loaded)} / ${fmtBytes(e.total)} · ${pct.toFixed(0)}%` +
      ` · ${fmtBytes(instantRate || avg)}/s` +
      (remaining !== null && isFinite(remaining) ? ` · about ${fmtDur(remaining)} remaining` : "");
  });

  xhr.addEventListener("load", () => {
    let d = {}; try { d = JSON.parse(xhr.responseText); } catch {}
    card.remove();
    if (xhr.status >= 200 && xhr.status < 300) {
      toast("Upload complete — the host is now extracting and installing it", "ok");
    } else {
      toast(d.error || `Upload failed (HTTP ${xhr.status})`, "err");
    }
  });
  xhr.addEventListener("error", () => { card.remove(); toast("Upload failed: connection lost", "err"); });
  xhr.addEventListener("abort", () => { card.remove(); toast("Upload cancelled", "ok"); });

  const fd = new FormData();
  fd.append("display_name", displayName);
  fd.append("archive", file);
  xhr.open("POST", "/api/imports/upload");
  xhr.withCredentials = true;
  xhr.setRequestHeader("X-Bonghos-CSRF", csrfToken);
  xhr.send(fd);
}

function importWizard() {
  const name = el("input", { placeholder: "My Awesome Server" });
  const slugPrev = el("div", { class: "hint mono" }, "");
  name.addEventListener("input", async () => {
    if (!name.value.trim()) { slugPrev.textContent = ""; return; }
    try { const d = await api("/servers/slug-preview", { method: "POST", json: { name: name.value } });
      slugPrev.textContent = "directory: servers/minecraft-java/modded/" + d.slug; } catch {}
  });
  const method = el("select", {},
    el("option", { value: "upload" }, "Upload archive from this device"),
    el("option", { value: "url" }, "Download from URL (host downloads it)"),
    el("option", { value: "local" }, "Archive already on the Linux machine"),
    el("option", { value: "dir" }, "Existing server directory on the Linux machine"));
  const url = el("input", { placeholder: "https://…/server-pack.zip" });
  const localPath = el("input", { placeholder: "/home/user/Downloads/pack.tar.gz" });
  const dirPath = el("input", { placeholder: "/home/user/old-server" });
  const dirMode = el("select", {},
    el("option", { value: "copy" }, "Copy into Bonghos (source untouched)"),
    el("option", { value: "move" }, "Move into Bonghos"),
    el("option", { value: "link" }, "Link in place (advanced — excluded from Bonghos migration)"));
  const fileInput = el("input", { type: "file", accept: ".zip,.tar,.gz,.tgz,.xz,.zst,.7z,.rar", style: "display:none" });
  const chosen = el("div", { class: "hint mono" }, "");
  const dropzone = el("div", { class: "dropzone", tabindex: "0", role: "button",
    "aria-label": "Drag a server-pack archive here, or choose one" },
    el("p", {}, "Drag a server-pack archive here"),
    el("p", { class: "muted" }, "or"),
    el("button", { class: "btn ghost", type: "button", onclick: (e) => { e.preventDefault(); fileInput.click(); } }, "Choose Archive"),
    chosen);

  function setChosen(f) {
    if (!f) { chosen.textContent = ""; return; }
    chosen.textContent = `${f.name} — ${fmtBytes(f.size)}`;
    if (!name.value.trim()) {
      // Offer the archive name as a starting display name.
      name.value = f.name.replace(/\.(zip|tar|tgz|gz|xz|zst|7z|rar)$/i, "").replace(/[._]+/g, " ").trim();
      name.dispatchEvent(new Event("input"));
    }
  }
  fileInput.addEventListener("change", () => setChosen(fileInput.files[0]));
  dropzone.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") { e.preventDefault(); fileInput.click(); }
  });
  ["dragenter", "dragover"].forEach((ev) => dropzone.addEventListener(ev, (e) => {
    e.preventDefault(); e.stopPropagation(); dropzone.classList.add("dragover");
  }));
  ["dragleave", "drop"].forEach((ev) => dropzone.addEventListener(ev, (e) => {
    e.preventDefault(); e.stopPropagation(); dropzone.classList.remove("dragover");
  }));
  dropzone.addEventListener("drop", (e) => {
    const f = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
    if (!f) return;
    // DataTransfer files cannot be assigned to an input in every browser, so
    // keep the dropped file separately and prefer it when submitting.
    dropzone.droppedFile = f;
    setChosen(f);
  });
  const rowUpload = el("div", { class: "field-row" },
    el("label", {}, "Archive (.zip, .tar.gz, .tar.xz, .tar.zst, .7z, .rar)", dropzone), fileInput);
  const rowURL = el("div", { class: "field-row hidden" }, el("label", {}, "URL", url), el("span", { class: "hint" }, "HTTPS only by default. The download runs on the Linux host and continues if you close this page."));
  const rowLocal = el("div", { class: "field-row hidden" }, el("label", {}, "Absolute path to archive", localPath));
  const rowDir = el("div", { class: "field-row hidden" }, el("label", {}, "Absolute path to directory", dirPath), el("label", {}, "Mode", dirMode));
  method.addEventListener("change", () => {
    rowUpload.classList.toggle("hidden", method.value !== "upload");
    rowURL.classList.toggle("hidden", method.value !== "url");
    rowLocal.classList.toggle("hidden", method.value !== "local");
    rowDir.classList.toggle("hidden", method.value !== "dir");
  });

  modal("Import server", [
    el("div", { class: "field-row" }, el("label", {}, "Display name", name), slugPrev),
    el("div", { class: "field-row" }, el("label", {}, "Method", method)),
    rowUpload, rowURL, rowLocal, rowDir,
  ], [
    ["Cancel", "ghost", (c) => c()],
    ["Import", "primary", async (c) => {
      try {
        if (method.value === "upload") {
          const f = dropzone.droppedFile || fileInput.files[0];
          if (!f) throw new Error("Choose or drop an archive file first.");
          if (!name.value.trim()) throw new Error("Enter a display name for the project.");
          c(); navigate("servers");
          uploadArchive(f, name.value);
        } else if (method.value === "url") {
          await api("/imports/url", { method: "POST", json: { url: url.value, display_name: name.value } });
          c(); navigate("servers");
          toast("Download started on the host — you can close this page", "ok");
        } else if (method.value === "local") {
          await api("/imports/local-archive", { method: "POST", json: { path: localPath.value, display_name: name.value } });
          c(); navigate("servers");
        } else {
          if (dirMode.value === "link") {
            c();
            confirmModal("Link external directory",
              "Linked directories stay outside the Bonghos home: they are NOT included in exports or the normal Bonghos migration, and Bonghos will operate on files in place. Continue?",
              "Link it", async () => {
                await api("/imports/existing-directory", { method: "POST", json: { path: dirPath.value, display_name: name.value, mode: "link", confirm_link: true } });
                navigate("servers");
              });
            return;
          }
          await api("/imports/existing-directory", { method: "POST", json: { path: dirPath.value, display_name: name.value, mode: dirMode.value } });
          c(); navigate("servers");
        }
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

// ----- activity --------------------------------------------------------------
async function pageActivity(main) {
  const list = await api("/activity");
  main.innerHTML = "";
  main.append(
    el("h1", {}, "Activity"),
    el("p", { class: "page-sub" }, "Who did what, and when."),
    el("table", {},
      el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "User"), el("th", {}, "Action"), el("th", {}, "Target"), el("th", {}, "Detail"))),
      el("tbody", {}, (list || []).map((a2) => el("tr", {},
        el("td", {}, fmtTime(a2.at)), el("td", {}, a2.username), el("td", {}, a2.action.replace(/_/g, " ")),
        el("td", { class: "mono" }, a2.target || ""), el("td", { class: "muted" }, a2.detail || ""))))));
}

// ----- users -----------------------------------------------------------------
async function pageUsers(main) {
  const users = await api("/users");
  main.innerHTML = "";
  const rows = (users || []).map((u) => el("tr", {},
    el("td", {}, u.Username),
    el("td", {}, el("span", { class: "tag" }, u.Role)),
    el("td", {}, u.Disabled ? "Disabled" : "Active"),
    el("td", { class: "row-actions" },
      u.ID !== S.me.id ? [
        el("button", { class: "btn ghost", onclick: () => changeRole(u) }, "Role"),
        el("button", { class: "btn ghost", onclick: async () => {
          try { await api(`/users/${u.ID}/disable`, { method: "POST", json: { disabled: !u.Disabled } }); renderPage(); }
          catch (e) { toast(e.message, "err"); }
        } }, u.Disabled ? "Enable" : "Disable"),
        el("button", { class: "btn ghost", onclick: async () => {
          try { await api(`/users/${u.ID}/revoke-sessions`, { method: "POST", json: {} }); toast("Sessions revoked", "ok"); }
          catch (e) { toast(e.message, "err"); }
        } }, "Revoke sessions"),
        el("button", { class: "btn danger", onclick: () =>
          confirmModal("Delete user", `Delete account "${u.Username}"?`, "Delete", async () => {
            try { await api(`/users/${u.ID}`, { method: "DELETE" }); renderPage(); } catch (e) { toast(e.message, "err"); }
          }) }, "Delete"),
      ] : el("span", { class: "muted" }, "you"))));
  main.append(
    el("div", { class: "toolbar" }, el("h1", {}, "Users"), el("div", { class: "spacer" }),
      el("button", { class: "btn primary", onclick: inviteUser }, "Invite user")),
    el("table", {},
      el("thead", {}, el("tr", {}, el("th", {}, "Username"), el("th", {}, "Role"), el("th", {}, "Status"), el("th", {}, ""))),
      el("tbody", {}, rows)));
}

function inviteUser() {
  const role = el("select", {},
    ...["Admin", "Member", "Viewer"].map((r) => el("option", { value: r }, r)));
  modal("Invite user", [
    el("div", { class: "field-row" }, el("label", {}, "Role", role),
      el("span", { class: "hint" }, "The invitation link is valid for 72 hours and works once. The new user sets their own password and authenticator during activation.")),
  ], [
    ["Cancel", "ghost", (c) => c()],
    ["Create invitation", "primary", async (c) => {
      try {
        const d = await api("/users/invite", { method: "POST", json: { role: role.value } });
        c();
        const link = location.origin + d.activation_path;
        modal("Invitation created", [
          el("p", {}, "Share this one-time link with the new user:"),
          el("input", { value: link, readonly: "", onclick: (e) => e.target.select() }),
          el("p", { class: "muted" }, "Expires " + fmtTime(d.expires_at) + "."),
        ], [["Done", "primary", (c2) => c2()]]);
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

function changeRole(u) {
  const role = el("select", {},
    ...["Owner", "Admin", "Member", "Viewer"].map((r) => el("option", { value: r, selected: r === u.Role ? "" : null }, r)));
  modal("Change role", [el("div", { class: "field-row" }, el("label", {}, u.Username, role))], [
    ["Cancel", "ghost", (c) => c()],
    ["Save", "primary", async (c) => {
      c();
      try { await api(`/users/${u.ID}/role`, { method: "POST", json: { role: role.value } }); renderPage(); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

// ----- host ------------------------------------------------------------------
async function pageHost(main) {
  const d = await api("/host");
  main.innerHTML = "";
  main.append(
    el("h1", {}, "Host"),
    el("p", { class: "page-sub" }, d.note),
    el("div", { class: "grid cols-3" },
      statCard("Memory available", fmtBytes(d.mem_available), "of " + fmtBytes(d.mem_total)),
      statCard("Disk free", fmtBytes(d.disk_free), "of " + fmtBytes(d.disk_total)),
      statCard("Load (1m)", (d.load1 || 0).toFixed(2), "")),
    el("div", { class: "card", style: "margin-top:16px" },
      el("h3", {}, "Services & panel"),
      el("dl", { class: "kv" },
        el("dt", {}, "Panel address"), el("dd", { class: "mono" }, `${d.bind_address}:${d.port}`),
        el("dt", {}, "Bonghos home"), el("dd", { class: "mono" }, d.home),
        el("dt", {}, "systemd"), el("dd", {}, d.systemd ? "available" : "unavailable"),
        el("dt", {}, "bonghos.service"), el("dd", {}, d.service_bonghos || "—"),
        el("dt", {}, "bonghos-minecraft.service"), el("dd", {}, d.service_minecraft || "—"),
        el("dt", {}, "Version"), el("dd", {}, d.version))));
}

// ---------------------------------------------------------------------------
// activation page (invited users land on /activate/<token>)
// ---------------------------------------------------------------------------
async function activationFlow(token) {
  document.body.innerHTML = "";
  const wrap = el("div", { class: "login-wrap" });
  document.body.append(wrap, el("div", { id: "toast-host" }), el("div", { id: "modal-host" }));
  try {
    const info = await fetch(`/api/invitations/${token}`).then((r) => r.json());
    if (info.error) throw new Error(info.error);
    const user = el("input", { autocomplete: "username" });
    const p1 = el("input", { type: "password", autocomplete: "new-password" });
    const p2 = el("input", { type: "password", autocomplete: "new-password" });
    const code = el("input", { inputmode: "numeric", maxlength: "6" });
    const secretBox = el("div", { class: "muted mono", style: "word-break:break-all" });
    let secret = "";
    const genBtn = el("button", { class: "btn", type: "button", onclick: async () => {
      const csrf = await fetch("/api/auth/csrf").then((r) => r.json());
      const d = await fetch(`/api/invitations/${token}/totp`, { method: "POST",
        headers: { "Content-Type": "application/json", "X-Bonghos-CSRF": csrf.csrf },
        body: JSON.stringify({ username: user.value }) }).then((r) => r.json());
      secret = d.secret;
      secretBox.textContent = "Secret: " + d.secret + "\nURI: " + d.uri;
    } }, "Generate authenticator secret");
    const form = el("form", { class: "login-card", onsubmit: async (e) => {
      e.preventDefault();
      if (p1.value !== p2.value) return toast("Passwords do not match", "err");
      if (!secret) return toast("Generate the authenticator secret first", "err");
      const csrf = await fetch("/api/auth/csrf").then((r) => r.json());
      const res = await fetch(`/api/invitations/${token}/activate`, { method: "POST",
        headers: { "Content-Type": "application/json", "X-Bonghos-CSRF": csrf.csrf },
        body: JSON.stringify({ username: user.value, password: p1.value, totp_secret: secret, totp_code: code.value }) });
      const d = await res.json();
      if (!res.ok) return toast(d.error, "err");
      form.innerHTML = "";
      form.append(
        el("div", { class: "brand" }, "Account created"),
        el("p", {}, "Store these one-time recovery codes safely:"),
        el("pre", { class: "mono" }, (d.recovery_codes || []).join("\n")),
        el("a", { class: "btn primary", href: "/", style: "text-align:center" }, "Go to sign-in"));
    } },
      el("div", { class: "brand" }, "Activate your Bonghos account"),
      el("p", { class: "muted" }, `You are joining as ${info.role}.`),
      el("label", {}, "Username", user),
      el("label", {}, "Password (min 10 chars)", p1),
      el("label", {}, "Confirm password", p2),
      genBtn, secretBox,
      el("label", {}, "6-digit code from your authenticator", code),
      el("button", { class: "btn primary" }, "Activate"));
    wrap.append(form);
  } catch (e) {
    wrap.append(el("div", { class: "login-card" },
      el("div", { class: "brand" }, "Invitation"),
      el("p", { class: "error" }, e.message || "This invitation is invalid or expired.")));
  }
}

// ---------------------------------------------------------------------------
const activateMatch = location.pathname.match(/^\/activate\/([A-Za-z0-9_-]+)/);
if (activateMatch) activationFlow(activateMatch[1]);
else boot();
