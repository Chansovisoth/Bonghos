/* Bonghos SPA — dependency-free vanilla JS */
"use strict";

// ---------------------------------------------------------------------------
// tiny helpers
// ---------------------------------------------------------------------------
const $ = (sel, root = document) => root.querySelector(sel);

let solarIconData = {};
const INLINE_SOLAR_ICONS = {
  "alt-arrow-left-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m15 5l-6 7l6 7"/>',
  },
  "alt-arrow-right-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m9 5l6 7l-6 7"/>',
  },
  "moon-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21.75 15.5A9.75 9.75 0 0 1 8.5 2.25A10 10 0 1 0 21.75 15.5Z"/>',
  },
  "sun-2-linear": {
    body: '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="12" r="4"/><path d="M12 2V1m0 22v-1m10-10h1M1 12h1m17.07-7.07l.7-.7M4.23 19.77l.7-.7m14.14 0l.7.7M4.23 4.23l.7.7"/></g>',
  },
  "storage-refresh": {
    body: '<path d="M0 0h24v24H0z" fill="none"/><path fill="currentColor" d="M12.077 19q-2.931 0-4.966-2.033q-2.034-2.034-2.034-4.964t2.034-4.966T12.077 5q1.783 0 3.339.847q1.555.847 2.507 2.365V5h1v5.23h-5.23v-1h3.7q-.782-1.495-2.198-2.363T12.077 6q-2.5 0-4.25 1.75T6.077 12t1.75 4.25t4.25 1.75q1.925 0 3.475-1.1t2.175-2.9h1.062q-.662 2.246-2.514 3.623T12.077 19"/>',
  },
  "wrap-text": {
    body: '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"><path d="M4 7h16M4 17h5m-5-5h13.5a2.5 2.5 0 0 1 2.5 2.5v0a2.5 2.5 0 0 1-2.5 2.5h-5"/><path d="M15 15.5L12.5 17l2.5 1.5z"/></g>',
  },
};
const BUTTON_ICONS = {
  "Activate": "check-circle-linear",
  "Back": "undo-left-linear",
  "Ban": "lock-keyhole-linear",
  "Cancel": "close-circle-linear",
  "Choose Archive": "folder-open-linear",
  "Clear view": "trash-bin-trash-linear",
  "Continue": "play-linear",
  "Copy visible": "copy-linear",
  "Deop": "key-linear",
  "Disable": "close-circle-linear",
  "Discard": "undo-left-linear",
  "Download world": "download-linear",
  "Duplicate": "copy-linear",
  "Enable": "check-circle-linear",
  "Force stop": "danger-triangle-linear",
  "Generate authenticator secret": "key-linear",
  "Invite user": "add-circle-linear",
  "Kick": "close-circle-linear",
  "Make active": "check-circle-linear",
  "Op": "key-linear",
  "Pause": "pause-linear",
  "Pause autoscroll": "pause-linear",
  "Refresh": "refresh-linear",
  "Restart": "restart-linear",
  "Resume": "play-linear",
  "Reset world": "refresh-linear",
  "Resume autoscroll": "play-linear",
  "Revoke sessions": "logout-2-linear",
  "Review & accept": "shield-check-linear",
  "Role": "key-linear",
  "Run now": "play-linear",
  "Save changes": "diskette-linear",
  "Servers": "server-square-linear",
  "Sign out": "logout-2-linear",
  "Start": "play-linear",
  "Stop": "stop-linear",
  "Use crop": "gallery-linear",
  "Verify": "shield-check-linear",
  "list": "users-group-rounded-linear",
  "save-all": "diskette-linear",
};

function buttonIconName(label) {
  if (BUTTON_ICONS[label]) return BUTTON_ICONS[label];
  if (/^Save\b/.test(label)) return "diskette-linear";
  if (/^(Delete|Remove|Clear)\b/.test(label)) return "trash-bin-trash-linear";
  if (/^(New|Create|Add)\b/.test(label)) return "add-circle-linear";
  if (/^(Upload|Import)\b/.test(label)) return "upload-linear";
  if (/^Download\b/.test(label)) return "download-linear";
  if (/^Open\b/.test(label)) return "folder-open-linear";
  if (/^Copy\b/.test(label)) return "copy-linear";
  if (/^Edit\b/.test(label)) return "pen-new-square-linear";
  if (/^Restore\b/.test(label)) return "archive-down-minimlistic-linear";
  if (/^(Protect|Accept)\b/.test(label)) return "shield-check-linear";
  if (/^Unprotect\b/.test(label)) return "shield-keyhole-linear";
  if (/^Send\b/.test(label)) return "send-square-linear";
  if (/^Rename\b/.test(label)) return "pen-new-square-linear";
  return "";
}

function hydrateSolarIcon(svg) {
  const data = solarIconData[svg.dataset.solarIcon] || INLINE_SOLAR_ICONS[svg.dataset.solarIcon];
  if (data) svg.innerHTML = data.body;
}

function solarIcon(name, className = "") {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "icon" + (className ? " " + className : ""));
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.dataset.solarIcon = name;
  hydrateSolarIcon(svg);
  return svg;
}

function gameVersionIcon() {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "icon game-version-icon");
  svg.setAttribute("viewBox", "0 0 512 512");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.innerHTML = '<path d="M0 0h512v512H0z" fill="none"/><path fill="currentColor" d="M483.13 245.38C461.92 149.49 430 98.31 382.65 84.33A107.1 107.1 0 0 0 352 80c-13.71 0-25.65 3.34-38.28 6.88C298.5 91.15 281.21 96 256 96s-42.51-4.84-57.76-9.11C185.6 83.34 173.67 80 160 80a115.7 115.7 0 0 0-31.73 4.32c-47.1 13.92-79 65.08-100.52 161C4.61 348.54 16 413.71 59.69 428.83a56.6 56.6 0 0 0 18.64 3.22c29.93 0 53.93-24.93 70.33-45.34c18.53-23.1 40.22-34.82 107.34-34.82c59.95 0 84.76 8.13 106.19 34.82c13.47 16.78 26.2 28.52 38.9 35.91c16.89 9.82 33.77 12 50.16 6.37c25.82-8.81 40.62-32.1 44-69.24c2.57-28.48-1.39-65.89-12.12-114.37M208 240h-32v32a16 16 0 0 1-32 0v-32h-32a16 16 0 0 1 0-32h32v-32a16 16 0 0 1 32 0v32h32a16 16 0 0 1 0 32m84 4a20 20 0 1 1 20-20a20 20 0 0 1-20 20m44 44a20 20 0 1 1 20-19.95A20 20 0 0 1 336 288m0-88a20 20 0 1 1 20-20a20 20 0 0 1-20 20m44 44a20 20 0 1 1 20-20a20 20 0 0 1-20 20"/>';
  return svg;
}

const LIFECYCLE_LOADING_STEP_SECONDS = 0.2;
const LIFECYCLE_LOADING_CYCLE_MS = LIFECYCLE_LOADING_STEP_SECONDS * 12 * 1000;
let lifecycleLoadingIconId = 0;
function lifecycleLoadingIcon(onCycleEnd = null) {
  const id = `lifecycle-loading-${++lifecycleLoadingIconId}`;
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "icon lifecycle-loading-icon");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.innerHTML = `
    <rect width="10" height="10" x="1" y="1" fill="currentColor" rx="1">
      <animate id="${id}-a" fill="freeze" attributeName="x" begin="0;${id}-l.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
      <animate id="${id}-d" fill="freeze" attributeName="y" begin="${id}-c.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
      <animate id="${id}-g" fill="freeze" attributeName="x" begin="${id}-f.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
      <animate id="${id}-j" fill="freeze" attributeName="y" begin="${id}-i.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
    </rect>
    <rect width="10" height="10" x="1" y="13" fill="currentColor" rx="1">
      <animate id="${id}-b" fill="freeze" attributeName="y" begin="${id}-a.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
      <animate id="${id}-e" fill="freeze" attributeName="x" begin="${id}-d.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
      <animate id="${id}-h" fill="freeze" attributeName="y" begin="${id}-g.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
      <animate id="${id}-k" fill="freeze" attributeName="x" begin="${id}-j.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
    </rect>
    <rect width="10" height="10" x="13" y="13" fill="currentColor" rx="1">
      <animate id="${id}-c" fill="freeze" attributeName="x" begin="${id}-b.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
      <animate id="${id}-f" fill="freeze" attributeName="y" begin="${id}-e.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="13;1"/>
      <animate id="${id}-i" fill="freeze" attributeName="x" begin="${id}-h.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
      <animate id="${id}-l" fill="freeze" attributeName="y" begin="${id}-k.end" dur="${LIFECYCLE_LOADING_STEP_SECONDS}s" values="1;13"/>
    </rect>`;
  if (onCycleEnd) svg.querySelector(`#${id}-l`)?.addEventListener("endEvent", onCycleEnd);
  return svg;
}

function decorateButton(button) {
  if (button.querySelector(":scope > .icon")) return;
  const name = buttonIconName(button.textContent.trim());
  if (name) button.prepend(solarIcon(name));
}

function setButtonLabel(button, label) {
  const textNode = [...button.childNodes].find((node) => node.nodeType === Node.TEXT_NODE && node.nodeValue.trim());
  if (textNode) textNode.nodeValue = label;
  else button.append(document.createTextNode(label));
}

async function loadSolarIcons() {
  document.querySelectorAll("button").forEach(decorateButton);
  try {
    const response = await fetch("/solar-icons.json", { credentials: "same-origin" });
    if (!response.ok) throw new Error("icon asset returned " + response.status);
    const collection = await response.json();
    solarIconData = collection.icons || {};
    document.querySelectorAll(".icon[data-solar-icon]").forEach(hydrateSolarIcon);
  } catch (error) {
    console.warn("Solar icons unavailable:", error);
  }
}

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
  if (tag === "button") decorateButton(n);
  return n;
};
loadSolarIcons();
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

function recordMatchesSearch(record, query, ...formattedValues) {
  if (!query) return true;
  return `${JSON.stringify(record)} ${formattedValues.join(" ")}`.toLowerCase().includes(query);
}

function pageSearchInput(label, onSearch) {
  const input = el("input", {
    class: "page-search",
    type: "search",
    placeholder: `Search ${label.toLowerCase()}`,
    "aria-label": `Search ${label.toLowerCase()}`,
  });
  input.addEventListener("input", () => onSearch(input.value.trim().toLowerCase()));
  return input;
}

function toast(msg, kind = "") {
  const t = el("div", { class: "toast " + kind, role: "status" }, msg);
  $("#toast-host").append(t);
  setTimeout(() => t.remove(), 6000);
}

let modalRestoreFocus = null;

function closeActiveModal() {
  const host = $("#modal-host");
  if (!host || !host.firstElementChild) return false;
  host.innerHTML = "";
  const restoreFocus = modalRestoreFocus;
  modalRestoreFocus = null;
  if (restoreFocus && restoreFocus.isConnected) restoreFocus.focus();
  return true;
}

function modal(title, bodyNodes, actions) {
  const host = $("#modal-host");
  modalRestoreFocus = document.activeElement;
  host.innerHTML = "";
  const close = closeActiveModal;
  const m = el("div", { class: "overlay", onclick: (e) => { if (e.target === m) close(); } },
    el("div", { class: "modal", role: "dialog", "aria-modal": "true", "aria-label": title },
      el("h2", {}, title),
      ...bodyNodes,
      el("div", { class: "actions" },
        ...actions.map(([label, cls, fn]) => {
          const bottomAction = /^(Cancel|Discard)$/i.test(label);
          return el("button", {
            class: "btn " + cls + (bottomAction ? " modal-bottom-action" : ""),
            onclick: () => fn(close),
          }, label);
        }))));
  host.append(m);
  requestAnimationFrame(() => m.querySelector("input, select, textarea, button")?.focus());
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
  syncThemeToggles(dark);
}

function syncThemeToggles(dark = document.documentElement.dataset.resolvedTheme === "dark") {
  document.querySelectorAll("[data-theme-toggle]").forEach((button) => {
    const label = dark ? "Switch to light mode" : "Switch to dark mode";
    button.setAttribute("aria-label", label);
    button.setAttribute("title", label);
    button.setAttribute("aria-pressed", String(dark));
  });
}

function initializeThemeToggles() {
  document.querySelectorAll("[data-theme-toggle]").forEach((button) => {
    button.append(
      el("span", { class: "theme-toggle-icon theme-toggle-sun", "aria-hidden": "true" }, solarIcon("sun-2-linear")),
      el("span", { class: "theme-toggle-icon theme-toggle-moon", "aria-hidden": "true" }, solarIcon("moon-linear")),
    );
    button.addEventListener("click", () => {
      const dark = document.documentElement.dataset.resolvedTheme === "dark";
      setTheme(dark ? "light" : "dark");
    });
  });
  syncThemeToggles();
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
initializeThemeToggles();

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
  { id: 1, slug: "bio1", display_name: "Bio1 Survival - Long Local Demo Server Name", provider: "curseforge", modloader: "neoforge", modloader_version: "21.1.228", minecraft_version: "1.21.1", source_type: "direct-url", startup_script: "run.sh", restart_policy: "on-failure", autostart_enabled: true, demo_icon: "demo-server-bio1.png" },
  { id: 2, slug: "creative-lab", display_name: "Creative Lab", provider: "modrinth", modloader: "fabric", modloader_version: "0.16.10", minecraft_version: "1.21.1", source_type: "archive-upload", external_directory: false, demo_icon: "demo-server-creative-lab.png" },
];
const DEMO_BOTS = [
  { id: 1, name: "Server alerts", provider: "telegram", destination_id: "-1001234567890", enabled: true, notify_server_started: true, notify_server_stopped: true, notify_player_joined: true, notify_player_left: true, token_configured: true },
  { id: 2, name: "Staff channel", provider: "discord", destination_id: "123456789012345678", enabled: false, notify_server_started: true, notify_server_stopped: true, notify_player_joined: false, notify_player_left: false, token_configured: true },
];
const DEMO_CONSOLE = [
  "[19:27:36] [Server thread/INFO]: Starting minecraft server version 1.20.1",
  "[19:27:43] [Server thread/INFO]: Loading Forge mods from /home/klaude/bonghos/servers/minecraft-java/modded/bio1/mods",
  "[19:28:12] [Server thread/WARN]: Can't keep up! Is the server overloaded? Running 2475ms behind",
  "[19:28:40] [Server thread/INFO]: Steve joined the game",
  "[19:29:04] [Server thread/ERROR]: Example datapack warning for visual review only",
];
const DEMO_METRICS = Array.from({ length: 60 }, (_, i) => ({
  collected_at: new Date(Date.now() - (59 - i) * 60000).toISOString(),
  cpu_percent: 18 + Math.sin(i / 6) * 11 + (i % 13),
  host_cpu_percent: 34 + Math.sin(i / 5) * 18 + (i % 7),
  cpu_temp_celsius: 54 + Math.sin(i / 8) * 7,
  cpu_cores: Array.from({ length: 8 }, (_, core) => ({
    index: core,
    usage_percent: Math.max(2, Math.min(100, 28 + Math.sin((i + core) / 4) * 24 + core * 3)),
    temp_celsius: 50 + Math.sin((i + core) / 9) * 7 + core * 0.6,
  })),
  rss_bytes: (2600 + Math.sin(i / 8) * 220 + i * 3) * 1024 * 1024,
  jvm_xms_bytes: 2 * 1024 * 1024 * 1024,
  jvm_xmx_bytes: 6 * 1024 * 1024 * 1024,
  host_mem_total: 32 * 1024 * 1024 * 1024,
  host_mem_avail: (18 - Math.sin(i / 9) * 1.4) * 1024 * 1024 * 1024,
  load1: 0.65 + Math.sin(i / 7) * 0.32,
  disk_total: 512 * 1024 * 1024 * 1024,
  bonghos_dir_bytes: (60.4 + i * 0.006) * 1024 * 1024 * 1024,
  server_dir_bytes: (14.2 + i * 0.005) * 1024 * 1024 * 1024,
  backup_dir_bytes: 42.6 * 1024 * 1024 * 1024,
  system_dir_bytes: 1.8 * 1024 * 1024 * 1024,
  online_players: i % 17 > 9 ? 4 : 3,
  disk_free: (186 - i * 0.08) * 1024 * 1024 * 1024,
  uptime_seconds: 86400 + i * 60,
  java_pid: 4281,
}));

const CONSOLE_LINE_LIMIT = 1000;

function demoDelay() {
  return new Promise((resolve) => setTimeout(resolve, 80));
}

async function demoApi(path, opts = {}) {
  await demoDelay();
  const method = opts.method || "GET";
  const clean = path.split("?")[0];
  const query = new URL(path, "http://bonghos.demo").searchParams;
  if (method !== "GET") {
    if (clean === "/server/start") { S.status = { state: "running" }; return { ok: true }; }
    if (clean === "/server/stop") { S.status = { state: "stopped" }; return { ok: true }; }
    if (clean === "/server/restart") {
      S.status = { state: "stopping" };
      lifecyclePendingSettled(S.status.state);
      await demoDelay();
      S.status = { state: "running" };
      return { ok: true };
    }
    if (clean === "/server/command") {
      S.consoleLines.push("> " + ((opts.json && opts.json.command) || ""));
      return { ok: true };
    }
    if (clean === "/servers/slug-preview") {
      const name = (opts.json && opts.json.name) || "server";
      return { slug: name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "server" };
    }
    if (method === "POST" && clean === "/bots") {
      const created = { id: Math.max(0, ...DEMO_BOTS.map((bot) => bot.id)) + 1, ...opts.json, token_configured: true };
      delete created.token;
      DEMO_BOTS.push(created);
      return { ...created };
    }
    const botMatch = clean.match(/^\/bots\/(\d+)$/);
    if (botMatch) {
      const index = DEMO_BOTS.findIndex((bot) => bot.id === Number(botMatch[1]));
      if (index < 0) throw new Error("Notification bot not found");
      if (method === "DELETE") {
        DEMO_BOTS.splice(index, 1);
        return { ok: true };
      }
      if (method === "PATCH") {
        const next = { ...opts.json };
        delete next.token;
        Object.assign(DEMO_BOTS[index], next);
        return { ...DEMO_BOTS[index] };
      }
    }
    if (method === "POST" && /^\/bots\/\d+\/test$/.test(clean)) return { ok: true };
    const updateMatch = clean.match(/^\/servers\/(\d+)$/);
    if (method === "PATCH" && updateMatch) {
      const server = DEMO_SERVERS.find((entry) => entry.id === Number(updateMatch[1]));
      if (!server) throw new Error("Server not found");
      if (opts.json?.display_name !== undefined) {
        const displayName = String(opts.json.display_name).trim();
        if (!displayName) throw new Error("Display name is required");
        if ([...displayName].length > 120) throw new Error("Display name must be 120 characters or fewer");
        server.display_name = displayName;
      }
      return { ...server };
    }
    const duplicateMatch = clean.match(/^\/servers\/(\d+)\/duplicate$/);
    if (duplicateMatch) {
      const source = DEMO_SERVERS.find((server) => server.id === Number(duplicateMatch[1]));
      const displayName = (opts.json && opts.json.display_name) || ((source && source.display_name) || "Server") + " Copy";
      const slugBase = displayName.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "server-copy";
      const clone = { ...source, id: Math.max(...DEMO_SERVERS.map((server) => server.id)) + 1, display_name: displayName, slug: slugBase, autostart_enabled: false };
      DEMO_SERVERS.push(clone);
      return { operation_id: "demo-duplicate", server: clone };
    }
    if (/^\/servers\/\d+\/world\/reset$/.test(clean)) {
      return { ok: true, backup_id: "demo-pre-reset-world" };
    }
    return { ok: true };
  }
  switch (clean) {
    case "/auth/csrf": return { csrf: "demo-csrf-token" };
    case "/auth/me": return DEMO_ME;
    case "/bots": return DEMO_BOTS.map((bot) => ({ ...bot }));
    case "/version": return { version: "0.1.1-demo" };
    case "/servers": return { servers: DEMO_SERVERS, active_id: 1 };
    case "/server/status": return S.status;
    case "/server/console/history": return { lines: DEMO_CONSOLE.slice(-CONSOLE_LINE_LIMIT), limit: CONSOLE_LINE_LIMIT, source: "demo" };
    case "/overview": return {
      state: S.status.state,
      version: "0.1.1-demo",
      instance: DEMO_SERVERS[0],
      motd: "A precise Bonghos local demo",
      lan_ip: "192.168.1.42",
      port: "25565",
      max_players: "20",
      last_backup: { created_at: new Date(Date.now() - 5 * 3600000).toISOString() },
      next_schedule_at: new Date(Date.now() + 3 * 3600000).toISOString(),
      sample: DEMO_METRICS[DEMO_METRICS.length - 1],
    };
    case "/host": return {
      bind_address: "127.0.0.1", port: 8080, home: "/home/demo/bonghos",
      metrics_interval_seconds: 10,
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
    case "/metrics/config": return { interval_seconds: 10 };
    case "/metrics/storage": {
      const sample = DEMO_METRICS[DEMO_METRICS.length - 1];
      return {
        collected_at: new Date().toISOString(),
        disk_total: sample.disk_total,
        disk_free: sample.disk_free,
        bonghos_dir_bytes: sample.bonghos_dir_bytes,
        server_dir_bytes: sample.server_dir_bytes,
        backup_dir_bytes: sample.backup_dir_bytes,
        system_dir_bytes: sample.system_dir_bytes,
      };
    }
    case "/players": return { players: [
      { username: "iKlaude", online: true, op: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 7342 },
      { username: "Alex", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 3922 },
      { username: "Long_Name_With_Underscores", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 18422 },
      { username: "OfflineMiner", online: false, last_seen_at: new Date(Date.now() - 86400000).toISOString(), observed_playtime_seconds: 7521 },
    ] };
    case "/files": return [
      { name: "world", is_dir: true, size: 0, mod_time: new Date(Date.now() - 3600000).toISOString() },
      { name: "mods", is_dir: true, size: 0, mod_time: new Date(Date.now() - 4200000).toISOString() },
      { name: "server-icon.png", is_dir: false, size: 12175, mod_time: new Date(Date.now() - 4800000).toISOString() },
      { name: "server.properties", is_dir: false, size: 914, mod_time: new Date(Date.now() - 7200000).toISOString() },
      { name: "user_jvm_args.txt", is_dir: false, size: 72, mod_time: new Date(Date.now() - 5400000).toISOString() },
      { name: "eula.txt", is_dir: false, size: 9, mod_time: new Date(Date.now() - 8200000).toISOString() },
      { name: "ops.json", is_dir: false, size: 328, mod_time: new Date(Date.now() - 9200000).toISOString() },
      { name: "forge-1.20.1-47.3.0.jar", is_dir: false, size: 8427356, mod_time: new Date(Date.now() - 10200000).toISOString() },
      { name: "run.sh", is_dir: false, size: 124, mod_time: new Date(Date.now() - 11200000).toISOString() },
    ];
    case "/files/content": return { content: "motd=A precise Bonghos local demo\nserver-port=25565\nmax-players=20\n" };
    case "/configuration": return {
      eula: true,
      instance: DEMO_SERVERS.find((server) => server.id === Number(query.get("server_id"))) || DEMO_SERVERS[0],
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
  overviewReturn: false,
  lifecyclePending: null,
  onlinePlayerCount: null,
  consoleLines: [],
  consolePaused: false,
  consoleWrap: false,
  consoleSearch: "",
  consoleHistoryRequest: 0,
  commandHistory: [],
  commandHistoryAt: -1,
  perf: [],
  perfStorage: null,
  performanceTarget: "",
  serverTargetId: null,
  managedServerId: null,
  perfIntervalSeconds: 2,
  uptimeBase: null,
};
const can = (p) => S.me && S.me.permissions && S.me.permissions.includes(p);

function setUptimeBaseline(sample) {
  const seconds = Number(sample?.uptime_seconds);
  if (!Number.isFinite(seconds) || seconds < 0 || !sample?.java_pid) {
    S.uptimeBase = null;
    return;
  }
  S.uptimeBase = {
    seconds: Math.floor(seconds),
    at: Date.now(),
    pid: sample.java_pid,
  };
}

function currentUptimeSeconds() {
  if (!S.uptimeBase) return null;
  return S.uptimeBase.seconds + Math.floor((Date.now() - S.uptimeBase.at) / 1000);
}

function updateUptimeDisplay() {
  const node = $("#uptime-value");
  if (!node) return;
  const seconds = currentUptimeSeconds();
  node.textContent = seconds === null ? "—" : fmtDur(seconds);
}

setInterval(updateUptimeDisplay, 1000);

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
const PERFORMANCE_INTERVAL_KEY = "bonghos.performance.interval";
const PERFORMANCE_INTERVAL_OPTIONS = [1, 2, 3, 5, 10, 30, 60];
let demoPerformanceTimer = null;
let performanceStorageRequest = 0;
let navigationJumpStartTimer = null;
let navigationJumpTimer = null;

function savedPerformanceInterval() {
  const saved = localStorage.getItem(PERFORMANCE_INTERVAL_KEY);
  if (saved === null) return 2;
  const seconds = Number(saved);
  // Older builds stored 0 for the removed "Server default" option. Treat it
  // like any invalid value and migrate the browser to the 2-second UI default.
  return PERFORMANCE_INTERVAL_OPTIONS.includes(seconds) ? seconds : 2;
}

S.perfIntervalSeconds = savedPerformanceInterval();

function performanceSubscription() {
  return { action: "subscribe", topic: "performance", interval_seconds: S.perfIntervalSeconds };
}

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
  if (next === currentPageTopic) {
    if (next === "performance") wsSend(performanceSubscription());
    syncDemoPerformanceStream();
    return;
  }
  if (currentPageTopic && !BASE_TOPICS.includes(currentPageTopic)) {
    wsSend({ action: "unsubscribe", topic: currentPageTopic });
  }
  currentPageTopic = next;
  if (next) wsSend(next === "performance" ? performanceSubscription() : { action: "subscribe", topic: next });
  syncDemoPerformanceStream();
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
    updatePerformanceFreshness();
  };
  ws.onmessage = (ev) => {
    let m; try { m = JSON.parse(ev.data); } catch { return; }
    handleEvent(m);
  };
  ws.onclose = () => {
    updatePerformanceFreshness();
    if (!S.me) return;
    setTimeout(connectWS, wsRetry);
    wsRetry = Math.min(wsRetry * 2, 15000);
  };
}

function handleEvent(m) {
  const { topic, type, data } = m;
  if (topic === "console" && type === "line") {
    S.consoleLines.push(data.line);
    trimConsoleLines();
    appendConsoleLine(data.line);
  } else if (type === "status") {
    S.status = data || S.status;
    markLifecyclePendingSettled(S.status.state);
    renderStatusPill();
    if (S.page === "overview" && !S.lifecyclePending) renderPage();
  } else if (type === "sample" && ((S.page === "performance" && topic === "performance") || (S.page === "overview" && topic === "overview"))) {
    appendPerformanceSample(data);
    setUptimeBaseline(data);
    updateUptimeDisplay();
    updateLiveStats(data);
  } else if (topic === "players") {
    refreshPlayerCount();
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

function trimConsoleLines(lines = S.consoleLines) {
  if (lines.length <= CONSOLE_LINE_LIMIT) return lines;
  const trimmed = lines.slice(lines.length - CONSOLE_LINE_LIMIT);
  if (lines === S.consoleLines) S.consoleLines = trimmed;
  return trimmed;
}

// ---------------------------------------------------------------------------
// auth flow
// ---------------------------------------------------------------------------
const mobileNavQuery = window.matchMedia("(max-width: 980px)");
const sidebarToggle = $("#sidebar-toggle");
sidebarToggle.append(solarIcon("hamburger-menu-linear"));

function setSidebarOpen(open) {
  const app = $("#app-view");
  const next = !!open && mobileNavQuery.matches;
  app.classList.toggle("sidebar-open", next);
  document.body.classList.toggle("mobile-nav-open", next);
  sidebarToggle.setAttribute("aria-expanded", String(next));
  sidebarToggle.setAttribute("aria-label", next ? "Close navigation" : "Open navigation");
}

function closeMobileSidebar() {
  if (!$("#app-view").classList.contains("sidebar-open")) return false;
  setSidebarOpen(false);
  sidebarToggle.focus();
  return true;
}

sidebarToggle.addEventListener("click", () =>
  setSidebarOpen(!$("#app-view").classList.contains("sidebar-open")));
$("#sidebar-scrim").addEventListener("click", closeMobileSidebar);
mobileNavQuery.addEventListener("change", (event) => { if (!event.matches) setSidebarOpen(false); });

function showLogin() {
  setSidebarOpen(false);
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
  refreshPlayerCount();
  connectWS();
  refreshServers().then(() => navigate(hashPage() || S.page, { replaceHash: true }));
}

// ---------------------------------------------------------------------------
// navigation
// ---------------------------------------------------------------------------
const PAGES = [
  { section: "Operate", id: "overview", label: "Overview", icon: "home-2-linear", perm: "server.view" },
  { section: "Operate", id: "console", label: "Console", icon: "command-linear", perm: "server.console.view" },
  { section: "Operate", id: "performance", label: "Performance", icon: "chart-2-linear", perm: "server.view" },
  { section: "Operate", id: "players", label: "Players", icon: "users-group-rounded-linear", perm: "server.players.view" },
  { section: "Manage", id: "servers", label: "Servers", icon: "server-square-linear", perm: "server.view" },
  { section: "Manage", id: "files", label: "Files", icon: "folder-with-files-linear", perm: "server.files.manage" },
  { section: "Manage", id: "configuration", label: "Configuration", icon: "tuning-2-linear", perm: "server.configuration.manage" },
  { section: "Manage", id: "backups", label: "Backups", icon: "archive-down-minimlistic-linear", perm: "server.backups.view" },
  { section: "Manage", id: "schedules", label: "Schedules", icon: "calendar-linear", perm: "server.schedules.manage" },
  { section: "System", id: "activity", label: "Activity", icon: "history-linear", perm: "server.configuration.manage" },
  { section: "System", id: "users", label: "Users", icon: "users-group-two-rounded-linear", perm: "users.manage" },
  { section: "System", id: "security", label: "Security", icon: "shield-keyhole-linear", perm: "users.manage" },
  { section: "System", id: "settings", label: "Settings", icon: "settings-linear", perm: "server.view" },
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
    const label = el("span", { class: "nav-item-label" }, page.label,
      page.id === "players"
        ? el("span", { class: "nav-player-count", id: "nav-player-count" }, `· ${S.onlinePlayerCount ?? "—"}`)
        : null);
    nav.append(el("div", { class: "nav-item", "data-page": page.id, tabindex: "0", onclick: () => navigate(page.id), onkeydown: (e) => {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(page.id); }
    } }, el("span", { class: "nav-icon", "aria-hidden": "true" }, solarIcon(page.icon)), label));
  }
}

function setOnlinePlayerCount(players) {
  S.onlinePlayerCount = (players || []).filter((player) => player.online).length;
  const count = $("#nav-player-count");
  if (count) count.textContent = `· ${S.onlinePlayerCount}`;
}

async function refreshPlayerCount() {
  if (!can("server.players.view")) return;
  try { setOnlinePlayerCount((await api("/players")).players || []); } catch {}
}

function pageAllowed(page) {
  const entry = PAGES.find((p) => p.id === page);
  return !!entry && (!entry.perm || can(entry.perm));
}

function defaultPage() {
  return (PAGES.find((p) => !p.perm || can(p.perm)) || PAGES[0]).id;
}

function hashPage() {
  let page = "";
  try { page = decodeURIComponent((location.hash || "").replace(/^#/, "")).trim(); }
  catch { page = ""; }
  return pageAllowed(page) ? page : "";
}

function syncHash(page, replace) {
  if (location.hash === "#" + page) return;
  const url = new URL(location.href);
  url.hash = page;
  const fn = replace ? "replaceState" : "pushState";
  history[fn](null, "", url);
}

function navigate(page, opts = {}) {
  const next = pageAllowed(page) ? page : defaultPage();
  S.page = next;
  S.performanceTarget = next === "performance" ? (opts.performanceTarget || "") : "";
  S.serverTargetId = next === "servers" ? (opts.serverTargetId ?? null) : null;
  S.managedServerId = next === "files" || next === "configuration" ? (opts.serverId ?? null) : null;
  S.overviewReturn = !!opts.fromOverview && (next === "players" || next === "servers");
  setSidebarOpen(false);
  if (!opts.fromHash) syncHash(next, !!opts.replaceHash);
  syncPageSubscription(next);
  document.querySelectorAll(".nav-item").forEach((n) =>
    n.classList.toggle("active", n.dataset.page === next));
  renderPage();
}

function serverScopedPath(path) {
  if (!S.managedServerId) return path;
  return `${path}${path.includes("?") ? "&" : "?"}server_id=${encodeURIComponent(S.managedServerId)}`;
}

function navigateFromHash() {
  if (!S.me) return;
  const page = hashPage();
  if (page) navigate(page, { fromHash: true });
  else navigate(defaultPage(), { replaceHash: true });
}

window.addEventListener("hashchange", navigateFromHash);
window.addEventListener("popstate", navigateFromHash);

function escapeBack() {
  if (!S.me || S.page !== "files" || !fileEscapeAction) return false;
  fileEscapeAction();
  return true;
}

document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape" || event.defaultPrevented) return;
  if (closeActiveModal() || closeMobileSidebar() || escapeBack()) event.preventDefault();
});

async function refreshServers() {
  try {
    const d = await api("/servers");
    S.servers = d.servers || [];
    S.activeId = d.active_id || 0;
    const st = await api("/server/status").catch(() => null);
    if (st) S.status = st;
    renderServerPicker();
  } catch (e) { /* server list may 403 for some roles */ }
}

function lifecyclePendingSettled(state) {
  const pending = S.lifecyclePending;
  if (!pending) return false;
  if (pending.action === "restart" && state !== pending.target) pending.departed = true;
  return state === pending.target && (pending.action !== "restart" || pending.departed);
}

function finishLifecyclePending(pending = S.lifecyclePending) {
  if (!pending || S.lifecyclePending !== pending || !pending.settled) return false;
  if (pending.completionTimer) clearTimeout(pending.completionTimer);
  S.lifecyclePending = null;
  if (S.page === "overview" || S.page === "console") renderPage();
  return true;
}

function armLifecycleCompletionFallback(pending = S.lifecyclePending) {
  if (!pending || S.lifecyclePending !== pending || !pending.settled) return;
  if (pending.completionTimer) clearTimeout(pending.completionTimer);
  const startedAt = Number(pending.cycleStartedAt) || Date.now();
  const elapsed = Math.max(0, Date.now() - startedAt);
  const remaining = LIFECYCLE_LOADING_CYCLE_MS - (elapsed % LIFECYCLE_LOADING_CYCLE_MS);
  pending.completionTimer = setTimeout(
    () => finishLifecyclePending(pending),
    remaining,
  );
}

function markLifecyclePendingSettled(state = S.status.state) {
  const pending = S.lifecyclePending;
  if (!pending || !lifecyclePendingSettled(state)) return false;
  if (!pending.settled) {
    pending.settled = true;
    armLifecycleCompletionFallback(pending);
  }
  return true;
}

function renderServerPicker() {
  const host = $("#server-picker"); host.innerHTML = "";
  const active = S.servers.find((s) => s.id === S.activeId);
  host.append(
    el("div", { class: "server-picker-head" },
      el("span", { class: "server-kicker" }, "Active project"),
      renderStatusPillNode({ id: "status-pill" })),
    el("div", { class: "server-name", title: active ? active.display_name : "None selected" },
      active ? active.display_name : "None selected"));
}

function renderStatusPillNode(opts = {}) {
  const st = S.status.state || "stopped";
  const attrs = { class: "status-label " + st + (opts.compact ? " compact" : "") };
  if (opts.id) attrs.id = opts.id;
  return el("div", attrs,
    el("span", { class: "status-square", "aria-hidden": "true" }), st.charAt(0).toUpperCase() + st.slice(1));
}
function renderStatusPill() {
  const p = $("#status-pill");
  if (p) p.replaceWith(renderStatusPillNode({ id: "status-pill" }));
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
      case "settings": return await pageSettings(main);
    }
  } catch (err) {
    main.innerHTML = "";
    main.append(el("h1", {}, "Something went wrong"), el("p", { class: "muted" }, err.message));
  }
}

function overviewBackButton() {
  if (!S.overviewReturn) return null;
  return el("button", {
    class: "btn ghost page-back-button", type: "button",
    "aria-label": "Back to Overview", title: "Back to Overview",
    onclick: () => navigate("overview"),
  }, solarIcon("alt-arrow-left-linear"));
}

function pageHeader(title, subtitle, actions = [], leading = null) {
  const heading = el("h1", {}, title);
  const titleNode = leading
    ? el("div", { class: "title has-leading" },
        el("div", { class: "page-title-heading-row" }, leading, heading),
        subtitle ? el("p", { class: "page-sub" }, subtitle) : null)
    : el("div", { class: "title" },
        heading,
        subtitle ? el("p", { class: "page-sub" }, subtitle) : null);
  return el("div", { class: "page-header" },
    titleNode,
    el("div", { class: "spacer" }),
    actions.length ? el("div", { class: "actions" }, actions) : null);
}

function projectContextSubtitle(prefix, project, isActive, includeArticle = true) {
  if (!project) return "No server project selected.";
  const state = isActive ? "Active" : "Non-Active";
  return el("span", { class: "project-context" },
    prefix,
    includeArticle ? (isActive ? " an " : " a ") : " ",
    el("strong", { class: "project-context-state" }, state),
    " project “",
    el("span", { class: "project-context-name" }, project.display_name),
    "”.");
}

// ----- overview -------------------------------------------------------------
async function pageOverview(main) {
  const d = await api("/overview");
  S.status = { state: d.state, detail: d.supervisor };
  renderStatusPill();
  const s = d.sample || {};
  setUptimeBaseline(s);
  const inst = d.instance;

  // Health, host and trends live here together. Knowing whether the server is
  // healthy should not require visiting three tabs.
  let host = null, events = [], history = [], players = null;
  try { host = await api("/host"); } catch {}
  try { events = (await api("/events?limit=25")).events || []; } catch {}
  try { history = await api("/metrics?hours=1") || []; } catch {}
  try { players = (await api("/players")).players || []; } catch {}
  if (players) setOnlinePlayerCount(players);
  S.perf = [];
  history.forEach(appendPerformanceSample);
  appendPerformanceSample(s);

  const hostMemTotal = Number(s.host_mem_total || host?.mem_total) || 0;
  const hostMemAvailable = Number(s.host_mem_avail || host?.mem_available) || 0;
  const memUsed = hostMemTotal > 0 ? hostMemTotal - hostMemAvailable : 0;
  const diskTotal = Number(host?.disk_total || s.disk_total) || 0;
  const diskFree = Number(host?.disk_free || s.disk_free) || 0;
  const hostCPU = Number(s.host_cpu_percent);
  const loadAverage = Number(s.load1 ?? host?.load1);
  const onlinePlayers = (players || []).filter((player) => player.online);
  const onlineCount = players ? onlinePlayers.length : Number(s.online_players || 0);
  const maxPlayers = Number(d.max_players || s.max_players || 20);

  main.innerHTML = "";
  main.append(
    pageHeader(inst ? inst.display_name : "Overview", "Server state, resource pressure, backups, and recent events for the active project.", [
      lifecycleButtons(true),
    ]),

    // What is happening right now.
    el("div", { class: "grid cols-4 overview-stat-grid" },
      serverStatusCard(d.state, inst),
      statCard("Uptime", currentUptimeSeconds() === null ? "—" : fmtDur(currentUptimeSeconds()), s.java_pid ? "Java PID " + s.java_pid : "not running", "uptime-value"),
      playerSummaryCard(onlinePlayers, onlineCount, maxPlayers),
      statCard("CPU", Number.isFinite(hostCPU) ? hostCPU.toFixed(1) + "%" : "—", "whole-machine average",
        "overview-live-cpu", "performance-host-cpu-card")),

    // Host health, previously a separate tab.
    el("div", { class: "grid cols-4 flow-section overview-stat-grid" },
      statCard("Process memory", fmtBytes(s.rss_bytes), "resident set (not Java heap)",
        "overview-live-rss", "allocated-memory-card"),
      statCard("Host memory", hostMemTotal > 0 ? fmtBytes(memUsed) : "—",
        hostMemTotal > 0 ? "of " + fmtBytes(hostMemTotal) : "",
        "overview-live-host-memory", "host-memory-card"),
      statCard("Disk free", diskTotal > 0 ? fmtBytes(diskFree) : "—",
        diskTotal > 0 ? "of " + fmtBytes(diskTotal) : "Visit Performance to measure",
        "overview-live-disk-free", "performance-machine-storage-card"),
      statCard("Load average", Number.isFinite(loadAverage) ? loadAverage.toFixed(2) : "—", "1 minute",
        "overview-live-load", "performance-load-card")),

    // Trends, previously the Performance tab.
    el("div", { class: "grid cols-2 flow-section" },
      trendCard("CPU", S.perf, (x) => x.host_cpu_percent, (v) => v.toFixed(0) + "%", "overview-trend-cpu",
        { min: 0, max: 100, axisFormat: (value) => value.toFixed(0) + "%" }),
      trendCard("Process memory", S.perf, (x) => x.rss_bytes, fmtBytes, "overview-trend-memory",
        { min: 0, max: overviewMemoryCeiling(S.perf), axisFormat: fmtBytes })),

    el("div", { class: "grid cols-2 flow-section" },
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
          el("dt", {}, "LAN IP"), el("dd", {},
            d.lan_ip ? el("button", {
              class: "copy-value mono", type: "button", title: "Copy LAN IP",
              "aria-label": `Copy LAN IP ${d.lan_ip}`,
              onclick: () => copyText(d.lan_ip, "LAN IP copied"),
            }, el("span", {}, d.lan_ip), solarIcon("copy-linear")) : "—"),
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

async function copyText(value, successMessage) {
  try {
    await navigator.clipboard.writeText(value);
  } catch {
    const input = el("textarea", { style: "position:fixed;opacity:0;pointer-events:none" });
    input.value = value;
    document.body.append(input);
    input.select();
    document.execCommand("copy");
    input.remove();
  }
  toast(successMessage, "ok");
}

// eventRow renders one timeline entry, coloured by severity.
function eventRow(e) {
  return el("li", { class: "timeline-item sev-" + (e.severity || "info") },
    el("span", { class: "timeline-time mono" }, fmtTime(e.occurred_at)),
    el("span", { class: "timeline-msg" }, e.message || e.event));
}

// trendCard draws a compact interactive sparkline plus the latest value.
function trendCard(title, samples, pick, fmt, id = "", chartOptions = {}) {
  const points = (samples || []).map((sample) => ({
    timestamp: sampleTimestamp(sample),
    value: Number(pick(sample)),
  })).filter((point) => point.timestamp && Number.isFinite(point.value));
  const latest = points.length ? points[points.length - 1].value : 0;
  const attrs = { class: "card graph-card" };
  if (id) attrs.id = id;
  return el("div", attrs,
    el("div", { class: "metric-label" }, title),
    el("div", { class: "metric-value" }, fmt(latest)),
    el("div", { class: "metric-note" }, "last hour"),
    points.length > 1 ? overviewSparklineNode(title, points, fmt, chartOptions) : el("p", { class: "muted" }, "Collecting samples…"));
}

function overviewMemoryCeiling(samples) {
  return Math.max(1, ...(samples || []).flatMap((sample) =>
    [Number(sample.rss_bytes), Number(sample.jvm_xmx_bytes)].filter(Number.isFinite)));
}

function updateOverviewTrendCharts() {
  if (S.page !== "overview") return;
  const cpu = $("#overview-trend-cpu");
  const memory = $("#overview-trend-memory");
  cpu?.replaceWith(trendCard("CPU", S.perf, (sample) => sample.host_cpu_percent,
    (value) => value.toFixed(0) + "%", "overview-trend-cpu",
    { min: 0, max: 100, axisFormat: (value) => value.toFixed(0) + "%" }));
  memory?.replaceWith(trendCard("Process memory", S.perf, (sample) => sample.rss_bytes,
    fmtBytes, "overview-trend-memory",
    { min: 0, max: overviewMemoryCeiling(S.perf), axisFormat: fmtBytes }));
}

function statCard(title, value, sub, valueId = "", performanceTarget = "") {
  const valueAttrs = { class: "metric-value" };
  if (valueId) valueAttrs.id = valueId;
  const attrs = { class: "card metric" + (performanceTarget ? " overview-performance-card" : "") };
  if (performanceTarget) {
    attrs.href = "#performance";
    attrs["aria-label"] = `${title}: ${value}. Open in Performance.`;
    attrs.onclick = (event) => {
      event.preventDefault();
      navigate("performance", { performanceTarget });
    };
  }
  return el(performanceTarget ? "a" : "div", attrs,
    el("div", { class: "metric-label" + (performanceTarget ? " overview-performance-label-row" : "") },
      title, performanceTarget ? solarIcon("alt-arrow-right-linear", "player-summary-arrow") : null),
    el("div", valueAttrs, String(value)),
    el("div", { class: "metric-note" }, sub || ""));
}

function overviewPlayerFace(player, moreCount = 0) {
  const username = player?.username || "MHF_Steve";
  const fallback = el("span", { class: "overview-player-face-fallback", "aria-hidden": "true" },
    String(username).charAt(0).toUpperCase());
  const image = el("img", {
    src: getPlayerFaceUrl(username), alt: "", width: "36", height: "36",
    loading: "lazy", decoding: "async", referrerpolicy: "no-referrer",
    onerror: () => handlePlayerFaceError(image, fallback, username, 64),
  });
  return el("span", {
    class: "overview-player-face" + (moreCount ? " is-more" : ""),
    title: moreCount ? `${moreCount} more online` : username,
  }, image, moreCount ? el("span", { class: "overview-player-more" }, `+${moreCount}`) : null);
}

function playerFaceStack(players, capacity) {
  if (!players.length) return null;
  const nodes = [];
  if (players.length <= capacity) {
    nodes.push(...players.slice(0, capacity).map((player) => overviewPlayerFace(player)));
  } else {
    const normalCount = Math.max(0, capacity - 1);
    nodes.push(...players.slice(0, normalCount).map((player) => overviewPlayerFace(player)));
    nodes.push(overviewPlayerFace(players[normalCount], players.length - normalCount));
  }
  return el("span", { class: `player-face-stack capacity-${capacity}`, "aria-hidden": "true" }, ...nodes);
}

function playerSummaryCard(players, onlineCount, maxPlayers) {
  const value = `${onlineCount} / ${maxPlayers}`;
  return el("a", {
    class: "card metric player-summary-card", href: "#players",
    "aria-label": `${onlineCount} of ${maxPlayers} players online. Open players.`,
    onclick: (event) => { event.preventDefault(); navigate("players", { fromOverview: true }); },
  },
  el("div", { class: "metric-label player-summary-label-row" },
    "Players online", solarIcon("alt-arrow-right-linear", "player-summary-arrow")),
  el("div", { class: "player-summary-main" },
    el("div", { class: "metric-value player-summary-count" }, value),
    el("span", { class: "player-face-variants" },
      playerFaceStack(players, 6), playerFaceStack(players, 5), playerFaceStack(players, 4),
      playerFaceStack(players, 3), playerFaceStack(players, 2), playerFaceStack(players, 1))),
  el("div", { class: "metric-note" }, ""));
}

function serverStatusCard(state, server) {
  const normalized = String(state || "stopped").toLowerCase();
  const label = normalized.charAt(0).toUpperCase() + normalized.slice(1);
  const icon = server
    ? el("span", { class: "server-status-icon" }, serverCardIcon(server))
    : null;
  const targetId = server?.id ?? S.activeId;
  return el("a", {
    class: "card metric server-status-card " + normalized,
    href: "#servers",
    "aria-label": `Server status: ${label}. Open ${server?.display_name || "the active server"} in Servers.`,
    onclick: (event) => {
      event.preventDefault();
      navigate("servers", { fromOverview: true, serverTargetId: targetId });
    },
  },
    icon,
    el("div", { class: "metric-label server-status-label-row" },
      "Server status", solarIcon("alt-arrow-right-linear", "player-summary-arrow")),
    el("div", { class: "metric-value server-status-value" },
      el("span", { class: "server-status-state" },
        el("span", { class: "status-square", "aria-hidden": "true" }), label)),
    el("div", { class: "metric-note" }, ""));
}

function lifecycleButtons(includeServers = false) {
  const st = S.status.state || "stopped";
  const running = st === "running" || st === "starting";
  markLifecyclePendingSettled(st);
  const pending = S.lifecyclePending;
  const wrap = el("div", { class: "row-actions" + (includeServers ? " overview-lifecycle-actions" : "") });
  const showLoading = (button) => {
    const icon = button.querySelector(":scope > .icon");
    const pendingAction = S.lifecyclePending;
    const loadingIcon = lifecycleLoadingIcon(() => finishLifecyclePending(pendingAction));
    if (pendingAction) pendingAction.cycleStartedAt = Date.now();
    if (icon) icon.replaceWith(loadingIcon);
    else button.prepend(loadingIcon);
    button.disabled = true;
    button.setAttribute("aria-busy", "true");
    if (pendingAction?.settled) armLifecycleCompletionFallback(pendingAction);
  };
  const act = async (path, label, action = "", target = "", button = null) => {
    if (action && target) S.lifecyclePending = { action, target, departed: action !== "restart", settled: false, completionTimer: null };
    if (button) showLoading(button);
    try {
      await api(path, { method: "POST", json: {} });
      toast(label + " requested", "ok");
      if (target) markLifecyclePendingSettled(S.status.state);
    } catch (e) {
      if (action && S.lifecyclePending?.action === action) {
        if (S.lifecyclePending.completionTimer) clearTimeout(S.lifecyclePending.completionTimer);
        S.lifecyclePending = null;
      }
      toast(e.message, "err");
      if (S.page === "overview" || S.page === "console") renderPage();
    }
  };
  const lifecycleButton = (action, path, label, target, className) => {
    const button = el("button", {
      class: className,
      onclick: (event) => act(path, label, action, target, event.currentTarget),
    }, label);
    if (pending?.action === action) showLoading(button);
    return button;
  };
  const serversButton = () => el("button", { class: "btn", onclick: () => navigate("servers", { fromOverview: true }) },
    solarIcon("server-square-linear"),
    "Servers",
    solarIcon("alt-arrow-right-linear", "redirect-icon"));
  if (pending) {
    if (includeServers && pending.action === "start") wrap.append(serversButton());
    if (pending.action === "start" && can("server.start"))
      wrap.append(lifecycleButton("start", "/server/start", "Start", "running", "btn primary"));
    if (pending.action === "stop" && can("server.stop"))
      wrap.append(lifecycleButton("stop", "/server/stop", "Stop", "stopped", "btn"));
    if (pending.action === "restart" && can("server.restart"))
      wrap.append(lifecycleButton("restart", "/server/restart", "Restart", "running", "btn"));
    if (includeServers && pending.action === "stop") wrap.append(serversButton());
    return wrap;
  }
  if (includeServers && !running) wrap.append(serversButton());
  if (can("server.start") && !running)
    wrap.append(lifecycleButton("start", "/server/start", "Start", "running", "btn primary"));
  if (can("server.stop") && running)
    wrap.append(lifecycleButton("stop", "/server/stop", "Stop", "stopped", "btn"));
  if (can("server.restart") && running)
    wrap.append(lifecycleButton("restart", "/server/restart", "Restart", "running", "btn"));
  if (can("server.force_stop") && st !== "stopped")
    wrap.append(el("button", { class: "btn danger", onclick: () =>
      confirmModal("Force stop", "Force stop kills the Java process immediately. Unsaved world data may be lost. Continue?",
        "Force stop", async () => {
          try { await api("/server/force-stop", { method: "POST", json: { confirm: true } }); toast("Force stop sent", "ok"); }
          catch (e) { toast(e.message, "err"); }
        }) }, "Force stop"));
  if (includeServers && running) wrap.append(serversButton());
  return wrap;
}

// ----- console --------------------------------------------------------------
async function pageConsole(main) {
  main.innerHTML = "";
  const stopped = (S.status.state || "stopped") === "stopped";
  const box = el("div", { class: "console" + (S.consolePaused ? " paused" : "") + (S.consoleWrap ? " is-wrapped" : ""), id: "console-box", role: "log", "aria-live": S.consolePaused ? "off" : "polite" });
  const search = el("input", { value: S.consoleSearch, placeholder: "Search buffer", "aria-label": "Search console buffer" });
  const input = el("input", { placeholder: can("server.console.use") ? (stopped ? "Start the server to send commands" : "Command, for example: say hello") : "Read-only console", spellcheck: "false", autocomplete: "off" });
  if (!can("server.console.use") || stopped) input.disabled = true;
  search.addEventListener("input", () => { S.consoleSearch = search.value; renderConsoleLines(box); });
  renderConsoleLines(box);
  if (!S.consoleLines.length) renderConsolePlaceholder(box, "Loading console history...");
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
  const toggleConsoleWrap = () => {
    S.consoleWrap = !S.consoleWrap;
    const actionLabel = S.consoleWrap ? "Disable text wrapping" : "Wrap text";
    box.classList.toggle("is-wrapped", S.consoleWrap);
    mobileWrapButton.classList.toggle("is-active", S.consoleWrap);
    desktopWrapButton.classList.toggle("is-on", S.consoleWrap);
    [mobileWrapButton, desktopWrapButton].forEach((button) => {
      button.setAttribute("aria-pressed", String(S.consoleWrap));
      button.setAttribute("aria-label", actionLabel);
      button.setAttribute("title", actionLabel);
    });
    desktopWrapButton.querySelector(".bot-power-label").textContent = S.consoleWrap ? "On" : "Off";
  };
  const mobileWrapButton = el("button", {
    class: "btn ghost console-icon-control console-wrap-control console-wrap-mobile" + (S.consoleWrap ? " is-active" : ""),
    "aria-label": S.consoleWrap ? "Disable text wrapping" : "Wrap text",
    "aria-pressed": String(S.consoleWrap),
    title: S.consoleWrap ? "Disable text wrapping" : "Wrap text",
    onclick: toggleConsoleWrap,
  }, solarIcon("wrap-text"));
  const desktopWrapButton = el("button", {
    class: "bot-power console-wrap-desktop-toggle" + (S.consoleWrap ? " is-on" : ""),
    type: "button",
    "aria-label": S.consoleWrap ? "Disable text wrapping" : "Wrap text",
    "aria-pressed": String(S.consoleWrap),
    title: S.consoleWrap ? "Disable text wrapping" : "Wrap text",
    onclick: toggleConsoleWrap,
  },
  el("span", { class: "bot-power-track", "aria-hidden": "true" }, el("span", {})),
  el("span", { class: "bot-power-label" }, S.consoleWrap ? "On" : "Off"));
  main.append(
    pageHeader("Console", "Live Minecraft output and command entry for the active server.", [renderStatusPillNode(), lifecycleButtons()]),
    el("div", { class: "console-shell" },
      el("div", { class: "console-toolbar" },
        search,
        el("button", {
          class: "btn ghost console-icon-control",
          "aria-label": S.consolePaused ? "Resume console" : "Pause console",
          title: S.consolePaused ? "Resume console" : "Pause console",
          onclick: () => { S.consolePaused = !S.consolePaused; pageConsole(main); },
        }, S.consolePaused ? "Resume" : "Pause"),
        mobileWrapButton,
        el("button", { class: "btn ghost console-icon-control", "aria-label": "Copy console", title: "Copy console", onclick: async () => {
          try { await navigator.clipboard.writeText(S.consoleLines.join("\n")); toast("Console buffer copied", "ok"); }
          catch { toast("Copy failed in this browser", "err"); }
        } }, "Copy"),
        el("button", { class: "btn ghost console-clear-control", onclick: () => { S.consoleHistoryRequest++; box.innerHTML = ""; S.consoleLines = []; } }, "Clear"),
        !DEMO_MODE ? el("span", { class: "status-label " + (S.ws && S.ws.readyState === WebSocket.OPEN ? "running" : "stopped") },
          el("span", { class: "status-square", "aria-hidden": "true" }),
          S.ws && S.ws.readyState === WebSocket.OPEN ? "Connected" : "Reconnecting") : null,
        el("span", { class: "console-wrap-desktop-group" },
          el("span", { class: "console-wrap-desktop-label" }, "Wrap text"),
          desktopWrapButton)),
      box,
      el("div", { class: "console-input" },
        input,
        can("server.console.use") && !stopped ? [
          el("button", { class: "btn ghost", onclick: () => sendQuick("list") }, "list"),
          el("button", { class: "btn ghost", onclick: () => sendQuick("save-all") }, "save-all"),
        ] : null)));
  box.scrollTop = box.scrollHeight;
  loadConsoleHistory(box);
}

async function loadConsoleHistory(box) {
  const requestId = ++S.consoleHistoryRequest;
  const liveStart = S.consoleLines.length;
  try {
    const d = await api(`/server/console/history?limit=${CONSOLE_LINE_LIMIT}`);
    if (requestId !== S.consoleHistoryRequest || S.page !== "console") return;
    const history = Array.isArray(d.lines) ? d.lines : [];
    const liveTail = S.consoleLines.slice(liveStart);
    S.consoleLines = trimConsoleLines(history.concat(liveTail));
    const target = box && box.isConnected ? box : $("#console-box");
    if (!target) return;
    renderConsoleLines(target);
    if (!S.consoleLines.length) renderConsolePlaceholder(target, "No console history yet.");
    if (!S.consolePaused) target.scrollTop = target.scrollHeight;
  } catch (err) {
    console.warn("Console history unavailable:", err);
    const target = box && box.isConnected ? box : $("#console-box");
    if (target && !S.consoleLines.length) renderConsolePlaceholder(target, "Console history unavailable.");
  }
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
  if (q && !box.childNodes.length) renderConsolePlaceholder(box, "No matching console lines.");
}

function renderConsolePlaceholder(box, message) {
  box.innerHTML = "";
  box.append(el("div", { class: "console-line console-placeholder" }, message));
}

function appendConsoleLine(line) {
  const box = $("#console-box");
  if (!box) return;
  if (S.consoleSearch && !String(line).toLowerCase().includes(S.consoleSearch.toLowerCase())) return;
  if (box.querySelector(".console-placeholder")) box.innerHTML = "";
  const atBottom = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
  box.append(consoleLineNode(line));
  while (box.childNodes.length > CONSOLE_LINE_LIMIT) box.firstChild.remove();
  if (!S.consolePaused && atBottom) box.scrollTop = box.scrollHeight;
}

// ----- players --------------------------------------------------------------
async function pagePlayers(main) {
  const d = await api("/players");
  const players = d.players || [];
  setOnlinePlayerCount(players);
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
    pageHeader("Players", "Observed online and recent players. Whitelist, operator, ban, and IP-ban lists are not exposed as separate read APIs yet.", [search], overviewBackButton()),
    el("div", { class: "toolbar" },
      el("span", { class: "status-label running" }, el("span", { class: "status-square" }), players.filter((p) => p.online).length + " online"),
      el("span", { class: "status-label" }, el("span", { class: "status-square" }), players.length + " observed")),
    el("div", { class: "table-wrap players-table" },
      el("table", {},
        el("thead", {}, el("tr", {},
          el("th", {}, "Player"), el("th", {}, "Status"), el("th", { class: "mobile-hide" }, "Last seen"),
          el("th", { class: "mobile-hide" }, "Observed playtime"), el("th", {}, ""))),
        tbody)));
  draw();
}

function playerRow(p) {
  const fallback = el("span", { class: "player-avatar player-avatar-fallback", "aria-hidden": "true" },
    String(p.username || "?").charAt(0).toUpperCase());
  const avatar = el("img", {
    class: "player-avatar",
    src: getPlayerFaceUrl(p.username),
    alt: "",
    width: "32",
    height: "32",
    loading: "lazy",
    decoding: "async",
    referrerpolicy: "no-referrer",
    onerror: () => handlePlayerFaceError(avatar, fallback, p.username),
  });
  return el("tr", {},
    el("td", {}, el("div", { class: "player-identity" }, avatar,
      el("span", { class: "player-name-block" },
        el("span", { class: "player-name-line" },
          el("strong", {}, p.username),
          p.op ? el("span", { class: "player-op" }, "OP") : null),
        el("span", { class: "mobile-only mobile-row-detail" }, p.online ? "Online" : "Offline")))),
    el("td", {}, p.online ? "Online" : "Offline"),
    el("td", { class: "mobile-hide" }, fmtTime(p.last_seen_at)),
    el("td", { class: "mobile-hide" }, fmtDur(p.observed_playtime_seconds)),
    el("td", { class: "table-actions" }, can("server.players.manage") ? playerActions(p) : ""));
}

function getPlayerFaceUrl(username, size = 64) {
  const name = encodeURIComponent(String(username || "MHF_Steve"));
  if (DEMO_MODE) return directPlayerFaceUrl(username, size);
  return `/api/players/avatar?username=${name}&size=${encodeURIComponent(size)}`;
}

function directPlayerFaceUrl(username, size = 64) {
  return `https://minotar.net/helm/${encodeURIComponent(String(username || "MHF_Steve"))}/${size}.png`;
}

function handlePlayerFaceError(image, fallback, username, size = 64) {
  if (!DEMO_MODE && !image.dataset.directFallback) {
    image.dataset.directFallback = "true";
    image.src = directPlayerFaceUrl(username, size);
    return;
  }
  image.replaceWith(fallback);
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
  const actions = [];
  if (p.online) actions.push({ label: "Kick", icon: "close-circle-linear", danger: true, run: act("Kick", true) });
  actions.push(
    { label: "Ban", icon: "lock-keyhole-linear", danger: true, run: act("Ban", true) },
    { label: "Op", icon: "key-linear", run: act("Op", false) },
    { label: "Deop", icon: "key-linear", run: act("Deop", false) });
  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    ...actions.map((action) => el("button", { class: "btn ghost", onclick: action.run }, action.label)));
  const mobile = overflowActionsMenu(`Actions for ${p.username}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
}

// ----- files ----------------------------------------------------------------
let filePath = "";
let fileEscapeAction = null;

const FILE_ICON_GROUPS = [
  ["archive-linear", new Set(["7z", "bz2", "gz", "jar", "rar", "tar", "tgz", "war", "xz", "zip"])],
  ["gallery-linear", new Set(["avif", "bmp", "gif", "ico", "jpeg", "jpg", "png", "svg", "webp"])],
  ["music-note-linear", new Set(["aac", "flac", "m4a", "mp3", "ogg", "opus", "wav"])],
  ["video-frame-linear", new Set(["avi", "m4v", "mkv", "mov", "mp4", "webm"])],
  ["database-linear", new Set(["dat", "db", "mca", "mcr", "nbt", "sqlite", "sqlite3"])],
  ["command-linear", new Set(["bat", "cmd", "mcfunction", "ps1", "sh", "zsh"])],
  ["settings-linear", new Set(["cfg", "conf", "config", "ini", "json", "properties", "toml", "yaml", "yml"])],
  ["code-file-linear", new Set(["c", "cc", "cpp", "cs", "css", "go", "h", "hpp", "htm", "html", "java", "js", "jsx", "kt", "kts", "less", "lua", "php", "py", "rb", "rs", "sass", "scss", "sql", "ts", "tsx", "xml"])],
];

function fileIconName(entry) {
  if (entry.is_dir) return "folder-linear";
  const name = String(entry.name || "").toLowerCase();
  if (name === "eula.txt") return "shield-check-linear";
  if (["banned-ips.json", "banned-players.json", "ops.json", "usercache.json", "whitelist.json"].includes(name)) {
    return "users-group-rounded-linear";
  }
  if (name === "server-icon.png") return "gallery-linear";
  const extension = name.includes(".") ? name.split(".").pop() : "";
  for (const [icon, extensions] of FILE_ICON_GROUPS) {
    if (extensions.has(extension)) return icon;
  }
  return "document-text-linear";
}

function fileIdentity(entry) {
  return el("span", { class: "file-identity" },
    el("span", { class: "file-type-icon", "aria-hidden": "true" }, solarIcon(fileIconName(entry))),
    el("span", {}, entry.name));
}

async function pageFiles(main, path = filePath) {
  // A deep link from elsewhere (for example the Configuration page naming the
  // file that owns the JVM settings) opens that file straight away.
  if (S.pendingFileOpen) {
    const target = S.pendingFileOpen;
    S.pendingFileOpen = null;
    return openFileEditor(main, target);
  }
  filePath = path;
  fileEscapeAction = path
    ? () => pageFiles(main, path.split("/").filter(Boolean).slice(0, -1).join("/"))
    : null;
  const entries = await api(serverScopedPath("/files?path=" + encodeURIComponent(path)));
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
      await fetch("/api" + serverScopedPath("/files/upload?path=" + encodeURIComponent(path)),
        { method: "POST", body: fd, headers: { "X-Bonghos-CSRF": csrfToken }, credentials: "same-origin" });
      toast("Uploaded", "ok"); pageFiles(main, path);
    } catch (e) { toast(e.message, "err"); }
  });
  const fileRow = (e2) => el("tr", {},
    el("td", { class: "mono", style: "cursor:pointer", onclick: () => {
      if (e2.is_dir) pageFiles(main, (path ? path + "/" : "") + e2.name);
      else openFileEditor(main, (path ? path + "/" : "") + e2.name);
    } }, fileIdentity(e2)),
    el("td", { class: "file-size-column" }, e2.is_dir ? "—" : fmtBytes(e2.size)),
    el("td", { class: "mobile-hide" }, fmtTime(e2.mod_time)),
    el("td", { class: "table-actions file-actions-cell" }, fileActions(main, path, e2)));
  const tbody = el("tbody");
  const draw = (query = "") => {
    const visible = (entries || []).filter((entry) => recordMatchesSearch(entry, query, fmtBytes(entry.size), fmtTime(entry.mod_time)));
    tbody.replaceChildren(...(visible.length
      ? visible.map(fileRow)
      : [el("tr", {}, el("td", { colspan: "4", class: "muted" }, (entries || []).length ? "No matching files." : "Empty directory"))]));
  };
  const search = pageSearchInput("files", draw);
  const project = S.servers.find((server) => server.id === S.managedServerId)
    || S.servers.find((server) => server.id === S.activeId);
  const subtitle = projectContextSubtitle("Currently in", project, project?.id === S.activeId);
  main.append(
    pageHeader("Files", subtitle, [
      search,
      el("button", { class: "btn", title: "Upload", onclick: () => upInput.click() }, solarIcon("upload-linear"), "Upload"),
      el("button", { class: "btn", title: "New folder", onclick: () => mkdirPrompt(main, path) }, solarIcon("folder-linear"), "New folder"),
      upInput,
    ]),
    crumbs,
    el("div", { class: "file-list" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", { class: "file-size-column" }, "Size"), el("th", { class: "mobile-hide" }, "Modified"), el("th", {}, ""))),
        tbody)));
  draw();
}

function fileActions(main, path, entry) {
  const rel = (path ? path + "/" : "") + entry.name;
  const download = !entry.is_dir ? "/api" + serverScopedPath("/files/download?path=" + encodeURIComponent(rel)) : "";
  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    download ? el("a", { class: "btn ghost", href: download }, solarIcon("download-linear"), "Download") : "",
    el("button", { class: "btn ghost", onclick: () => renameEntry(main, path, entry.name) }, "Rename"),
    el("button", { class: "btn danger", onclick: () => deleteEntry(main, path, entry.name) }, "Delete"));
  const items = [];
  if (download) items.push(el("a", { class: "action-menu-item", role: "menuitem", href: download }, solarIcon("download-linear"), "Download"));
  items.push(
    el("button", { class: "action-menu-item", type: "button", role: "menuitem", onclick: () => renameEntry(main, path, entry.name) }, solarIcon("pen-new-square-linear"), "Rename"),
    el("button", { class: "action-menu-item danger", type: "button", role: "menuitem", onclick: () => deleteEntry(main, path, entry.name) }, solarIcon("trash-bin-trash-linear"), "Delete"));
  return el("div", { class: "responsive-row-actions" }, desktop,
    overflowActionsMenu(`Actions for ${entry.name}`, items, "mobile-row-actions"));
}

async function openFileEditor(main, rel) {
  let data;
  try { data = await api(serverScopedPath("/files/content?path=" + encodeURIComponent(rel))); }
  catch (e) { toast(e.message, "err"); return; }
  main.innerHTML = "";
  const ta = el("textarea", { class: "editor", spellcheck: "false" });
  ta.value = data.content;
  let baseline = data.content;
  const leaveEditor = () => {
    if (ta.value === baseline) {
      pageFiles(main);
      return;
    }
    confirmModal("Discard changes", `Discard unsaved changes to "${rel}"?`, "Discard", async () => pageFiles(main));
  };
  fileEscapeAction = leaveEditor;
  main.append(
    el("div", { class: "toolbar" },
      el("h1", { class: "mono", style: "font-size:1rem" }, rel),
      el("div", { class: "spacer" }),
      el("button", { class: "btn ghost", title: "Back to files", onclick: leaveEditor }, solarIcon("folder-open-linear"), "Back"),
      el("button", { class: "btn primary", title: "Save file", onclick: async () => {
        try {
          await api(serverScopedPath("/files/content"), { method: "POST", json: { path: rel, content: ta.value } });
          baseline = ta.value;
          toast("Saved (a .bonghos-backup copy of important files is kept)", "ok");
        } catch (e) { toast(e.message, "err"); }
      } }, solarIcon("diskette-linear"), "Save")),
    ta);
}

function mkdirPrompt(main, path) {
  const inp = el("input", { placeholder: "folder-name" });
  modal("New folder", [el("div", { class: "field-row" }, inp)], [
    ["Cancel", "ghost", (c) => c()],
    ["Create", "primary", async (c) => {
      c();
      try { await api(serverScopedPath("/files/mkdir"), { method: "POST", json: { path: (path ? path + "/" : "") + inp.value } }); pageFiles(main, path); }
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
      try { await api(serverScopedPath("/files/rename"), { method: "POST", json: { from, to } }); pageFiles(main, path); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function deleteEntry(main, path, name) {
  const rel = (path ? path + "/" : "") + name;
  confirmModal("Delete", `Delete ${rel}? This cannot be undone.`, "Delete", async () => {
    try { await api(serverScopedPath("/files/delete"), { method: "POST", json: { path: rel, confirm: true } }); pageFiles(main, path); }
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
  navigate("files", { serverId: S.managedServerId });
}

const SERVER_ICON_MAX_BYTES = 10 * 1024 * 1024;
const SERVER_ICON_TYPES = new Set(["image/png", "image/jpeg", "image/webp"]);

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () => resolve(reader.result));
    reader.addEventListener("error", () => reject(new Error("Could not read that image")));
    reader.readAsDataURL(file);
  });
}

function loadCropImage(src) {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.addEventListener("load", () => resolve(image));
    image.addEventListener("error", () => reject(new Error("Could not decode that image")));
    image.src = src;
  });
}

async function openServerIconCropper(file, onCropped) {
  if (!SERVER_ICON_TYPES.has(file.type)) {
    return toast("Choose a PNG, JPEG, or WebP image", "err");
  }
  if (file.size > SERVER_ICON_MAX_BYTES) {
    return toast("Server icons must be 10 MiB or smaller", "err");
  }

  let sourceURL;
  let image;
  try {
    sourceURL = await readFileAsDataURL(file);
    image = await loadCropImage(sourceURL);
  } catch (error) {
    return toast(error.message, "err");
  }

  const stageSize = 320;
  const canvas = el("canvas", {
    class: "server-icon-crop-canvas",
    width: stageSize,
    height: stageSize,
    role: "img",
    "aria-label": "Cropped server icon preview",
  });
  const context = canvas.getContext("2d");
  const zoom = el("input", { type: "range", min: "100", max: "400", step: "1", value: "100", "aria-label": "Crop zoom" });
  const zoomValue = el("span", { class: "mono" }, "100%");
  let cropSize = Math.min(image.naturalWidth, image.naturalHeight);
  let centerX = image.naturalWidth / 2;
  let centerY = image.naturalHeight / 2;
  let drag = null;

  function cropRect() {
    const size = Math.max(1, Math.min(image.naturalWidth, image.naturalHeight, cropSize));
    centerX = Math.max(size / 2, Math.min(image.naturalWidth - size / 2, centerX));
    centerY = Math.max(size / 2, Math.min(image.naturalHeight - size / 2, centerY));
    return { x: centerX - size / 2, y: centerY - size / 2, size };
  }

  function drawCrop() {
    const crop = cropRect();
    context.clearRect(0, 0, stageSize, stageSize);
    context.imageSmoothingEnabled = true;
    context.imageSmoothingQuality = "high";
    context.drawImage(image, crop.x, crop.y, crop.size, crop.size, 0, 0, stageSize, stageSize);
    context.save();
    context.strokeStyle = "rgba(255, 255, 255, 0.38)";
    context.lineWidth = 1;
    context.beginPath();
    for (const point of [stageSize / 3, stageSize * 2 / 3]) {
      context.moveTo(point, 0); context.lineTo(point, stageSize);
      context.moveTo(0, point); context.lineTo(stageSize, point);
    }
    context.stroke();
    context.restore();
  }

  zoom.addEventListener("input", () => {
    const factor = Number(zoom.value) / 100;
    cropSize = Math.min(image.naturalWidth, image.naturalHeight) / factor;
    zoomValue.textContent = `${zoom.value}%`;
    drawCrop();
  });
  canvas.addEventListener("pointerdown", (event) => {
    event.preventDefault();
    canvas.setPointerCapture(event.pointerId);
    drag = { pointerId: event.pointerId, x: event.clientX, y: event.clientY };
    canvas.classList.add("is-dragging");
  });
  canvas.addEventListener("pointermove", (event) => {
    if (!drag || drag.pointerId !== event.pointerId) return;
    const bounds = canvas.getBoundingClientRect();
    const visible = cropRect().size;
    centerX -= (event.clientX - drag.x) * visible / bounds.width;
    centerY -= (event.clientY - drag.y) * visible / bounds.height;
    drag.x = event.clientX;
    drag.y = event.clientY;
    drawCrop();
  });
  const endDrag = (event) => {
    if (!drag || drag.pointerId !== event.pointerId) return;
    drag = null;
    canvas.classList.remove("is-dragging");
  };
  canvas.addEventListener("pointerup", endDrag);
  canvas.addEventListener("pointercancel", endDrag);
  drawCrop();

  const stage = el("div", { class: "server-icon-crop-stage" }, canvas);
  const controls = el("div", { class: "server-icon-crop-controls" },
    el("div", { class: "server-icon-crop-label" }, el("span", {}, "Zoom"), zoomValue),
    zoom,
    el("p", { class: "hint" }, `Drag to reposition. The saved icon will be converted from ${image.naturalWidth}×${image.naturalHeight} to 64×64 PNG.`));

  modal("Crop server icon", [stage, controls], [
    ["Cancel", "ghost", (close) => close()],
    ["Use crop", "primary", (close) => {
      const crop = cropRect();
      const size = Math.max(1, Math.floor(crop.size));
      const x = Math.max(0, Math.min(image.naturalWidth - size, Math.round(crop.x)));
      const y = Math.max(0, Math.min(image.naturalHeight - size, Math.round(crop.y)));
      const output = document.createElement("canvas");
      output.width = 64;
      output.height = 64;
      const outputContext = output.getContext("2d");
      outputContext.imageSmoothingEnabled = true;
      outputContext.imageSmoothingQuality = "high";
      outputContext.drawImage(image, x, y, size, size, 0, 0, 64, 64);
      onCropped({ file, x, y, size, previewURL: output.toDataURL("image/png") });
      close();
    }],
  ]);
}

function serverIconConfigurationCard(server, onChange) {
  const preview = el("div", { class: "configuration-server-icon-preview" }, serverCardIcon(server));
  const showPending = (pending) => preview.replaceChildren(el("img", {
    class: "server-card-icon",
    src: pending.previewURL,
    alt: "Pending server icon",
    width: 64,
    height: 64,
  }));
  const input = el("input", {
    class: "server-icon-file-input",
    type: "file",
    accept: "image/png,image/jpeg,image/webp",
    tabindex: "-1",
  });
  input.addEventListener("change", () => {
    const file = input.files && input.files[0];
    input.value = "";
    if (file) openServerIconCropper(file, (pending) => {
      showPending(pending);
      onChange(pending);
    });
  });
  const changeButton = el("button", { class: "btn", onclick: () => input.click() },
    solarIcon("gallery-linear"), "Change icon");
  const card = el("div", { class: "card configuration-server-icon-card" },
    preview,
    el("div", { class: "configuration-server-icon-copy" },
      el("strong", {}, "Minecraft server icon"),
      el("p", { class: "muted" }, "Upload a PNG, JPEG, or WebP image. Crop it here, then click Save changes to store the 64×64 PNG."),
      can("server.icon.manage") ? el("div", { class: "actions" }, changeButton, input) :
        el("p", { class: "hint" }, "You do not have permission to change this icon.")));
  card.resetPreview = () => preview.replaceChildren(serverCardIcon(server));
  return card;
}

async function pageConfiguration(main) {
  const d = await api(serverScopedPath("/configuration"));
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
  const propInputs = {};
  const propRows = commonProps.filter((k) => k in props).map((k) => {
    const v = el("input", { value: props[k] });
    propInputs[k] = v;
    return el("div", { class: "field-row" }, el("label", {}, k, v));
  });

  const auto = el("input", { type: "checkbox" }); auto.checked = !!inst.autostart_enabled;
  const recover = el("input", { type: "checkbox" }); recover.checked = !!inst.recover_after_unclean_shutdown;
  const delay = el("input", { type: "number", min: "0", value: inst.boot_delay_seconds || 0, style: "width:110px" });
  const policy = el("select", {},
    ...["never", "on-failure", "always"].map((p) => el("option", { value: p, selected: p === (inst.restart_policy || "never") ? "" : null }, p)));

  const automationCard = el("div", { class: "card" },
    el("div", { class: "field-row" }, el("label", { class: "check-row" }, auto, " Start this server when the machine boots")),
    el("div", { class: "field-row" }, el("label", { class: "check-row" }, recover, " Recover after unclean shutdown (power loss)")),
    el("div", { class: "field-row" }, el("label", {}, "Boot delay (seconds)", delay)),
    el("div", { class: "field-row" }, el("label", {}, "Crash restart policy", policy),
      el("span", { class: "hint" }, "Crash-loop protection pauses automatic restarts after repeated rapid crashes.")));

  let pendingIcon = null;
  let iconChangeVersion = 0;
  const iconCard = serverIconConfigurationCard(inst, (pending) => {
    pendingIcon = pending;
    iconChangeVersion++;
    updateActions();
  });

  const currentState = () => ({
    xms: xms.value,
    xmx: xmx.value,
    startupScript: scriptSel.value,
    javaSelection: javaSel.value,
    properties: Object.fromEntries(Object.entries(propInputs).map(([key, input]) => [key, input.value])),
    autostartEnabled: auto.checked,
    recoverAfterUncleanShutdown: recover.checked,
    bootDelaySeconds: delay.value,
    restartPolicy: policy.value,
    iconChangeVersion,
  });
  let baseline = currentState();
  let saving = false;

  const saveButtons = [];
  const discardButtons = [];
  const bottomActions = el("div", { class: "configuration-action-bar hidden" });

  function isDirty() {
    return JSON.stringify(currentState()) !== JSON.stringify(baseline);
  }

  function updateActions() {
    const dirty = isDirty();
    saveButtons.forEach((button) => {
      button.disabled = saving || !dirty;
      setButtonLabel(button, saving ? "Saving..." : "Save changes");
    });
    discardButtons.forEach((button) => { button.disabled = saving || !dirty; });
    bottomActions.classList.toggle("hidden", !dirty);
  }

  function discardChanges() {
    xms.value = baseline.xms;
    xmx.value = baseline.xmx;
    scriptSel.value = baseline.startupScript;
    javaSel.value = baseline.javaSelection;
    Object.entries(propInputs).forEach(([key, input]) => { input.value = baseline.properties[key]; });
    auto.checked = baseline.autostartEnabled;
    recover.checked = baseline.recoverAfterUncleanShutdown;
    delay.value = baseline.bootDelaySeconds;
    policy.value = baseline.restartPolicy;
    pendingIcon = null;
    iconChangeVersion = baseline.iconChangeVersion;
    iconCard.resetPreview();
    updateActions();
  }

  async function saveChanges() {
    if (saving || !isDirty()) return;
    const next = currentState();
    saving = true;
    updateActions();
    try {
      let restartRequired = false;
      if (next.xms !== baseline.xms || next.xmx !== baseline.xmx) {
        await api(serverScopedPath("/configuration/jvm"), { method: "POST", json: { xms: next.xms, xmx: next.xmx } });
        restartRequired = true;
      }
      if (next.startupScript !== baseline.startupScript) {
        await api(serverScopedPath("/configuration/startup-script"), { method: "POST", json: { script: next.startupScript } });
      }

      const instanceChanges = {};
      if (next.javaSelection !== baseline.javaSelection) instanceChanges.java_selection = next.javaSelection;
      if (next.autostartEnabled !== baseline.autostartEnabled) instanceChanges.autostart_enabled = next.autostartEnabled;
      if (next.recoverAfterUncleanShutdown !== baseline.recoverAfterUncleanShutdown) instanceChanges.recover_after_unclean_shutdown = next.recoverAfterUncleanShutdown;
      if (next.bootDelaySeconds !== baseline.bootDelaySeconds) instanceChanges.boot_delay_seconds = Number(next.bootDelaySeconds);
      if (next.restartPolicy !== baseline.restartPolicy) instanceChanges.restart_policy = next.restartPolicy;
      if (Object.keys(instanceChanges).length) {
        await api(`/servers/${inst.id}`, { method: "PATCH", json: instanceChanges });
      }

      for (const [key, value] of Object.entries(next.properties)) {
        if (value === baseline.properties[key]) continue;
        await api(serverScopedPath("/configuration/property"), { method: "POST", json: { key, value } });
        restartRequired = true;
      }

      if (pendingIcon && next.iconChangeVersion !== baseline.iconChangeVersion) {
        if (DEMO_MODE) {
          const demoServer = DEMO_SERVERS.find((item) => item.id === inst.id);
          if (demoServer) {
            demoServer.demo_icon = pendingIcon.previewURL;
            demoServer.icon_revision = (demoServer.icon_revision || 0) + 1;
            inst.demo_icon = demoServer.demo_icon;
            inst.icon_revision = demoServer.icon_revision;
          }
        } else {
          const form = new FormData();
          form.append("icon", pendingIcon.file, pendingIcon.file.name);
          form.append("crop", `${pendingIcon.x},${pendingIcon.y},${pendingIcon.size}`);
          const result = await api(`/servers/${inst.id}/icon`, { method: "POST", body: form });
          inst.icon_revision = result.icon_revision;
          const cached = S.servers.find((item) => item.id === inst.id);
          if (cached) cached.icon_revision = result.icon_revision;
        }
        pendingIcon = null;
        iconCard.resetPreview();
      }

      baseline = currentState();
      toast("Configuration saved" + (restartRequired ? " - restart required to apply some changes" : ""), "ok");
    } catch (e) {
      toast(e.message, "err");
    } finally {
      saving = false;
      updateActions();
    }
  }

  function actionButton(label, className, handler, collection) {
    const button = el("button", { class: "btn " + className, onclick: handler, disabled: "" }, label);
    collection.push(button);
    return button;
  }

  const headerDiscard = actionButton("Discard", "ghost", discardChanges, discardButtons);
  const headerSave = actionButton("Save changes", "primary", saveChanges, saveButtons);
  const bottomDiscard = actionButton("Discard", "ghost", discardChanges, discardButtons);
  const bottomSave = actionButton("Save changes", "primary", saveChanges, saveButtons);
  bottomActions.append(bottomDiscard, bottomSave);

  const controls = [xms, xmx, scriptSel, javaSel, auto, recover, delay, policy, ...Object.values(propInputs)];
  controls.forEach((control) => {
    control.addEventListener("input", updateActions);
    control.addEventListener("change", updateActions);
  });

  const openServerProperties = () => {
    if (!isDirty()) return openFileInEditor("server.properties");
    confirmModal(
      "Open server.properties",
      "Open server.properties and discard the unsaved changes on this page?",
      "Open file",
      async () => openFileInEditor("server.properties"),
      false,
    );
  };

  main.append(
    pageHeader("Configuration", projectContextSubtitle("Editing", inst, inst.id === S.activeId, false), [headerDiscard, headerSave]),
    d.eula ? document.createDocumentFragment() : el("div", { class: "notice" },
      "The Minecraft EULA has not been accepted for this project. The server will not start until it is. ",
      el("button", { class: "btn inline-offset", onclick: () =>
        confirmModal("Accept Minecraft EULA",
          "By accepting you agree to the Minecraft End User License Agreement (https://aka.ms/MinecraftEULA). Bonghos never accepts it silently on your behalf.",
          "I accept the EULA", async () => {
            try { await api(serverScopedPath("/configuration/eula"), { method: "POST", json: { accept: true } }); toast("EULA accepted", "ok"); renderPage(); }
            catch (e) { toast(e.message, "err"); }
          }, false) }, "Review & accept")),
    el("div", { class: "grid cols-2" },
      el("div", { class: "card" },
        el("h3", {}, "JVM memory"),
        jvmSourceNote(d.jvm),
        el("div", { class: "field-row" }, el("label", {}, "Minimum (-Xms)", xms)),
        el("div", { class: "field-row" }, el("label", {}, "Maximum (-Xmx)", xmx))),
      el("div", { class: "card" },
        el("h3", {}, "Startup"),
        el("div", { class: "field-row" }, el("label", {}, "Startup script", scriptSel)),
        el("div", { class: "field-row flow-section" }, el("label", {}, "Java installation", javaSel)))),
    el("h2", {}, "Server icon"),
    iconCard,
    el("div", { class: "configuration-section-heading" },
      el("h2", {}, "server.properties"),
      can("server.files.manage") ? el("button", { class: "btn ghost", onclick: openServerProperties }, "Open server.properties") : null),
    el("div", { class: "card" }, propRows.length ? propRows : el("p", { class: "muted" }, "No server.properties found yet (it is created on first start).")),
    el("h2", {}, "Automation"),
    automationCard,
    bottomActions);

  updateActions();
}

// ----- backups --------------------------------------------------------------
async function pageBackups(main) {
  const list = await api("/backups");
  main.innerHTML = "";
  const createBackup = async (type, label) => {
    try { await api("/backups", { method: "POST", json: { type } }); toast(label + " backup started", "ok"); }
    catch (e) { toast(e.message, "err"); }
  };
  const backupConfirmation = {
    world: "Create a backup of the world and player data? Online backups briefly pause world saving while a consistent archive is created.",
    full: "Create a full server backup? This includes the world, player data, configuration, mods, and other server files. Online backups briefly pause world saving.",
  };
  const mkBtn = (type, label) => el("button", { class: "btn backup-create-button", onclick: () => {
    const message = backupConfirmation[type];
    if (!message) return createBackup(type, label);
    confirmModal(
      label,
      message,
      `Create ${label.toLowerCase()}`,
      () => createBackup(type, label),
      false,
    );
  } }, label);
  const backupRow = (b) => el("tr", {},
    el("td", { class: "mono" },
      el("span", {}, b.backup_id),
      el("span", { class: "mobile-only mobile-row-detail" }, `${b.backup_type.replace(/_/g, " ")} · ${fmtBytes(b.compressed_size)}`)),
    el("td", {}, b.backup_type.replace(/_/g, " ")),
    el("td", { class: "mobile-hide" }, b.consistency_mode + " / " + b.trigger_type),
    el("td", {}, fmtBytes(b.compressed_size)),
    el("td", { class: "mobile-hide" }, b.verification_status || "—"),
    el("td", { class: "mobile-hide" }, fmtTime(b.created_at)),
    el("td", { class: "table-actions" }, backupActions(b)));
  const tbody = el("tbody");
  const draw = (query = "") => {
    const visible = (list || []).filter((backup) => recordMatchesSearch(backup, query, fmtBytes(backup.compressed_size), fmtTime(backup.created_at)));
    tbody.replaceChildren(...(visible.length
      ? visible.map(backupRow)
      : [el("tr", {}, el("td", { colspan: "7", class: "muted" }, (list || []).length ? "No matching backups." : "No backups yet."))]));
  };
  const search = pageSearchInput("backups", draw);
  main.append(
    pageHeader("Backups", "Verified archives, retention decisions, and restore controls. Online backups briefly pause world saving.", [
      search,
      can("server.backups.create") ? mkBtn("world", "World backup") : null,
      can("server.backups.create") ? mkBtn("full", "Full backup") : null,
      can("server.backups.create") ? mkBtn("configuration", "Config backup") : null,
    ]),
    el("div", { class: "progress hidden", id: "backup-progress" }, el("div", { style: "width:0%" })),
    el("div", { class: "table-wrap backups-table" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "ID"), el("th", {}, "Type"), el("th", { class: "mobile-hide" }, "Mode"), el("th", {}, "Size"), el("th", { class: "mobile-hide" }, "Verified"), el("th", { class: "mobile-hide" }, "Created"), el("th", {}, ""))),
        tbody)));
  draw();
}

function backupActions(backup) {
  const actions = [];
  if (can("server.backups.restore")) actions.push({
    label: "Restore", icon: "archive-down-minimlistic-linear", run: () => restoreBackup(backup),
  });
  actions.push(
    { label: "Verify", icon: "shield-check-linear", run: async () => {
      try { await api(`/backups/${backup.backup_id}/verify`, { method: "POST", json: {} }); toast("Verified", "ok"); renderPage(); }
      catch (error) { toast(error.message, "err"); }
    } },
    { label: backup.protected ? "Unprotect" : "Protect", icon: backup.protected ? "shield-keyhole-linear" : "shield-check-linear", run: async () => {
      try { await api(`/backups/${backup.backup_id}/protect`, { method: "POST", json: { protected: !backup.protected } }); renderPage(); }
      catch (error) { toast(error.message, "err"); }
    } },
    { label: "Delete", icon: "trash-bin-trash-linear", danger: true, run: () =>
      confirmModal("Delete backup", `Delete backup ${backup.backup_id}?` + (backup.protected ? " It is PROTECTED." : ""), "Delete", async () => {
        try { await api(`/backups/${backup.backup_id}`, { method: "DELETE" }); renderPage(); }
        catch (error) { toast(error.message, "err"); }
      }) });

  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    ...actions.map((action) => el("button", { class: "btn " + (action.danger ? "danger" : "ghost"), onclick: action.run }, action.label)));
  const mobile = overflowActionsMenu(`Actions for backup ${backup.backup_id}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
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
  const scheduleRow = (s) => el("tr", {},
    el("td", {}, s.name, s.enabled ? "" : el("span", { class: "tag inline-offset" }, "disabled")),
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
          }) }, "Delete")] : ""));
  const tbody = el("tbody");
  const draw = (query = "") => {
    const visible = (list || []).filter((schedule) => recordMatchesSearch(schedule, query, fmtTime(schedule.next_run_at)));
    tbody.replaceChildren(...(visible.length
      ? visible.map(scheduleRow)
      : [el("tr", {}, el("td", { colspan: "6", class: "muted" }, (list || []).length ? "No matching schedules." : "No schedules yet."))]));
  };
  const search = pageSearchInput("schedules", draw);
  main.append(
    pageHeader("Schedules", "Persistent Linux-host schedules with next run, last result, and manual run controls.", [
      search,
      can("server.schedules.manage") ? el("button", { class: "btn primary", onclick: () => scheduleForm(null) }, "New schedule") : null,
    ]),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", {}, "Action"), el("th", {}, "When"), el("th", {}, "Next run"), el("th", {}, "Last result"), el("th", {}, ""))),
        tbody)));
  draw();
}

function timezoneOffsetMinutes(timeZone, date) {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone, year: "numeric", month: "2-digit", day: "2-digit",
    hour: "2-digit", minute: "2-digit", hourCycle: "h23",
  }).formatToParts(date).reduce((out, part) => {
    if (part.type !== "literal") out[part.type] = part.value;
    return out;
  }, {});
  return Math.round((Date.UTC(+parts.year, +parts.month - 1, +parts.day, +parts.hour, +parts.minute) - date.getTime()) / 60000);
}

function timezoneSelect(selected) {
  const local = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  const zones = typeof Intl.supportedValuesOf === "function" ? Intl.supportedValuesOf("timeZone") : [local, "UTC"];
  const unique = [...new Set(["UTC", selected, local, ...zones].filter(Boolean))];
  const now = new Date();
  now.setSeconds(0, 0);
  const records = unique.map((zone) => {
    let offset = 0;
    try { offset = timezoneOffsetMinutes(zone, now); } catch { return null; }
    const sign = offset >= 0 ? "+" : "-";
    const absolute = Math.abs(offset);
    const gmt = `GMT${sign}${String(Math.floor(absolute / 60)).padStart(2, "0")}:${String(absolute % 60).padStart(2, "0")}`;
    const segments = zone.split("/");
    const city = segments[segments.length - 1].replace(/_/g, " ");
    return { zone, offset, label: `${gmt} - ${city}${segments.length > 1 ? ` (${zone})` : ""}` };
  }).filter(Boolean).sort((a, b) => a.offset - b.offset || a.label.localeCompare(b.label));
  return el("select", { "aria-label": "Timezone" },
    ...records.map((record) => el("option", { value: record.zone, selected: record.zone === selected ? "" : null }, record.label)));
}

function scheduleForm(s) {
  const typeAliases = { interval: "fixed_interval", cron: "advanced_cron" };
  const offlineAliases = { skip: "skip_when_offline", start_first: "start_then_execute", run_anyway: "skip_when_offline" };
  const missedAliases = { skip: "skip_missed_run", run_once_on_boot: "run_once_after_startup" };
  const initialType = typeAliases[s?.schedule_type] || s?.schedule_type || "daily";
  const initialAction = s?.action === "backup" ? "create_backup" : (s?.action || "restart_server");
  const initialOffline = offlineAliases[s?.offline_policy] || s?.offline_policy || "skip_when_offline";
  const initialMissed = missedAliases[s?.missed_run_policy] || s?.missed_run_policy || "skip_missed_run";
  let initialPayload = s?.action_payload || {};
  if (typeof initialPayload === "string") {
    try { initialPayload = JSON.parse(initialPayload); } catch { initialPayload = {}; }
  }

  const name = el("input", { value: s ? s.name : "", required: "", maxlength: "120" });
  const description = el("input", { value: s?.description || "", maxlength: "240", placeholder: "Optional" });
  const type = el("select", {},
    ...[["once", "Once"], ["hourly", "Hourly"], ["daily", "Daily"], ["weekly", "Weekly"],
      ["monthly", "Monthly"], ["fixed_interval", "Fixed interval"], ["advanced_cron", "Advanced cron"]]
      .map(([value, label]) => el("option", { value, selected: initialType === value ? "" : null }, label)));
  const expressionHost = el("div", { class: "schedule-expression" });
  const expressionCache = { [initialType]: s?.schedule_expression || "" };
  let renderedType = initialType;
  let readExpression = () => "";

  const renderExpression = (scheduleType) => {
    expressionHost.innerHTML = "";
    const saved = expressionCache[scheduleType] || "";
    if (scheduleType === "once") {
      const input = el("input", { type: "datetime-local", value: saved.replace(" ", "T") });
      expressionHost.append(el("label", {}, "Date and time", input));
      readExpression = () => input.value.replace("T", " ");
    } else if (scheduleType === "hourly") {
      const input = el("input", { type: "number", min: "0", max: "59", step: "1", value: saved || "0" });
      expressionHost.append(el("label", {}, "Minute of each hour", input));
      readExpression = () => input.value;
    } else if (scheduleType === "daily") {
      const input = el("input", { type: "time", value: saved || "04:00" });
      expressionHost.append(el("label", {}, "Time", input));
      readExpression = () => input.value;
    } else if (scheduleType === "weekly") {
      const [savedDay = "MON", savedTime = "04:00"] = saved.toUpperCase().split(/\s+/);
      const day = el("select", {}, ...[["MON", "Monday"], ["TUE", "Tuesday"], ["WED", "Wednesday"], ["THU", "Thursday"], ["FRI", "Friday"], ["SAT", "Saturday"], ["SUN", "Sunday"]]
        .map(([value, label]) => el("option", { value, selected: savedDay === value ? "" : null }, label)));
      const time = el("input", { type: "time", value: savedTime || "04:00" });
      expressionHost.append(el("div", { class: "grid cols-2" }, el("label", {}, "Day", day), el("label", {}, "Time", time)));
      readExpression = () => `${day.value} ${time.value}`;
    } else if (scheduleType === "monthly") {
      const [savedDay = "1", savedTime = "04:00"] = saved.split(/\s+/);
      const day = el("input", { type: "number", min: "1", max: "31", step: "1", value: savedDay || "1" });
      const time = el("input", { type: "time", value: savedTime || "04:00" });
      expressionHost.append(el("div", { class: "grid cols-2" }, el("label", {}, "Day of month", day), el("label", {}, "Time", time)));
      readExpression = () => `${day.value} ${time.value}`;
    } else if (scheduleType === "fixed_interval") {
      const seconds = Math.max(60, Number(saved) || 3600);
      const choices = [[86400, "days"], [3600, "hours"], [60, "minutes"]];
      const [factor, selectedUnit] = choices.find(([value]) => seconds % value === 0) || [60, "minutes"];
      const amount = el("input", { type: "number", min: "1", step: "1", value: String(Math.max(1, seconds / factor)) });
      const unit = el("select", {}, ...choices.map(([value, label]) => el("option", { value: String(value), selected: label === selectedUnit ? "" : null }, label)));
      expressionHost.append(el("div", { class: "grid cols-2" }, el("label", {}, "Every", amount), el("label", {}, "Unit", unit)));
      readExpression = () => String(Math.round(Number(amount.value) * Number(unit.value)));
    } else {
      const input = el("input", { value: saved || "0 4 * * *", placeholder: "0 4 * * *", spellcheck: "false" });
      expressionHost.append(el("label", {}, "Five-field cron expression", input));
      readExpression = () => input.value.trim();
    }
    renderedType = scheduleType;
  };
  renderExpression(initialType);
  type.addEventListener("change", () => {
    expressionCache[renderedType] = readExpression();
    renderExpression(type.value);
  });

  const selectedZone = s?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  const timezone = timezoneSelect(selectedZone);
  const action = el("select", {},
    ...[["restart_server", "Restart server"], ["stop_server", "Stop server"], ["start_server", "Start server"],
      ["send_console_command", "Run console command"], ["broadcast_message", "Broadcast message"],
      ["save_all", "Save world"], ["create_backup", "Create backup"]]
      .map(([value, label]) => el("option", { value, selected: initialAction === value ? "" : null }, label)));
  const actionDetailHost = el("div", { class: "schedule-action-detail" });
  const command = el("input", { value: initialPayload.command || "", placeholder: "say Server restart in 5 minutes", spellcheck: "false" });
  const message = el("input", { value: initialPayload.message || "", placeholder: "Server maintenance starts soon" });
  const backupAliases = { world: "world_and_player_data", configuration: "configuration_only", full: "full_server" };
  const initialBackupType = backupAliases[initialPayload.backup_type || initialPayload.type] || initialPayload.backup_type || "full_server";
  const backupType = el("select", {},
    ...[["full_server", "Full server"], ["world_and_player_data", "World and player data"], ["configuration_only", "Configuration only"]]
      .map(([value, label]) => el("option", { value, selected: initialBackupType === value ? "" : null }, label)));
  const offline = el("select", {},
    ...[["skip_when_offline", "Skip when server is offline"], ["start_then_execute", "Start server, then run"], ["wait_until_online", "Wait up to 10 minutes"]]
      .map(([value, label]) => el("option", { value, selected: initialOffline === value ? "" : null }, label)));
  const offlineRow = el("div", { class: "field-row" }, el("label", {}, "If server is offline", offline));
  const renderActionDetail = () => {
    actionDetailHost.innerHTML = "";
    if (action.value === "send_console_command") actionDetailHost.append(el("div", { class: "field-row" }, el("label", {}, "Command", command)));
    if (action.value === "broadcast_message") actionDetailHost.append(el("div", { class: "field-row" }, el("label", {}, "Message", message)));
    if (action.value === "create_backup") actionDetailHost.append(el("div", { class: "field-row" }, el("label", {}, "Backup scope", backupType)));
    offlineRow.classList.toggle("hidden", !["send_console_command", "broadcast_message", "save_all"].includes(action.value));
  };
  action.addEventListener("change", renderActionDetail);

  const missed = el("select", {},
    ...[["skip_missed_run", "Skip the missed run"], ["run_once_after_startup", "Run once after Bonghos starts"]]
      .map(([value, label]) => el("option", { value, selected: initialMissed === value ? "" : null }, label)));
  const conflict = el("select", {},
    ...[["skip", "Skip this run"], ["retry_later", "Retry after 2 minutes"]]
      .map(([value, label]) => el("option", { value, selected: (s?.conflict_policy || "skip") === value ? "" : null }, label)));
  const enabled = el("input", { type: "checkbox" });
  enabled.checked = s ? !!s.enabled : true;
  renderActionDetail();

  modal(s ? "Edit schedule" : "New schedule", [
    el("h3", {}, "Schedule"),
    el("div", { class: "field-row" }, el("label", {}, "Name", name)),
    el("div", { class: "field-row" }, el("label", {}, "Description", description)),
    el("div", { class: "field-row" }, el("label", {}, "Repeats", type)),
    el("div", { class: "field-row" }, expressionHost),
    el("div", { class: "field-row" }, el("label", {}, "Timezone", timezone)),
    el("h3", {}, "Minecraft action"),
    el("div", { class: "field-row" }, el("label", {}, "Action", action)),
    actionDetailHost,
    el("h3", {}, "Run behavior"),
    offlineRow,
    el("div", { class: "field-row" }, el("label", {}, "If a run was missed", missed)),
    el("div", { class: "field-row" }, el("label", {}, "If another operation is running", conflict)),
    el("div", { class: "field-row" }, el("label", { class: "check-row" }, enabled, " Enabled")),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Save", "primary", async (close) => {
      const expression = readExpression().trim();
      if (!name.value.trim()) { toast("Schedule name is required", "err"); name.focus(); return; }
      if (!expression) { toast("Schedule time is required", "err"); return; }
      if (action.value === "send_console_command" && !command.value.trim()) { toast("Console command is required", "err"); command.focus(); return; }
      if (action.value === "broadcast_message" && !message.value.trim()) { toast("Broadcast message is required", "err"); message.focus(); return; }
      let actionPayload = null;
      if (action.value === "send_console_command") actionPayload = { command: command.value.trim() };
      if (action.value === "broadcast_message") actionPayload = { message: message.value.trim() };
      if (action.value === "create_backup") actionPayload = { backup_type: backupType.value };
      const body = {
        name: name.value.trim(), description: description.value.trim(), enabled: enabled.checked,
        schedule_type: type.value, schedule_expression: expression, timezone: timezone.value,
        action: action.value, action_payload: actionPayload,
        offline_policy: offline.value, missed_run_policy: missed.value, conflict_policy: conflict.value,
      };
      try {
        if (s) await api(`/schedules/${s.id}`, { method: "PATCH", json: body });
        else await api("/schedules", { method: "POST", json: body });
        close(); renderPage();
      } catch (error) { toast(error.message, "err"); }
    }]]);
}

// ----- performance ----------------------------------------------------------
async function pagePerformance(main) {
  const [history, overview] = await Promise.all([
    api("/metrics?hours=1"),
    api("/overview").catch(() => null),
  ]);
  S.perf = [];
  S.perfStorage = null;
  (history || []).forEach(appendPerformanceSample);
  const current = overview?.sample;
  if (current) {
    appendPerformanceSample(current);
    setUptimeBaseline(current);
  }
  if (overview?.state) S.status = { state: overview.state, detail: overview.supervisor };
  const intervalSelect = el("select", {
    id: "performance-interval",
    onchange: (event) => setPerformanceInterval(Number(event.target.value)),
  },
  ...PERFORMANCE_INTERVAL_OPTIONS.map((seconds) =>
    el("option", { value: String(seconds) },
      `Every ${formatInterval(seconds)}${seconds === 2 ? " (Default)" : ""}`)));
  intervalSelect.value = String(S.perfIntervalSeconds);

  main.innerHTML = "";
  main.append(
    pageHeader("Performance", "Live Java process and Linux host telemetry for the active server. History covers the last hour.", [
      el("div", { class: "performance-interval-control" },
        el("label", { for: "performance-interval" }, "Update interval"),
        el("span", { class: "performance-interval-row" },
          intervalSelect,
          el("button", {
            class: "btn ghost icon-button performance-interval-refresh",
            id: "performance-interval-refresh",
            type: "button",
            title: "Refresh",
            "aria-label": "Refresh",
            onclick: refreshPerformanceMetrics,
          }, solarIcon("storage-refresh")))),
    ]),

    el("div", { class: "performance-feedbar", "aria-live": "polite" },
      el("span", { class: "performance-feed-state", id: "performance-feed-state" },
        el("span", { class: "status-square", "aria-hidden": "true" }),
        el("span", { id: "performance-feed-label" }, "Connecting")),
      el("span", { class: "performance-feed-detail", id: "performance-feed-detail" }),
      el("span", { class: "performance-feed-window mono", id: "performance-feed-window" })),

    el("section", { class: "performance-domain performance-cpu-domain", "aria-labelledby": "performance-cpu-title" },
      el("div", { class: "performance-section-heading" },
        performanceSectionTitle("performance-cpu-title", "CPU",
          "Whole-machine load, Java process usage, temperatures, and every logical CPU.", "command-linear")),
      el("div", { class: "performance-domain-readouts" },
        performanceReadout("Machine usage", "performance-host-cpu", "Average across all logical CPUs"),
        performanceReadout("Average temperature", "performance-cpu-temp", "Best available Linux CPU sensors"),
        performanceReadout("Java process", "performance-process-cpu", "100% = one full CPU core"),
        performanceReadout("Load average", "performance-load", "1 minute")),
      el("div", { class: "grid cols-2 performance-domain-charts" },
        performanceChartPanel("Machine CPU usage", "Average utilization across all logical CPUs", "performance-chart-host-cpu"),
        performanceChartPanel("CPU temperature", "Average of available CPU sensors", "performance-chart-cpu-temp")),
      el("div", { class: "performance-core-heading" },
        el("h3", {}, "Logical CPUs"),
        el("span", { class: "metric-note" }, "Temperature is shown only when Linux exposes a matching per-core sensor.")),
      el("div", { class: "performance-core-grid", id: "performance-core-grid" })),

    el("section", { class: "performance-domain flow-section", "aria-labelledby": "performance-memory-title" },
      el("div", { class: "performance-section-heading" },
        performanceSectionTitle("performance-memory-title", "Memory",
          "Host physical memory, configured Java allocation, and current resident process memory.", "server-2-linear")),
      el("div", { class: "performance-meter-grid" },
        performanceMeter("Machine memory", "host-memory"),
        performanceMeter("Java memory (RSS / -Xmx)", "allocated-memory")),
      el("div", { class: "grid cols-2 performance-domain-charts" },
        performanceChartPanel("Machine memory", "Physical memory used by the host", "performance-chart-host-memory"),
        performanceChartPanel("Java resident memory", "Process RSS compared with configured -Xmx", "performance-chart-rss"))),

    el("section", { class: "performance-domain flow-section", "aria-labelledby": "performance-storage-title" },
      el("div", { class: "performance-section-heading has-action" },
        performanceSectionTitle("performance-storage-title", "Storage",
          "Machine filesystem capacity and storage managed by Bonghos.", "database-linear"),
        el("button", {
          class: "btn ghost icon-button performance-storage-refresh",
          id: "performance-storage-refresh",
          type: "button",
          title: "Refresh",
          "aria-label": "Refresh",
          onclick: refreshPerformanceStorage,
        }, solarIcon("storage-refresh"))),
      el("div", { class: "performance-storage-visual", id: "performance-storage-visual" })));

  syncPageSubscription("performance");
  updatePerformanceView(current || latestPerformanceSample());
  renderStorageVisual();
  activatePendingPerformanceTarget();
  refreshPerformanceStorage();
}

function formatInterval(seconds) {
  return seconds === 1 ? "1 second" : `${seconds} seconds`;
}

function setPerformanceInterval(seconds) {
  S.perfIntervalSeconds = PERFORMANCE_INTERVAL_OPTIONS.includes(seconds) ? seconds : 2;
  localStorage.setItem(PERFORMANCE_INTERVAL_KEY, String(S.perfIntervalSeconds));
  wsSend(performanceSubscription());
  syncDemoPerformanceStream();
  updatePerformanceFreshness();
}

async function refreshPerformanceMetrics() {
  if (S.page !== "performance") return;
  const button = $("#performance-interval-refresh");
  if (button?.disabled) return;
  if (button) {
    button.disabled = true;
    button.classList.add("is-loading");
    button.setAttribute("aria-label", "Refreshing performance");
  }
  try {
    const overview = await api("/overview");
    if (S.page !== "performance") return;
    const received = overview?.sample;
    if (!received) throw new Error("No performance sample was returned");
    const sample = DEMO_MODE
      ? { ...(latestPerformanceSample() || received), collected_at: new Date().toISOString() }
      : received;
    appendPerformanceSample(sample);
    setUptimeBaseline(sample);
    if (overview?.state) {
      S.status = { state: overview.state, detail: overview.supervisor };
      renderStatusPill();
    }
    updatePerformanceView(latestPerformanceSample());
    updatePerformanceFreshness();
  } catch (error) {
    toast("Performance refresh failed: " + error.message, "err");
  } finally {
    const currentButton = $("#performance-interval-refresh");
    if (currentButton) {
      currentButton.disabled = false;
      currentButton.classList.remove("is-loading");
      currentButton.setAttribute("aria-label", "Refresh");
      currentButton.setAttribute("title", "Refresh");
    }
  }
}

function sampleTimestamp(sample) {
  const timestamp = Date.parse(sample?.collected_at || sample?.at || "");
  return Number.isFinite(timestamp) ? timestamp : 0;
}

function appendPerformanceSample(sample) {
  if (!sample || !sampleTimestamp(sample)) return;
  const timestamp = sampleTimestamp(sample);
  const existing = S.perf.findIndex((entry) => sampleTimestamp(entry) === timestamp);
  if (existing >= 0) S.perf[existing] = { ...S.perf[existing], ...sample };
  else S.perf.push(sample);
  const cutoff = Date.now() - 60 * 60 * 1000;
  S.perf = S.perf
    .filter((entry) => sampleTimestamp(entry) >= cutoff)
    .sort((a, b) => sampleTimestamp(a) - sampleTimestamp(b))
    .slice(-5000);
}

function latestPerformanceSample() {
  return S.perf.length ? S.perf[S.perf.length - 1] : null;
}

function performanceReadout(label, id, note) {
  return el("div", { class: "performance-readout", id: `${id}-card` },
    el("div", { class: "metric-label" }, label),
    el("div", { class: "performance-readout-value mono", id }, "—"),
    el("div", { class: "metric-note", id: `${id}-note` }, note));
}

function performanceSectionTitle(id, label, description, icon) {
  return el("div", { class: "performance-section-title" },
    el("h2", { id }, solarIcon(icon, "performance-section-icon"), el("span", {}, label)),
    el("p", { class: "muted" }, description));
}

function performanceMeter(label, id) {
  return el("div", { class: "performance-meter", id: `${id}-card` },
    el("div", { class: "performance-meter-head" },
      el("span", { class: "metric-label" }, label),
      el("strong", { class: "mono", id: `${id}-percent` }, "—")),
    el("div", { class: "performance-meter-track", role: "meter", "aria-valuemin": "0", "aria-valuemax": "100", id: `${id}-meter` },
      el("span", { class: "performance-meter-fill", id: `${id}-fill` })),
    el("div", { class: "performance-meter-detail", id: `${id}-detail` }, "Waiting for a sample"));
}

function performanceChartPanel(title, description, id) {
  return el("div", { class: "card performance-chart-panel" },
    el("div", { class: "performance-chart-heading" },
      el("div", {}, el("h3", {}, title), el("p", { class: "metric-note" }, description))),
    el("div", { class: "performance-chart-host", id }));
}

function setNodeText(id, value) {
  const node = $("#" + id);
  if (node) node.textContent = value;
}

function activatePendingPerformanceTarget() {
  if (S.page !== "performance" || !S.performanceTarget) return false;
  const target = document.getElementById(S.performanceTarget);
  if (!target) return false;
  S.performanceTarget = "";
  activateNavigationTarget(target, "performance");
  return true;
}

function activateNavigationTarget(target, page) {
  if (navigationJumpStartTimer) clearTimeout(navigationJumpStartTimer);
  if (navigationJumpTimer) clearTimeout(navigationJumpTimer);
  document.querySelectorAll(".navigation-jump-highlight").forEach((node) =>
    node.classList.remove("navigation-jump-highlight"));
  const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  requestAnimationFrame(() => {
    target.scrollIntoView({ behavior: reduceMotion ? "auto" : "smooth", block: "center", inline: "nearest" });
    navigationJumpStartTimer = setTimeout(() => {
      if (!target.isConnected || S.page !== page) return;
      target.classList.add("navigation-jump-highlight");
      navigationJumpTimer = setTimeout(() => {
        target.classList.remove("navigation-jump-highlight");
        navigationJumpTimer = null;
      }, 2000);
      navigationJumpStartTimer = null;
    }, reduceMotion ? 0 : 350);
  });
}

function updatePerformanceMeter(id, used, total, detail) {
  const valid = Number.isFinite(used) && Number.isFinite(total) && total > 0;
  const percent = valid ? Math.max(0, Math.min(100, used / total * 100)) : 0;
  const meter = $("#" + id + "-meter");
  const fill = $("#" + id + "-fill");
  if (meter) meter.setAttribute("aria-valuenow", valid ? percent.toFixed(1) : "0");
  if (fill) {
    fill.style.width = `${percent}%`;
    fill.dataset.pressure = percent >= 90 ? "danger" : percent >= 80 ? "warning" : "normal";
  }
  setNodeText(id + "-percent", valid ? percent.toFixed(1) + "%" : "—");
  setNodeText(id + "-detail", valid ? detail : "Not available");
}

function updateLiveStats(sample) {
  if (S.page === "performance") updatePerformanceView(sample);
  if (S.page === "overview" && sample) {
    const hostCPU = Number(sample.host_cpu_percent);
    const rss = Number(sample.rss_bytes);
    const hostTotal = Number(sample.host_mem_total);
    const hostAvail = Number(sample.host_mem_avail);
    const diskTotal = Number(sample.disk_total);
    const diskFree = Number(sample.disk_free);
    const load = Number(sample.load1);
    setNodeText("overview-live-cpu", Number.isFinite(hostCPU) ? hostCPU.toFixed(1) + "%" : "—");
    setNodeText("overview-live-rss", Number.isFinite(rss) ? fmtBytes(rss) : "—");
    setNodeText("overview-live-host-memory", hostTotal > 0 && Number.isFinite(hostAvail) ? fmtBytes(hostTotal - hostAvail) : "—");
    setNodeText("overview-live-disk-free", diskTotal > 0 && Number.isFinite(diskFree) ? fmtBytes(diskFree) : "—");
    setNodeText("overview-live-load", Number.isFinite(load) ? load.toFixed(2) : "—");
    updateOverviewTrendCharts();
  }
}

function updatePerformanceView(sample = latestPerformanceSample()) {
  if (S.page !== "performance" || !sample) {
    updatePerformanceFreshness();
    return;
  }
  const processCPU = Number(sample.cpu_percent);
  const hostCPU = Number(sample.host_cpu_percent);
  const cpuTemp = sample.cpu_temp_celsius == null ? NaN : Number(sample.cpu_temp_celsius);
  const rss = Number(sample.rss_bytes);
  const xms = Number(sample.jvm_xms_bytes);
  const xmx = Number(sample.jvm_xmx_bytes);
  const hostTotal = Number(sample.host_mem_total);
  const hostAvail = Number(sample.host_mem_avail);
  const hostUsed = hostTotal - hostAvail;

  setNodeText("performance-host-cpu", Number.isFinite(hostCPU) ? hostCPU.toFixed(1) + "%" : "—");
  setNodeText("performance-cpu-temp", Number.isFinite(cpuTemp) ? cpuTemp.toFixed(1) + " °C" : "Unavailable");
  setNodeText("performance-process-cpu", Number.isFinite(processCPU) ? processCPU.toFixed(1) + "%" : "—");
  setNodeText("performance-load", Number.isFinite(Number(sample.load1)) ? Number(sample.load1).toFixed(2) : "—");

  updatePerformanceMeter("host-memory", hostUsed, hostTotal,
    `${fmtBytes(hostUsed)} / ${fmtBytes(hostTotal)} · ${fmtBytes(hostAvail)} available`);
  updatePerformanceMeter("allocated-memory", rss, xmx,
    xmx > 0 ? `${fmtBytes(rss)} / ${fmtBytes(xmx)} · ${fmtBytes(xms)} min (-Xms) · ${fmtBytes(xmx)} max (-Xmx)` : "No JVM allocation detected in project configuration");

  renderCPUCoreGrid(sample);
  renderPerformanceCharts();
  updatePerformanceFreshness();
}

function renderPerformanceCharts() {
  const samples = S.perf;
  const configuredXmx = Number(latestPerformanceSample()?.jvm_xmx_bytes) || 0;
  const charts = [
    ["#performance-chart-host-cpu", samples, {
      label: "Whole-machine CPU usage over the last hour", min: 0, fixedMax: 100, axisFormat: (v) => v.toFixed(0) + "%",
      series: [{ label: "Machine", tone: "accent", area: true, value: (s) => Number(s.host_cpu_percent), format: (v) => v.toFixed(1) + "%" }],
    }],
    ["#performance-chart-cpu-temp", samples.filter((s) => s.cpu_temp_celsius != null), {
      label: "Average CPU sensor temperature over the last hour", axisFormat: (v) => v.toFixed(0) + " °C",
      series: [{ label: "Average", tone: "warning", area: true, value: (s) => Number(s.cpu_temp_celsius), format: (v) => v.toFixed(1) + " °C" }],
    }],
    ["#performance-chart-host-memory", samples, {
      label: "Machine physical memory usage over the last hour", min: 0, axisFormat: fmtBytes,
      series: [{ label: "Machine used", tone: "accent", area: true, value: (s) => Number(s.host_mem_total) - Number(s.host_mem_avail), format: fmtBytes }],
    }],
    ["#performance-chart-rss", samples, {
      label: "Java resident memory and configured maximum over the last hour", min: 0, axisFormat: fmtBytes,
      series: [
        { label: "Java RSS", tone: "info", area: true, value: (s) => Number(s.rss_bytes), format: fmtBytes },
        { label: "Configured -Xmx", tone: "warning", value: (s) => Number(s.jvm_xmx_bytes) || configuredXmx, format: fmtBytes },
      ],
    }],
  ];
  charts.forEach(([selector, chartData, options]) => {
    const host = $(selector);
    if (host) host.replaceChildren(timeSeriesChart(chartData, options));
  });
}

function renderCPUCoreGrid(sample) {
  const host = $("#performance-core-grid");
  if (!host) return;
  const cores = Array.isArray(sample?.cpu_cores) ? sample.cpu_cores : [];
  if (!cores.length) {
    host.replaceChildren(el("div", { class: "performance-chart-empty" }, "Per-core CPU data is not available yet."));
    return;
  }
  host.replaceChildren(...cores.map((core) => {
    const usage = Math.max(0, Math.min(100, Number(core.usage_percent) || 0));
    const temperature = core.temp_celsius == null ? null : Number(core.temp_celsius);
    const pressure = usage >= 90 ? "danger" : usage >= 75 ? "warning" : "normal";
    return el("div", { class: "performance-core" },
      el("div", { class: "performance-core-head" },
        el("strong", { class: "mono" }, `C${core.index}`),
        el("span", { class: "mono" }, usage.toFixed(1) + "%")),
      el("div", { class: "performance-core-track", role: "meter", "aria-label": `CPU C${core.index} usage`, "aria-valuemin": "0", "aria-valuemax": "100", "aria-valuenow": usage.toFixed(1) },
        el("span", { class: "performance-core-fill", style: `width:${usage}%`, "data-pressure": pressure })),
      el("div", { class: "performance-core-temp mono" }, Number.isFinite(temperature) ? temperature.toFixed(1) + " °C" : "temp —"));
  }));
}

async function refreshPerformanceStorage() {
  if (S.page !== "performance") return;
  const request = ++performanceStorageRequest;
  const button = $("#performance-storage-refresh");
  if (button) {
    button.disabled = true;
    button.classList.add("is-loading");
    button.setAttribute("aria-label", "Refreshing storage stats");
    button.setAttribute("title", "Refresh");
  }
  if (!S.perfStorage) renderStorageVisual();
  try {
    const snapshot = await api("/metrics/storage");
    if (request !== performanceStorageRequest || S.page !== "performance") return;
    S.perfStorage = snapshot;
    renderStorageVisual();
  } catch (error) {
    if (request !== performanceStorageRequest || S.page !== "performance") return;
    if (!S.perfStorage) {
      const host = $("#performance-storage-visual");
      host?.replaceChildren(el("div", { class: "card performance-chart-empty" }, "Storage stats could not be loaded."));
    }
    toast("Storage refresh failed: " + error.message, "err");
  } finally {
    if (request !== performanceStorageRequest || S.page !== "performance") return;
    const currentButton = $("#performance-storage-refresh");
    if (currentButton) {
      currentButton.disabled = false;
      currentButton.classList.remove("is-loading");
      currentButton.setAttribute("aria-label", "Refresh");
      currentButton.setAttribute("title", "Refresh");
    }
  }
}

function renderStorageVisual(sample = S.perfStorage) {
  const host = $("#performance-storage-visual");
  if (!host) return;
  if (!sample) {
    host.replaceChildren(el("div", { class: "card performance-chart-empty performance-storage-loading" }, "Reading filesystem storage…"));
    return;
  }
  const diskTotal = Math.max(0, Number(sample.disk_total) || 0);
  const diskFree = Math.max(0, Math.min(diskTotal, Number(sample.disk_free) || 0));
  const bonghosTotal = Math.max(0, Number(sample.bonghos_dir_bytes) || 0);
  const diskUsed = Math.max(0, diskTotal - diskFree);
  const bonghosOnDisk = Math.min(diskUsed, bonghosTotal);
  const otherUsed = Math.max(0, diskUsed - bonghosOnDisk);
  let remaining = bonghosTotal;
  const servers = Math.min(remaining, Math.max(0, Number(sample.server_dir_bytes) || 0));
  remaining -= servers;
  const backups = Math.min(remaining, Math.max(0, Number(sample.backup_dir_bytes) || 0));
  remaining -= backups;
  const system = Math.min(remaining, Math.max(0, Number(sample.system_dir_bytes) || 0));
  remaining -= system;
  const timestamp = sample.collected_at || sample.at;
  host.replaceChildren(el("div", { class: "grid cols-2 performance-storage-distributions" },
    storageDonutChart({
      id: "performance-machine-storage-card",
      title: "Machine filesystem",
      description: "Filesystem containing Bonghos",
      total: diskTotal,
      timestamp,
      emptyMessage: "Filesystem capacity is not available.",
      segments: [
        { label: "Other used", value: otherUsed, tone: "warning" },
        { label: "Bonghos", value: bonghosOnDisk, tone: "accent" },
        { label: "Available", value: diskFree, tone: "empty" },
      ],
    }),
    storageDonutChart({
      title: "Bonghos",
      description: "Servers, backups, system files, and other managed data",
      total: bonghosTotal,
      timestamp,
      emptyMessage: "Bonghos storage size is not available.",
      segments: [
        { label: "Servers", value: servers, tone: "accent" },
        { label: "Backups", value: backups, tone: "info" },
        { label: "System", value: system, tone: "success" },
        { label: "Other", value: remaining, tone: "warning" },
      ],
    })));
  activatePendingPerformanceTarget();
}

function storageDonutChart({ id = "", title, description, total, segments, timestamp, emptyMessage }) {
  const attrs = { class: "card performance-storage-panel" };
  if (id) attrs.id = id;
  const heading = el("div", { class: "performance-chart-heading" },
    el("div", {}, el("h3", {}, title), el("p", { class: "metric-note" }, description)));
  if (total <= 0) return el("div", attrs, heading,
    el("div", { class: "performance-chart-empty" }, emptyMessage));

  const svg = svgElement("svg", { class: "performance-donut", viewBox: "0 0 240 240", role: "img", "aria-label": `${title} distribution` });
  const centerValue = el("strong", { class: "mono" }, fmtBytes(total));
  const centerLabel = el("span", {}, "total");
  const detail = el("div", { class: "performance-donut-detail mono" }, `${fmtBytes(total)} total · ${fmtTime(timestamp)}`);
  let offset = 0;
  const entries = [];
  let activeEntry = null;
  const showTotal = () => {
    activeEntry = null;
    entries.forEach((entry) => {
      if (entry.circle) svg.append(entry.circle);
      entry.circle?.classList.remove("is-active");
      entry.row.classList.remove("is-active");
    });
    centerValue.textContent = fmtBytes(total);
    centerLabel.textContent = "total";
    detail.textContent = `${fmtBytes(total)} total · ${fmtTime(timestamp)}`;
  };
  const showEntry = (entry) => {
    if (activeEntry === entry) return;
    activeEntry = entry;
    // SVG uses paint order for stacking. Re-append the active slice so its
    // scale and shadow render above every neighboring segment.
    if (entry.circle) svg.append(entry.circle);
    entries.forEach((candidate) => {
      const active = candidate === entry;
      candidate.circle?.classList.toggle("is-active", active);
      candidate.row.classList.toggle("is-active", active);
    });
    centerValue.textContent = fmtBytes(entry.segment.value);
    centerLabel.textContent = entry.segment.label;
    detail.textContent = `${entry.segment.label}: ${fmtBytes(entry.segment.value)} (${(entry.segment.value / total * 100).toFixed(1)}%) · ${fmtTime(timestamp)}`;
  };
  segments.forEach((segment) => {
    const percent = segment.value / total * 100;
    let circle = null;
    if (percent > 0) {
      circle = svgElement("circle", {
        cx: "120", cy: "120", r: "78", pathLength: "100", fill: "none", "stroke-width": "42",
        "stroke-dasharray": `${percent} ${100 - percent}`, "stroke-dashoffset": String(-offset),
        class: `performance-donut-segment tone-${segment.tone}`, tabindex: "0",
        "aria-label": `${segment.label}: ${fmtBytes(segment.value)}, ${percent.toFixed(1)} percent`,
      });
      svg.append(circle);
    }
    offset += percent;
    const row = el("div", { class: "performance-donut-legend-row", tabindex: "0" },
      el("span", {}, el("span", { class: `performance-chart-swatch tone-${segment.tone}` }), segment.label),
      el("strong", { class: "mono" }, fmtBytes(segment.value)),
      el("span", { class: "mono" }, percent.toFixed(1) + "%"));
    const entry = { segment, circle, row };
    entries.push(entry);
    row.addEventListener("pointerenter", () => showEntry(entry));
    row.addEventListener("pointerleave", showTotal);
    [row, circle].filter(Boolean).forEach((target) => {
      target.addEventListener("focus", () => showEntry(entry));
      target.addEventListener("blur", showTotal);
    });
  });
  // The active SVG slice is re-appended for correct paint order. Delegating
  // pointer tracking to the stable SVG prevents that DOM move from swallowing
  // the slice's pointerleave event and leaving it visually selected.
  const entryHasVisibleFocus = (entry) => [entry?.circle, entry?.row]
    .filter(Boolean).some((target) => target === document.activeElement && target.matches(":focus-visible"));
  svg.addEventListener("pointermove", (event) => {
    const entry = entries.find((candidate) => candidate.circle === event.target);
    if (entry) showEntry(entry);
    else if (activeEntry && !entryHasVisibleFocus(activeEntry)) showTotal();
  });
  svg.addEventListener("pointerleave", () => {
    const focused = entries.find(entryHasVisibleFocus);
    if (focused) showEntry(focused);
    else showTotal();
  });
  return el("div", attrs, heading,
    el("div", { class: "performance-donut-layout" },
      el("div", { class: "performance-donut-plot" }, svg, el("div", { class: "performance-donut-center" }, centerValue, centerLabel)),
      el("div", { class: "performance-donut-legend" }, ...entries.map((entry) => entry.row), detail)));
}

function chartSamples(samples, limit = 240) {
  if (samples.length <= limit) return samples;
  const result = [];
  for (let index = 0; index < limit; index++) {
    result.push(samples[Math.round(index / (limit - 1) * (samples.length - 1))]);
  }
  return result;
}

function svgElement(tag, attrs = {}) {
  const node = document.createElementNS("http://www.w3.org/2000/svg", tag);
  Object.entries(attrs).forEach(([key, value]) => node.setAttribute(key, value));
  return node;
}

function timeSeriesChart(inputSamples, options) {
  const samples = chartSamples((inputSamples || []).filter((sample) => sampleTimestamp(sample)));
  if (!samples.length) return el("div", { class: "performance-chart-empty" }, "No samples in this time window.");

  const width = 720, height = 220, pad = 8;
  const series = options.series.map((entry) => ({
    ...entry,
    values: samples.map((sample) => {
      const value = entry.value(sample);
      return Number.isFinite(value) ? value : 0;
    }),
  }));
  const allValues = series.flatMap((entry) => entry.values);
  const min = options.min ?? Math.min(...allValues);
  let max = options.fixedMax ?? Math.max(...allValues, options.floorMax ?? -Infinity);
  if (!Number.isFinite(max)) max = min + 1;
  if (max <= min) max = min + 1;
  const firstAt = sampleTimestamp(samples[0]);
  const lastAt = sampleTimestamp(samples[samples.length - 1]);
  const timeSpan = Math.max(1, lastAt - firstAt);
  const xAt = (sample) => pad + (sampleTimestamp(sample) - firstAt) / timeSpan * (width - pad * 2);
  const yAt = (value) => height - pad - (value - min) / (max - min) * (height - pad * 2);

  const frame = el("div", { class: "performance-chart-frame", tabindex: "0", "aria-label": options.label });
  const svg = svgElement("svg", { viewBox: `0 0 ${width} ${height}`, preserveAspectRatio: "none", role: "img", "aria-hidden": "true" });
  for (let tick = 0; tick <= 4; tick++) {
    const y = pad + tick / 4 * (height - pad * 2);
    svg.append(svgElement("line", { x1: pad, y1: y, x2: width - pad, y2: y, class: "performance-chart-gridline" }));
  }
  for (let tick = 0; tick <= 4; tick++) {
    const x = pad + tick / 4 * (width - pad * 2);
    svg.append(svgElement("line", { x1: x, y1: pad, x2: x, y2: height - pad, class: "performance-chart-gridline" }));
  }

  series.forEach((entry) => {
    const points = entry.values.map((value, index) => ({ x: xAt(samples[index]), y: yAt(value) }));
    let pathData = "";
    points.forEach((point, index) => {
      if (!index) pathData = `M ${point.x} ${point.y}`;
      else if (options.step) pathData += ` H ${point.x} V ${point.y}`;
      else pathData += ` L ${point.x} ${point.y}`;
    });
    if (entry.area) {
      const area = `${pathData} L ${points[points.length - 1].x} ${height - pad} L ${points[0].x} ${height - pad} Z`;
      svg.append(svgElement("path", { d: area, class: `performance-chart-area tone-${entry.tone}` }));
    }
    svg.append(svgElement("path", { d: pathData, class: `performance-chart-line tone-${entry.tone}` }));
  });

  const crosshair = svgElement("line", { y1: pad, y2: height - pad, class: "performance-chart-crosshair" });
  crosshair.setAttribute("visibility", "hidden");
  svg.append(crosshair);
  const markers = series.map((entry) => {
    const marker = svgElement("circle", { r: 5, class: `performance-chart-marker tone-${entry.tone}` });
    marker.setAttribute("visibility", "hidden");
    svg.append(marker);
    return marker;
  });
  const overlay = svgElement("rect", { x: 0, y: 0, width, height, class: "performance-chart-overlay" });
  svg.append(overlay);

  const tooltip = el("div", { class: "performance-chart-tooltip", hidden: "" });
  const legend = el("div", { class: "performance-chart-legend" }, ...series.map((entry) =>
    el("span", {}, el("span", { class: `performance-chart-swatch tone-${entry.tone}` }),
      `${entry.label} `, el("strong", { class: "mono" }, entry.format(entry.values[entry.values.length - 1])))));
  const range = el("div", { class: "performance-chart-range mono" },
    el("span", {}, new Date(firstAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })),
    el("span", {}, `${options.axisFormat(min)} – ${options.axisFormat(max)}`),
    el("span", {}, new Date(lastAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })),
  );

  let activeIndex = samples.length - 1;
  const showSample = (index) => {
    activeIndex = Math.max(0, Math.min(samples.length - 1, index));
    const sample = samples[activeIndex];
    const x = xAt(sample);
    crosshair.setAttribute("visibility", "visible");
    crosshair.setAttribute("x1", x);
    crosshair.setAttribute("x2", x);
    markers.forEach((marker, seriesIndex) => {
      marker.setAttribute("visibility", "visible");
      marker.setAttribute("cx", x);
      marker.setAttribute("cy", yAt(series[seriesIndex].values[activeIndex]));
    });
    tooltip.hidden = false;
    tooltip.classList.toggle("align-right", x / width > 0.7);
    tooltip.style.left = `${x / width * 100}%`;
    tooltip.replaceChildren(
      el("time", { class: "mono", datetime: sample.collected_at || sample.at }, fmtTime(sample.collected_at || sample.at)),
      ...series.map((entry) => el("div", {},
        el("span", {}, el("span", { class: `performance-chart-swatch tone-${entry.tone}` }), entry.label),
        el("strong", { class: "mono" }, entry.format(entry.values[activeIndex])))));
    frame.setAttribute("aria-label", `${options.label}. ${fmtTime(sample.collected_at || sample.at)}. ${series.map((entry) => `${entry.label} ${entry.format(entry.values[activeIndex])}`).join(", ")}`);
  };
  const hideSample = () => {
    if (frame === document.activeElement) return;
    crosshair.setAttribute("visibility", "hidden");
    markers.forEach((marker) => marker.setAttribute("visibility", "hidden"));
    tooltip.hidden = true;
  };
  overlay.addEventListener("pointermove", (event) => {
    const rect = svg.getBoundingClientRect();
    const pointerX = (event.clientX - rect.left) / Math.max(1, rect.width) * width;
    let nearest = 0;
    let distance = Infinity;
    samples.forEach((sample, index) => {
      const nextDistance = Math.abs(xAt(sample) - pointerX);
      if (nextDistance < distance) { nearest = index; distance = nextDistance; }
    });
    showSample(nearest);
  });
  overlay.addEventListener("pointerleave", hideSample);
  overlay.addEventListener("pointerdown", (event) => { event.preventDefault(); frame.focus(); });
  frame.addEventListener("focus", () => showSample(activeIndex));
  frame.addEventListener("blur", hideSample);
  frame.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    showSample(activeIndex + (event.key === "ArrowRight" ? 1 : -1));
  });

  frame.append(legend, svg, range, tooltip);
  return frame;
}

function updatePerformanceFreshness() {
  if (S.page !== "performance") return;
  const label = $("#performance-feed-label");
  const state = $("#performance-feed-state");
  const detail = $("#performance-feed-detail");
  const windowNode = $("#performance-feed-window");
  if (!label || !state) return;
  const interval = S.perfIntervalSeconds;
  const latest = latestPerformanceSample();
  const age = latest ? Math.max(0, Date.now() - sampleTimestamp(latest)) : Infinity;
  const connected = DEMO_MODE || (S.ws && S.ws.readyState === WebSocket.OPEN);
  const fresh = age <= Math.max(interval * 2500, 15000);
  state.className = "performance-feed-state " + (connected && fresh ? "is-live" : connected ? "is-waiting" : "is-offline");
  label.textContent = DEMO_MODE ? "Demo live" : !connected ? "Reconnecting" : fresh ? "Live" : "Waiting for sample";
  if (detail) {
    const process = latest?.java_pid ? `PID ${latest.java_pid} · uptime ${fmtDur(currentUptimeSeconds() ?? latest.uptime_seconds)}` : "Java process stopped";
    detail.textContent = latest
      ? `Last sample ${relativeSampleAge(age)} · ${fmtTime(latest.collected_at || latest.at)} · ${process}`
      : "No sample received yet";
  }
  if (windowNode) windowNode.textContent = `${S.perf.length} samples · 1 hour`;
}

function relativeSampleAge(milliseconds) {
  if (!Number.isFinite(milliseconds)) return "never";
  const seconds = Math.floor(milliseconds / 1000);
  if (seconds < 2) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  return `${Math.floor(seconds / 60)}m ago`;
}

function syncDemoPerformanceStream() {
  if (demoPerformanceTimer) clearInterval(demoPerformanceTimer);
  demoPerformanceTimer = null;
  if (!DEMO_MODE || (S.page !== "performance" && S.page !== "overview")) return;
  const seconds = S.page === "performance" ? S.perfIntervalSeconds : 10;
  demoPerformanceTimer = setInterval(() => {
    const previous = latestPerformanceSample() || DEMO_METRICS[DEMO_METRICS.length - 1];
    const tick = Date.now() / 1000;
    const sample = {
      ...previous,
      collected_at: new Date().toISOString(),
      cpu_percent: Math.max(0, 32 + Math.sin(tick / 7) * 18 + Math.sin(tick / 2) * 5),
      host_cpu_percent: Math.max(0, Math.min(100, 46 + Math.sin(tick / 6) * 24)),
      cpu_temp_celsius: 57 + Math.sin(tick / 10) * 6,
      cpu_cores: Array.from({ length: 8 }, (_, core) => ({
        index: core,
        usage_percent: Math.max(1, Math.min(100, 38 + Math.sin((tick + core * 2) / 5) * 30 + core * 2)),
        temp_celsius: 52 + Math.sin((tick + core) / 10) * 6 + core * 0.5,
      })),
      rss_bytes: Math.max(0, Number(previous.rss_bytes) + Math.sin(tick / 11) * 6 * 1024 * 1024),
      host_mem_avail: 18 * 1024 * 1024 * 1024 - Math.sin(tick / 9) * 1.4 * 1024 * 1024 * 1024,
      load1: Math.max(0, 0.72 + Math.sin(tick / 8) * 0.38),
      online_players: Math.max(0, Math.round(3 + Math.sin(tick / 20))),
      uptime_seconds: Number(previous.uptime_seconds || 0) + seconds,
    };
    appendPerformanceSample(sample);
    setUptimeBaseline(sample);
    updateLiveStats(sample);
  }, seconds * 1000);
}

setInterval(updatePerformanceFreshness, 1000);

// overviewSparklineNode keeps Overview graphs visually light while exposing
// exact samples through pointer and keyboard interaction.
function overviewSparklineNode(title, points, fmt, options = {}) {
  const W = 320, H = 56, pad = 2;
  const values = points.map((point) => point.value);
  const observedMax = Math.max(...values, 1);
  const min = Number.isFinite(options.min) ? options.min : Math.min(...values, 0);
  const max = Math.max(observedMax, Number.isFinite(options.max) ? options.max : observedMax);
  const span = (max - min) || 1;
  const firstAt = points[0].timestamp;
  const lastAt = points[points.length - 1].timestamp;
  const timeSpan = Math.max(1, points[points.length - 1].timestamp - firstAt);
  const xAt = (point, index) => points.length > 1
    ? pad + ((point.timestamp - firstAt) / timeSpan) * (W - pad * 2)
    : pad + (index / Math.max(1, points.length - 1)) * (W - pad * 2);
  const yAt = (point) => H - pad - ((point.value - min) / span) * (H - pad * 2);
  const linePoints = points.map((point, index) => `${xAt(point, index).toFixed(1)},${yAt(point).toFixed(1)}`).join(" ");
  const svg = svgElement("svg", {
    class: "sparkline", viewBox: `0 0 ${W} ${H}`, preserveAspectRatio: "none",
    role: "img", "aria-label": `${title} over the last hour`,
  });
  const line = svgElement("polyline", { points: linePoints, class: "overview-sparkline-line" });
  const crosshair = svgElement("line", { y1: pad, y2: H - pad, class: "overview-sparkline-crosshair" });
  const marker = el("span", { class: "overview-sparkline-marker", "aria-hidden": "true" });
  const overlay = svgElement("rect", { x: 0, y: 0, width: W, height: H, class: "overview-sparkline-overlay" });
  const tooltip = el("div", { class: "overview-sparkline-tooltip mono", role: "status", "aria-live": "polite", hidden: "" });
  const axisFormat = options.axisFormat || fmt;
  const range = el("div", { class: "overview-sparkline-range mono", "aria-hidden": "true" },
    el("span", {}, new Date(firstAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })),
    el("span", {}, `${axisFormat(min)} – ${axisFormat(max)}`),
    el("span", {}, new Date(lastAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })));
  const plot = el("div", { class: "overview-sparkline-plot" }, svg, marker, tooltip);
  const wrapper = el("div", {
    class: "overview-sparkline", tabindex: "0",
    "aria-label": `${title} history from ${fmtTime(firstAt)} to ${fmtTime(lastAt)}, range ${axisFormat(min)} to ${axisFormat(max)}. Focus and use left or right arrow keys to inspect samples.`,
  }, plot, range);
  svg.append(line, crosshair, overlay);
  let activeIndex = points.length - 1;

  const showPoint = (index) => {
    activeIndex = Math.max(0, Math.min(points.length - 1, index));
    const point = points[activeIndex];
    const x = xAt(point, activeIndex);
    const y = yAt(point);
    crosshair.setAttribute("x1", x);
    crosshair.setAttribute("x2", x);
    marker.style.left = `${x / W * 100}%`;
    marker.style.top = `${y / H * 100}%`;
    wrapper.classList.add("is-active");
    tooltip.hidden = false;
    tooltip.textContent = `${fmtTime(point.timestamp)} · ${title} ${fmt(point.value)}`;
    tooltip.style.left = `${x / W * 100}%`;
    tooltip.style.top = `${y / H * 100}%`;
    tooltip.dataset.align = x < W * 0.22 ? "left" : x > W * 0.78 ? "right" : "center";
  };
  const hidePoint = () => {
    wrapper.classList.remove("is-active");
    tooltip.hidden = true;
  };
  const pointFromPointer = (event) => {
    const rect = svg.getBoundingClientRect();
    const pointerX = (event.clientX - rect.left) / Math.max(1, rect.width) * W;
    let nearest = 0;
    let distance = Infinity;
    points.forEach((point, index) => {
      const nextDistance = Math.abs(xAt(point, index) - pointerX);
      if (nextDistance < distance) { nearest = index; distance = nextDistance; }
    });
    return nearest;
  };

  overlay.addEventListener("pointermove", (event) => showPoint(pointFromPointer(event)));
  overlay.addEventListener("pointerleave", hidePoint);
  overlay.addEventListener("pointerdown", (event) => {
    event.preventDefault();
    wrapper.focus();
    showPoint(pointFromPointer(event));
  });
  wrapper.addEventListener("focus", () => showPoint(activeIndex));
  wrapper.addEventListener("blur", hidePoint);
  wrapper.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    showPoint(activeIndex + (event.key === "ArrowRight" ? 1 : -1));
  });
  return wrapper;
}

// ----- servers (projects + import) ------------------------------------------
function serverCardIcon(server) {
  const fallback = el("div", { class: "server-card-icon server-card-icon-fallback", "aria-hidden": "true" },
    solarIcon("server-square-linear"));
  const revision = encodeURIComponent(server.icon_revision || 0);
  const icon = el("img", {
    class: "server-card-icon",
    src: DEMO_MODE ? server.demo_icon : `/api/servers/${server.id}/icon?v=${revision}`,
    alt: "",
    width: 64,
    height: 64,
    loading: "lazy",
    decoding: "async",
  });
  icon.addEventListener("error", () => icon.replaceWith(fallback), { once: true });
  return icon;
}

const PROVIDER_ICONS = {
  forge: {
    viewBox: "0 0 24 24",
    body: '<path fill="none" stroke="currentColor" stroke-width="1.5" d="m18.66 8.286l.368-.368c.342-.343.514-.514.617-.692a1.56 1.56 0 0 0 0-1.562c-.103-.178-.275-.35-.617-.692s-.514-.514-.692-.616a1.56 1.56 0 0 0-1.562 0c-.178.102-.35.274-.692.616l-.368.368m-4.419 10.31l-5.523 5.524c-.343.343-.514.514-.692.617a1.56 1.56 0 0 1-1.562 0c-.179-.103-.35-.274-.692-.617c-.343-.342-.514-.514-.617-.692a1.56 1.56 0 0 1 0-1.562c.103-.178.274-.35.617-.692l5.523-5.523m-.736-.737l4.419 4.42c1.735 1.735 2.603 2.603 3.682 2.603s1.946-.868 3.682-2.604S22 13.783 22 12.705c0-1.079-.868-1.947-2.604-3.682l-4.419-4.42C13.242 2.869 12.374 2 11.295 2s-1.946.868-3.682 2.604s-2.604 2.604-2.604 3.682c0 1.079.868 1.947 2.604 3.682Z"/>',
  },
  fabric: {
    viewBox: "0 0 16 16",
    body: '<g fill-rule="evenodd" stroke-linejoin="round"><path fill="#38342a" d="M8 1v1H7v2H6v1H5v1H4v1H3v1H2v1H1v2h1v1h1v1h1v1h1v1h2v-1h1v-2h1v-1h1v-1h1V9h2V8h1V6h-1V5h-1V4h-1V3h-1V2H9V1z"/><path fill="#dbd0b4" d="M8 2v1h1v1h1v1h1v1h1V5h-1V4h-1V3H9V2zM7 4v2H5v1H4v1H3v2h1v1h1v1h2v-2H6V9h2v1h1V8H8V7h2V6H9V5H8V4z"/><path fill="#38342a" d="M8 4v1h1v1h1v1h1V6h-1V5H9V4z"/><path fill="#bcb29c" d="M9 4v1h1V4zm1 1v1h1V5zm1 1v1h1V6zm0 1h-1v1H9v2h1V9h1zm-2 3H7v2h1v-1h1zm-2 2H6v1h1z"/><path fill="#807a6d" d="M12 7v1h1V7z"/><path fill="#aea694" d="M2 9v1h1v1h1v1h1v1h1v-1H5v-1H4v-1H3V9z"/><path fill="#9a927e" d="M2 10v1h1v-1zm1 1v1h1v-1zm1 1v1h1v-1zm1 1v1h1v-1z"/><path fill="#c6bca5" d="M8 3v1h1V3zM6 5v1h1V5zm1 1v1h1V6zm1 1v1h2V7zM5 8v1h1V8zm1 1v1h2V9z"/></g>',
  },
};

const PROVIDER_FAVICONS = {
  curseforge: "/curseforge-favicon.png",
  fabric: "/fabric-favicon.png",
  forge: "/forge-favicon.png",
  modrinth: "/modrinth-favicon.png",
  neoforge: "/neoforge-favicon.png",
  quilt: "/quilt-favicon.png",
};

function normalizedProviderKey(value) {
  const compact = String(value || "unknown").trim().toLowerCase().replace(/[^a-z0-9]+/g, "");
  return {
    curse: "curseforge",
    curseforge: "curseforge",
    forgecdn: "curseforge",
    fabricloader: "fabric",
    fabricmc: "fabric",
    forge: "forge",
    minecraftforge: "forge",
    modrinth: "modrinth",
    neoforge: "neoforge",
    neoforged: "neoforge",
    quilt: "quilt",
    quiltmc: "quilt",
  }[compact] || compact;
}

function providerIconFallback(key) {
  const iconData = PROVIDER_ICONS[key === "neoforge" ? "forge" : key];
  if (!iconData) return solarIcon("server-square-linear", "provider-icon");
  const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  icon.setAttribute("class", "provider-icon");
  icon.setAttribute("viewBox", iconData.viewBox);
  icon.setAttribute("aria-hidden", "true");
  icon.innerHTML = iconData.body;
  return icon;
}

function serverProviderLabel(server) {
  const sourceHint = `${server.source_provider || ""} ${server.source_url_host || ""} ${server.source_type || ""}`.toLowerCase();
  let inferredProvider = "";
  if (sourceHint.includes("curseforge")) {
    inferredProvider = "curseforge";
  } else if (sourceHint.includes("modrinth")) {
    inferredProvider = "modrinth";
  } else if (sourceHint.includes("neoforge")) {
    inferredProvider = "neoforge";
  } else if (sourceHint.includes("minecraftforge") || sourceHint.includes("files.minecraftforge")) {
    inferredProvider = "forge";
  } else if (sourceHint.includes("fabricmc")) {
    inferredProvider = "fabric";
  } else if (sourceHint.includes("quiltmc")) {
    inferredProvider = "quilt";
  }
  const raw = String(server.modloader || (server.minecraft_version ? "vanilla" : "") || server.provider || inferredProvider || "unknown").trim();
  const key = normalizedProviderKey(raw);
  const favicon = PROVIDER_FAVICONS[key];
  const icon = favicon
    ? el("img", { class: "provider-icon", src: favicon, alt: "", width: "18", height: "18", loading: "lazy", decoding: "async" })
    : providerIconFallback(key);
  if (favicon) {
    icon.addEventListener("error", () => icon.replaceWith(providerIconFallback(key)), { once: true });
  }
  const names = { forge: "Forge", neoforge: "NeoForge", fabric: "Fabric", quilt: "Quilt", vanilla: "Vanilla", modrinth: "Modrinth", curseforge: "CurseForge" };
  const version = String(server.modloader_version || "").trim();
  return el("span", { class: "server-provider" }, icon, names[key] || raw, version ? " " + version : "");
}

function serverVersionSummary(server) {
  const items = [serverProviderLabel(server)];
  const gameVersion = String(server.minecraft_version || "").trim();
  if (gameVersion) items.push(el("span", { class: "server-game-version" }, gameVersionIcon(), gameVersion));
  items.push(document.createTextNode("imported via " + server.source_type));
  const children = [];
  items.forEach((item, index) => {
    if (index) children.push(el("span", { "aria-hidden": "true" }, "·"));
    children.push(item);
  });
  return el("div", { class: "server-provider-row muted" }, children);
}

function worldDownloadMenuItem(server) {
  const href = DEMO_MODE ? "demo-world.zip" : `/api/servers/${server.id}/world.zip`;
  return el("a", { class: "action-menu-item", role: "menuitem", href, download: `${server.slug}-world.zip` },
    solarIcon("download-linear"), "Download world");
}

function closeActionMenus(except = null) {
  document.querySelectorAll(".action-menu:not([hidden])").forEach((menu) => {
    if (menu === except) return;
    closeActionMenu(menu);
  });
}

function closeActionMenu(menu) {
  menu.hidden = true;
  menu.actionTrigger?.setAttribute("aria-expanded", "false");
  menu.removeAttribute("style");
  menu.actionOwner?.append(menu);
}

function positionActionMenu(menu, trigger) {
  const triggerRect = trigger.getBoundingClientRect();
  const gap = 4;
  const edge = 8;
  const width = menu.offsetWidth;
  const height = menu.offsetHeight;
  const left = Math.max(edge, Math.min(window.innerWidth - width - edge, triggerRect.right - width));
  let top = triggerRect.bottom + gap;
  if (top + height > window.innerHeight - edge) top = Math.max(edge, triggerRect.top - height - gap);
  Object.assign(menu.style, { position: "fixed", left: left + "px", right: "auto", top: top + "px" });
}

function overflowActionsMenu(label, items, className = "") {
  if (!items.length) return null;
  const menu = el("div", { class: "action-menu", role: "menu", hidden: "" }, ...items);
  const trigger = el("button", {
    class: "btn ghost small icon-button action-menu-trigger",
    type: "button",
    title: label,
    "aria-label": label,
    "aria-haspopup": "menu",
    "aria-expanded": "false",
    onclick: (event) => {
      event.stopPropagation();
      const open = menu.hidden;
      if (!open) {
        closeActionMenu(menu);
        return;
      }
      closeActionMenus();
      document.body.append(menu);
      menu.hidden = false;
      trigger.setAttribute("aria-expanded", "true");
      positionActionMenu(menu, trigger);
      menu.querySelector('[role="menuitem"]')?.focus();
    },
  }, solarIcon("menu-dots-bold"));

  menu.addEventListener("click", () => closeActionMenus());
  menu.addEventListener("keydown", (event) => {
    const options = [...menu.querySelectorAll('[role="menuitem"]')];
    const index = options.indexOf(document.activeElement);
    if (event.key === "Escape") {
      event.preventDefault();
      closeActionMenus();
      trigger.focus();
      return;
    }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const next = event.key === "Home" ? 0
      : event.key === "End" ? options.length - 1
        : (index + (event.key === "ArrowDown" ? 1 : -1) + options.length) % options.length;
    options[next]?.focus();
  });
  trigger.addEventListener("keydown", (event) => {
    if (event.key !== "Escape" || menu.hidden) return;
    event.preventDefault();
    closeActionMenus();
  });

  const owner = el("div", { class: "overflow-actions" + (className ? " " + className : "") }, trigger, menu);
  menu.actionOwner = owner;
  menu.actionTrigger = trigger;
  return owner;
}

function serverActionsMenu(server) {
  const items = [];
  if (can("server.configuration.manage")) items.push(
    el("button", { class: "action-menu-item", type: "button", role: "menuitem", onclick: () => renameProject(server) },
      solarIcon("pen-new-square-linear"), "Rename"));
  if (can("server.files.manage")) items.push(worldDownloadMenuItem(server));
  if (can("server.import.manage")) items.push(
    el("button", { class: "action-menu-item", type: "button", role: "menuitem", onclick: () => duplicateProject(server) },
      solarIcon("copy-linear"), "Duplicate"));
  if (can("server.configuration.manage")) items.push(
    el("button", { class: "action-menu-item danger", type: "button", role: "menuitem", onclick: () => resetWorld(server) },
      solarIcon("refresh-linear"), "Reset world"));
  if (can("server.import.manage")) items.push(
    el("button", { class: "action-menu-item danger", type: "button", role: "menuitem", onclick: () => deleteProject(server) },
      solarIcon("trash-bin-trash-linear"), "Delete"));
  return overflowActionsMenu(`Actions for ${server.display_name}`, items, "server-actions");
}

function serverManagementButton(server, page, label, icon) {
  return el("button", {
    class: "btn ghost icon-button server-card-quick-action",
    type: "button",
    title: label,
    "aria-label": `${label} for ${server.display_name}`,
    onclick: () => {
      if (page === "files") {
        filePath = "";
        fileEscapeAction = null;
        S.pendingFileOpen = null;
      }
      navigate(page, { serverId: server.id });
    },
  }, solarIcon(icon));
}

document.addEventListener("click", () => closeActionMenus());

function renameProject(server) {
  const name = el("input", { value: server.display_name, maxlength: "120", autocomplete: "off" });
  modal("Rename server", [
    el("p", { class: "muted" }, "Only the display name changes. The project slug and files stay unchanged."),
    el("div", { class: "field-row" }, el("label", {}, "Server name", name)),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Rename", "primary", async (close) => {
      try {
        const displayName = name.value.trim();
        if (!displayName) throw new Error("Enter a server name.");
        if (displayName === server.display_name) { close(); return; }
        const updated = await api(`/servers/${server.id}`, { method: "PATCH", json: { display_name: displayName } });
        close();
        toast(`Renamed server to ${updated.display_name}`, "ok");
        await renderPage();
      } catch (error) { toast(error.message, "err"); }
    }],
  ]);
  requestAnimationFrame(() => name.select());
}

function duplicateProject(server) {
  const name = el("input", { value: server.display_name + " Copy", maxlength: "120" });
  modal("Duplicate server", [
    el("p", { class: "muted" }, "Copies the complete server into a new managed project. Autostart stays disabled on the copy."),
    el("div", { class: "field-row" }, el("label", {}, "New server name", name)),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Duplicate", "primary", async (close) => {
      try {
        const displayName = name.value.trim();
        if (!displayName) throw new Error("Enter a name for the duplicate.");
        await api(`/servers/${server.id}/duplicate`, { method: "POST", json: { display_name: displayName } });
        close();
        toast("Server duplication started", "ok");
        renderPage();
      } catch (error) { toast(error.message, "err"); }
    }],
  ]);
}

function resetWorld(server) {
  confirmModal("Reset world",
    `Reset the world for "${server.display_name}"? Bonghos creates a verified safety backup first, then removes the current world so Minecraft generates a new one on the next start. The server must be stopped.`,
    "Reset world", async () => {
      try {
        const result = await api(`/servers/${server.id}/world/reset`, { method: "POST", json: { confirm: true } });
        toast("World reset. Safety backup: " + (result.backup_id || "created"), "ok");
      } catch (error) { toast(error.message, "err"); }
    });
}

async function pageServers(main) {
  await refreshServers();
  const ops = await api("/operations?active=true").catch(() => []);
  main.innerHTML = "";
  const serverCard = (s2) => el("div", { class: "card server-card", id: `server-card-${s2.id}` },
    serverCardIcon(s2),
    s2.id === S.activeId ? el("span", { class: "tag server-card-active-mobile" }, "active") : null,
    el("div", { class: "server-card-body" },
      el("div", { class: "toolbar compact" },
        el("strong", {}, s2.display_name),
        s2.id === S.activeId ? el("span", { class: "tag server-card-active-desktop" }, "active") : "",
        s2.external_directory ? el("span", { class: "tag" }, "external link") : "",
        el("div", { class: "spacer" })),
      el("div", { class: "muted mono" }, s2.slug),
      serverVersionSummary(s2),
      el("div", { class: "row-actions action-row-spaced server-card-actions" },
        el("div", { class: "spacer" }),
        s2.id !== S.activeId && can("server.configuration.manage")
          ? el("button", { class: "btn small", onclick: async () => {
              try { await api(`/servers/${s2.id}/select`, { method: "POST", json: {} }); toast("Active project changed", "ok"); refreshServers().then(renderPage); }
              catch (e) { toast(e.message, "err"); }
            } }, "Make active") : "",
        can("server.files.manage")
          ? serverManagementButton(s2, "files", "Files", "folder-with-files-linear") : "",
        can("server.configuration.manage")
          ? serverManagementButton(s2, "configuration", "Configuration", "tuning-2-linear") : "",
        serverActionsMenu(s2))));
  const cardsHost = el("div", { class: "grid cols-2" });
  const draw = (query = "") => {
    const visible = S.servers.filter((server) => recordMatchesSearch(server, query));
    cardsHost.replaceChildren(...(visible.length
      ? visible.map(serverCard)
      : [el("p", { class: "muted" }, S.servers.length ? "No matching servers." : "No servers imported yet — use “Import server”.")]));
  };
  const search = pageSearchInput("servers", draw);
  main.append(
    pageHeader("Servers", "Project inventory, active-project selection, and persistent import progress.", [
      search,
      can("server.import.manage") ? el("button", { class: "btn primary", onclick: importWizard }, "Import server") : null,
    ], overviewBackButton()),
    el("div", { id: "ops-host" }, ...(ops || []).map(opCard)),
    cardsHost);
  draw();
  activatePendingServerTarget();
}

function activatePendingServerTarget() {
  if (S.page !== "servers" || S.serverTargetId == null) return false;
  const target = document.getElementById(`server-card-${S.serverTargetId}`);
  if (!target) return false;
  S.serverTargetId = null;
  activateNavigationTarget(target, "servers");
  return true;
}

function opCard(op) {
  const pct = op.total_bytes > 0 ? Math.min(100, (op.bytes_processed / op.total_bytes) * 100) : null;
  return el("div", { class: "card list-card", "data-op": op.id },
    el("div", { class: "toolbar compact" },
      el("strong", {}, op.kind), el("span", { class: "tag" }, op.stage.replace(/_/g, " ")),
      el("div", { class: "spacer" }),
      ["completed", "failed", "cancelled"].includes(op.stage) ? "" :
        el("button", { class: "btn ghost", onclick: async () => {
          try { await api(`/operations/${op.id}/cancel`, { method: "POST", json: {} }); } catch (e) { toast(e.message, "err"); }
        } }, "Cancel")),
    el("div", { class: "progress" + (pct === null ? " indeterminate" : "") },
      el("div", { style: `width:${pct === null ? 35 : pct}%` })),
    el("div", { class: "muted detail-note" },
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
    el("label", { class: "check-row" }, delFiles, " Also delete the server files on disk"),
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

function uploadArchiveChunk(operationID, offset, chunk, onProgress, onXHR) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    onXHR(xhr);
    xhr.upload.addEventListener("progress", (event) => onProgress(event.loaded));
    xhr.addEventListener("load", () => {
      let data = {};
      try { data = JSON.parse(xhr.responseText); } catch {}
      if (xhr.status >= 200 && xhr.status < 300) resolve(data);
      else reject(new Error(data.error || `Upload chunk failed (HTTP ${xhr.status})`));
    });
    xhr.addEventListener("error", () => reject(new Error("Upload failed: connection lost")));
    xhr.addEventListener("abort", () => reject(new Error("Upload cancelled")));
    xhr.open("PUT", `/api/imports/upload/${encodeURIComponent(operationID)}/chunk`);
    xhr.withCredentials = true;
    xhr.setRequestHeader("Content-Type", "application/octet-stream");
    xhr.setRequestHeader("X-Bonghos-CSRF", csrfToken);
    xhr.setRequestHeader("X-Bonghos-Upload-Offset", String(offset));
    xhr.send(chunk);
  });
}

// Archives are uploaded as small, ordered requests. Besides making retries
// cheaper, this stays below reverse-proxy request limits (Cloudflare rejects a
// single request above the plan's upload cap) without loading the file into
// browser or server memory.
function uploadArchive(file, displayName) {
  const host = $("#ops-host");
  const started = Date.now();
  let lastTime = started, lastLoaded = 0, instantRate = 0;
  let operationID = "";
  let currentXHR = null;
  let cancelled = false;

  const bar = el("div", { style: "width:0%" });
  const line = el("div", { class: "muted detail-note" }, "Preparing upload…");
  const cancelBtn = el("button", { class: "btn ghost" }, "Cancel");
  const card = el("div", { class: "card list-card" },
    el("div", { class: "toolbar compact" },
      el("strong", {}, "Uploading " + file.name),
      el("span", { class: "tag" }, "browser → host"),
      el("div", { class: "spacer" }), cancelBtn),
    el("div", { class: "progress" }, bar), line);
  if (host) host.prepend(card); else toast("Upload started", "ok");

  function updateProgress(loaded) {
    const now = Date.now();
    const dt = (now - lastTime) / 1000;
    if (dt >= 0.5) {
      instantRate = (loaded - lastLoaded) / dt;
      lastTime = now; lastLoaded = loaded;
    }
    const avg = loaded / Math.max(0.001, (now - started) / 1000);
    const pct = (loaded / file.size) * 100;
    bar.style.width = pct.toFixed(1) + "%";
    const remaining = instantRate > 0 ? (file.size - loaded) / instantRate : null;
    line.textContent =
      `${fmtBytes(loaded)} / ${fmtBytes(file.size)} · ${pct.toFixed(0)}%` +
      ` · ${fmtBytes(instantRate || avg)}/s` +
      (remaining !== null && isFinite(remaining) ? ` · about ${fmtDur(remaining)} remaining` : "");
  }

  cancelBtn.addEventListener("click", async () => {
    if (cancelled) return;
    cancelled = true;
    currentXHR?.abort();
    if (operationID) {
      try { await api(`/operations/${operationID}/cancel`, { method: "POST" }); } catch {}
    }
    card.remove();
    toast("Upload cancelled", "ok");
  });

  (async () => {
    try {
      const session = await api("/imports/upload/start", {
        method: "POST",
        json: { display_name: displayName, filename: file.name, size: file.size },
      });
      operationID = session.operation_id;
      const chunkSize = Number(session.chunk_size) || 16 * 1024 * 1024;
      let offset = Number(session.offset) || 0;
      while (offset < file.size) {
        if (cancelled) return;
        const end = Math.min(file.size, offset + chunkSize);
        const baseOffset = offset;
        const result = await uploadArchiveChunk(
          operationID,
          offset,
          file.slice(offset, end),
          (chunkLoaded) => updateProgress(baseOffset + chunkLoaded),
          (xhr) => { currentXHR = xhr; },
        );
        currentXHR = null;
        offset = Number(result.offset);
        if (!Number.isFinite(offset) || offset !== end) throw new Error("Host returned an invalid upload offset");
        updateProgress(offset);
      }
      cancelBtn.disabled = true;
      line.textContent = "Upload received · verifying archive…";
      await api(`/imports/upload/${operationID}/finish`, { method: "POST" });
      card.remove();
      toast("Upload complete — the host is now extracting and installing it", "ok");
    } catch (error) {
      if (cancelled) return;
      if (operationID) {
        try { await api(`/operations/${operationID}/cancel`, { method: "POST" }); } catch {}
      }
      card.remove();
      toast(error.message, "err");
    }
  })();
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
              "Linked directories stay outside Bonghos storage: they are NOT included in exports or the normal Bonghos migration, and Bonghos will operate on files in place. Continue?",
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
  const activityRow = (event) => el("tr", {},
    el("td", {}, fmtTime(event.at)), el("td", {}, event.username), el("td", {}, event.action.replace(/_/g, " ")),
    el("td", { class: "mono" }, event.target || ""), el("td", { class: "muted" }, event.detail || ""));
  const tbody = el("tbody");
  const draw = (query = "") => {
    const visible = (list || []).filter((event) => recordMatchesSearch(event, query, fmtTime(event.at), event.action.replace(/_/g, " ")));
    tbody.replaceChildren(...(visible.length
      ? visible.map(activityRow)
      : [el("tr", {}, el("td", { colspan: "5", class: "muted" }, (list || []).length ? "No matching activity." : "No audit events recorded yet."))]));
  };
  const search = pageSearchInput("activity", draw);
  main.append(
    pageHeader("Activity", "Audit trail of account and server-management actions.", [search]),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "User"), el("th", {}, "Action"), el("th", {}, "Target"), el("th", {}, "Detail"))),
        tbody)));
  draw();
}

// ----- users -----------------------------------------------------------------
async function pageUsers(main) {
  const users = await api("/users");
  main.innerHTML = "";
  const userRow = (u) => el("tr", {},
    el("td", {}, el("div", { class: "user-identity" },
      el("strong", {}, u.Username),
      el("span", { class: "mobile-only mobile-row-detail" }, `${u.Role} · ${u.Disabled ? "Disabled" : "Active"}`))),
    el("td", {}, el("span", { class: "tag" }, u.Role)),
    el("td", {}, u.Disabled ? "Disabled" : "Active"),
    el("td", { class: "table-actions" }, userActions(u)));
  const tbody = el("tbody");
  const draw = (query = "") => {
    const visible = (users || []).filter((user) => recordMatchesSearch(user, query, user.Disabled ? "disabled" : "active"));
    tbody.replaceChildren(...(visible.length
      ? visible.map(userRow)
      : [el("tr", {}, el("td", { colspan: "4", class: "muted" }, (users || []).length ? "No matching users." : "No users returned by the API."))]));
  };
  const search = pageSearchInput("users", draw);
  main.append(
    pageHeader("Users", "Accounts, roles, invitations, sessions, and final-Owner protection.", [
      search,
      el("button", { class: "btn primary", onclick: inviteUser }, "Invite user"),
    ]),
    el("div", { class: "table-wrap users-table" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Username"), el("th", {}, "Role"), el("th", {}, "Status"), el("th", {}, ""))),
        tbody)));
  draw();
}

function userActions(user) {
  if (user.ID === S.me.id) return el("span", { class: "muted" }, "you");
  const toggleDisabled = async () => {
    try { await api(`/users/${user.ID}/disable`, { method: "POST", json: { disabled: !user.Disabled } }); renderPage(); }
    catch (error) { toast(error.message, "err"); }
  };
  const revokeSessions = async () => {
    try { await api(`/users/${user.ID}/revoke-sessions`, { method: "POST", json: {} }); toast("Sessions revoked", "ok"); }
    catch (error) { toast(error.message, "err"); }
  };
  const deleteUser = () => confirmModal("Delete user", `Delete account "${user.Username}"?`, "Delete", async () => {
    try { await api(`/users/${user.ID}`, { method: "DELETE" }); renderPage(); }
    catch (error) { toast(error.message, "err"); }
  });
  const actions = [
    { label: "Role", icon: "key-linear", run: () => changeRole(user) },
    { label: user.Disabled ? "Enable" : "Disable", icon: user.Disabled ? "check-circle-linear" : "close-circle-linear", run: toggleDisabled },
    { label: "Revoke sessions", icon: "logout-2-linear", run: revokeSessions },
    { label: "Delete", icon: "trash-bin-trash-linear", danger: true, run: deleteUser },
  ];
  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    ...actions.map((action) => el("button", { class: "btn " + (action.danger ? "danger" : "ghost"), onclick: action.run }, action.label)));
  const mobile = overflowActionsMenu(`Actions for ${user.Username}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""), type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
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
    el("div", { class: "grid cols-2 flow-section" },
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

const BOT_EVENT_FIELDS = [
  ["notify_server_started", "Server started", "After Minecraft is fully ready"],
  ["notify_server_stopped", "Server stopped", "After the process fully exits"],
  ["notify_player_joined", "Player joins", "Includes player and server names"],
  ["notify_player_left", "Player leaves", "Includes player and server names"],
];

function botProviderName(provider) {
  return provider === "telegram" ? "Telegram" : "Discord";
}

function botProviderMark(provider) {
  return el("span", { class: `bot-provider-mark ${provider}`, "aria-hidden": "true" },
    el("img", {
      class: "bot-provider-logo",
      src: provider === "telegram" ? "/Telegram.png" : "/Discord.png",
      alt: "",
    }));
}

async function patchBot(bot, patch, control = null) {
  if (control) control.disabled = true;
  try {
    const updated = await api(`/bots/${bot.id}`, { method: "PATCH", json: patch });
    Object.assign(bot, updated);
    return updated;
  } finally {
    if (control) control.disabled = false;
  }
}

function botEventToggle(bot, field, label, note) {
  const input = el("input", { type: "checkbox", "aria-label": `${label} for ${bot.name}` });
  input.checked = !!bot[field];
  input.addEventListener("change", async () => {
    const previous = !input.checked;
    try {
      await patchBot(bot, { [field]: input.checked }, input);
      toast(`${label} notifications ${input.checked ? "enabled" : "disabled"}`, "ok");
    } catch (error) {
      input.checked = previous;
      toast(error.message, "err");
    }
  });
  return el("label", { class: "bot-event-option check-row" },
    el("span", { class: "bot-event-copy" }, el("strong", {}, label), el("small", {}, note)),
    input);
}

function botPowerButton(bot) {
  const button = el("button", {
    class: "bot-power" + (bot.enabled ? " is-on" : ""),
    type: "button",
    "aria-pressed": String(!!bot.enabled),
    "aria-label": `${bot.enabled ? "Turn off" : "Turn on"} ${bot.name}`,
    onclick: async () => {
      const next = !bot.enabled;
      try {
        await patchBot(bot, { enabled: next }, button);
        button.classList.toggle("is-on", next);
        button.closest(".bot-card")?.classList.toggle("is-disabled", !next);
        button.setAttribute("aria-pressed", String(next));
        button.setAttribute("aria-label", `${next ? "Turn off" : "Turn on"} ${bot.name}`);
        button.querySelector(".bot-power-label").textContent = next ? "On" : "Off";
        toast(`${bot.name} turned ${next ? "on" : "off"}`, "ok");
      } catch (error) { toast(error.message, "err"); }
    },
  }, el("span", { class: "bot-power-track", "aria-hidden": "true" }, el("span", {})),
  el("span", { class: "bot-power-label" }, bot.enabled ? "On" : "Off"));
  return button;
}

function botCard(bot) {
  const destinationLabel = bot.provider === "telegram" ? "Chat" : "Channel";
  return el("article", { class: "card bot-card" + (bot.enabled ? "" : " is-disabled") },
    el("div", { class: "bot-card-head" },
      botProviderMark(bot.provider),
      el("div", { class: "bot-card-identity" },
        el("strong", {}, bot.name),
        el("span", { class: "muted" }, botProviderName(bot.provider))),
      botPowerButton(bot)),
    el("div", { class: "bot-destination" },
      el("span", {}, destinationLabel),
      el("code", {}, bot.destination_id)),
    el("div", { class: "bot-events-title" }, "Notify when"),
    el("div", { class: "bot-event-grid" },
      ...BOT_EVENT_FIELDS.map(([field, label, note]) => botEventToggle(bot, field, label, note))),
    el("div", { class: "bot-card-actions" },
      el("button", { class: "btn ghost small", onclick: () => botEditor(bot) }, "Edit"),
      el("button", { class: "btn ghost small", onclick: async (event) => {
        const button = event.currentTarget;
        button.disabled = true;
        try {
          await api(`/bots/${bot.id}/test`, { method: "POST", json: {} });
          toast(`Test sent through ${bot.name}`, "ok");
        } catch (error) { toast(error.message, "err"); }
        finally { button.disabled = false; }
      } }, "Send test"),
      el("div", { class: "spacer" }),
      el("button", { class: "btn danger small", onclick: () => removeBot(bot) }, "Remove")));
}

function botEditor(existing = null) {
  const name = el("input", { value: existing?.name || "", maxlength: "80", autocomplete: "off", placeholder: "Server alerts" });
  const provider = el("select", {},
    el("option", { value: "telegram" }, "Telegram"),
    el("option", { value: "discord" }, "Discord"));
  provider.value = existing?.provider || "telegram";
  const token = el("input", {
    type: "password", autocomplete: "new-password", spellcheck: "false",
    placeholder: existing ? "Leave blank to keep the current token" : "Paste the bot token",
  });
  const destination = el("input", { value: existing?.destination_id || "", autocomplete: "off", spellcheck: "false" });
  const destinationTitle = el("span", {});
  const destinationHint = el("p", { class: "hint" });
  const enabled = el("input", { type: "checkbox" });
  enabled.checked = existing ? !!existing.enabled : true;
  const eventInputs = {};
  const renderDestination = () => {
    const telegram = provider.value === "telegram";
    destinationTitle.textContent = telegram ? "Chat ID" : "Channel ID";
    destination.placeholder = telegram ? "-1001234567890 or @channel" : "123456789012345678";
    destinationHint.textContent = telegram
      ? "Add the bot to the chat first. Private chats, groups, and channels use a numeric chat ID; public channels may use @username."
      : "Invite the bot to the Discord server and grant Send Messages in this channel.";
  };
  provider.disabled = !!existing;
  if (!existing) provider.addEventListener("change", renderDestination);
  renderDestination();

  const notificationRows = existing ? [] : BOT_EVENT_FIELDS.map(([field, label, note]) => {
    const input = el("input", { type: "checkbox" });
    input.checked = existing ? !!existing[field] : true;
    eventInputs[field] = input;
    return el("label", { class: "bot-modal-event check-row" }, input,
      el("span", {}, el("strong", {}, label), el("small", {}, note)));
  });

  modal(existing ? "Edit bot" : "Add bot", [
    el("p", { class: "muted" }, "Tokens are encrypted on this machine and are never shown again after saving."),
    el("div", { class: "field-row" }, el("label", {}, "Name", name)),
    el("div", { class: "field-row" }, el("label", {}, "Provider", provider),
      existing ? el("p", { class: "hint" }, "The provider cannot be changed after a bot is added.") : null),
    el("div", { class: "field-row" }, el("label", {}, existing ? "New bot token (optional)" : "Bot token", token),
      existing ? el("p", { class: "hint" }, "Leave blank to keep the encrypted token already stored.") : null),
    el("div", { class: "field-row" }, el("label", {}, destinationTitle, destination), destinationHint),
    existing ? null : el("label", { class: "check-row bot-enabled-row" }, enabled, " Bot enabled"),
    existing ? null : el("h3", { class: "bot-modal-heading" }, "Notifications"),
    existing ? null : el("div", { class: "bot-modal-events" }, notificationRows),
  ], [
    ["Cancel", "ghost", (close) => close()],
    [existing ? "Save changes" : "Add bot", "primary", async (close) => {
      if (!name.value.trim()) { toast("Bot name is required", "err"); name.focus(); return; }
      if (!token.value.trim() && !existing) {
        toast("Bot token is required", "err");
        token.focus(); return;
      }
      if (!destination.value.trim()) { toast("Destination is required", "err"); destination.focus(); return; }
      const body = {
        name: name.value.trim(), destination_id: destination.value.trim(),
      };
      if (!existing) {
        body.provider = provider.value;
        body.enabled = enabled.checked;
        BOT_EVENT_FIELDS.forEach(([field]) => { body[field] = eventInputs[field].checked; });
      }
      if (token.value.trim()) body.token = token.value.trim();
      try {
        if (existing) await api(`/bots/${existing.id}`, { method: "PATCH", json: body });
        else await api("/bots", { method: "POST", json: body });
        close();
        toast(existing ? "Bot updated" : "Bot added", "ok");
        await renderPage();
      } catch (error) { toast(error.message, "err"); }
    }],
  ]);
}

function removeBot(bot) {
  confirmModal("Remove bot", `Remove “${bot.name}”? Its encrypted token and notification settings will be deleted.`,
    "Remove", async () => {
      try {
        await api(`/bots/${bot.id}`, { method: "DELETE" });
        toast("Bot removed", "ok");
        await renderPage();
      } catch (error) { toast(error.message, "err"); }
    });
}

function settingsSectionHeading(id, title, subtitle, action = null) {
  return el("div", { class: "settings-section-head" },
    el("div", {}, el("h2", { id }, title), subtitle ? el("p", { class: "muted" }, subtitle) : null),
    action);
}

function botsSettingsSection(bots) {
  return el("section", { class: "settings-page-section", "aria-labelledby": "bots-settings-title" },
    settingsSectionHeading("bots-settings-title", "Bots", "Send server and player activity to Telegram or Discord.",
      el("button", { class: "btn primary", onclick: () => botEditor() }, "Add bot")),
    bots.length
      ? el("div", { class: "bot-grid" }, ...bots.map(botCard))
      : el("div", { class: "card bot-empty" }, solarIcon("send-square-linear"),
        el("strong", {}, "No notification bots yet"),
        el("p", { class: "muted" }, "Add a Telegram or Discord bot to receive server alerts."),
        el("button", { class: "btn", onclick: () => botEditor() }, "Add bot")));
}

async function pageSettings(main) {
  const [version, bots, host] = await Promise.all([
    api("/version").catch(() => ({ version: "unknown" })),
    can("security.manage") ? api("/bots") : Promise.resolve([]),
    can("server.configuration.manage") ? api("/host").catch(() => null) : Promise.resolve(null),
  ]);
  const makeThemeButton = (value, label) => el("button", {
    class: "btn ghost" + (themeChoice() === value ? " active" : ""),
    "data-theme-choice": value,
    onclick: () => setTheme(value),
  }, label);
  main.innerHTML = "";
  main.append(
    pageHeader("Settings", "Local preferences, notification bots, and application metadata."),
    el("section", { class: "settings-page-section", "aria-labelledby": "general-settings-title" },
      settingsSectionHeading("general-settings-title", "General", "Appearance and local preferences."),
      el("div", { class: "card" },
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Theme"), el("p", { class: "muted" }, "System follows the operating-system color scheme and reacts to changes.")),
        el("div", { class: "theme-choice" },
          makeThemeButton("system", "System"),
          makeThemeButton("dark", "Dark"),
          makeThemeButton("light", "Light"))))),
    can("security.manage") ? botsSettingsSection(bots) : null,
    el("section", { class: "settings-page-section", "aria-labelledby": "about-settings-title" },
      settingsSectionHeading("about-settings-title", "About", "Application details and monitoring behavior."),
      el("div", { class: "card" },
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Monitoring"), el("p", { class: "muted" }, "Live metrics are subscribed only on Overview and Performance.")),
        el("dl", { class: "kv" },
          el("dt", {}, "WebSocket topics"), el("dd", { class: "mono" }, BASE_TOPICS.join(", ") + " + active page"),
          el("dt", {}, "Console buffer"), el("dd", {}, "Latest 1000 lines from server history plus live events"))),
      host ? el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Services & panel"), el("p", { class: "muted" }, host.note || "Linux services and local runtime paths.")),
        el("dl", { class: "kv" },
          el("dt", {}, "Panel address"), el("dd", { class: "mono" }, `${host.bind_address}:${host.port}`),
          el("dt", {}, "Bonghos"), el("dd", { class: "mono" }, host.home),
          el("dt", {}, "systemd"), el("dd", {}, host.systemd ? "available" : "unavailable"),
          el("dt", {}, "bonghos.service"), el("dd", {}, host.service_bonghos || "—"),
          el("dt", {}, "bonghos-minecraft.service"), el("dd", {}, host.service_minecraft || "—"))) : null,
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Application"), el("p", { class: "muted" }, "Runtime settings not exposed by the API are shown honestly rather than mocked.")),
        el("dl", { class: "kv" },
          el("dt", {}, "Version"), el("dd", { class: "mono" }, version.version),
          el("dt", {}, "Frontend"), el("dd", {}, "Dependency-free vanilla HTML, CSS, and JavaScript embedded in the Go binary"))))));
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
        el("div", { class: "brand activation-title" }, "Account created"),
        el("p", {}, "Store these one-time recovery codes safely:"),
        el("pre", { class: "mono" }, (d.recovery_codes || []).join("\n")),
        el("a", { class: "btn primary", href: "/", style: "text-align:center" }, "Go to sign-in"));
    } },
      el("div", { class: "brand activation-title" }, "Activate your Bonghos account"),
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
