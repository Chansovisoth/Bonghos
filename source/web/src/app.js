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
  const t = el("div", { class: "toast " + kind, role: "status" }, msg);
  $("#toast-host").append(t);
  setTimeout(() => t.remove(), 6000);
}

function modal(title, bodyNodes, actions) {
  const host = $("#modal-host");
  host.innerHTML = "";
  const close = () => (host.innerHTML = "");
  const m = el("div", { class: "overlay", onclick: (e) => { if (e.target === m) close(); } },
    el("div", { class: "modal", role: "dialog", "aria-modal": "true", "aria-label": title },
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
// theme
// ---------------------------------------------------------------------------
const THEME_KEY = "bonghos.theme";
const themeQuery = window.matchMedia("(prefers-color-scheme: dark)");

function themeChoice() {
  return localStorage.getItem(THEME_KEY) || "system";
}

function applyTheme(choice = themeChoice()) {
  const dark = choice === "dark" || (choice === "system" && themeQuery.matches);
  document.documentElement.dataset.theme = choice;
  document.documentElement.dataset.resolvedTheme = dark ? "dark" : "light";
}

function setTheme(choice) {
  if (choice === "system") localStorage.removeItem(THEME_KEY);
  else localStorage.setItem(THEME_KEY, choice);
  applyTheme(choice);
  document.querySelectorAll("[data-theme-choice]").forEach((b) =>
    b.classList.toggle("active", b.dataset.themeChoice === choice));
}

themeQuery.addEventListener("change", () => { if (themeChoice() === "system") applyTheme("system"); });
applyTheme();

// ---------------------------------------------------------------------------
// local demo mode
// ---------------------------------------------------------------------------
const DEMO_MODE = new URLSearchParams(location.search).has("demo");
const DEMO_PERMS = [
  "server.view", "server.start", "server.stop", "server.restart", "server.force_stop",
  "server.console.view", "server.console.use", "server.players.view", "server.players.manage",
  "server.files.manage", "server.configuration.manage", "server.icon.manage",
  "server.import.manage", "server.backups.view", "server.backups.create",
  "server.backups.restore", "server.schedules.manage", "users.manage",
  "security.manage", "host.manage", "portability.manage",
];
const DEMO_ME = { id: 1, username: "demo-owner", role: "owner", permissions: DEMO_PERMS };
const DEMO_SERVERS = [
  { id: 1, slug: "bio1", display_name: "Bio1 Survival - Long Local Demo Server Name", modloader: "forge", source_type: "direct-url", startup_script: "run.sh", restart_policy: "on-failure", autostart_enabled: true },
  { id: 2, slug: "creative-lab", display_name: "Creative Lab", modloader: "fabric", source_type: "archive-upload", external_directory: false },
];
const DEMO_CONSOLE = [
  "[19:27:36] [Server thread/INFO]: Starting minecraft server version 1.20.1",
  "[19:27:43] [Server thread/INFO]: Loading Forge mods from /home/klaude/bonghos/servers/minecraft-java/modded/bio1/mods",
  "[19:28:12] [Server thread/WARN]: Can't keep up! Is the server overloaded? Running 2475ms behind",
  "[19:28:40] [Server thread/INFO]: Steve joined the game",
  "[19:29:04] [Server thread/ERROR]: Example datapack warning for visual review only",
];
const DEMO_METRICS = Array.from({ length: 60 }, (_, i) => ({
  at: new Date(Date.now() - (59 - i) * 60000).toISOString(),
  cpu_percent: 18 + Math.sin(i / 6) * 11 + (i % 13),
  rss_bytes: (2600 + Math.sin(i / 8) * 220 + i * 3) * 1024 * 1024,
  online_players: i % 17 > 9 ? 4 : 3,
  disk_free: (186 - i * 0.08) * 1024 * 1024 * 1024,
}));

function demoDelay() {
  return new Promise((resolve) => setTimeout(resolve, 80));
}

async function demoApi(path, opts = {}) {
  await demoDelay();
  const method = opts.method || "GET";
  const clean = path.split("?")[0];
  if (method !== "GET") {
    if (clean === "/server/start") { S.status = { state: "running" }; return { ok: true }; }
    if (clean === "/server/stop") { S.status = { state: "stopped" }; return { ok: true }; }
    if (clean === "/server/restart") { S.status = { state: "restarting" }; return { ok: true }; }
    if (clean === "/server/command") {
      S.consoleLines.push("> " + ((opts.json && opts.json.command) || ""));
      return { ok: true };
    }
    if (clean === "/servers/slug-preview") {
      const name = (opts.json && opts.json.name) || "server";
      return { slug: name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "server" };
    }
    return { ok: true };
  }
  switch (clean) {
    case "/auth/csrf": return { csrf: "demo-csrf-token" };
    case "/auth/me": return DEMO_ME;
    case "/version": return { version: "0.1.1-demo" };
    case "/servers": return { servers: DEMO_SERVERS, active_id: 1 };
    case "/server/status": return S.status;
    case "/overview": return {
      state: S.status.state,
      version: "0.1.1-demo",
      instance: DEMO_SERVERS[0],
      motd: "A precise Bonghos local demo",
      port: "25565",
      max_players: "20",
      last_backup: { created_at: new Date(Date.now() - 5 * 3600000).toISOString() },
      next_schedule_at: new Date(Date.now() + 3 * 3600000).toISOString(),
      sample: DEMO_METRICS[DEMO_METRICS.length - 1],
    };
    case "/host": return {
      bind_address: "127.0.0.1", port: 8080, home: "/home/demo/bonghos",
      mem_total: 32 * 1024 * 1024 * 1024, mem_available: 18 * 1024 * 1024 * 1024,
      disk_total: 512 * 1024 * 1024 * 1024, disk_free: 186 * 1024 * 1024 * 1024,
      load1: 0.82, systemd: true, service_bonghos: "active", service_minecraft: "running",
      version: "0.1.1-demo",
      note: "Demo data only. Local listening does not prove public accessibility.",
    };
    case "/events": return { events: [
      { occurred_at: new Date(Date.now() - 9 * 60000).toISOString(), severity: "info", message: "Server reached ready state" },
      { occurred_at: new Date(Date.now() - 22 * 60000).toISOString(), severity: "warning", message: "Backup verification took longer than usual" },
      { occurred_at: new Date(Date.now() - 48 * 60000).toISOString(), severity: "info", message: "Scheduled restart completed" },
    ] };
    case "/metrics": return DEMO_METRICS;
    case "/players": return { players: [
      { username: "Steve", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 7342 },
      { username: "Alex", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 3922 },
      { username: "Long_Name_With_Underscores", online: false, last_seen_at: new Date(Date.now() - 86400000).toISOString(), observed_playtime_seconds: 18422 },
    ] };
    case "/files": return [
      { name: "world", is_dir: true, size: 0, mod_time: new Date(Date.now() - 3600000).toISOString() },
      { name: "server.properties", is_dir: false, size: 914, mod_time: new Date(Date.now() - 7200000).toISOString() },
      { name: "user_jvm_args.txt", is_dir: false, size: 72, mod_time: new Date(Date.now() - 5400000).toISOString() },
    ];
    case "/files/content": return { content: "motd=A precise Bonghos local demo\nserver-port=25565\nmax-players=20\n" };
    case "/configuration": return {
      eula: true,
      instance: DEMO_SERVERS[0],
      jvm: { xms: "2G", xmx: "6G", source_file: "user_jvm_args.txt", source_kind: "jvm_args_file", editable: true },
      scripts: [{ path: "run.sh", modloader: "forge", score: 98 }],
      java: [{ path: "/usr/lib/jvm/java-21-openjdk/bin/java", version: "21" }],
      properties: { motd: "A precise Bonghos local demo", "server-port": "25565", "max-players": "20", difficulty: "normal", gamemode: "survival", "white-list": "false", pvp: "true", "view-distance": "10", "online-mode": "true" },
    };
    case "/backups": return [
      { backup_id: "demo-full-20260803-1900", backup_type: "full_server", consistency_mode: "online", trigger_type: "manual", compressed_size: 4620000000, verification_status: "verified", created_at: new Date(Date.now() - 5 * 3600000).toISOString(), protected: true },
      { backup_id: "demo-world-20260802-0400", backup_type: "world_and_player_data", consistency_mode: "offline", trigger_type: "schedule", compressed_size: 2110000000, verification_status: "verified", created_at: new Date(Date.now() - 29 * 3600000).toISOString(), protected: false },
    ];
    case "/schedules": return [
      { id: 1, name: "Nightly verified backup", enabled: true, action: "backup", schedule_type: "daily", schedule_expression: "04:00", timezone: "Asia/Phnom_Penh", next_run_at: new Date(Date.now() + 3 * 3600000).toISOString(), last_result: "success" },
      { id: 2, name: "Weekly restart", enabled: false, action: "restart_server", schedule_type: "weekly", schedule_expression: "sun 05:00", timezone: "Asia/Phnom_Penh", next_run_at: null, last_result: "skipped" },
    ];
    case "/operations": return [];
    case "/activity": return [
      { at: new Date(Date.now() - 12 * 60000).toISOString(), username: "demo-owner", action: "backup_created", target: "bio1", detail: "full_server verified" },
      { at: new Date(Date.now() - 46 * 60000).toISOString(), username: "demo-owner", action: "configuration_saved", target: "user_jvm_args.txt", detail: "-Xmx changed to 6G" },
    ];
    case "/users": return [
      { ID: 1, Username: "demo-owner", Role: "owner", Disabled: false },
      { ID: 2, Username: "admin", Role: "admin", Disabled: false },
      { ID: 3, Username: "viewer", Role: "viewer", Disabled: true },
    ];
    default: return {};
  }
}

// ---------------------------------------------------------------------------
// API
// ---------------------------------------------------------------------------
let csrfToken = "";
async function api(path, opts = {}) {
  if (DEMO_MODE) return demoApi(path, opts);
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
  consolePaused: false,
  consoleSearch: "",
  commandHistory: [],
  commandHistoryAt: -1,
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
  if (DEMO_MODE) return;
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
  $(".otp-wrap")?.classList.remove("error");
  if (n === 2) {
    $("#login-step-2-who").textContent = "Signing in as " + $("#login-user").value.trim();
    $("#login-code").value = "";
    syncOTPCells();
    setTimeout(() => $("#login-code").focus(), 30);
  } else {
    setTimeout(() => $("#login-user").focus(), 30);
  }
}

function syncOTPCells() {
  const input = $("#login-code");
  const wrap = $(".otp-wrap");
  if (!input || !wrap) return;
  const code = input.value.replace(/\D/g, "").slice(0, 6);
  [...wrap.children].forEach((cell, i) => { cell.textContent = code[i] || ""; });
}

function installOTPControl() {
  const input = $("#login-code");
  const wrap = $(".otp-wrap");
  if (!input || !wrap) return;
  wrap.addEventListener("click", () => input.focus());
  input.addEventListener("focus", () => wrap.classList.add("focus"));
  input.addEventListener("blur", () => wrap.classList.remove("focus"));
  input.addEventListener("input", () => {
    const raw = input.value.trim();
    const digits = raw.replace(/\D/g, "");
    if (digits.length >= 6 && /^[0-9\s-]+$/.test(raw)) input.value = digits.slice(0, 6);
    syncOTPCells();
  });
  input.addEventListener("paste", () => setTimeout(syncOTPCells, 0));
  syncOTPCells();
}

async function boot() {
  if (DEMO_MODE) {
    csrfToken = "demo-csrf-token";
    S.me = DEMO_ME;
    S.status = { state: "running" };
    S.consoleLines = [...DEMO_CONSOLE];
    S.perf = [...DEMO_METRICS];
    enterApp();
    setTimeout(() => toast("Demo mode uses local mock data. No server changes are made.", "ok"), 200);
    return;
  }
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
    $(".otp-wrap")?.classList.add("error");
    $("#login-code").value = "";
    syncOTPCells();
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
  { section: "Operate", id: "overview", label: "Overview", icon: "01", perm: "server.view" },
  { section: "Operate", id: "console", label: "Console", icon: ">", perm: "server.console.view" },
  { section: "Operate", id: "performance", label: "Performance", icon: "%", perm: "server.view" },
  { section: "Operate", id: "players", label: "Players", icon: "P", perm: "server.players.view" },
  { section: "Manage", id: "servers", label: "Servers", icon: "S", perm: "server.view" },
  { section: "Manage", id: "files", label: "Files", icon: "F", perm: "server.files.manage" },
  { section: "Manage", id: "configuration", label: "Configuration", icon: "C", perm: "server.configuration.manage" },
  { section: "Manage", id: "backups", label: "Backups", icon: "B", perm: "server.backups.view" },
  { section: "Manage", id: "schedules", label: "Schedules", icon: "T", perm: "server.schedules.manage" },
  { section: "System", id: "activity", label: "Activity", icon: "A", perm: "server.configuration.manage" },
  { section: "System", id: "users", label: "Users", icon: "U", perm: "users.manage" },
  { section: "System", id: "security", label: "Security", icon: "!", perm: "users.manage" },
  { section: "System", id: "host", label: "Host", icon: "H", perm: "server.configuration.manage" },
  { section: "System", id: "settings", label: "Settings", icon: "*", perm: "server.view" },
];

function buildNav() {
  const nav = $("#nav"); nav.innerHTML = "";
  let lastSection = "";
  for (const page of PAGES) {
    if (page.perm && !can(page.perm)) continue;
    if (page.section !== lastSection) {
      nav.append(el("div", { class: "nav-section" }, page.section));
      lastSection = page.section;
    }
    nav.append(el("div", { class: "nav-item", "data-page": page.id, tabindex: "0", onclick: () => navigate(page.id), onkeydown: (e) => {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(page.id); }
    } }, el("span", { class: "nav-icon", "aria-hidden": "true" }, page.icon), page.label));
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
    el("div", {},
      el("div", { class: "muted", style: "margin-bottom:6px" }, "Active project"),
      el("div", { class: "server-name" }, active ? active.display_name : "None selected")),
    renderStatusPillNode());
}

function renderStatusPillNode() {
  const st = S.status.state || "stopped";
  return el("div", { class: "status-label " + st, id: "status-pill" },
    el("span", { class: "status-square", "aria-hidden": "true" }), st.charAt(0).toUpperCase() + st.slice(1));
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
      case "security": return await pageSecurity(main);
      case "host": return await pageHost(main);
      case "settings": return await pageSettings(main);
    }
  } catch (err) {
    main.innerHTML = "";
    main.append(el("h1", {}, "Something went wrong"), el("p", { class: "muted" }, err.message));
  }
}

function pageHeader(title, subtitle, actions = []) {
  return el("div", { class: "page-header" },
    el("div", { class: "title" },
      el("h1", {}, title),
      subtitle ? el("p", { class: "page-sub" }, subtitle) : null),
    el("div", { class: "spacer" }),
    actions.length ? el("div", { class: "actions" }, actions) : null);
}

// ----- overview -------------------------------------------------------------
async function pageOverview(main) {
  const d = await api("/overview");
  S.status = { state: d.state, detail: d.supervisor };
  renderStatusPill();
  const s = d.sample || {};
  const inst = d.instance;

  // Health, host and trends live here together. Knowing whether the server is
  // healthy should not require visiting three tabs.
  let host = null, events = [], history = [];
  try { host = await api("/host"); } catch {}
  try { events = (await api("/events?limit=25")).events || []; } catch {}
  try { history = await api("/metrics?hours=1") || []; } catch {}

  const memUsed = (host && host.mem_total) ? host.mem_total - host.mem_available : 0;

  main.innerHTML = "";
  main.append(
    pageHeader(inst ? inst.display_name : "Overview", "Server state, resource pressure, backups, and recent events for the active project.", [
      renderStatusPillNode(),
      lifecycleButtons(),
    ]),

    // What is happening right now.
    el("div", { class: "grid cols-4" },
      statCard("Uptime", fmtDur(s.uptime_seconds), s.java_pid ? "Java PID " + s.java_pid : "not running"),
      statCard("Players online", (s.online_players ?? 0) + (s.max_players ? " / " + s.max_players : ""), ""),
      statCard("Process memory", fmtBytes(s.rss_bytes), "resident set (not Java heap)"),
      statCard("CPU", (s.cpu_percent ?? 0).toFixed(1) + "%", "of one core = 100%")),

    // Host health, previously a separate tab.
    el("div", { class: "grid cols-4", style: "margin-top:16px" },
      statCard("Host memory", host ? fmtBytes(memUsed) : "—",
        host ? "of " + fmtBytes(host.mem_total) : ""),
      statCard("Disk free", fmtBytes(s.disk_free), "of " + fmtBytes(s.disk_total)),
      statCard("Load average", host && host.load1 != null ? host.load1.toFixed(2) : "—", "1 minute"),
      statCard("Services", serviceSummary(d, host), "")),

    // Trends, previously the Performance tab.
    el("div", { class: "grid cols-2", style: "margin-top:16px" },
      trendCard("CPU", history, (x) => x.cpu_percent, (v) => v.toFixed(0) + "%"),
      trendCard("Process memory", history, (x) => x.rss_bytes, fmtBytes)),

    el("div", { class: "grid cols-2", style: "margin-top:16px" },
      // The timeline: what the server did, in its own words.
      el("div", { class: "card" },
        el("h3", {}, "Recent activity"),
        events.length
          ? el("ul", { class: "timeline" }, ...events.map(eventRow))
          : el("p", { class: "muted" }, "Nothing recorded yet.")),

      el("div", { class: "card" },
        el("h3", {}, "Project"),
        inst ? el("dl", { class: "kv" },
          el("dt", {}, "MOTD"), el("dd", {}, d.motd || "—"),
          el("dt", {}, "Port"), el("dd", {}, d.port || "25565"),
          el("dt", {}, "Modloader"), el("dd", {}, inst.modloader || "unknown"),
          el("dt", {}, "Startup script"), el("dd", { class: "mono" }, inst.startup_script || "not selected"),
          el("dt", {}, "Restart policy"), el("dd", {}, inst.restart_policy || "never"),
          el("dt", {}, "Autostart"), el("dd", {}, inst.autostart_enabled ? "enabled" : "disabled"),
          el("dt", {}, "Last backup"), el("dd", {},
            d.last_backup ? fmtTime(d.last_backup.created_at) : "none yet"),
          el("dt", {}, "Next schedule"), el("dd", {},
            d.next_schedule_at ? fmtTime(d.next_schedule_at) : "none"))
          : el("p", { class: "muted" }, "No active project selected."))));
}

function serviceSummary(d, host) {
  const mc = (d.state && d.state !== "stopped") ? "running" : "stopped";
  const cp = host && host.service_bonghos ? host.service_bonghos : "active";
  const sup = host && host.service_minecraft ? host.service_minecraft : mc;
  return cp === "active" ? "Panel ok · Minecraft " + sup : cp + " · Minecraft " + sup;
}

// eventRow renders one timeline entry, coloured by severity.
function eventRow(e) {
  return el("li", { class: "timeline-item sev-" + (e.severity || "info") },
    el("span", { class: "timeline-time mono" }, fmtTime(e.occurred_at)),
    el("span", { class: "timeline-msg" }, e.message || e.event));
}

// trendCard draws a small sparkline plus the latest value.
function trendCard(title, samples, pick, fmt) {
  const values = (samples || []).map(pick).filter((v) => typeof v === "number");
  const latest = values.length ? values[values.length - 1] : 0;
  return el("div", { class: "card" },
    el("div", { class: "metric-label" }, title),
    el("div", { class: "metric-value" }, fmt(latest)),
    el("div", { class: "metric-note" }, "last hour"),
    values.length > 1 ? sparklineNode(values) : el("p", { class: "muted" }, "Collecting samples…"));
}

function statCard(title, value, sub) {
  return el("div", { class: "card metric" },
    el("div", { class: "metric-label" }, title),
    el("div", { class: "metric-value" }, String(value)),
    el("div", { class: "metric-note" }, sub || ""));
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
  const stopped = (S.status.state || "stopped") === "stopped";
  const box = el("div", { class: "console" + (S.consolePaused ? " paused" : ""), id: "console-box", role: "log", "aria-live": S.consolePaused ? "off" : "polite" });
  const search = el("input", { value: S.consoleSearch, placeholder: "Search buffer", "aria-label": "Search console buffer" });
  const input = el("input", { placeholder: can("server.console.use") ? (stopped ? "Start the server to send commands" : "Command, for example: say hello") : "Read-only console", spellcheck: "false", autocomplete: "off" });
  if (!can("server.console.use") || stopped) input.disabled = true;
  search.addEventListener("input", () => { S.consoleSearch = search.value; renderConsoleLines(box); });
  renderConsoleLines(box);
  input.addEventListener("keydown", async (e) => {
    if (e.key === "ArrowUp" && S.commandHistory.length) {
      e.preventDefault();
      S.commandHistoryAt = Math.max(0, S.commandHistoryAt < 0 ? S.commandHistory.length - 1 : S.commandHistoryAt - 1);
      input.value = S.commandHistory[S.commandHistoryAt];
      return;
    }
    if (e.key === "ArrowDown" && S.commandHistory.length) {
      e.preventDefault();
      S.commandHistoryAt = Math.min(S.commandHistory.length, S.commandHistoryAt + 1);
      input.value = S.commandHistoryAt >= S.commandHistory.length ? "" : S.commandHistory[S.commandHistoryAt];
      return;
    }
    if (e.key !== "Enter" || !input.value.trim()) return;
    const cmd = input.value.trim(); input.value = "";
    S.commandHistory.push(cmd);
    S.commandHistoryAt = -1;
    try { await api("/server/command", { method: "POST", json: { command: cmd } }); }
    catch (err) { toast(err.message, "err"); }
  });
  const sendQuick = async (cmd) => {
    try { await api("/server/command", { method: "POST", json: { command: cmd } }); toast("Command sent: " + cmd, "ok"); }
    catch (err) { toast(err.message, "err"); }
  };
  main.append(
    pageHeader("Console", "Live Minecraft output and command entry for the active server.", [renderStatusPillNode(), lifecycleButtons()]),
    el("div", { class: "console-shell" },
      el("div", { class: "console-toolbar" },
        search,
        el("button", { class: "btn ghost", onclick: () => { S.consolePaused = !S.consolePaused; pageConsole(main); } }, S.consolePaused ? "Resume autoscroll" : "Pause autoscroll"),
        el("button", { class: "btn ghost", onclick: async () => {
          try { await navigator.clipboard.writeText(S.consoleLines.join("\n")); toast("Console buffer copied", "ok"); }
          catch { toast("Copy failed in this browser", "err"); }
        } }, "Copy"),
        el("button", { class: "btn ghost", onclick: () => { box.innerHTML = ""; S.consoleLines = []; } }, "Clear view"),
        el("span", { class: "status-label " + (DEMO_MODE || (S.ws && S.ws.readyState === WebSocket.OPEN) ? "running" : "stopped") },
          el("span", { class: "status-square", "aria-hidden": "true" }),
          DEMO_MODE ? "Demo stream" : (S.ws && S.ws.readyState === WebSocket.OPEN ? "Connected" : "Reconnecting"))),
      box,
      el("div", { class: "console-input" },
        input,
        can("server.console.use") && !stopped ? [
          el("button", { class: "btn ghost", onclick: () => sendQuick("list") }, "list"),
          el("button", { class: "btn ghost", onclick: () => sendQuick("save-all") }, "save-all"),
        ] : null)));
  box.scrollTop = box.scrollHeight;
}

function consoleLineNode(line) {
  let cls = "";
  if (/ERROR|SEVERE|FATAL/.test(line)) cls = "errline";
  else if (/WARN/.test(line)) cls = "warnline";
  return el("div", { class: "console-line " + cls }, line);
}

function renderConsoleLines(box) {
  const q = (S.consoleSearch || "").toLowerCase();
  box.innerHTML = "";
  for (const line of S.consoleLines) {
    if (q && !String(line).toLowerCase().includes(q)) continue;
    box.append(consoleLineNode(line));
  }
}

function appendConsoleLine(line) {
  const box = $("#console-box");
  if (!box) return;
  if (S.consoleSearch && !String(line).toLowerCase().includes(S.consoleSearch.toLowerCase())) return;
  const atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
  box.append(consoleLineNode(line));
  while (box.childNodes.length > 1000) box.firstChild.remove();
  if (!S.consolePaused && atBottom) box.scrollTop = box.scrollHeight;
}

// ----- players --------------------------------------------------------------
async function pagePlayers(main) {
  const d = await api("/players");
  const players = d.players || [];
  const search = el("input", { placeholder: "Search players", "aria-label": "Search players" });
  const tbody = el("tbody");
  const draw = () => {
    const q = search.value.trim().toLowerCase();
    const visible = players.filter((p) => !q || String(p.username).toLowerCase().includes(q));
    tbody.innerHTML = "";
    tbody.append(...(visible.length ? visible.map(playerRow) : [el("tr", {}, el("td", { colspan: "5", class: "muted" }, players.length ? "No matching players." : "No players seen yet."))]));
  };
  search.addEventListener("input", draw);
  main.innerHTML = "";
  main.append(
    pageHeader("Players", "Observed online and recent players. Whitelist, operator, ban, and IP-ban lists are not exposed as separate read APIs yet.", [search]),
    el("div", { class: "toolbar" },
      el("span", { class: "status-label running" }, el("span", { class: "status-square" }), players.filter((p) => p.online).length + " online"),
      el("span", { class: "status-label" }, el("span", { class: "status-square" }), players.length + " observed")),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {},
          el("th", {}, "Player"), el("th", {}, "Status"), el("th", {}, "Last seen"),
          el("th", {}, "Observed playtime"), el("th", {}, ""))),
        tbody)));
  draw();
}

function playerRow(p) {
  return el("tr", {},
    el("td", {}, el("span", { class: "pill " + (p.online ? "running" : "") },
      el("span", { class: "dot" }), p.username)),
    el("td", {}, p.online ? "Online" : "Offline"),
    el("td", {}, fmtTime(p.last_seen_at)),
    el("td", {}, fmtDur(p.observed_playtime_seconds)),
    el("td", { class: "row-actions" }, can("server.players.manage") ? playerActions(p) : ""));
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
  // A deep link from elsewhere (for example the Configuration page naming the
  // file that owns the JVM settings) opens that file straight away.
  if (S.pendingFileOpen) {
    const target = S.pendingFileOpen;
    S.pendingFileOpen = null;
    return openFileEditor(main, target);
  }
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
    pageHeader("Files", "Constrained file manager for the active server directory.", [
      el("button", { class: "btn", onclick: () => upInput.click() }, "Upload"),
      el("button", { class: "btn", onclick: () => mkdirPrompt(main, path) }, "New folder"),
      upInput,
    ]),
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
// jvmSourceNote explains which file actually controls the memory settings and
// offers a direct route to it. Some packs regenerate their argument file at
// launch, so saying "detected in user_jvm_args.txt" without that context leads
// people to edit a file whose contents are discarded on the next restart.
function jvmSourceNote(jvm) {
  if (!jvm || !jvm.source_file) {
    return el("p", { class: "muted" }, "No JVM configuration detected.");
  }
  const where = jvm.variable
    ? `${jvm.variable} in ${jvm.source_file}`
    : jvm.source_file;

  const openBtn = el("button", {
    class: "btn ghost small",
    title: "Open this file in the file editor",
    onclick: () => openFileInEditor(jvm.source_file),
  }, "Open " + jvm.source_file);

  return el("div", { class: "jvm-source" },
    el("p", { class: "muted" },
      `Controlled by ${where} (${(jvm.source_kind || "").replace(/_/g, " ")})`,
      jvm.editable ? "" : " — not safely editable here"),
    jvm.note ? el("p", { class: "notice" }, jvm.note) : null,
    can("server.files.manage") ? openBtn : null);
}

// openFileInEditor jumps to the Files page with the given path already open,
// so the authoritative file is one click away from where it is named.
function openFileInEditor(path) {
  if (!can("server.files.manage")) {
    return toast("You do not have permission to edit files", "err");
  }
  S.pendingFileOpen = path;
  navigate("files");
}

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
    pageHeader("Configuration", "Startup, Java, memory, Minecraft properties, and recovery policy for the active project."),
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
        jvmSourceNote(d.jvm),
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
    pageHeader("Backups", "Verified archives, retention decisions, and restore controls. Online backups briefly pause world saving.", [
      can("server.backups.create") ? mkBtn("world", "World backup") : null,
      can("server.backups.create") ? mkBtn("full", "Full backup") : null,
      can("server.backups.create") ? mkBtn("configuration", "Config backup") : null,
    ]),
    el("div", { class: "progress hidden", id: "backup-progress" }, el("div", { style: "width:0%" })),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "ID"), el("th", {}, "Type"), el("th", {}, "Mode"), el("th", {}, "Size"), el("th", {}, "Verified"), el("th", {}, "Created"), el("th", {}, ""))),
        el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "7", class: "muted" }, "No backups yet."))))));
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
    pageHeader("Schedules", "Persistent Linux-host schedules with next run, last result, and manual run controls.", [
      can("server.schedules.manage") ? el("button", { class: "btn primary", onclick: () => scheduleForm(null) }, "New schedule") : null,
    ]),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", {}, "Action"), el("th", {}, "When"), el("th", {}, "Next run"), el("th", {}, "Last result"), el("th", {}, ""))),
        el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "6", class: "muted" }, "No schedules yet."))))));
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
    pageHeader("Performance", "One-hour process and host history. Process memory is Java RSS, not configured heap."),
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

// sparklineNode returns a standalone SVG element, for callers building a card
// rather than filling a pre-existing container.
function sparklineNode(values) {
  const W = 320, H = 56, pad = 2;
  const max = Math.max(...values, 1), min = Math.min(...values, 0);
  const span = (max - min) || 1;
  const pts = values.map((v, i) => {
    const x = pad + (i / Math.max(1, values.length - 1)) * (W - pad * 2);
    const y = H - pad - ((v - min) / span) * (H - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
  const ns = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(ns, "svg");
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  svg.setAttribute("class", "sparkline");
  svg.setAttribute("preserveAspectRatio", "none");
  svg.setAttribute("role", "img");
  const line = document.createElementNS(ns, "polyline");
  line.setAttribute("points", pts);
  line.setAttribute("fill", "none");
  line.setAttribute("stroke", "currentColor");
  line.setAttribute("stroke-width", "2");
  svg.appendChild(line);
  return svg;
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
    pageHeader("Servers", "Project inventory, active-project selection, and persistent import progress.", [
      can("server.import.manage") ? el("button", { class: "btn primary", onclick: importWizard }, "Import server") : null,
    ]),
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
          if (DEMO_MODE) {
            toast("Demo import simulated for " + f.name, "ok");
            return;
          }
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
    pageHeader("Activity", "Audit trail of account and server-management actions."),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "User"), el("th", {}, "Action"), el("th", {}, "Target"), el("th", {}, "Detail"))),
        el("tbody", {}, (list || []).length ? (list || []).map((a2) => el("tr", {},
          el("td", {}, fmtTime(a2.at)), el("td", {}, a2.username), el("td", {}, a2.action.replace(/_/g, " ")),
          el("td", { class: "mono" }, a2.target || ""), el("td", { class: "muted" }, a2.detail || "")))
          : el("tr", {}, el("td", { colspan: "5", class: "muted" }, "No audit events recorded yet."))))));
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
    pageHeader("Users", "Accounts, roles, invitations, sessions, and final-Owner protection.", [
      el("button", { class: "btn primary", onclick: inviteUser }, "Invite user"),
    ]),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Username"), el("th", {}, "Role"), el("th", {}, "Status"), el("th", {}, ""))),
        el("tbody", {}, rows.length ? rows : el("tr", {}, el("td", { colspan: "4", class: "muted" }, "No users returned by the API."))))));
}

function inviteUser() {
  const role = el("select", {},
    ...[["admin", "Admin"], ["member", "Member"], ["viewer", "Viewer"]].map(([v, r]) => el("option", { value: v }, r)));
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
    ...[["owner", "Owner"], ["admin", "Admin"], ["member", "Member"], ["viewer", "Viewer"]]
      .map(([v, r]) => el("option", { value: v, selected: v === u.Role ? "" : null }, r)));
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
    pageHeader("Host", "Linux host dependencies, services, storage, and exact local runtime paths."),
    el("div", { class: "notice" }, d.note),
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

async function pageSecurity(main) {
  const users = await api("/users").catch(() => []);
  const activity = await api("/activity").catch(() => []);
  const activeUsers = (users || []).filter((u) => !u.Disabled).length;
  const disabledUsers = (users || []).filter((u) => u.Disabled).length;
  main.innerHTML = "";
  main.append(
    pageHeader("Security", "Authentication posture, account exposure, and sensitive-account operations."),
    el("div", { class: "grid cols-3" },
      statCard("Signed in as", S.me.username, S.me.role),
      statCard("Active accounts", activeUsers, disabledUsers + " disabled"),
      statCard("TOTP policy", "Mandatory", "enforced at setup and invitation activation")),
    el("div", { class: "grid cols-2", style: "margin-top:14px" },
      el("div", { class: "card" },
        el("h3", {}, "Current session"),
        el("dl", { class: "kv" },
          el("dt", {}, "Username"), el("dd", {}, S.me.username),
          el("dt", {}, "Role"), el("dd", {}, S.me.role),
          el("dt", {}, "Permissions"), el("dd", { class: "mono" }, (S.me.permissions || []).join(", ")))),
      el("div", { class: "card" },
        el("h3", {}, "Recovery and sessions"),
        el("p", { class: "muted" }, "The current API supports revoking another user's sessions from Users. It does not expose recovery-code inventory, session lists, or TOTP reset flows in the Web UI yet."))),
    el("h2", {}, "Recent security activity"),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "Actor"), el("th", {}, "Action"), el("th", {}, "Target"))),
        el("tbody", {}, (activity || []).slice(0, 12).map((a2) => el("tr", {},
          el("td", {}, fmtTime(a2.at)),
          el("td", {}, a2.username),
          el("td", {}, a2.action.replace(/_/g, " ")),
          el("td", { class: "mono" }, a2.target || "")))))));
}

async function pageSettings(main) {
  const version = await api("/version").catch(() => ({ version: "unknown" }));
  const makeThemeButton = (value, label) => el("button", {
    class: "btn ghost" + (themeChoice() === value ? " active" : ""),
    "data-theme-choice": value,
    onclick: () => setTheme(value),
  }, label);
  main.innerHTML = "";
  main.append(
    pageHeader("Settings", "Local UI preferences and application metadata."),
    el("div", { class: "card" },
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Theme"), el("p", { class: "muted" }, "System follows the operating-system color scheme and reacts to changes.")),
        el("div", { class: "theme-choice" },
          makeThemeButton("system", "System"),
          makeThemeButton("dark", "Dark"),
          makeThemeButton("light", "Light"))),
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Monitoring"), el("p", { class: "muted" }, "Live metrics are subscribed only on Overview and Performance.")),
        el("dl", { class: "kv" },
          el("dt", {}, "WebSocket topics"), el("dd", { class: "mono" }, BASE_TOPICS.join(", ") + " + active page"),
          el("dt", {}, "Console buffer"), el("dd", {}, "Last 1000 lines in this browser"))),
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Application"), el("p", { class: "muted" }, "Runtime settings not exposed by the API are shown honestly rather than mocked.")),
        el("dl", { class: "kv" },
          el("dt", {}, "Version"), el("dd", { class: "mono" }, version.version),
          el("dt", {}, "Frontend"), el("dd", {}, "Dependency-free vanilla HTML, CSS, and JavaScript embedded in the Go binary")))));
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
    const qrBox = el("div", { class: "qr-box hidden" });
    const secretBox = el("div", { class: "muted mono", style: "word-break:break-all" });
    let secret = "";
    const genBtn = el("button", { class: "btn", type: "button", onclick: async () => {
      const csrf = await fetch("/api/auth/csrf").then((r) => r.json());
      const d = await fetch(`/api/invitations/${token}/totp`, { method: "POST",
        headers: { "Content-Type": "application/json", "X-Bonghos-CSRF": csrf.csrf },
        body: JSON.stringify({ username: user.value }) }).then((r) => r.json());
      if (d.error) return toast(d.error, "err");
      secret = d.secret;
      // The QR is generated server-side, so the browser needs no QR library
      // and this page keeps working offline. Without it the secret below is
      // still everything an authenticator app needs.
      // The SVG is built by Bonghos from integer coordinates and contains no
      // user input, but it arrives as markup, so refuse anything that does not
      // look like the plain shape we generate.
      const svgOK = typeof d.qr_svg === "string" &&
        d.qr_svg.startsWith("<svg ") && !/<script|onload=|xlink:href/i.test(d.qr_svg);
      if (svgOK) {
        qrBox.innerHTML = d.qr_svg;
        qrBox.prepend(el("p", { class: "muted" }, "Scan this with your authenticator app:"));
        qrBox.classList.remove("hidden");
      }
      secretBox.textContent =
        (svgOK ? "If scanning does not work, enter this secret manually:\n\n" : "") +
        "Secret: " + d.secret + "\nURI: " + d.uri;
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
      genBtn, qrBox, secretBox,
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
else {
  installOTPControl();
  boot();
}
