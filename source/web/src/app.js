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
  "arrow-left-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m11 5l-7 7m0 0l7 7m-7-7h16"/>',
  },
  "alt-arrow-right-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m9 5l6 7l-6 7"/>',
  },
  "alt-arrow-down-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m5 9l7 6l7-6"/>',
  },
  "alt-arrow-up-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-linejoin="miter" stroke-width="1.5" d="m5 15l7-6l7 6"/>',
  },
  "close-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-linecap="square" stroke-width="1.5" d="m6 6l12 12M18 6L6 18"/>',
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
  "global-linear": {
    body: '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.25 2.46 3.4 5.46 3.4 9S14.25 18.54 12 21M12 3C9.75 5.46 8.6 8.46 8.6 12s1.15 6.54 3.4 9"/></g>',
  },
  "wrap-text": {
    body: '<g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"><path d="M4 7h16M4 17h5m-5-5h13.5a2.5 2.5 0 0 1 2.5 2.5v0a2.5 2.5 0 0 1-2.5 2.5h-5"/><path d="M15 15.5L12.5 17l2.5 1.5z"/></g>',
  },
  "filter-linear": {
    body: '<path fill="none" stroke="currentColor" stroke-width="1.5" d="M19 3H5c-1.414 0-2.121 0-2.56.412S2 4.488 2 5.815v.69c0 1.037 0 1.556.26 1.986s.733.698 1.682 1.232l2.913 1.64c.636.358.955.537 1.183.735c.474.411.766.895.898 1.49c.064.284.064.618.064 1.285v2.67c0 .909 0 1.364.252 1.718c.252.355.7.53 1.594.88c1.879.734 2.818 1.101 3.486.683S15 19.452 15 17.542v-2.67c0-.666 0-1 .064-1.285a2.68 2.68 0 0 1 .899-1.49c.227-.197.546-.376 1.182-.735l2.913-1.64c.948-.533 1.423-.8 1.682-1.23c.26-.43.26-.95.26-1.988v-.69c0-1.326 0-1.99-.44-2.402C21.122 3 20.415 3 19 3Z"/>',
  },
  "plain-2-linear": {
    body: '<g fill="none" stroke="currentColor" stroke-width="1.5"><path d="m17.498 18.485l3.13-9.391c1.248-3.745 1.873-5.618.884-6.606c-.988-.989-2.86-.364-6.606.884l-9.331 3.11c-2.082.694-3.123 1.041-3.439 1.804q-.112.271-.133.564c-.059.824.717 1.6 2.269 3.151l.283.283c.254.254.382.382.478.523c.19.28.297.607.31.945c.008.171-.019.35-.072.705c-.196 1.304-.294 1.956-.179 2.458c.23 1 1.004 1.785 2 2.028c.5.123 1.154.034 2.46-.143l.072-.01c.368-.05.552-.075.729-.064c.32.019.63.124.898.303c.147.098.279.23.541.492l.252.252c1.51 1.51 2.265 2.265 3.066 2.226c.22-.011.438-.062.64-.152c.734-.323 1.072-1.336 1.747-3.362Z"/><path stroke-linecap="round" d="M6 18L21 3"/></g>',
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
  "Sign in with a passkey": "key-linear",
  "Start": "play-linear",
  "Stop": "stop-linear",
  "Unban": "check-circle-linear",
  "Use crop": "gallery-linear",
  "Verify": "shield-check-linear",
  "Whitelist": "shield-check-linear",
  "Unwhitelist": "shield-keyhole-linear",
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
  if (/^Send\b/.test(label)) return "plain-2-linear";
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

const LIFECYCLE_LOADING_CYCLE_MS = 2000;
let lifecycleLoadingIconId = 0;
function lifecycleLoadingIcon(onCycleEnd = null) {
  const id = `lifecycle-loading-${++lifecycleLoadingIconId}`;
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "icon lifecycle-loading-icon");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.innerHTML = `
    <path d="M0 0h24v24H0z" fill="none"/>
    <defs>
      <filter id="${id}-blur">
        <feGaussianBlur in="SourceGraphic" result="y" stdDeviation="1"/>
        <feColorMatrix in="y" result="z" values="1 0 0 0 0 0 1 0 0 0 0 0 1 0 0 0 0 0 18 -7"/>
        <feBlend in="SourceGraphic" in2="z"/>
      </filter>
    </defs>
    <g filter="url(#${id}-blur)">
      <circle cx="5" cy="12" r="4" fill="currentColor">
        <animate class="lifecycle-loading-cycle" attributeName="cx" calcMode="spline" dur="2s" keySplines=".36,.62,.43,.99;.79,0,.58,.57" repeatCount="indefinite" values="5;8;5"/>
      </circle>
      <circle cx="19" cy="12" r="4" fill="currentColor">
        <animate attributeName="cx" calcMode="spline" dur="2s" keySplines=".36,.62,.43,.99;.79,0,.58,.57" repeatCount="indefinite" values="19;16;19"/>
      </circle>
      <animateTransform attributeName="transform" dur="0.75s" repeatCount="indefinite" type="rotate" values="0 12 12;360 12 12"/>
    </g>`;
  if (onCycleEnd) svg.querySelector(".lifecycle-loading-cycle")?.addEventListener("repeatEvent", onCycleEnd);
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
const fmtTimeToMinute = (iso) => iso ? new Date(iso).toLocaleString(undefined, {
  year: "numeric", month: "numeric", day: "numeric", hour: "numeric", minute: "2-digit",
}) : "—";

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

function pageFilterMenu(label, modes, onChange, initialValue = modes[0]?.[0]) {
  let selected = initialValue;
  let control;
  const items = modes.map(([value, optionLabel]) => el("button", {
    class: "action-menu-item page-filter-option",
    type: "button", role: "menuitem",
    onclick: () => {
      selected = value;
      update();
      onChange(value);
    },
  }, solarIcon("check-circle-linear", "page-filter-selected-icon"), optionLabel));
  control = overflowActionsMenu(label, items, "page-filter-menu", solarIcon("filter-linear"));
  const update = () => {
    items.forEach((item, index) => {
      const active = modes[index][0] === selected;
      item.classList.toggle("is-active", active);
      item.setAttribute("aria-current", active ? "true" : "false");
    });
    const selectedLabel = modes.find(([value]) => value === selected)?.[1] || modes[0]?.[1] || "Filter";
    const trigger = control.querySelector(".action-menu-trigger");
    trigger.title = `${label}: ${selectedLabel}`;
    trigger.setAttribute("aria-label", `${label}: ${selectedLabel}`);
  };
  update();
  return control;
}

function toast(msg, kind = "", iconName = "") {
  const t = el("div", { class: `toast ${kind}${iconName ? " has-icon" : ""}`, role: "status" },
    iconName ? solarIcon(iconName, "toast-icon") : null,
    el("span", {}, msg));
  $("#toast-host").append(t);
  setTimeout(() => t.remove(), 6000);
}

function offlineWifiIcon() {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "offline-wifi-icon");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.innerHTML = `
    <path d="M0 0h24v24H0z" fill="none"/>
    <path fill="currentColor" d="M12,21L15.6,16.2C14.6,15.45 13.35,15 12,15C10.65,15 9.4,15.45 8.4,16.2L12,21">
      <animate attributeName="opacity" dur="1.55s" keyTimes="0;0.16129;0.80645;0.87097;1" repeatCount="indefinite" values="0;1;1;0;0"/>
    </path>
    <path fill="currentColor" d="M12,9C9.3,9 6.81,9.89 4.8,11.4L6.6,13.8C8.1,12.67 9.97,12 12,12C14.03,12 15.9,12.67 17.4,13.8L19.2,11.4C17.19,9.89 14.7,9 12,9Z">
      <animate attributeName="opacity" dur="1.55s" keyTimes="0;0.16129;0.32258;0.80645;0.87097;1" repeatCount="indefinite" values="0;0;1;1;0;0"/>
    </path>
    <path fill="currentColor" d="M12,3C7.95,3 4.21,4.34 1.2,6.6L3,9C5.5,7.12 8.62,6 12,6C15.38,6 18.5,7.12 21,9L22.8,6.6C19.79,4.34 16.05,3 12,3">
      <animate attributeName="opacity" dur="1.55s" keyTimes="0;0.32258;0.48387;0.80645;0.87097;1" repeatCount="indefinite" values="0;0;1;1;0;0"/>
    </path>`;
  return svg;
}

let offlineWarningToast = null;

function syncConnectivityWarning() {
  if (offlineWarningToast && !offlineWarningToast.isConnected) offlineWarningToast = null;
  if (navigator.onLine !== false) {
    offlineWarningToast?.remove();
    offlineWarningToast = null;
    return;
  }
  if (offlineWarningToast) return;
  const host = $("#toast-host");
  if (!host) return;
  offlineWarningToast = el("div", { class: "toast warn offline-warning-toast", role: "alert" },
    offlineWifiIcon(),
    el("div", { class: "offline-warning-copy" },
      el("strong", {}, "No internet connection"),
      el("span", { class: "muted" }, "Bonghos will reconnect automatically when your connection returns.")));
  host.append(offlineWarningToast);
}

function startConnectivityMonitor() {
  window.addEventListener("offline", syncConnectivityWarning);
  window.addEventListener("online", syncConnectivityWarning);
  syncConnectivityWarning();
}

let modalRestoreFocus = null;
let modalOnClose = null;

function closeActiveModal() {
  const host = $("#modal-host");
  if (!host || !host.firstElementChild) return false;
  host.innerHTML = "";
  const restoreFocus = modalRestoreFocus;
  const onClose = modalOnClose;
  modalRestoreFocus = null;
  modalOnClose = null;
  if (restoreFocus && restoreFocus.isConnected) restoreFocus.focus();
  if (onClose) onClose();
  return true;
}

function modal(title, bodyNodes, actions, onClose = null, modalClass = "") {
  const host = $("#modal-host");
  modalRestoreFocus = document.activeElement;
  modalOnClose = onClose;
  host.innerHTML = "";
  const close = closeActiveModal;
  const m = el("div", { class: "overlay", onclick: (e) => { if (e.target === m) close(); } },
    el("div", { class: `modal${modalClass ? " " + modalClass : ""}`, role: "dialog", "aria-modal": "true", "aria-label": title },
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
const DEMO_PARAMS = new URLSearchParams(location.search);
const DEMO_MODE = DEMO_PARAMS.has("demo");
const DEMO_VIEW = (DEMO_PARAMS.get("demo") || "app").toLowerCase();
const DEMO_DEBUG_BOTS = DEMO_MODE && DEMO_PARAMS.has("debug-bots");
const DEMO_ROLES = ["owner", "admin", "member", "viewer"];
const requestedDemoRole = (DEMO_PARAMS.get("demo-role") || "owner").toLowerCase();
const DEMO_INITIAL_ROLE = DEMO_ROLES.includes(requestedDemoRole) ? requestedDemoRole : "owner";
const DEMO_VIEW_ASSIGNABLE_ROLES = ["admin", "member", "viewer"];
const DEMO_ACTION_ASSIGNABLE_ROLES = ["admin", "member"];
const DEMO_PERMISSION_CATALOG = [
  ["server.view", "Access", "View server", "See Overview, Servers, and live status.", [], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["server.performance.view", "Access", "View performance", "Open machine and Java performance telemetry.", ["server.view"], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["server.performance.test", "Access", "Test internet speed", "Run a manual bandwidth test that can temporarily affect connected players.", ["server.performance.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.start", "Lifecycle", "Start server", "Start the active Minecraft server.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.stop", "Lifecycle", "Stop server", "Request a clean server shutdown.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.restart", "Lifecycle", "Restart server", "Stop and start the active server.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.force_stop", "Lifecycle", "Force stop server", "Immediately terminate a stuck server process.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.console.view", "Console and players", "View console", "Read server console output.", ["server.view"], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["server.console.use", "Console and players", "Use console", "Send unrestricted Minecraft server commands.", ["server.console.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.players.view", "Console and players", "View players", "See online and known players.", ["server.view"], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["server.players.manage", "Console and players", "Manage players", "Kick, ban, whitelist, or grant operator access to players.", ["server.players.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.files.manage", "Server files and configuration", "Manage files", "Read, upload, edit, move, and delete project files.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.configuration.manage", "Server files and configuration", "Manage configuration", "Change launch settings and server properties.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.icon.manage", "Server files and configuration", "Manage server icons", "Upload or remove project icons.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.import.manage", "Server files and configuration", "Manage projects", "Import, duplicate, reset, or delete server projects.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.backups.view", "Backups and automation", "View backups", "See backups and their storage location.", ["server.view"], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["server.backups.create", "Backups and automation", "Manage backups", "Create, verify, protect, and delete backups.", ["server.backups.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.backups.restore", "Backups and automation", "Restore backups", "Replace server data from a backup.", ["server.backups.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["server.schedules.manage", "Backups and automation", "Manage schedules", "Create and run lifecycle, console, and backup automation.", ["server.view", "server.start", "server.stop", "server.restart", "server.console.use", "server.backups.create"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["activity.view", "People and integrations", "View activity", "Review account and server administration history.", [], DEMO_VIEW_ASSIGNABLE_ROLES],
  ["users.manage", "People and integrations", "Manage users", "Invite users and manage lower-role accounts and sessions.", [], ["admin", "member"]],
  ["roles.manage", "People and integrations", "Manage role permissions", "Configure permissions for lower roles.", [], ["admin"]],
  ["playit.manage", "People and integrations", "Manage Playit.gg", "Configure the public game-server tunnel and Playit agent.", ["server.view"], ["admin"]],
  ["bots.manage", "People and integrations", "Manage notification bots", "Configure Discord and Telegram bots.", ["server.view"], DEMO_ACTION_ASSIGNABLE_ROLES],
  ["host.view", "System", "View host", "See Bonghos installation, listener, and service details.", [], DEMO_VIEW_ASSIGNABLE_ROLES],
].map(([id, group, label, description, requires, assignable_roles]) => ({ id, group, label, description, requires, assignable_roles }));
const DEMO_PERMS = DEMO_PERMISSION_CATALOG.map(({ id }) => id);
const DEMO_ROLE_DEFAULTS = {
  owner: [...DEMO_PERMS],
  admin: DEMO_PERMS.filter((permission) => permission !== "roles.manage" && permission !== "playit.manage"),
  member: ["server.view", "server.start", "server.stop", "server.restart", "server.players.view"],
  viewer: ["server.view", "server.players.view", "server.console.view"],
};
const DEMO_ROLE_PERMISSIONS = Object.fromEntries(
  Object.entries(DEMO_ROLE_DEFAULTS).map(([role, permissions]) => [role, [...permissions]]));
const DEMO_ME = {
  id: 1,
  username: "demo-user",
  role: DEMO_INITIAL_ROLE,
  permissions: [...DEMO_ROLE_PERMISSIONS[DEMO_INITIAL_ROLE]],
};
const DEMO_ROLE_REVISIONS = { owner: 0, admin: 0, member: 0, viewer: 0 };
const DEMO_ROLE_CUSTOMIZED = { owner: false, admin: false, member: false, viewer: false };
const demoUserSnapshot = () => ({ ...DEMO_ME, permissions: [...DEMO_ME.permissions] });
const DEMO_SERVERS = [
  { id: 1, slug: "bio1", display_name: "Bio1 Survival - Long Local Demo Server Name", provider: "curseforge", modloader: "neoforge", modloader_version: "21.1.228", minecraft_version: "1.21.1", source_type: "direct-url", server_directory: "servers/minecraft-java/modded/bio1", external_directory: false, startup_script: "run.sh", restart_policy: "on-failure", autostart_enabled: true, created_at: new Date(Date.now() - 30 * 86400000).toISOString(), updated_at: new Date(Date.now() - 2 * 3600000).toISOString(), demo_icon: "demo-server-bio1.png" },
  { id: 2, slug: "creative-lab", display_name: "Creative Lab", provider: "modrinth", modloader: "fabric", modloader_version: "0.16.10", minecraft_version: "1.21.1", source_type: "archive-upload", server_directory: "servers/minecraft-java/modded/creative-lab", external_directory: false, created_at: new Date(Date.now() - 12 * 86400000).toISOString(), updated_at: new Date(Date.now() - 86400000).toISOString(), demo_icon: "demo-server-creative-lab.png" },
];
const DEMO_BOTS = [
  { id: 1, name: "Server alerts", provider: "telegram", destination_id: "-1001234567890", destinations: [{ id: "-1001234567890", name: "Server staff", type: "supergroup", photo_url: "/demo-server-bio1.png", forum: true, thread_id: 23, thread_name: "Server alerts" }, { id: "-1009876543210", name: "Players", type: "supergroup", photo_url: "/demo-server-creative-lab.png" }], discovered_destinations: [{ id: "-1001234567890", name: "Server staff", type: "supergroup", photo_url: "/demo-server-bio1.png", discovered_at: new Date(Date.now() - 21 * 86400000).toISOString() }, { id: "-1009876543210", name: "Players", type: "supergroup", photo_url: "/demo-server-creative-lab.png", discovered_at: new Date(Date.now() - 14 * 86400000).toISOString() }, { id: "-1005555555555", name: "Build Team", type: "supergroup", discovered_at: new Date(Date.now() - 2 * 86400000).toISOString() }], enabled: true, notify_server_started: true, notify_server_stopped: true, notify_player_joined: true, notify_player_left: true, token_configured: true },
  { id: 2, name: "Staff channel", provider: "discord", dns_server: "", destination_id: "123456789012345678", destinations: [{ id: "123456789012345678", name: "bot-spam", type: "channel", guild_id: "223456789012345678", guild_name: "Bonghos Community", guild_icon: "demo" }], discovered_destinations: [{ id: "223456789012345678", name: "Bonghos Community", type: "guild", guild_id: "223456789012345678", guild_name: "Bonghos Community", guild_icon: "demo", discovered_at: new Date(Date.now() - 18 * 86400000).toISOString() }, { id: "323456789012345678", name: "Creative Server", type: "guild", guild_id: "323456789012345678", guild_name: "Creative Server", discovered_at: new Date(Date.now() - 5 * 86400000).toISOString() }], enabled: false, notify_server_started: true, notify_server_stopped: true, notify_player_joined: false, notify_player_left: false, token_configured: true },
];
let DEMO_PLAYIT = {
  enabled: false, account_mode: "account", management_mode: "none",
  secret_configured: false, agent_id: "", agent_name: "", tunnel_id: "", public_address: "",
  local_port: 25565, claim_pending: false, claim_url: "", agent_online: false,
  account_status: "", tunnel_status: "",
  detections: [{ kind: "docker", name: "playit", state: "running", externally_managed: true }],
  daemon_available: false, managed_state: "inactive", notice: "",
};
const DEMO_TELEGRAM_GROUPS = [
  { id: "-1001234567890", name: "Server staff", type: "supergroup", photo_url: "/demo-server-bio1.png", forum: true, topics: [{ id: 23, name: "Server alerts" }, { id: 91, name: "Admin chat" }] },
  { id: "-1009876543210", name: "Players", type: "supergroup", photo_url: "/demo-server-creative-lab.png" },
  { id: "-1005555555555", name: "Build testing", type: "group" },
];
const DEMO_PASSKEYS = [
  { id: 1, name: "Laptop passkey", rp_id: location.hostname, created_at: new Date(Date.now() - 12 * 86400000).toISOString(), last_used_at: new Date(Date.now() - 18 * 60000).toISOString(), backup_eligible: true, backed_up: true },
  { id: 2, name: "YubiKey 5", rp_id: location.hostname, created_at: new Date(Date.now() - 36 * 86400000).toISOString(), backup_eligible: false, backed_up: false },
];
let DEMO_RECOVERY_CODES = Array.from({ length: 8 }, (_, index) => ({
  id: index + 1,
  created_at: new Date(Date.now() - 30 * 86400000).toISOString(),
  used_at: index === 1 ? new Date(Date.now() - 8 * 86400000).toISOString() : null,
}));
const DEMO_CONSOLE = [
  "[19:27:36] [Server thread/INFO]: Starting minecraft server version 1.20.1",
  "[19:27:43] [Server thread/INFO]: Loading Forge mods from /home/klaude/bonghos/servers/minecraft-java/modded/bio1/mods",
  "[19:28:12] [Server thread/WARN]: Can't keep up! Is the server overloaded? Running 2475ms behind",
  "[19:28:40] [Server thread/INFO]: Steve joined the game",
  "[19:29:04] [Server thread/ERROR]: Example datapack warning for visual review only",
];
const DEMO_INVITE_TOKEN = "demo-invite";
const DEMO_INVITE_SECRET = "JBSWY3DPEHPK3PXP";
const DEMO_INVITE_QR = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120" role="img" aria-label="Demo authenticator QR code"><rect width="120" height="120" fill="#fff"/><path fill="#000" d="M8 8h32v32H8zm8 8v16h16V16zm64-8h32v32H80zm8 8v16h16V16zM8 80h32v32H8zm8 8v16h16V88zm40-72h8v8h-8zm8 8h8v16h-8zM48 40h16v8H48zm24 0h8v16h-8zM48 56h8v16h-8zm16 0h16v8H64zm24-8h16v8H88zM48 80h8v24h-8zm16-8h8v16h-8zm8 16h16v8H72zm24-24h8v16h-8zm-8 16h8v24h-8zm16 8h8v24h-8z"/></svg>';
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

async function developmentBotApi(path, opts = {}) {
  const method = opts.method || "GET";
  const response = await fetch("/__dev" + path, {
    method,
    headers: {
      "Content-Type": "application/json",
      "X-Bonghos-Dev-Relay": "1",
    },
    body: method === "GET" ? undefined : JSON.stringify(opts.json || {}),
    credentials: "same-origin",
  });
  let payload = {};
  try { payload = await response.json(); } catch { /* handled below */ }
  if (!response.ok) throw new Error(payload.error || `Development relay returned HTTP ${response.status}`);
  return payload;
}

async function demoApi(path, opts = {}) {
  await demoDelay();
  const method = opts.method || "GET";
  const clean = path.split("?")[0];
  const query = new URL(path, "http://bonghos.demo").searchParams;
  if (DEMO_DEBUG_BOTS && /^\/bots(?:\/|$)/.test(clean)) {
    return developmentBotApi(clean, opts);
  }
  if (method !== "GET") {
    if (clean === "/auth/login") return demoUserSnapshot();
    if (clean === "/account/reauth/password") return { action_token: "demo-account-action" };
    if (clean === "/account/password") return { ok: true };
    if (clean === "/account/totp/begin") return {
      setup_token: "demo-totp-setup", secret: DEMO_INVITE_SECRET,
      uri: `otpauth://totp/Bonghos:demo-user?secret=${DEMO_INVITE_SECRET}&issuer=Bonghos`,
      qr_svg: DEMO_INVITE_QR,
    };
    if (clean === "/account/totp/finish") {
      DEMO_RECOVERY_CODES = Array.from({ length: 8 }, (_, index) => ({
        id: index + 20, created_at: new Date().toISOString(), used_at: null,
      }));
      return { recovery_codes: Array.from({ length: 8 }, (_, index) => `demo${index + 1}-code${index + 1}`) };
    }
    if (clean === "/account/recovery-codes/regenerate") {
      DEMO_RECOVERY_CODES = Array.from({ length: 8 }, (_, index) => ({
        id: index + 40, created_at: new Date().toISOString(), used_at: null,
      }));
      return { recovery_codes: Array.from({ length: 8 }, (_, index) => `fresh${index + 1}-code${index + 1}`) };
    }
    if (clean === `/invitations/${DEMO_INVITE_TOKEN}/totp`) return {
      secret: DEMO_INVITE_SECRET,
      uri: `otpauth://totp/Bonghos:invited-admin?secret=${DEMO_INVITE_SECRET}&issuer=Bonghos`,
      qr_svg: DEMO_INVITE_QR,
    };
    if (clean === `/invitations/${DEMO_INVITE_TOKEN}/activate`) return {
      recovery_codes: ["DEMO-4Q7P-9K2M", "DEMO-8N3R-6T5W", "DEMO-2Y9H-7C4X", "DEMO-5F8J-3L6V"],
    };
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
    if (method === "PUT" && clean === "/playit") {
      DEMO_PLAYIT = { ...DEMO_PLAYIT, ...opts.json };
      if (DEMO_PLAYIT.management_mode === "external") {
        DEMO_PLAYIT.claim_pending = false;
        DEMO_PLAYIT.claim_url = "";
      }
      return { ...DEMO_PLAYIT };
    }
    if (method === "POST" && clean === "/playit/claim") {
      DEMO_PLAYIT = {
        ...DEMO_PLAYIT, management_mode: "bonghos",
        account_mode: opts.json?.account_mode || "account", claim_pending: true,
        claim_url: "https://playit.gg/claim/b0a605de00",
      };
      return { ...DEMO_PLAYIT };
    }
    if (method === "POST" && clean === "/playit/claim/poll") {
      DEMO_PLAYIT = {
        ...DEMO_PLAYIT, claim_pending: false, claim_url: "", secret_configured: true,
        agent_id: "7c66e87b-demo-agent", agent_online: true, managed_state: "active",
        account_status: DEMO_PLAYIT.account_mode === "guest" ? "guest" : "verified",
      };
      return { state: "complete", config: { ...DEMO_PLAYIT } };
    }
    if (method === "POST" && clean === "/playit/tunnel") {
      DEMO_PLAYIT = {
        ...DEMO_PLAYIT, tunnel_id: "12d5b7c0-demo-tunnel",
        public_address: "bonghos-demo.gl.joinmc.link:25565", tunnel_status: "configured",
      };
      return { ...DEMO_PLAYIT };
    }
    if (method === "PUT" && clean === "/playit/agent") {
      const name = String(opts.json?.name || "").trim();
      if (!name) throw new Error("Enter a valid Playit agent name");
      DEMO_PLAYIT = { ...DEMO_PLAYIT, agent_name: name };
      return { ...DEMO_PLAYIT };
    }
    if (method === "DELETE" && clean === "/playit/tunnel") {
      DEMO_PLAYIT = { ...DEMO_PLAYIT, tunnel_id: "", public_address: "", tunnel_status: "" };
      return { ...DEMO_PLAYIT };
    }
    if (method === "POST" && clean === "/playit/guest-login") {
      return { url: "https://playit.gg/" };
    }
    if (method === "POST" && clean === "/playit/refresh") return { ...DEMO_PLAYIT };
    if (method === "POST" && clean === "/metrics/internet/speed-test") {
      await new Promise((resolve) => setTimeout(resolve, 900));
      return {
        tested_at: new Date().toISOString(), provider: "Cloudflare", latency_ms: 36.8,
        download_mbps: 286.4, upload_mbps: 91.7,
        download_bytes: 37 * 1024 * 1024, upload_bytes: 16 * 1024 * 1024, duration_ms: 4820,
      };
    }
    const rolePermissionsMatch = clean.match(/^\/roles\/(admin|member|viewer)\/permissions$/);
    if (method === "PUT" && rolePermissionsMatch) {
      const role = rolePermissionsMatch[1];
      if (Number(opts.json?.revision) !== DEMO_ROLE_REVISIONS[role]) throw new Error("Role permissions changed elsewhere; reload and try again");
      DEMO_ROLE_PERMISSIONS[role] = [...new Set(opts.json?.permissions || [])];
      DEMO_ROLE_REVISIONS[role]++;
      DEMO_ROLE_CUSTOMIZED[role] = true;
      return demoRolePermissionsPayload();
    }
    if (clean === "/servers/slug-preview") {
      const name = (opts.json && opts.json.name) || "server";
      return { slug: name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "server" };
    }
    if (method === "POST" && (clean === "/bots/telegram/discover" || /^\/bots\/\d+\/telegram\/discover$/.test(clean))) {
      return { bot_username: "bonghos_demo_bot", groups: DEMO_TELEGRAM_GROUPS.map((group) => ({ ...group })) };
    }
    if (method === "POST" && clean === "/bots") {
      const created = { id: Math.max(0, ...DEMO_BOTS.map((bot) => bot.id)) + 1, ...opts.json, token_configured: true };
      delete created.token;
      if (Array.isArray(created.destinations) && created.destinations.length) created.destination_id = created.destinations[0].id;
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
        if (Array.isArray(next.destinations) && next.destinations.length) DEMO_BOTS[index].destination_id = next.destinations[0].id;
        return { ...DEMO_BOTS[index] };
      }
    }
    const demoBotTest = clean.match(/^\/bots\/(\d+)\/test$/);
    if (method === "POST" && demoBotTest) {
      const bot = DEMO_BOTS.find((entry) => entry.id === Number(demoBotTest[1]));
      if (!bot) throw new Error("Notification bot not found");
      if (!bot.enabled) throw new Error("Notification bot is disabled");
      return { ok: true };
    }
    const passkeyMatch = clean.match(/^\/passkeys\/(\d+)$/);
    if (passkeyMatch) {
      const index = DEMO_PASSKEYS.findIndex((passkey) => passkey.id === Number(passkeyMatch[1]));
      if (index < 0) throw new Error("Passkey not found");
      if (method === "PATCH") {
        const name = String(opts.json?.name || "").trim();
        if (!name) throw new Error("Passkey name is required");
        if (name.length > 80) throw new Error("Passkey name must be 80 characters or fewer");
        DEMO_PASSKEYS[index].name = name;
        return { id: DEMO_PASSKEYS[index].id, name };
      }
      if (method === "DELETE") {
        DEMO_PASSKEYS.splice(index, 1);
        return { ok: true };
      }
    }
    const updateMatch = clean.match(/^\/servers\/(\d+)$/);
    if (method === "PATCH" && updateMatch) {
      const server = DEMO_SERVERS.find((entry) => entry.id === Number(updateMatch[1]));
      if (!server) throw new Error("Server not found");
      if (opts.json?.display_name !== undefined) {
        const displayName = String(opts.json.display_name).trim();
        if (!displayName) throw new Error("Display name is required");
        if ([...displayName].length > 120) throw new Error("Display name must be 120 characters or fewer");
        server.display_name = displayName;
        server.updated_at = new Date().toISOString();
      }
      return { ...server };
    }
    const duplicateMatch = clean.match(/^\/servers\/(\d+)\/duplicate$/);
    if (duplicateMatch) {
      const source = DEMO_SERVERS.find((server) => server.id === Number(duplicateMatch[1]));
      const displayName = (opts.json && opts.json.display_name) || ((source && source.display_name) || "Server") + " Copy";
      const slugBase = displayName.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "server-copy";
      const createdAt = new Date().toISOString();
      const clone = { ...source, id: Math.max(...DEMO_SERVERS.map((server) => server.id)) + 1, display_name: displayName, slug: slugBase, autostart_enabled: false, created_at: createdAt, updated_at: createdAt };
      DEMO_SERVERS.push(clone);
      return { operation_id: "demo-duplicate", server: clone };
    }
    if (/^\/servers\/\d+\/world\/reset$/.test(clean)) {
      return { ok: true, backup_id: "demo-pre-reset-world" };
    }
    return { ok: true };
  }
  const demoBotInvite = clean.match(/^\/bots\/(\d+)\/invite$/);
  if (demoBotInvite) {
    const bot = DEMO_BOTS.find((entry) => entry.id === Number(demoBotInvite[1]));
    if (!bot) throw new Error("Notification bot not found");
    return { url: bot.provider === "telegram"
      ? "https://t.me/bonghos_demo_bot?startgroup"
      : "https://discord.com/oauth2/authorize?client_id=1536799744431755275&scope=bot%20applications.commands&permissions=274877910016&integration_type=0" };
  }
  switch (clean) {
    case "/auth/csrf": return { csrf: "demo-csrf-token" };
    case "/auth/me": return demoUserSnapshot();
    case `/invitations/${DEMO_INVITE_TOKEN}`: return { role: "admin" };
    case "/bots": return DEMO_BOTS.map((bot) => ({ ...bot }));
    case "/playit": return { ...DEMO_PLAYIT };
    case "/version": return { version: "0.3.0-rc.1-demo" };
    case "/servers": return { servers: DEMO_SERVERS, active_id: 1 };
    case "/server/status": return S.status;
    case "/server/console/history": return { lines: DEMO_CONSOLE.slice(-CONSOLE_LINE_LIMIT), limit: CONSOLE_LINE_LIMIT, source: "demo" };
    case "/overview": return {
      state: S.status.state,
      version: "0.3.0-rc.1-demo",
      instance: DEMO_SERVERS[0],
      motd: "A precise Bonghos local demo",
      playit_address: DEMO_PLAYIT.enabled ? DEMO_PLAYIT.public_address : "",
      lan_ip: "192.168.1.42",
      port: "25565",
      max_players: "20",
      last_backup: { created_at: new Date(Date.now() - 5 * 3600000).toISOString() },
      next_schedule_at: new Date(Date.now() + 3 * 3600000).toISOString(),
      sample: DEMO_METRICS[DEMO_METRICS.length - 1],
    };
    case "/host": return {
      bind_address: "127.0.0.1", port: 8080, home: "/home/demo/bonghos",
      session_hours: 72,
      metrics_interval_seconds: 10,
      mem_total: 32 * 1024 * 1024 * 1024, mem_available: 18 * 1024 * 1024 * 1024,
      disk_total: 512 * 1024 * 1024 * 1024, disk_free: 186 * 1024 * 1024 * 1024,
      load1: 0.82, systemd: true, service_bonghos: "active", service_minecraft: "running",
      version: "0.3.0-rc.1-demo",
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
    case "/metrics/internet":
    case "/metrics/internet/refresh": {
      const checkedAt = new Date().toISOString();
      const connectionLatency = 18 + Math.sin(Date.now() / 5000) * 4;
      const httpsLatency = 32 + Math.sin(Date.now() / 9000) * 7;
      return {
        collected_at: checkedAt, status: "online",
        connection_latency_ms: connectionLatency, connection_successful_targets: 2, connection_total_targets: 2,
        connection_targets: [
          { name: "Cloudflare", reachable: true, latency_ms: connectionLatency - 1 },
          { name: "Google", reachable: true, latency_ms: connectionLatency + 1 },
        ],
        consecutive_failures: 0, reliability_successful: 10, reliability_total: 10,
        diagnostics_collected_at: checkedAt, dns_ok: true, dns_ms: 8.4,
        latency_ms: httpsLatency, successful_targets: 2, total_targets: 2,
        targets: [
          { name: "Cloudflare", reachable: true, latency_ms: httpsLatency - 2 },
          { name: "Google", reachable: true, latency_ms: httpsLatency + 2 },
        ],
      };
    }
    case "/players": return { players: [
      { username: "iKlaude", uuid: "03c69a88-5438-4b03-952a-17efcbcfe6f7", online: true, op: true, whitelisted: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 7342 },
      { username: "Alex", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 3922 },
      { username: "Long_Name_With_Underscores", online: true, last_seen_at: new Date().toISOString(), observed_playtime_seconds: 18422 },
      { username: "OfflineMiner", online: false, banned: true, last_seen_at: new Date(Date.now() - 86400000).toISOString(), observed_playtime_seconds: 7521 },
    ] };
    case "/files": {
      if (query.get("root") === "servers") {
        const path = query.get("path") || "";
        if (!path) return [{ name: "minecraft-java", is_dir: true, size: 0, mod_time: new Date().toISOString() }];
        if (path === "minecraft-java") return [
          { name: "modded", is_dir: true, size: 0, mod_time: new Date().toISOString() },
          { name: "vanilla", is_dir: true, size: 0, mod_time: new Date().toISOString() },
        ];
        if (path === "minecraft-java/modded") return DEMO_SERVERS.map((server) => ({
          name: server.slug, is_dir: true, size: 0, mod_time: server.updated_at,
        }));
        return [];
      }
      return [
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
    }
    case "/files/content": return { content: "motd=A precise Bonghos local demo\nserver-port=25565\nmax-players=20\n" };
    case "/configuration": return {
      eula: true,
      instance: DEMO_SERVERS.find((server) => server.id === Number(query.get("server_id"))) || DEMO_SERVERS[0],
      jvm: { xms: "2G", xmx: "6G", source_file: "user_jvm_args.txt", source_kind: "jvm_args_file", editable: true },
      scripts: [{ path: "run.sh", modloader: "forge", score: 98 }],
      java: [{ path: "/usr/lib/jvm/java-21-openjdk/bin/java", version: "21" }],
      properties: { motd: "A precise Bonghos local demo", "server-port": "25565", "max-players": "20", difficulty: "normal", gamemode: "survival", "white-list": "false", pvp: "true", "view-distance": "10", "simulation-distance": "10", "online-mode": "true" },
    };
    case "/backups": return [
      { backup_id: "demo-full-20260803-1900", backup_type: "full_server", consistency_mode: "online", trigger_type: "manual", compressed_size: 4620000000, verification_status: "verified", created_at: new Date(Date.now() - 5 * 3600000).toISOString(), protected: true },
      { backup_id: "demo-world-20260802-0400", backup_type: "world_and_player_data", consistency_mode: "offline", trigger_type: "schedule", compressed_size: 2110000000, verification_status: "verified", created_at: new Date(Date.now() - 29 * 3600000).toISOString(), protected: false },
    ];
    case "/backups/storage": return { path: "/home/demo/bonghos/backups", external: false, included_in_bonghos_size: true };
    case "/schedules": return [
      { id: 1, name: "Nightly verified backup", enabled: true, action: "backup", schedule_type: "daily", schedule_expression: "04:00", timezone: "Asia/Phnom_Penh", next_run_at: new Date(Date.now() + 3 * 3600000).toISOString(), last_result: "success" },
      { id: 2, name: "Weekly restart", enabled: false, action: "restart_server", schedule_type: "weekly", schedule_expression: "sun 05:00", timezone: "Asia/Phnom_Penh", next_run_at: null, last_result: "skipped" },
    ];
    case "/operations": return [];
    case "/activity": return [
      { at: new Date(Date.now() - 4 * 60000).toISOString(), username: "demo-user", action: "login_success", target: "", detail: "", remote_addr: "192.168.1.24" },
      { at: new Date(Date.now() - 12 * 60000).toISOString(), username: "demo-user", action: "backup_created", target: "bio1", detail: "full_server verified" },
      { at: new Date(Date.now() - 46 * 60000).toISOString(), username: "demo-user", action: "configuration_saved", target: "user_jvm_args.txt", detail: "-Xmx changed to 6G" },
    ];
    case "/users": return [
      { ID: DEMO_ME.id, Username: DEMO_ME.username, Role: DEMO_ME.role, Disabled: false },
      { ID: 2, Username: "admin", Role: "admin", Disabled: false },
      { ID: 3, Username: "viewer", Role: "viewer", Disabled: true },
    ];
    case "/roles/permissions": return demoRolePermissionsPayload();
    case "/passkeys": return DEMO_PASSKEYS.map((passkey) => ({ ...passkey }));
    case "/account/recovery-codes": return DEMO_RECOVERY_CODES.map((item) => ({ ...item }));
    case "/auth/turnstile": return { enabled: false };
    case "/security/turnstile": return { enabled: false, site_key: "", secret_configured: false };
    default: return {};
  }
}

function demoRolePermissionsPayload() {
  return {
    catalog: DEMO_PERMISSION_CATALOG.map((definition) => ({
      ...definition,
      requires: [...definition.requires],
      assignable_roles: [...definition.assignable_roles],
    })),
    roles: Object.fromEntries(["owner", "admin", "member", "viewer"].map((role) => [role, {
      permissions: [...DEMO_ROLE_PERMISSIONS[role]],
      defaults: [...DEMO_ROLE_DEFAULTS[role]],
      editable: role !== "owner",
      revision: DEMO_ROLE_REVISIONS[role],
      customized: DEMO_ROLE_CUSTOMIZED[role],
    }])),
  };
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
  if (!res.ok) {
    const error = new Error((data && data.error) || res.statusText);
    error.status = res.status;
    throw error;
  }
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
  fileEditorWrap: false,
  consoleSearch: "",
  consoleFilterMode: "all",
  consoleHistoryRequest: 0,
  commandHistory: [],
  commandHistoryAt: -1,
  perf: [],
  perfStorage: null,
  perfInternet: null,
  perfInternetHistory: [],
  perfSpeedTest: null,
  turnstile: { enabled: false },
  overviewCPUTrend: "machine",
  overviewMemoryTrend: "java",
  overviewMaxPlayers: 20,
  performanceTarget: "",
  serverTargetId: null,
  managedServerId: null,
  serverManagementReturn: false,
  consoleReturn: false,
  pendingFileOpen: null,
  perfIntervalSeconds: 2,
  uptimeBase: null,
};
const can = (p) => S.me && S.me.permissions && S.me.permissions.includes(p);
const canAny = (...permissions) => permissions.some(can);

function syncCurrentUserUI() {
  if (!S.me) return;
  $("#whoami").textContent = `${esc(S.me.username)} · ${esc(S.me.role)}`;
  const control = $("#demo-role-control");
  const select = $("#demo-role-select");
  control?.classList.toggle("hidden", !DEMO_MODE);
  if (select && DEMO_MODE) select.value = S.me.role;
}

function changeDemoUserRole(role) {
  if (!DEMO_MODE || !DEMO_ROLES.includes(role) || role === S.me?.role) return;
  closeActiveModal();
  DEMO_ME.role = role;
  DEMO_ME.permissions = [...DEMO_ROLE_PERMISSIONS[role]];
  S.me = demoUserSnapshot();

  const url = new URL(location.href);
  if (role === "owner") url.searchParams.delete("demo-role");
  else url.searchParams.set("demo-role", role);
  history.replaceState(null, "", url);

  syncCurrentUserUI();
  buildNav();
  if (!can("server.players.view")) S.onlinePlayerCount = null;
  navigate(pageAllowed(S.page) ? S.page : defaultPage(), { replaceHash: true });
  refreshPlayerCount();
  toast(`Demo user is now ${capitalizeFirst(role)}.`, "ok");
}

$("#demo-role-select")?.addEventListener("change", (event) =>
  changeDemoUserRole(event.currentTarget.value));

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
const BASE_TOPICS = ["overview", "overview_performance", "servers", "backups"];
const OVERVIEW_INTERVAL_SECONDS = 4;
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
let demoOverviewTimer = null;
let demoPerformanceTimer = null;
let performanceStorageRequest = 0;
let performanceStorageShowPercentage = false;
let performanceInternetRequest = 0;
let performanceInternetTimer = null;
let performanceInternetAbort = null;
let performanceSpeedAbort = null;
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

function baseTopicSubscription(topic) {
  return topic === "overview_performance"
    ? { action: "subscribe", topic, interval_seconds: OVERVIEW_INTERVAL_SECONDS }
    : { action: "subscribe", topic };
}

function baseTopicAllowed(topic) {
  if (topic === "overview_performance") return can("server.performance.view");
  if (topic === "backups") return can("server.backups.view");
  return can("server.view");
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
  if (page !== "performance") {
    stopPerformanceInternetPolling();
    cancelPerformanceSpeedTest();
  }
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
  ws.onopen = async () => {
    wsRetry = 1000;
    try {
      const me = await api("/auth/me");
      const changed = me.role !== S.me.role || JSON.stringify(me.permissions || []) !== JSON.stringify(S.me.permissions || []);
      S.me = me;
      syncCurrentUserUI();
      if (changed) {
        closeActiveModal();
        buildNav();
        if (!pageAllowed(S.page)) {
          navigate(defaultPage(), { replaceHash: true });
        } else {
          renderPage();
        }
      }
    } catch {
      showLogin();
      return;
    }
    // Always-on topics: status and long-running operations must keep arriving
    // whatever page is open, so an import or backup started elsewhere is seen.
    BASE_TOPICS.filter(baseTopicAllowed).forEach((topic) => wsSend(baseTopicSubscription(topic)));
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
  } else if (type === "sample" && topic === "overview_performance") {
    updateSidebarLiveStats(data);
    if (S.page !== "overview") return;
    appendPerformanceSample(data);
    setUptimeBaseline(data);
    updateUptimeDisplay();
    updateLiveStats(data);
  } else if (type === "sample" && S.page === "performance" && topic === "performance") {
    appendPerformanceSample(data);
    setUptimeBaseline(data);
    updateUptimeDisplay();
    updateLiveStats(data);
  } else if (topic === "overview" && type === "players") {
    refreshOverviewPlayers();
  } else if (topic === "players") {
    refreshPlayerCount();
    if (S.page === "players") renderPage();
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

let loginTurnstileToken = "";
let loginTurnstileWidget = null;
let loginTurnstileSiteKey = "";
let turnstileScriptPromise = null;

function turnstileEnabled() {
  return !DEMO_MODE && !!S.turnstile?.enabled && !!S.turnstile?.site_key;
}

function setLoginTurnstileStatus(message, tone = "") {
  const status = $("#login-turnstile-status");
  if (!status) return;
  status.textContent = message;
  status.className = "hint login-turnstile-status" + (tone ? ` is-${tone}` : "");
}

function syncLoginTurnstileActions() {
  const waiting = turnstileEnabled() && !loginTurnstileToken;
  const verify = $("#login-btn");
  if (verify) verify.disabled = waiting;
  updatePasskeyAvailability();
}

function loadTurnstileScript() {
  if (window.turnstile) return Promise.resolve(window.turnstile);
  if (turnstileScriptPromise) return turnstileScriptPromise;
  turnstileScriptPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    script.defer = true;
    script.onload = () => window.turnstile ? resolve(window.turnstile) : reject(new Error("Turnstile did not initialize"));
    script.onerror = () => reject(new Error("Could not load Turnstile"));
    document.head.append(script);
  });
  return turnstileScriptPromise;
}

function clearLoginTurnstileWidget() {
  if (loginTurnstileWidget !== null && window.turnstile) {
    try { window.turnstile.remove(loginTurnstileWidget); } catch {}
  }
  loginTurnstileWidget = null;
  loginTurnstileSiteKey = "";
  loginTurnstileToken = "";
  const container = $("#login-turnstile-widget");
  if (container) container.innerHTML = "";
}

async function prepareLoginTurnstile() {
  const wrap = $("#login-turnstile");
  if (!wrap) return;
  const enabled = turnstileEnabled();
  wrap.classList.toggle("hidden", !enabled);
  if (!enabled) {
    clearLoginTurnstileWidget();
    syncLoginTurnstileActions();
    return;
  }
  loginTurnstileToken = "";
  setLoginTurnstileStatus("Checking browser…");
  syncLoginTurnstileActions();
  try {
    const turnstile = await loadTurnstileScript();
    const siteKey = S.turnstile.site_key;
    if (loginTurnstileWidget !== null && loginTurnstileSiteKey !== siteKey) clearLoginTurnstileWidget();
    if (loginTurnstileWidget === null) {
      loginTurnstileSiteKey = siteKey;
      loginTurnstileWidget = turnstile.render("#login-turnstile-widget", {
        sitekey: siteKey,
        action: "login",
        theme: "auto",
        appearance: "interaction-only",
        callback: (token) => {
          loginTurnstileToken = token;
          setLoginTurnstileStatus("Browser verified", "ready");
          syncLoginTurnstileActions();
        },
        "expired-callback": () => {
          loginTurnstileToken = "";
          setLoginTurnstileStatus("Checking browser…");
          syncLoginTurnstileActions();
        },
        "error-callback": () => {
          loginTurnstileToken = "";
          setLoginTurnstileStatus("Security check could not load. Refresh and try again.", "error");
          syncLoginTurnstileActions();
        },
      });
    } else {
      turnstile.reset(loginTurnstileWidget);
    }
  } catch {
    setLoginTurnstileStatus("Security check could not load. Refresh and try again.", "error");
    syncLoginTurnstileActions();
  }
}

function consumeLoginTurnstileToken() {
  if (!turnstileEnabled()) return "";
  if (!loginTurnstileToken) throw new Error("Complete the security check.");
  const token = loginTurnstileToken;
  loginTurnstileToken = "";
  syncLoginTurnstileActions();
  return token;
}

function resetLoginTurnstile() {
  if (!turnstileEnabled()) return;
  loginTurnstileToken = "";
  setLoginTurnstileStatus("Checking browser…");
  if (loginTurnstileWidget !== null && window.turnstile) {
    try { window.turnstile.reset(loginTurnstileWidget); } catch {}
  }
  syncLoginTurnstileActions();
}

sidebarToggle.addEventListener("click", () =>
  setSidebarOpen(!$("#app-view").classList.contains("sidebar-open")));
$("#sidebar-scrim").addEventListener("click", closeMobileSidebar);
mobileNavQuery.addEventListener("change", (event) => { if (!event.matches) setSidebarOpen(false); });

function showLogin() {
  setSidebarOpen(false);
  stopPerformanceInternetPolling();
  cancelPerformanceSpeedTest();
  S.me = null;
  if (S.ws) try { S.ws.close(); } catch {}
  $("#app-view").classList.add("hidden");
  $("#login-view").classList.remove("hidden");
  loginStep(1);
  updatePasskeyAvailability();
  prepareLoginTurnstile();
}

function updatePasskeyAvailability() {
  const button = $("#login-passkey");
  if (!button) return;
  const available = DEMO_MODE || passkeysSupported();
  button.disabled = !available || (turnstileEnabled() && !loginTurnstileToken);
  button.title = available ? "Use a passkey, another device, or a security key" : "Passkeys require a supported browser over HTTPS or localhost";
}

// loginStep switches between the credential step and the authenticator step.
// Step two is always reached, whatever was typed in step one: the interface
// must not reveal whether an account exists any more than the API does.
let loginCodeMode = "totp";

function setLoginCodeMode(mode, focus = true) {
  loginCodeMode = mode === "recovery" ? "recovery" : "totp";
  const recovery = loginCodeMode === "recovery";
  $("#login-totp-mode")?.classList.toggle("hidden", recovery);
  $("#login-recovery-mode")?.classList.toggle("hidden", !recovery);
  const totpInput = $("#login-code"), recoveryInput = $("#login-recovery");
  if (totpInput) totpInput.disabled = recovery;
  if (recoveryInput) recoveryInput.disabled = !recovery;
  const switcher = $("#login-code-mode");
  if (switcher) switcher.textContent = recovery ? "Use an authenticator code" : "Use a recovery code";
  $("#login-step-2 .otp-wrap")?.classList.remove("error");
  $("#login-recovery")?.classList.remove("error");
  if (focus) setTimeout(() => (recovery ? $("#login-recovery") : $("#login-code"))?.focus(), 30);
}

function normalizeRecoveryCode(value) {
  const compact = String(value || "").toLowerCase().replace(/[^0-9a-f]/g, "").slice(0, 10);
  return compact.length > 5 ? `${compact.slice(0, 5)}-${compact.slice(5)}` : compact;
}

function currentLoginCode() {
  const input = loginCodeMode === "recovery" ? $("#login-recovery") : $("#login-code");
  return (input?.value || "").trim();
}

function loginStep(n) {
  const s1 = $("#login-step-1"), s2 = $("#login-step-2");
  if (!s1 || !s2) return;
  s1.classList.toggle("hidden", n !== 1);
  s2.classList.toggle("hidden", n !== 2);
  $("#login-error").classList.add("hidden");
  $(".otp-wrap")?.classList.remove("error");
  if (n === 2) {
    $("#login-step-2-who").textContent = "Signing in as " + $("#login-user").value.trim();
    $("#login-code").value = DEMO_MODE && DEMO_VIEW === "login" ? "123456" : "";
    $("#login-recovery").value = "";
    setLoginCodeMode("totp", false);
    syncOTPCells();
    setTimeout(() => $("#login-code").focus(), 30);
  } else {
    setTimeout(() => $("#login-user").focus(), 30);
  }
}

function syncOTPCells(input = $("#login-code"), wrap = $(".otp-wrap")) {
  if (!input || !wrap) return;
  const code = input.value.replace(/\D/g, "").slice(0, 6);
  [...wrap.children].forEach((cell, i) => { cell.textContent = code[i] || ""; });
}

function installOTPControl(input = $("#login-code"), wrap = $(".otp-wrap")) {
  if (!input || !wrap) return;
  wrap.addEventListener("click", () => input.focus());
  input.addEventListener("focus", () => wrap.classList.add("focus"));
  input.addEventListener("blur", () => wrap.classList.remove("focus"));
  input.addEventListener("input", () => {
    input.value = input.value.replace(/\D/g, "").slice(0, 6);
    syncOTPCells(input, wrap);
  });
  input.addEventListener("paste", () => setTimeout(() => syncOTPCells(input, wrap), 0));
  syncOTPCells(input, wrap);
}

async function boot() {
  if (DEMO_MODE) {
    csrfToken = "demo-csrf-token";
    if (DEMO_VIEW === "login") {
      showLogin();
      $("#login-form > .muted").textContent = "Demo sign-in · use any non-empty credentials and authenticator code.";
      $("#login-user").value = "demo-user";
      $("#login-pass").value = "demo-password";
      return;
    }
    S.me = demoUserSnapshot();
    S.status = { state: "running" };
    S.consoleLines = [...DEMO_CONSOLE];
    S.perf = [...DEMO_METRICS];
    enterApp();
    setTimeout(() => toast("Demo mode uses local mock data. No server changes are made.", "ok"), 200);
    return;
  }
  const [c, turnstile] = await Promise.all([
    api("/auth/csrf"),
    api("/auth/turnstile").catch(() => ({ enabled: false })),
  ]);
  csrfToken = c.csrf;
  S.turnstile = turnstile || { enabled: false };
  try {
    S.me = await api("/auth/me");
    enterApp();
  } catch { showLogin(); }
}

$("#login-back").addEventListener("click", () => loginStep(1));

$("#login-code-mode").addEventListener("click", () => {
  setLoginCodeMode(loginCodeMode === "totp" ? "recovery" : "totp");
});

$("#login-recovery").addEventListener("input", (event) => {
  event.currentTarget.value = normalizeRecoveryCode(event.currentTarget.value);
  event.currentTarget.classList.remove("error");
});

$("#login-passkey").addEventListener("click", async () => {
  const button = $("#login-passkey");
  const errorBox = $("#login-error");
  errorBox.classList.add("hidden");
  button.disabled = true;
  try {
    if (DEMO_MODE) {
      S.me = demoUserSnapshot();
      enterApp();
      toast("Demo passkey sign-in completed locally.", "ok");
      return;
    }
    if (!passkeysSupported()) throw new Error("Passkeys require a supported browser over HTTPS or localhost.");
    const turnstileToken = consumeLoginTurnstileToken();
    const started = await api("/auth/passkey/begin", {
      method: "POST", json: { turnstile_token: turnstileToken },
    });
    const credential = await navigator.credentials.get({
      publicKey: passkeyRequestOptions(started.options.publicKey),
    });
    S.me = await api(`/auth/passkey/finish?flow=${encodeURIComponent(started.flow)}`, {
      method: "POST",
      json: passkeyCredentialJSON(credential),
    });
    const nextCSRF = await api("/auth/csrf");
    csrfToken = nextCSRF.csrf;
    enterApp();
  } catch (error) {
    errorBox.textContent = passkeyError(error, "Passkey sign-in failed.");
    errorBox.classList.remove("hidden");
  } finally {
    resetLoginTurnstile();
    updatePasskeyAvailability();
  }
});

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
    const turnstileToken = consumeLoginTurnstileToken();
    S.me = await api("/auth/login", { method: "POST", json: {
      username: $("#login-user").value.trim(),
      password: $("#login-pass").value,
      code: currentLoginCode(),
      turnstile_token: turnstileToken,
    }});
    const c = await api("/auth/csrf"); csrfToken = c.csrf;
    $("#login-pass").value = ""; $("#login-code").value = ""; $("#login-recovery").value = "";
    enterApp();
  } catch (err) {
    const eb = $("#login-error"); eb.textContent = err.message; eb.classList.remove("hidden");
    if (loginCodeMode === "recovery") {
      $("#login-recovery").value = "";
      $("#login-recovery").classList.add("error");
      $("#login-recovery").focus();
    } else {
      $("#login-step-2 .otp-wrap")?.classList.add("error");
      $("#login-code").value = "";
      syncOTPCells();
      $("#login-code").focus();
    }
  } finally {
    resetLoginTurnstile();
    syncLoginTurnstileActions();
  }
});

$("#logout-btn").addEventListener("click", async () => {
  try { await api("/auth/logout", { method: "POST", json: {} }); } catch {}
  showLogin();
});

function enterApp() {
  $("#login-view").classList.add("hidden");
  $("#app-view").classList.remove("hidden");
  syncCurrentUserUI();
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
  { section: "Operate", id: "performance", label: "Performance", icon: "chart-2-linear", perm: "server.performance.view" },
  { section: "Operate", id: "players", label: "Players", icon: "users-group-rounded-linear", perm: "server.players.view" },
  { section: "Manage", id: "servers", label: "Servers", icon: "server-square-linear", perm: "server.view" },
  { section: "Manage", id: "files", label: "Files", icon: "folder-with-files-linear", perm: "server.files.manage" },
  { section: "Manage", id: "configuration", label: "Configuration", icon: "tuning-2-linear", perm: "server.configuration.manage" },
  { section: "Manage", id: "backups", label: "Backups", icon: "archive-down-minimlistic-linear", perm: "server.backups.view" },
  { section: "Manage", id: "schedules", label: "Schedules", icon: "calendar-linear", perm: "server.schedules.manage" },
  { section: "System", id: "activity", label: "Activity", icon: "history-linear", perm: "activity.view" },
  { section: "System", id: "users", label: "Users", icon: "users-group-two-rounded-linear", anyPerm: ["users.manage", "roles.manage"] },
  { section: "Account", id: "account", label: "Account", icon: "users-group-rounded-linear", fallbackOnly: true, accountPage: true },
  { section: "Account", id: "security", label: "Security", icon: "shield-keyhole-linear", accountPage: true },
  { section: "Account", id: "settings", label: "Settings", icon: "settings-linear", accountPage: true },
];

function buildNav() {
  const nav = $("#nav"); nav.innerHTML = "";
  let lastSection = "";
  for (const page of PAGES) {
    if (!pageAvailable(page)) continue;
    if (page.section !== lastSection) {
      nav.append(el("div", { class: "nav-section" }, page.section));
      lastSection = page.section;
    }
    const label = el("span", { class: "nav-item-label" }, page.label,
      page.id === "players"
        ? el("span", { class: "nav-player-count", id: "nav-player-count" }, `· ${S.onlinePlayerCount ?? "—"}`)
        : null);
    const openPage = () => {
      if (page.id === "files") {
        filePath = "";
        fileBrowseRoot = "project";
        fileEscapeAction = null;
        S.pendingFileOpen = null;
      }
      navigate(page.id);
    };
    nav.append(el("div", { class: "nav-item", "data-page": page.id, tabindex: "0", onclick: openPage, onkeydown: (e) => {
      if (e.key === "Enter" || e.key === " ") { e.preventDefault(); openPage(); }
    } }, el("span", { class: "nav-icon", "aria-hidden": "true" }, solarIcon(page.icon)), label));
  }
}

function setOnlinePlayerCount(players) {
  S.onlinePlayerCount = (players || []).filter((player) => player.online).length;
  const count = $("#nav-player-count");
  if (count) count.textContent = `· ${S.onlinePlayerCount}`;
}

function updateSidebarLiveStats(sample) {
  const onlinePlayers = Number(sample?.online_players);
  if (!Number.isFinite(onlinePlayers) || onlinePlayers < 0) return;
  S.onlinePlayerCount = Math.floor(onlinePlayers);
  const count = $("#nav-player-count");
  if (count) count.textContent = `· ${S.onlinePlayerCount}`;
}

async function refreshPlayerCount() {
  if (!can("server.players.view")) return;
  try { setOnlinePlayerCount((await api("/players")).players || []); } catch {}
}

async function refreshOverviewPlayers() {
  if (S.page !== "overview" || !can("server.players.view")) return;
  try {
    const players = (await api("/players")).players || [];
    setOnlinePlayerCount(players);
    const onlinePlayers = players.filter((player) => player.online);
    const card = $("#overview-player-summary");
    if (card) card.replaceWith(playerSummaryCard(onlinePlayers, onlinePlayers.length, S.overviewMaxPlayers));
  } catch {}
}

function pageAllowed(page) {
  const entry = PAGES.find((p) => p.id === page);
  return !!entry && pageAvailable(entry);
}

function defaultPage() {
  if (!hasPrimaryPage()) return "account";
  return (PAGES.find((page) => !page.fallbackOnly && !page.accountPage && basePageAvailable(page)) || PAGES.find(pageAvailable) || PAGES[0]).id;
}

function pageAvailable(page) {
  if (page.fallbackOnly) return !hasPrimaryPage();
  return basePageAvailable(page);
}

function basePageAvailable(page) {
  return (!page.perm || can(page.perm)) && (!page.anyPerm || canAny(...page.anyPerm));
}

function hasPrimaryPage() {
  return PAGES.some((page) => !page.fallbackOnly && !page.accountPage && basePageAvailable(page));
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
  S.overviewReturn = !!opts.fromOverview && (next === "players" || next === "servers" || next === "configuration" || next === "performance");
  S.serverManagementReturn = !!opts.fromServers && (next === "files" || next === "configuration");
  S.consoleReturn = !!opts.fromConsole && (next === "servers" || next === "files" || next === "configuration");
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

function updateServerNameTicker(ticker) {
  if (!ticker?.isConnected) return;
  const text = ticker.querySelector(".server-name-text:not(.server-name-clone)");
  if (!text) return;
  const link = ticker.querySelector(".server-name-link");
  const textWidth = Math.max(text.scrollWidth, text.getBoundingClientRect().width);
  const overflowing = ticker.clientWidth > 0 && textWidth > ticker.clientWidth + 1;
  ticker.classList.toggle("is-overflowing", overflowing);
  if (overflowing) {
    const seconds = Math.max(10, (textWidth + 32) / 28);
    ticker.style.setProperty("--ticker-duration", `${seconds.toFixed(2)}s`);
    if (!link) ticker.tabIndex = 0;
  } else {
    ticker.style.removeProperty("--ticker-duration");
  }
  if (link || !overflowing) ticker.removeAttribute("tabindex");
}

let serverNameTickerResizeFrame = 0;
let serverNameTickerObserver = null;
function scheduleServerNameTickerUpdate(ticker = $(".server-name")) {
  cancelAnimationFrame(serverNameTickerResizeFrame);
  serverNameTickerResizeFrame = requestAnimationFrame(() => updateServerNameTicker(ticker));
}
window.addEventListener("resize", () => {
  scheduleServerNameTickerUpdate();
});

function renderServerPicker() {
  const host = $("#server-picker"); host.innerHTML = "";
  const header = $("#sidebar-project-header");
  header?.querySelector(":scope > .server-picker-icon")?.remove();
  const active = S.servers.find((s) => s.id === S.activeId);
  const name = active ? active.display_name : "None selected";
  const icon = active
    ? el("span", { class: "server-status-icon server-picker-icon", "aria-hidden": "true" }, serverCardIcon(active))
    : null;
  const track = el("div", { class: "server-name-track" },
    el("span", { class: "server-name-text" }, name),
    el("span", { class: "server-name-text server-name-clone", "aria-hidden": "true" }, name));
  const ticker = el("div", { class: "server-name", title: name, "aria-label": name },
    active ? el("a", {
      class: "server-name-link",
      href: "#servers",
      "aria-label": `Open ${name} in Servers`,
      onclick: (event) => {
        event.preventDefault();
        navigate("servers", { serverTargetId: active.id });
      },
    }, track) : track);
  if (icon && header) header.prepend(icon);
  host.append(
    el("div", { class: "server-picker-head" },
      el("span", { class: "server-kicker" }, "Active project"),
      renderStatusPillNode({ id: "status-pill" })),
    ticker);
  serverNameTickerObserver?.disconnect();
  serverNameTickerObserver = typeof ResizeObserver === "function"
    ? new ResizeObserver(() => scheduleServerNameTickerUpdate(ticker))
    : null;
  serverNameTickerObserver?.observe(ticker);
  scheduleServerNameTickerUpdate(ticker);
  document.fonts?.ready.then(() => scheduleServerNameTickerUpdate(ticker));
}

function renderStatusPillNode(opts = {}) {
  const st = S.status.state || "stopped";
  const attrs = { class: "status-label " + st + (opts.compact ? " compact" : "") };
  if (opts.id) attrs.id = opts.id;
  return el("div", attrs,
    el("span", { class: "status-square", "aria-hidden": "true" }), capitalizeFirst(st));
}

function capitalizeFirst(value) {
  const text = String(value || "");
  return text.charAt(0).toUpperCase() + text.slice(1);
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
  document.body.classList.toggle("security-page-active", S.page === "security");
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
      case "account": return pageAccount(main);
      case "security": return await pageSecurity(main);
      case "settings": return await pageSettings(main);
    }
  } catch (err) {
    main.innerHTML = "";
    main.append(el("h1", {}, "Something went wrong"), el("p", { class: "muted" }, err.message));
  }
}

function overviewBackButton() {
  if (S.consoleReturn) return el("button", {
    class: "btn ghost page-back-button", type: "button",
    "aria-label": "Back to Console", title: "Back to Console",
    onclick: () => navigate("console"),
  }, solarIcon("alt-arrow-left-linear"));
  if (!S.overviewReturn) return null;
  return el("button", {
    class: "btn ghost page-back-button", type: "button",
    "aria-label": "Back to Overview", title: "Back to Overview",
    onclick: () => navigate("overview"),
  }, solarIcon("alt-arrow-left-linear"));
}

function managedPageBackButton() {
  if (!S.serverManagementReturn) return overviewBackButton();
  return el("button", {
    class: "btn ghost page-back-button", type: "button",
    "aria-label": "Back to Servers", title: "Back to Servers",
    onclick: () => navigate("servers", { serverTargetId: S.managedServerId }),
  }, solarIcon("alt-arrow-left-linear"));
}

function pageHeader(title, subtitle, actions = [], leading = null) {
  const heading = el("h1", {}, title);
  const hasSearchFilter = actions.some((action) => action?.classList?.contains("page-search-filter-controls"));
  const neighboringActionCount = actions.filter((action) => action?.matches?.("button, a")).length;
  const actionsClass = [
    "actions",
    hasSearchFilter ? "has-search-filter" : "",
    hasSearchFilter && neighboringActionCount > 1 ? "has-many-actions" : "",
  ].filter(Boolean).join(" ");
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
    actions.length ? el("div", { class: actionsClass }, actions) : null);
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
  const showPerformance = can("server.performance.view");
  const showPlayers = can("server.players.view");
  const showBackups = can("server.backups.view");
  const showSchedules = can("server.schedules.manage");
  const s = d.sample || {};
  if (showPerformance) setUptimeBaseline(s);
  else S.uptimeBase = null;
  const inst = d.instance;

  // Health, host and trends live here together. Knowing whether the server is
  // healthy should not require visiting three tabs.
  let host = null, events = [], history = [], players = null;
  if (showPerformance && can("host.view")) try { host = await api("/host"); } catch {}
  try { events = (await api("/events?limit=25")).events || []; } catch {}
  if (showPerformance) try { history = await api("/metrics?hours=1") || []; } catch {}
  if (showPlayers) try { players = (await api("/players")).players || []; } catch {}
  if (players) setOnlinePlayerCount(players);
  S.perf = [];
  if (showPerformance) {
    history.forEach(appendPerformanceSample);
    appendPerformanceSample(s);
  }

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
  S.overviewMaxPlayers = maxPlayers;
  const summaryCards = [serverStatusCard(d.state, inst)];
  if (showPerformance) {
    summaryCards.push(
      statCard("Uptime", currentUptimeSeconds() === null ? "—" : fmtDur(currentUptimeSeconds()), s.java_pid ? "Java PID " + s.java_pid : "not running", "uptime-value"),
    );
  }
  if (showPlayers) summaryCards.push(playerSummaryCard(onlinePlayers, onlineCount, maxPlayers));
  if (showPerformance) {
    summaryCards.push(statCard("Disk free", diskTotal > 0 ? fmtBytes(diskFree) : "—",
      diskTotal > 0 ? "of " + fmtBytes(diskTotal) : "Visit Performance to measure",
      "overview-live-disk-free", "performance-machine-storage-card"));
  }
  const projectDetails = [
    el("dt", {}, "MOTD"), el("dd", {}, d.motd || "—"),
    ...(d.playit_address ? [
      el("dt", {}, "Playit IP"), el("dd", {}, el("button", {
        class: "copy-value mono", type: "button", title: "Copy Playit IP",
        "aria-label": `Copy Playit IP ${d.playit_address}`,
        onclick: () => copyText(d.playit_address, "Playit IP copied"),
      }, el("span", {}, d.playit_address), solarIcon("copy-linear"))),
    ] : []),
    el("dt", {}, "LAN IP"), el("dd", {},
      d.lan_ip ? el("button", {
        class: "copy-value mono", type: "button", title: "Copy LAN IP",
        "aria-label": `Copy LAN IP ${d.lan_ip}`,
        onclick: () => copyText(d.lan_ip, "LAN IP copied"),
      }, el("span", {}, d.lan_ip), solarIcon("copy-linear")) : "—"),
    el("dt", {}, "Port"), el("dd", {}, d.port || "25565"),
    el("dt", {}, "Modloader"), el("dd", {}, inst?.modloader || "unknown"),
    el("dt", {}, "Startup script"), el("dd", { class: "mono" }, inst?.startup_script || "not selected"),
    el("dt", {}, "Restart policy"), el("dd", {}, inst?.restart_policy || "never"),
    el("dt", {}, "Autostart"), el("dd", {}, inst?.autostart_enabled ? "enabled" : "disabled"),
  ];
  if (showBackups) projectDetails.push(
    el("dt", {}, "Last backup"), el("dd", {}, d.last_backup ? fmtTime(d.last_backup.created_at) : "none yet"));
  if (showSchedules) projectDetails.push(
    el("dt", {}, "Next schedule"), el("dd", {}, d.next_schedule_at ? fmtTime(d.next_schedule_at) : "none"));

  const overviewSections = [
    pageHeader(inst ? inst.display_name : "Overview", "Server state, resource pressure, backups, and recent events for the active project.", [
      lifecycleButtons(true),
    ]),

    // What is happening right now.
    el("div", { class: "grid cols-4 overview-stat-grid" }, ...summaryCards),
  ];
  if (showPerformance) overviewSections.push(
    // Host health, previously a separate tab.
    el("div", { class: "grid cols-4 flow-section overview-stat-grid" },
      statCard("CPU", Number.isFinite(hostCPU) ? hostCPU.toFixed(1) + "%" : "—", "whole-machine average",
        "overview-live-cpu", "performance-host-cpu-card"),
      statCard("Load average", Number.isFinite(loadAverage) ? loadAverage.toFixed(2) : "—", "1 minute",
        "overview-live-load", "performance-load-card"),
      statCard("Process memory", fmtBytes(s.rss_bytes), "resident set (not Java heap)",
        "overview-live-rss", "allocated-memory-card"),
      statCard("Host memory", hostMemTotal > 0 ? fmtBytes(memUsed) : "—",
        hostMemTotal > 0 ? "of " + fmtBytes(hostMemTotal) : "",
        "overview-live-host-memory", "host-memory-card")),

    // Trends, previously the Performance tab.
    el("div", { class: "grid cols-2 flow-section" },
      overviewCPUTrendCard(),
      overviewMemoryTrendCard()));
  overviewSections.push(
    el("div", { class: "grid cols-2 flow-section" },
      // The timeline: what the server did, in its own words.
      el("div", { class: "card" },
        el("h3", {}, "Recent activity"),
        events.length
          ? el("ul", { class: "timeline" }, ...events.map(eventRow))
          : el("p", { class: "muted" }, "Nothing recorded yet.")),

      el("div", { class: "card" },
        el("h3", {}, "Project"),
        inst ? el("dl", { class: "kv" }, ...projectDetails)
            : el("p", { class: "muted" }, "No active project selected."))));
  main.replaceChildren(...overviewSections);
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
  toast(successMessage, "ok", "copy-linear");
}

// eventRow renders one timeline entry, coloured by severity.
function eventRow(e) {
  return el("li", { class: "timeline-item sev-" + (e.severity || "info") },
    el("span", { class: "timeline-time mono" }, fmtTime(e.occurred_at)),
    el("span", { class: "timeline-msg" }, e.message || e.event));
}

// trendCard draws a compact interactive sparkline plus the latest value.
function trendCard(title, samples, pick, fmt, id = "", chartOptions = {}) {
  const { headerControl = null, ...sparklineOptions } = chartOptions;
  const points = (samples || []).map((sample) => ({
    timestamp: sampleTimestamp(sample),
    value: Number(pick(sample)),
  })).filter((point) => point.timestamp && Number.isFinite(point.value));
  const latest = points.length ? points[points.length - 1].value : 0;
  const attrs = { class: "card graph-card" };
  if (id) attrs.id = id;
  const chart = points.length > 1
    ? overviewSparklineNode(title, points, fmt, {
      ...sparklineOptions,
      summary: { value: fmt(latest), note: "last hour" },
    })
    : null;
  return el("div", attrs,
    el("div", { class: "overview-trend-card-head" },
      el("div", { class: "metric-label" }, title),
      headerControl),
    chart || el("div", {},
      el("div", { class: "metric-value" }, fmt(latest)),
      el("div", { class: "metric-note" }, "last hour"),
      el("p", { class: "muted" }, "Collecting samples…")));
}

function overviewMemoryCeiling(samples) {
  return Math.max(1, ...(samples || []).flatMap((sample) =>
    [Number(sample.rss_bytes), Number(sample.jvm_xmx_bytes)].filter(Number.isFinite)));
}

function overviewMachineMemoryCeiling(samples) {
  return Math.max(1, ...(samples || []).map((sample) => Number(sample.host_mem_total)).filter(Number.isFinite));
}

function overviewTrendSelect(label, value, options, onChange, cardId) {
  const select = el("select", { class: "overview-trend-select", "aria-label": label },
    ...options.map(([optionValue, optionLabel]) =>
      el("option", { value: optionValue, selected: optionValue === value ? "" : null }, optionLabel)));
  select.addEventListener("change", () => {
    onChange(select.value);
    updateOverviewTrendCharts(cardId);
    requestAnimationFrame(() => document.querySelector(`#${cardId} .overview-trend-select`)?.focus());
  });
  return select;
}

function overviewCPUTrendCard() {
  const javaProcess = S.overviewCPUTrend === "java";
  return trendCard("CPU", S.perf,
    javaProcess ? (sample) => sample.cpu_percent : (sample) => sample.host_cpu_percent,
    (value) => value.toFixed(0) + "%", "overview-trend-cpu", {
      min: 0,
      ...(javaProcess ? {} : { max: 100 }),
      axisFormat: (value) => value.toFixed(0) + "%",
      headerControl: overviewTrendSelect("CPU graph source", S.overviewCPUTrend, [
        ["machine", "Machine Usage"],
        ["java", "Java Process"],
      ], (value) => { S.overviewCPUTrend = value; }, "overview-trend-cpu"),
    });
}

function overviewMemoryTrendCard() {
  const machine = S.overviewMemoryTrend === "machine";
  const machineUsed = (sample) => {
    const total = Number(sample.host_mem_total);
    const available = Number(sample.host_mem_avail);
    return total > 0 && Number.isFinite(available) ? total - available : NaN;
  };
  return trendCard("Memory", S.perf,
    machine ? machineUsed : (sample) => sample.rss_bytes,
    fmtBytes, "overview-trend-memory", {
      min: 0,
      max: machine ? overviewMachineMemoryCeiling(S.perf) : overviewMemoryCeiling(S.perf),
      axisFormat: fmtBytes,
      headerControl: overviewTrendSelect("Memory graph source", S.overviewMemoryTrend, [
        ["machine", "Machine"],
        ["java", "Java memory"],
      ], (value) => { S.overviewMemoryTrend = value; }, "overview-trend-memory"),
    });
}

function updateOverviewTrendCharts(forceCardId = "") {
  if (S.page !== "overview") return;
  const cpu = $("#overview-trend-cpu");
  const memory = $("#overview-trend-memory");
  const focusedCardId = document.activeElement?.closest?.(".graph-card")?.id || "";
  if (forceCardId === "overview-trend-cpu" || focusedCardId !== "overview-trend-cpu")
    cpu?.replaceWith(overviewCPUTrendCard());
  if (forceCardId === "overview-trend-memory" || focusedCardId !== "overview-trend-memory")
    memory?.replaceWith(overviewMemoryTrendCard());
}

function statCard(title, value, sub, valueId = "", performanceTarget = "") {
  if (!can("server.performance.view")) performanceTarget = "";
  const valueAttrs = { class: "metric-value" };
  if (valueId) valueAttrs.id = valueId;
  const attrs = { class: "card metric" + (performanceTarget ? " overview-performance-card" : "") };
  if (performanceTarget) {
    attrs.href = "#performance";
    attrs["aria-label"] = `${title}: ${value}. Open in Performance.`;
    attrs.onclick = (event) => {
      event.preventDefault();
      navigate("performance", { performanceTarget, fromOverview: true });
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
    id: "overview-player-summary", class: "card metric player-summary-card", href: "#players",
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
  const icons = server ? [
    el("span", { class: "server-status-icon server-status-icon-ambient", "aria-hidden": "true" }, serverCardIcon(server)),
    el("span", { class: "server-status-icon server-status-icon-main", "aria-hidden": "true" }, serverCardIcon(server)),
  ] : [];
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
    ...icons,
    el("div", { class: "metric-label server-status-label-row" },
      "Server status", solarIcon("alt-arrow-right-linear", "player-summary-arrow")),
    el("div", { class: "metric-value server-status-value" },
      el("span", { class: "server-status-state" },
        el("span", { class: "status-square", "aria-hidden": "true" }), label)),
    el("div", { class: "metric-note" }, ""));
}

function lifecycleButtons(includeServers = false, managementSource = "overview") {
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
  const forceStop = () => confirmModal(
    "Force stop",
    "Force stop kills the Java process immediately. Unsaved world data may be lost. Continue?",
    "Force stop",
    async () => {
      try { await api("/server/force-stop", { method: "POST", json: { confirm: true } }); toast("Force stop sent", "ok"); }
      catch (e) { toast(e.message, "err"); }
    },
  );
  const appendOverviewManagementActions = () => {
    if (!includeServers) return;
    if (can("server.configuration.manage")) {
      wrap.append(el("button", {
        class: "btn ghost mobile-icon-only overview-configuration-button",
        type: "button",
        title: "Configuration",
        "aria-label": "Configuration",
        onclick: () => navigate("configuration", managementSource === "console" ? { fromConsole: true } : { fromOverview: true }),
      }, solarIcon("tuning-2-linear")));
    }
    const menuItems = [
      el("button", {
        class: "action-menu-item", type: "button", role: "menuitem",
        onclick: () => navigate("servers", managementSource === "console" ? { fromConsole: true } : { fromOverview: true }),
      }, solarIcon("server-square-linear"), "Servers"),
    ];
    if (can("server.force_stop") && st !== "stopped") {
      menuItems.push(el("button", {
        class: "action-menu-item danger", type: "button", role: "menuitem", onclick: forceStop,
      }, solarIcon("danger-triangle-linear"), "Force stop"));
    }
    wrap.append(overflowActionsMenu("More server actions", menuItems, "overview-lifecycle-menu"));
  };
  if (pending) {
    if (pending.action === "start" && can("server.start"))
      wrap.append(lifecycleButton("start", "/server/start", "Start", "running", "btn primary"));
    if (pending.action === "stop" && can("server.stop"))
      wrap.append(lifecycleButton("stop", "/server/stop", "Stop", "stopped", "btn"));
    if (pending.action === "restart" && can("server.restart"))
      wrap.append(lifecycleButton("restart", "/server/restart", "Restart", "running", "btn"));
    appendOverviewManagementActions();
    return wrap;
  }
  if (can("server.start") && !running)
    wrap.append(lifecycleButton("start", "/server/start", "Start", "running", "btn primary"));
  if (can("server.stop") && running)
    wrap.append(lifecycleButton("stop", "/server/stop", "Stop", "stopped", "btn"));
  if (can("server.restart") && running)
    wrap.append(lifecycleButton("restart", "/server/restart", "Restart", "running", "btn"));
  if (!includeServers && can("server.force_stop") && st !== "stopped")
    wrap.append(el("button", { class: "btn danger", onclick: forceStop }, "Force stop"));
  appendOverviewManagementActions();
  return wrap;
}

// ----- console --------------------------------------------------------------
async function pageConsole(main) {
  main.innerHTML = "";
  const stopped = (S.status.state || "stopped") === "stopped";
  const box = el("div", { class: "console" + (S.consolePaused ? " paused" : "") + (S.consoleWrap ? " is-wrapped" : ""), id: "console-box", role: "log", "aria-live": S.consolePaused ? "off" : "polite" });
  const search = el("input", { class: "page-search", type: "search", value: S.consoleSearch, placeholder: "Search buffer", "aria-label": "Search console buffer" });
  const input = el("input", { placeholder: can("server.console.use") ? (stopped ? "Start the server to send commands" : "Command, for example: say hello") : "Read-only console", spellcheck: "false", autocomplete: "off" });
  if (!can("server.console.use") || stopped) input.disabled = true;
  search.addEventListener("input", () => { S.consoleSearch = search.value; renderConsoleLines(box); });
  const filter = pageFilterMenu("Filter console", [
    ["all", "All lines"],
    ["errors", "Errors only"],
    ["warnings", "Warnings only"],
    ["info", "Info only"],
    ["players", "Player events only"],
    ["chat", "Chat only"],
  ], (value) => {
    S.consoleFilterMode = value;
    renderConsoleLines(box);
  }, S.consoleFilterMode);
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
    pageHeader("Console", "Live Minecraft output and command entry for the active server.", [renderStatusPillNode(), lifecycleButtons(true, "console")]),
    el("div", { class: "console-shell" },
      el("div", { class: "console-toolbar" },
        el("div", { class: "page-search-filter-controls console-search-controls" }, search, filter),
        el("button", {
          class: "btn ghost console-icon-control",
          "aria-label": S.consolePaused ? "Resume console" : "Pause console",
          title: S.consolePaused ? "Resume console" : "Pause console",
          onclick: () => { S.consolePaused = !S.consolePaused; pageConsole(main); },
        }, S.consolePaused ? "Resume" : "Pause"),
        mobileWrapButton,
        el("button", { class: "btn ghost console-icon-control", "aria-label": "Copy console", title: "Copy console", onclick: async () => {
          try { await navigator.clipboard.writeText(S.consoleLines.join("\n")); toast("Console buffer copied", "ok", "copy-linear"); }
          catch { toast("Copy failed in this browser", "err", "copy-linear"); }
        } }, "Copy"),
        el("button", { class: "btn ghost console-clear-control", onclick: () => { S.consoleHistoryRequest++; box.innerHTML = ""; S.consoleLines = []; } }, "Clear"),
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
    if (!consoleLineMatchesFilter(line, q)) continue;
    box.append(consoleLineNode(line));
  }
  if ((q || S.consoleFilterMode !== "all") && !box.childNodes.length) renderConsolePlaceholder(box, "No matching console lines.");
}

function consoleLineMatchesFilter(line, query = (S.consoleSearch || "").toLowerCase()) {
  const text = String(line || "");
  if (query && !text.toLowerCase().includes(query)) return false;
  if (S.consoleFilterMode === "errors") return /ERROR|SEVERE|FATAL/i.test(text);
  if (S.consoleFilterMode === "warnings") return /WARN(?:ING)?/i.test(text);
  if (S.consoleFilterMode === "info") return /(?:\[|\/)INFO(?:\]|:)/i.test(text);
  if (S.consoleFilterMode === "players") return /joined the game|left the game|logged in with entity id|lost connection:/i.test(text);
  if (S.consoleFilterMode === "chat") return /(?:\]:|\[CHAT\])\s*(?:\[Not Secure\]\s*)?<[^>]+>\s+/i.test(text);
  return true;
}

function renderConsolePlaceholder(box, message) {
  box.innerHTML = "";
  box.append(el("div", { class: "console-line console-placeholder" }, message));
}

function appendConsoleLine(line) {
  const box = $("#console-box");
  if (!box) return;
  if (!consoleLineMatchesFilter(line)) return;
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
  const search = el("input", { class: "page-search", type: "search", placeholder: "Search players", "aria-label": "Search players" });
  const tbody = el("tbody");
  let filterMode = "name-asc";
  const byName = (a, b) => String(a.username || "").localeCompare(String(b.username || ""), undefined, { sensitivity: "base" });
  const draw = () => {
    const q = search.value.trim().toLowerCase();
    let visible = players.filter((p) => !q || String(p.username).toLowerCase().includes(q));
    if (filterMode === "online-only") visible = visible.filter((p) => p.online);
    if (filterMode === "offline-only") visible = visible.filter((p) => !p.online);
    if (filterMode === "op-only") visible = visible.filter((p) => p.op);
    if (filterMode === "banned-only") visible = visible.filter((p) => p.banned);
    if (filterMode === "whitelisted-only") visible = visible.filter((p) => p.whitelisted);
    visible.sort((a, b) => {
      if (filterMode === "name-desc") return -byName(a, b);
      if (filterMode === "online-first") return Number(b.online) - Number(a.online) || byName(a, b);
      if (filterMode === "offline-first") return Number(a.online) - Number(b.online) || byName(a, b);
      if (filterMode === "last-seen-newest") return (Date.parse(b.last_seen_at) || 0) - (Date.parse(a.last_seen_at) || 0) || byName(a, b);
      if (filterMode === "last-seen-oldest") return (Date.parse(a.last_seen_at) || 0) - (Date.parse(b.last_seen_at) || 0) || byName(a, b);
      if (filterMode === "playtime-most") return Number(b.observed_playtime_seconds || 0) - Number(a.observed_playtime_seconds || 0) || byName(a, b);
      if (filterMode === "playtime-least") return Number(a.observed_playtime_seconds || 0) - Number(b.observed_playtime_seconds || 0) || byName(a, b);
      return byName(a, b);
    });
    tbody.innerHTML = "";
    tbody.append(...(visible.length ? visible.map((player) => playerRow(player, draw)) : [el("tr", {}, el("td", { colspan: "5", class: "muted" }, players.length ? "No matching players." : "No players seen yet."))]));
  };
  const filterModes = [
    ["name-asc", "Name: A-Z"],
    ["name-desc", "Name: Z-A"],
    ["online-first", "Status: Online first"],
    ["offline-first", "Status: Offline first"],
    ["last-seen-newest", "Last seen: Newest"],
    ["last-seen-oldest", "Last seen: Oldest"],
    ["playtime-most", "Playtime: Most"],
    ["playtime-least", "Playtime: Least"],
    ["online-only", "Online only"],
    ["offline-only", "Offline only"],
    ["op-only", "OP only"],
    ["whitelisted-only", "Whitelisted only"],
    ["banned-only", "BANNED only"],
  ];
  const filterControl = pageFilterMenu("Filter players", filterModes, (value) => {
    filterMode = value;
    draw();
  }, filterMode);
  search.addEventListener("input", draw);
  main.innerHTML = "";
  main.append(
    pageHeader("Players", "Observed online and recent players with their server access state.", [
      el("div", { class: "page-search-filter-controls players-search-controls" }, search, filterControl),
    ], overviewBackButton()),
    el("div", { class: "toolbar" },
      el("span", { class: "status-label running" }, el("span", { class: "status-square" }), players.filter((p) => p.online).length + " Online"),
      el("span", { class: "status-label" }, el("span", { class: "status-square" }), players.length + " Observed")),
    el("div", { class: "table-wrap players-table" },
      el("table", {},
        el("thead", {}, el("tr", {},
          el("th", {}, "Player"), el("th", {}, "Status"), el("th", { class: "mobile-hide" }, "Last seen"),
          el("th", { class: "mobile-hide player-observed-playtime" }, "Playtime"), el("th", {}, ""))),
        tbody)));
  draw();
}

function playerRow(p, onUpdated) {
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
  const profileURL = playerNameMCProfileURL(p.uuid, p.username);
  const playerSkin = profileURL
    ? el("a", {
      class: "player-avatar-link",
      href: profileURL,
      target: "_blank",
      rel: "noopener noreferrer",
      title: `View ${p.username} on NameMC`,
      "aria-label": `View ${p.username} on NameMC`,
    }, avatar)
    : avatar;
  const username = String(p.username || "Unknown");
  const usernameTicker = el("span", {
    class: "player-name-ticker",
    "aria-label": username,
    onpointerenter: () => updatePlayerNameTicker(usernameTicker),
    onfocus: () => updatePlayerNameTicker(usernameTicker),
  },
    el("span", { class: "player-name-ticker-track" },
      el("strong", { class: "player-name-ticker-text" }, username),
      el("strong", { class: "player-name-ticker-text player-name-ticker-clone", "aria-hidden": "true" }, username)));
  const row = el("tr", {
    onclick: (event) => {
      if (!window.matchMedia("(max-width: 820px)").matches || event.target.closest("a, button, input, select, textarea, [role='menuitem']")) return;
      updatePlayerNameTicker(usernameTicker);
      if (!usernameTicker.classList.contains("is-overflowing")) return;
      const activate = !usernameTicker.classList.contains("is-ticker-active");
      $$(".player-name-ticker.is-ticker-active").forEach((ticker) => ticker.classList.remove("is-ticker-active"));
      usernameTicker.classList.toggle("is-ticker-active", activate);
    },
  },
    el("td", {}, el("div", { class: "player-identity" }, playerSkin,
      el("span", { class: "player-name-block" },
        el("span", { class: "player-name-line" },
          usernameTicker,
          p.op ? el("span", { class: "player-tag" }, "OP") : null,
          p.whitelisted ? el("span", { class: "player-tag", title: "Whitelisted", "aria-label": "Whitelisted" }, "WL") : null,
          p.banned ? el("span", { class: "player-tag" }, "BANNED") : null),
        el("span", { class: "mobile-only mobile-row-detail player-status" + (p.online ? "" : " is-offline") }, p.online ? "Online" : "Offline")))),
    el("td", {}, el("span", { class: "player-status" + (p.online ? "" : " is-offline") }, p.online ? "Online" : "Offline")),
    el("td", { class: "mobile-hide" }, fmtTime(p.last_seen_at)),
    el("td", { class: "mobile-hide player-observed-playtime" }, fmtDur(p.observed_playtime_seconds)),
    el("td", { class: "table-actions" }, can("server.players.manage") ? playerActions(p, onUpdated) : ""));
  requestAnimationFrame(() => updatePlayerNameTicker(usernameTicker));
  document.fonts?.ready.then(() => updatePlayerNameTicker(usernameTicker));
  return row;
}

function updatePlayerNameTicker(ticker) {
  if (!ticker?.isConnected) return;
  const text = ticker.querySelector(".player-name-ticker-text:not(.player-name-ticker-clone)");
  if (!text) return;
  const range = document.createRange();
  range.selectNodeContents(text);
  const textWidth = Math.max(text.scrollWidth, range.getBoundingClientRect().width);
  range.detach?.();
  const overflowing = ticker.clientWidth > 0 && textWidth > ticker.clientWidth + 1;
  ticker.classList.toggle("is-overflowing", overflowing);
  if (overflowing) {
    ticker.style.setProperty("--player-ticker-duration", `${Math.max(6, (textWidth + 24) / 24).toFixed(2)}s`);
    ticker.tabIndex = 0;
  } else {
    ticker.classList.remove("is-ticker-active");
    ticker.style.removeProperty("--player-ticker-duration");
    ticker.removeAttribute("tabindex");
  }
}

let playerNameTickerResizeFrame = 0;
window.addEventListener("resize", () => {
  cancelAnimationFrame(playerNameTickerResizeFrame);
  playerNameTickerResizeFrame = requestAnimationFrame(() => {
    playerNameTickerResizeFrame = requestAnimationFrame(() => {
      $$(".player-name-ticker").forEach(updatePlayerNameTicker);
    });
  });
});

function playerNameMCProfileURL(uuid, username = "") {
  const value = String(uuid || "").trim();
  if (/^(?:[0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/i.test(value)) {
    return `https://namemc.com/profile/${encodeURIComponent(value)}`;
  }
  const name = String(username || "").trim();
  if (!/^[A-Za-z0-9_]{1,16}$/.test(name)) return "";
  return `https://namemc.com/profile/${encodeURIComponent(name)}`;
}

function pageAccount(main) {
  main.innerHTML = "";
  main.append(
    pageHeader("Account", "Personal security and appearance remain available without a primary workspace page."),
    el("div", { class: "card account-no-access-card" },
      solarIcon("shield-keyhole-linear"),
      el("div", {},
        el("h3", {}, "No primary workspace assigned"),
        el("p", { class: "muted" }, `The ${capitalizeFirst(S.me.role)} role currently has no server, activity, or user-management page. An Owner can change the role configuration.`)),
      el("div", { class: "actions" },
        el("button", { class: "btn ghost", onclick: () => navigate("security") }, "Account security"),
        el("button", { class: "btn ghost", onclick: () => navigate("settings") }, "Appearance"))),
  );
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

function playerActions(p, onUpdated) {
  const act = (label, apiAction, needsReason, onSuccess = null, buttonStyle = "danger") => () => {
    const reason = el("input", { placeholder: "Reason (optional)" });
    modal(`${label} ${p.username}`,
      needsReason ? [el("div", { class: "field-row" }, reason)] : [el("p", {}, `Confirm ${label.toLowerCase()} for ${p.username}?`)],
      [["Cancel", "ghost", (c) => c()],
       [label, buttonStyle, async (c) => {
         c();
         try {
           await api("/players/action", { method: "POST", json: { action: apiAction, username: p.username, reason: reason.value || "" } });
           if (onSuccess) onSuccess();
           toast(`${label} sent for ${p.username}`, "ok");
         } catch (e) { toast(e.message, "err"); }
       }]]);
  };
  const actions = [
    p.online ? { slot: "kick", label: "Kick", icon: "close-circle-linear", danger: true, run: act("Kick", "kick", true) } : null,
    {
      slot: "ban",
      label: p.banned ? "Unban" : "Ban",
      icon: p.banned ? "check-circle-linear" : "lock-keyhole-linear",
      danger: !p.banned,
      run: act(p.banned ? "Unban" : "Ban", p.banned ? "pardon" : "ban", !p.banned, () => {
        p.banned = !p.banned;
        if (onUpdated) onUpdated();
      }, p.banned ? "primary" : "danger"),
    },
    {
      slot: "whitelist",
      label: p.whitelisted ? "Unwhitelist" : "Whitelist",
      icon: p.whitelisted ? "shield-keyhole-linear" : "shield-check-linear",
      run: act(p.whitelisted ? "Unwhitelist" : "Whitelist", p.whitelisted ? "whitelist_remove" : "whitelist_add", false, () => {
        p.whitelisted = !p.whitelisted;
        if (onUpdated) onUpdated();
      }, "primary"),
    },
    {
      slot: "op",
      label: p.op ? "Deop" : "Op",
      icon: "key-linear",
      danger: p.op,
      run: act(p.op ? "Deop" : "Op", p.op ? "deop" : "op", false, () => {
        p.op = !p.op;
        if (onUpdated) onUpdated();
      }, p.op ? "danger" : "primary"),
    },
  ];
  const desktop = el("div", { class: "desktop-row-actions player-action-grid" },
    ...actions.map((action) => action
      ? el("button", { class: `btn ghost player-action-${action.slot}`, onclick: action.run },
        action.slot === "whitelist" ? solarIcon(action.icon) : null, action.label)
      : el("span", { class: "player-action-placeholder", "aria-hidden": "true" })));
  const mobile = overflowActionsMenu(`Actions for ${p.username}`,
    actions.filter(Boolean).map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
}

// ----- files ----------------------------------------------------------------
let filePath = "";
let fileEscapeAction = null;
let fileBrowseRoot = "project";

function currentFileProject() {
  return S.servers.find((server) => server.id === S.managedServerId)
    || S.servers.find((server) => server.id === S.activeId)
    || null;
}

function projectServersPath(project) {
  if (!project || project.external_directory) return "";
  const fallback = project.slug ? `servers/minecraft-java/modded/${project.slug}` : "";
  const directory = String(project.server_directory || fallback).replace(/\\/g, "/").replace(/^\/+|\/+$/g, "");
  return directory.startsWith("servers/") ? directory.slice("servers/".length) : "";
}

function projectFolderName(project) {
  const directory = String(project?.server_directory || "").replace(/\\/g, "/").replace(/\/+$/g, "");
  return directory.split("/").filter(Boolean).pop() || project?.slug || "project";
}

function projectAtServersPath(path) {
  const clean = String(path || "").replace(/^\/+|\/+$/g, "");
  return S.servers
    .map((project) => ({ project, directory: projectServersPath(project) }))
    .filter(({ directory }) => directory && (clean === directory || clean.startsWith(directory + "/")))
    .sort((a, b) => b.directory.length - a.directory.length)[0] || null;
}

function appendFileQuery(path, key, value) {
  return `${path}${path.includes("?") ? "&" : "?"}${encodeURIComponent(key)}=${encodeURIComponent(value)}`;
}

function fileRequestScope(path = filePath) {
  if (fileBrowseRoot === "servers") {
    const match = projectAtServersPath(path);
    if (match) {
      const relative = path.slice(match.directory.length).replace(/^\/+/, "");
      return { path: relative, project: match.project, serversRoot: false, writable: true };
    }
    return { path, project: null, serversRoot: true, writable: false };
  }
  return { path, project: currentFileProject(), serversRoot: false, writable: true };
}

function fileScopeEndpoint(endpoint, scope) {
  if (scope.serversRoot) return appendFileQuery(endpoint, "root", "servers");
  if (scope.project?.id) return appendFileQuery(endpoint, "server_id", scope.project.id);
  return serverScopedPath(endpoint);
}

function filePathEndpoint(endpoint, path) {
  const scope = fileRequestScope(path);
  return { scope, url: appendFileQuery(fileScopeEndpoint(endpoint, scope), "path", scope.path) };
}

function parentFileLocation(path) {
  const parts = String(path || "").split("/").filter(Boolean);
  if (parts.length) return { root: fileBrowseRoot, path: parts.slice(0, -1).join("/") };
  if (fileBrowseRoot !== "project") return null;
  const projectPath = projectServersPath(currentFileProject());
  if (!projectPath) return null;
  const projectParts = projectPath.split("/").filter(Boolean);
  return { root: "servers", path: projectParts.slice(0, -1).join("/") };
}

function fileContextSubtitle(scope) {
  if (scope.project) {
    const state = scope.project.id === S.activeId ? "Active" : "Non-Active";
    return el("span", { class: "project-context" },
      el("strong", { class: "project-context-state" }, state),
      " project “",
      el("span", { class: "project-context-name" }, scope.project.display_name),
      "”.");
  }
  return "Browsing outside of project.";
}

function activeProjectFilesButton(main, scope) {
  const activeProject = S.servers.find((server) => server.id === S.activeId);
  if (!activeProject || scope.project?.id === activeProject.id) return null;
  return el("button", {
    class: "btn ghost small file-active-project-button",
    type: "button",
    title: `Back to active project: ${activeProject.display_name}`,
    onclick: () => {
      S.managedServerId = activeProject.id;
      filePath = "";
      fileBrowseRoot = "project";
      fileEscapeAction = null;
      S.pendingFileOpen = null;
      pageFiles(main, "", "project");
    },
  }, solarIcon("arrow-left-linear"), "Active");
}

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

const FILE_IMAGE_PREVIEW_EXTENSIONS = new Set(["avif", "bmp", "gif", "ico", "jpeg", "jpg", "png", "webp"]);

function fileExtension(path) {
  const name = String(path || "").toLowerCase().split("/").pop() || "";
  return name.includes(".") ? name.split(".").pop() : "";
}

function isPreviewableImage(path) {
  return FILE_IMAGE_PREVIEW_EXTENSIONS.has(fileExtension(path));
}

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

function fileIdentity(entry, modified = "") {
  return el("span", { class: "file-identity" },
    el("span", { class: "file-type-icon", "aria-hidden": "true" }, solarIcon(fileIconName(entry))),
    el("span", { class: "file-identity-copy" },
      el("span", { class: "file-name" }, entry.name),
      modified ? el("small", { class: "file-modified-mobile mobile-only" }, modified) : null));
}

async function pageFiles(main, path = filePath, root = fileBrowseRoot) {
  // A deep link from elsewhere (for example the Configuration page naming the
  // file that owns the JVM settings) opens that file straight away.
  if (S.pendingFileOpen) {
    const pending = S.pendingFileOpen;
    S.pendingFileOpen = null;
    return openFile(main, pending.path, pending.returnTo);
  }
  if (root === "servers") {
    const projectMatch = projectAtServersPath(path);
    if (projectMatch) {
      S.managedServerId = projectMatch.project.id;
      path = path.slice(projectMatch.directory.length).replace(/^\/+/, "");
      root = "project";
    }
  }
  filePath = path;
  fileBrowseRoot = root;
  const parentLocation = parentFileLocation(path);
  fileEscapeAction = parentLocation
    ? () => pageFiles(main, parentLocation.path, parentLocation.root)
    : null;
  const directoryRequest = filePathEndpoint("/files", path);
  const entries = await api(directoryRequest.url);
  main.innerHTML = "";
  let selectionMode = false;
  const selectedPaths = new Map();
  let lastVisibleEntries = [];
  let selectButton;
  let selectAllInput;
  let nameHeader;
  let fileList;
  let selectionAnchorPath = "";
  let selectionDrag = null;
  let suppressSelectionClick = false;
  let selectionAutoScrollFrame = 0;
  const renderedFileRows = new Map();
  const crumbs = el("div", { class: "breadcrumb" },
    el("span", { onclick: () => pageFiles(main, "", fileBrowseRoot) },
      fileBrowseRoot === "servers" ? "servers" : projectFolderName(currentFileProject())));
  let acc = "";
  for (const part of path.split("/").filter(Boolean)) {
    acc += (acc ? "/" : "") + part;
    const p = acc;
    crumbs.append(" / ", el("span", { onclick: () => pageFiles(main, p, fileBrowseRoot) }, part));
  }
  const locationBar = el("div", { class: "file-location-bar" },
    crumbs,
    activeProjectFilesButton(main, directoryRequest.scope));
  const upInput = el("input", { type: "file", multiple: true, style: "display:none" });
  upInput.addEventListener("change", async () => {
    const fd = new FormData();
    for (const f of upInput.files) fd.append("file", f);
    try {
      const uploadScope = fileRequestScope(path);
      const uploadURL = appendFileQuery(fileScopeEndpoint("/files/upload", uploadScope), "path", uploadScope.path);
      await fetch("/api" + uploadURL,
        { method: "POST", body: fd, headers: { "X-Bonghos-CSRF": csrfToken }, credentials: "same-origin" });
      toast("Uploaded", "ok"); pageFiles(main, path, fileBrowseRoot);
    } catch (e) { toast(e.message, "err"); }
  });
  const entryPath = (entry) => (path ? path + "/" : "") + entry.name;
  const syncRenderedSelection = (rel) => {
    const row = renderedFileRows.get(rel);
    if (!row) return;
    const selected = selectedPaths.has(rel);
    row.classList.toggle("is-selected", selected);
    const checkbox = row.querySelector(".file-selection-checkbox");
    if (checkbox) checkbox.checked = selected;
  };
  const setEntrySelected = (entry, selected) => {
    const rel = entryPath(entry);
    if (selected) selectedPaths.set(rel, entry);
    else selectedPaths.delete(rel);
    syncRenderedSelection(rel);
  };
  const visibleIndex = (rel) => lastVisibleEntries.findIndex((entry) => entryPath(entry) === rel);
  const selectVisibleRange = (fromIndex, toIndex, selected = true) => {
    const first = Math.max(0, Math.min(fromIndex, toIndex));
    const last = Math.min(lastVisibleEntries.length - 1, Math.max(fromIndex, toIndex));
    for (let index = first; index <= last; index += 1) {
      setEntrySelected(lastVisibleEntries[index], selected);
    }
    updateSelectionUI();
  };
  const selectFromAnchor = (entry) => {
    const rel = entryPath(entry);
    const currentIndex = visibleIndex(rel);
    const anchorIndex = visibleIndex(selectionAnchorPath);
    if (anchorIndex < 0 || currentIndex < 0) {
      setEntrySelected(entry, true);
      selectionAnchorPath = rel;
      updateSelectionUI();
      return;
    }
    selectVisibleRange(anchorIndex, currentIndex, true);
  };
  const applySelectionDragAtPoint = (clientX, clientY) => {
    if (!selectionDrag) return;
    const row = document.elementFromPoint(clientX, clientY)?.closest?.("tr[data-file-index]");
    if (!row || !fileList?.contains(row)) return;
    const nextIndex = Number(row.dataset.fileIndex);
    if (!Number.isInteger(nextIndex) || nextIndex === selectionDrag.lastIndex) return;
    const previousFirst = Math.min(selectionDrag.originIndex, selectionDrag.lastIndex);
    const previousLast = Math.max(selectionDrag.originIndex, selectionDrag.lastIndex);
    const nextFirst = Math.min(selectionDrag.originIndex, nextIndex);
    const nextLast = Math.max(selectionDrag.originIndex, nextIndex);
    const affectedFirst = Math.min(previousFirst, nextFirst);
    const affectedLast = Math.max(previousLast, nextLast);
    for (let index = affectedFirst; index <= affectedLast; index += 1) {
      const selected = index >= nextFirst && index <= nextLast
        ? selectionDrag.selected
        : selectionDrag.baseline[index];
      setEntrySelected(lastVisibleEntries[index], selected);
    }
    updateSelectionUI();
    selectionDrag.lastIndex = nextIndex;
  };
  const runSelectionAutoScroll = () => {
    if (!selectionDrag) return;
    const edge = 72;
    const y = selectionDrag.clientY;
    let delta = 0;
    if (y < edge) delta = -Math.ceil((edge - y) / 5);
    else if (y > window.innerHeight - edge) delta = Math.ceil((y - (window.innerHeight - edge)) / 5);
    if (delta) {
      window.scrollBy(0, delta);
      applySelectionDragAtPoint(selectionDrag.clientX, selectionDrag.clientY);
    }
    selectionAutoScrollFrame = requestAnimationFrame(runSelectionAutoScroll);
  };
  const handleSelectionPointerMove = (event) => {
    if (!selectionDrag) return;
    if ((event.buttons & 1) === 0) {
      stopSelectionDrag();
      return;
    }
    selectionDrag.clientX = event.clientX;
    selectionDrag.clientY = event.clientY;
    applySelectionDragAtPoint(event.clientX, event.clientY);
  };
  const stopSelectionDrag = () => {
    if (!selectionDrag) return;
    selectionDrag = null;
    document.body.classList.remove("file-selection-dragging");
    document.removeEventListener("pointermove", handleSelectionPointerMove);
    document.removeEventListener("pointerup", stopSelectionDrag);
    document.removeEventListener("pointercancel", stopSelectionDrag);
    window.removeEventListener("blur", stopSelectionDrag);
    if (selectionAutoScrollFrame) cancelAnimationFrame(selectionAutoScrollFrame);
    selectionAutoScrollFrame = 0;
    setTimeout(() => { suppressSelectionClick = false; }, 0);
  };
  const releaseSuppressedSelectionClick = () => {
    document.removeEventListener("pointerup", releaseSuppressedSelectionClick);
    document.removeEventListener("pointercancel", releaseSuppressedSelectionClick);
    window.removeEventListener("blur", releaseSuppressedSelectionClick);
    setTimeout(() => { suppressSelectionClick = false; }, 0);
  };
  const beginSelectionDrag = (event, entry, index) => {
    if (!selectionMode || event.pointerType !== "mouse" || event.button !== 0
      || event.target.closest?.("a, button")) return;
    event.preventDefault();
    suppressSelectionClick = true;
    if (event.shiftKey) {
      selectFromAnchor(entry);
      document.addEventListener("pointerup", releaseSuppressedSelectionClick);
      document.addEventListener("pointercancel", releaseSuppressedSelectionClick);
      window.addEventListener("blur", releaseSuppressedSelectionClick);
      return;
    }
    const rel = entryPath(entry);
    const selected = !selectedPaths.has(rel);
    const baseline = lastVisibleEntries.map((visibleEntry) => selectedPaths.has(entryPath(visibleEntry)));
    selectionAnchorPath = rel;
    setEntrySelected(entry, selected);
    updateSelectionUI();
    selectionDrag = {
      selected, baseline, originIndex: index, lastIndex: index,
      clientX: event.clientX, clientY: event.clientY,
    };
    document.body.classList.add("file-selection-dragging");
    document.addEventListener("pointermove", handleSelectionPointerMove);
    document.addEventListener("pointerup", stopSelectionDrag);
    document.addEventListener("pointercancel", stopSelectionDrag);
    window.addEventListener("blur", stopSelectionDrag);
    selectionAutoScrollFrame = requestAnimationFrame(runSelectionAutoScroll);
  };
  const fileRow = (entry, index) => {
    const rel = entryPath(entry);
    const checkbox = el("input", {
      class: "file-selection-checkbox", type: "checkbox",
      "aria-label": `Select ${entry.name}`,
      checked: selectedPaths.has(rel) ? "checked" : null,
      onclick: (event) => {
        event.stopPropagation();
        if (suppressSelectionClick) {
          event.preventDefault();
          setTimeout(() => syncRenderedSelection(rel), 0);
          return;
        }
        if (event.shiftKey && selectionAnchorPath) {
          event.preventDefault();
          selectFromAnchor(entry);
        } else {
          selectionAnchorPath = rel;
        }
      },
      onchange: (event) => {
        if (suppressSelectionClick) return;
        setEntrySelected(entry, event.currentTarget.checked);
        updateSelectionUI();
      },
    });
    const row = el("tr", {
      class: selectedPaths.has(rel) ? "is-selected" : "",
      "data-file-index": String(index),
      onpointerdown: (event) => beginSelectionDrag(event, entry, index),
    },
      el("td", { class: "mono", style: "cursor:pointer", onclick: (event) => {
        if (selectionMode) {
          if (suppressSelectionClick) return;
          if (event.shiftKey && selectionAnchorPath) selectFromAnchor(entry);
          else {
            selectionAnchorPath = rel;
            setEntrySelected(entry, !selectedPaths.has(rel));
          }
          updateSelectionUI();
        } else if (entry.is_dir) pageFiles(main, rel, fileBrowseRoot);
        else openFile(main, rel);
      } }, el("span", { class: "file-selectable-identity" },
        selectionMode ? checkbox : null,
        fileIdentity(entry, fmtTime(entry.mod_time || entry.modified)))),
      el("td", { class: "file-size-column" }, entry.is_dir ? "—" : fmtBytes(entry.size)),
      el("td", { class: "mobile-hide" }, fmtTime(entry.mod_time || entry.modified)),
      el("td", { class: "table-actions file-actions-cell" },
        fileActions(main, path, entry, directoryRequest.scope.writable)));
    renderedFileRows.set(rel, row);
    return row;
  };
  const parentRow = () => el("tr", { class: "file-parent-row" },
    el("td", {
      class: "mono", colspan: "4",
      style: selectionMode ? "cursor:default" : "cursor:pointer",
      title: selectionMode ? "Parent directory cannot be selected" : "Parent directory",
      onclick: () => {
        if (!selectionMode) pageFiles(main, parentLocation.path, parentLocation.root);
      },
    }, el("span", { class: "file-selectable-identity" },
      selectionMode ? el("input", {
        class: "file-selection-checkbox", type: "checkbox", disabled: "disabled",
        "aria-label": "Parent directory cannot be selected",
      }) : null,
      fileIdentity({ name: "..", is_dir: true }))));
  const tbody = el("tbody");
  const selectedCount = el("strong", { class: "file-selection-count" }, "0 selected");
  const bulkCopyButton = el("button", { class: "btn ghost small", disabled: "disabled" }, solarIcon("copy-linear"), "Copy");
  const bulkMoveButton = el("button", { class: "btn ghost small", disabled: "disabled" }, solarIcon("folder-open-linear"), "Move");
  const bulkDeleteButton = el("button", { class: "btn danger small", disabled: "disabled" }, solarIcon("trash-bin-trash-linear"), "Delete");
  const selectionBar = el("div", { class: "file-selection-bar", hidden: "" },
    selectedCount,
    el("div", { class: "spacer" }),
    bulkCopyButton, bulkMoveButton, bulkDeleteButton,
    el("button", {
      class: "file-selection-close", type: "button", title: "Exit selection mode",
      "aria-label": "Exit selection mode", onclick: () => setSelectionMode(false),
    }, solarIcon("close-linear")));
  const updateSelectionUI = () => {
    const count = selectedPaths.size;
    selectedCount.textContent = `${count} selected`;
    [bulkCopyButton, bulkMoveButton, bulkDeleteButton].forEach((button) => { button.disabled = count === 0; });
    selectionBar.hidden = !selectionMode;
    fileList?.classList.toggle("has-selection-bar", selectionMode);
    fileList?.classList.toggle("is-selecting", selectionMode);
    selectButton?.classList.toggle("primary", selectionMode);
    selectButton?.setAttribute("aria-pressed", String(selectionMode));
    if (nameHeader) nameHeader.classList.toggle("is-selecting", selectionMode);
    if (selectAllInput) {
      selectAllInput.hidden = !selectionMode;
      const visiblePaths = lastVisibleEntries.map(entryPath);
      const selectedVisible = visiblePaths.filter((rel) => selectedPaths.has(rel)).length;
      selectAllInput.checked = visiblePaths.length > 0 && selectedVisible === visiblePaths.length;
      selectAllInput.indeterminate = selectedVisible > 0 && selectedVisible < visiblePaths.length;
    }
  };
  const setSelectionMode = (enabled) => {
    stopSelectionDrag();
    selectionMode = enabled;
    if (!enabled) {
      selectedPaths.clear();
      selectionAnchorPath = "";
    }
    updateSelectionUI();
    draw(search.value.trim().toLowerCase());
  };
  let filterMode = "folders-first";
  const byName = (a, b) => String(a.name || "").localeCompare(String(b.name || ""), undefined, { sensitivity: "base", numeric: true });
  const draw = (query = "") => {
    let visible = (entries || []).filter((entry) => recordMatchesSearch(entry, query, fmtBytes(entry.size), fmtTime(entry.mod_time || entry.modified)));
    if (filterMode === "folders-only") visible = visible.filter((entry) => entry.is_dir);
    if (filterMode === "files-only") visible = visible.filter((entry) => !entry.is_dir);
    visible.sort((a, b) => {
      if (filterMode === "folders-first") return Number(b.is_dir) - Number(a.is_dir) || byName(a, b);
      if (filterMode === "name-desc") return -byName(a, b);
      if (filterMode === "modified-newest") return (Date.parse(b.mod_time || b.modified) || 0) - (Date.parse(a.mod_time || a.modified) || 0) || byName(a, b);
      if (filterMode === "modified-oldest") return (Date.parse(a.mod_time || a.modified) || 0) - (Date.parse(b.mod_time || b.modified) || 0) || byName(a, b);
      if (filterMode === "size-largest") return Number(b.size || 0) - Number(a.size || 0) || byName(a, b);
      if (filterMode === "size-smallest") return Number(a.size || 0) - Number(b.size || 0) || byName(a, b);
      return byName(a, b);
    });
    lastVisibleEntries = visible;
    renderedFileRows.clear();
    const rows = visible.length
      ? visible.map((entry, index) => fileRow(entry, index))
      : [el("tr", { class: "file-empty-row" }, el("td", { colspan: "4", class: "muted file-empty-cell" }, (entries || []).length ? "No matching files." : "Empty directory"))];
    if (parentLocation) rows.unshift(parentRow());
    tbody.replaceChildren(...rows);
    updateSelectionUI();
  };
  const search = pageSearchInput("files", draw);
  const filter = pageFilterMenu("Filter files", [
    ["folders-first", "Folders first"],
    ["name", "Name: A-Z"],
    ["name-desc", "Name: Z-A"],
    ["modified-newest", "Modified: Newest"],
    ["modified-oldest", "Modified: Oldest"],
    ["size-largest", "Size: Largest"],
    ["size-smallest", "Size: Smallest"],
    ["folders-only", "Folders only"],
    ["files-only", "Files only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  selectButton = el("button", {
    class: "btn",
    type: "button",
    title: directoryRequest.scope.writable ? "Select files" : "Selection is available inside a project",
    disabled: directoryRequest.scope.writable ? null : "disabled",
    "aria-pressed": "false",
    onclick: () => setSelectionMode(!selectionMode),
  }, "Select");
  selectAllInput = el("input", {
    class: "file-selection-checkbox", type: "checkbox", hidden: "",
    "aria-label": "Select all visible files",
    onchange: (event) => {
      for (const entry of lastVisibleEntries) {
        setEntrySelected(entry, event.currentTarget.checked);
      }
      updateSelectionUI();
    },
  });
  nameHeader = el("th", {}, el("span", { class: "file-name-heading" }, selectAllInput, "Name"));
  bulkCopyButton.addEventListener("click", () => fileDestinationPicker(main, path, selectedPaths, "copy"));
  bulkMoveButton.addEventListener("click", () => fileDestinationPicker(main, path, selectedPaths, "move"));
  bulkDeleteButton.addEventListener("click", () => bulkDeleteEntries(main, path, selectedPaths));
  const createMenu = fileCreateMenu(main, path, directoryRequest.scope.writable);
  const subtitle = fileContextSubtitle(directoryRequest.scope);
  const headerActions = [
    el("div", { class: "page-search-filter-controls" }, search, filter),
    el("button", {
      class: "btn",
      title: directoryRequest.scope.writable ? "Upload" : "Upload is available inside a project",
      disabled: directoryRequest.scope.writable ? null : "disabled",
      onclick: () => upInput.click(),
    }, solarIcon("upload-linear"), "Upload"),
    el("div", { class: "file-select-create-group" }, selectButton, createMenu),
    upInput,
  ];
  fileList = el("div", { class: "file-list" },
    el("table", {},
      el("thead", {}, el("tr", {}, nameHeader, el("th", { class: "file-size-column" }, "Size"), el("th", { class: "mobile-hide" }, "Modified"), el("th", {}, ""))),
      tbody));
  main.append(
    pageHeader("Files", subtitle, headerActions, managedPageBackButton()),
    locationBar,
    selectionBar,
    fileList);
  draw();
}

function fileActions(main, path, entry, writable = true) {
  const rel = (path ? path + "/" : "") + entry.name;
  const download = !entry.is_dir ? "/api" + filePathEndpoint("/files/download", rel).url : "";
  const editItems = () => writable ? [
    el("button", { class: "action-menu-item", type: "button", role: "menuitem", onclick: () => renameEntry(main, path, entry.name) }, solarIcon("pen-new-square-linear"), "Rename"),
    el("button", { class: "action-menu-item danger", type: "button", role: "menuitem", onclick: () => deleteEntry(main, path, entry.name) }, solarIcon("trash-bin-trash-linear"), "Delete"),
  ] : [];
  const desktopEditItems = editItems();
  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    download ? el("a", {
      class: "btn ghost small icon-button file-desktop-download",
      href: download, title: "Download", "aria-label": `Download ${entry.name}`,
    }, solarIcon("download-linear")) : "",
    desktopEditItems.length ? overflowActionsMenu(`Edit ${entry.name}`, desktopEditItems) : "");
  const mobileItems = download ? [el("a", {
    class: "action-menu-item", role: "menuitem",
    href: download, title: "Download", "aria-label": `Download ${entry.name}`,
  }, solarIcon("download-linear"), "Download"), ...editItems()] : editItems();
  if (!mobileItems.length && !download) return "";
  return el("div", { class: "responsive-row-actions" }, desktop,
    mobileItems.length ? overflowActionsMenu(`Actions for ${entry.name}`, mobileItems, "mobile-row-actions") : "");
}

function openFile(main, rel, returnTo = null) {
  if (isPreviewableImage(rel)) return openFileImagePreview(main, rel, returnTo);
  return openFileEditor(main, rel, returnTo);
}

function fileViewerBack(main, returnTo) {
  fileEscapeAction = null;
  if (returnTo?.page === "configuration") {
    navigate("configuration", { serverId: returnTo.serverId, fromOverview: returnTo.fromOverview, fromServers: returnTo.fromServers, fromConsole: returnTo.fromConsole });
    return;
  }
  pageFiles(main);
}

function openFileImagePreview(main, rel, returnTo = null) {
  const previewRequest = filePathEndpoint("/files/preview", rel);
  const downloadRequest = filePathEndpoint("/files/download", rel);
  const preview = DEMO_MODE && rel.toLowerCase().endsWith("server-icon.png")
    ? (S.servers.find((server) => server.id === (S.managedServerId || S.activeId))?.demo_icon || DEMO_SERVERS[0].demo_icon)
    : "/api" + previewRequest.url;
  const download = DEMO_MODE ? preview : "/api" + downloadRequest.url;
  const back = () => fileViewerBack(main, returnTo);
  const image = el("img", { src: preview, alt: `Preview of ${rel}`, decoding: "async" });
  const error = el("p", { class: "muted hidden" }, "This image could not be previewed. You can still download it.");
  image.addEventListener("error", () => {
    image.classList.add("hidden");
    error.classList.remove("hidden");
  }, { once: true });
  fileEscapeAction = back;
  main.innerHTML = "";
  main.append(
    el("div", { class: "toolbar" },
      el("h1", { class: "mono", style: "font-size:1rem" }, rel),
      el("div", { class: "spacer" }),
      el("button", { class: "btn ghost", title: returnTo ? "Back to Configuration" : "Back to files", onclick: back }, solarIcon("folder-open-linear"), "Back"),
      el("a", { class: "btn", href: download, download: rel.split("/").pop() || "image" }, solarIcon("download-linear"), "Download")),
    el("div", { class: "file-image-viewer" }, image, error));
}

async function openFileEditor(main, rel, returnTo = null) {
  const fileRequest = filePathEndpoint("/files/content", rel);
  let data;
  try { data = await api(fileRequest.url); }
  catch (e) { toast(e.message, "err"); return; }
  main.innerHTML = "";
  const ta = el("textarea", {
    class: "editor" + (S.fileEditorWrap ? " is-wrapped" : ""),
    spellcheck: "false", wrap: S.fileEditorWrap ? "soft" : "off",
    readonly: fileRequest.scope.writable ? null : "readonly",
  });
  ta.value = data.content;
  let baseline = data.content;
  const toggleEditorWrap = () => {
    S.fileEditorWrap = !S.fileEditorWrap;
    const actionLabel = S.fileEditorWrap ? "Disable text wrapping" : "Wrap text";
    ta.classList.toggle("is-wrapped", S.fileEditorWrap);
    ta.setAttribute("wrap", S.fileEditorWrap ? "soft" : "off");
    mobileWrapButton.classList.toggle("is-active", S.fileEditorWrap);
    desktopWrapButton.classList.toggle("is-on", S.fileEditorWrap);
    [mobileWrapButton, desktopWrapButton].forEach((button) => {
      button.setAttribute("aria-pressed", String(S.fileEditorWrap));
      button.setAttribute("aria-label", actionLabel);
      button.setAttribute("title", actionLabel);
    });
    desktopWrapButton.querySelector(".bot-power-label").textContent = S.fileEditorWrap ? "On" : "Off";
  };
  const mobileWrapButton = el("button", {
    class: "btn ghost file-editor-wrap-control file-editor-wrap-mobile mobile-icon-only" + (S.fileEditorWrap ? " is-active" : ""),
    type: "button",
    "aria-label": S.fileEditorWrap ? "Disable text wrapping" : "Wrap text",
    "aria-pressed": String(S.fileEditorWrap),
    title: S.fileEditorWrap ? "Disable text wrapping" : "Wrap text",
    onclick: toggleEditorWrap,
  }, solarIcon("wrap-text"));
  const desktopWrapButton = el("button", {
    class: "bot-power file-editor-wrap-desktop-toggle" + (S.fileEditorWrap ? " is-on" : ""),
    type: "button",
    "aria-label": S.fileEditorWrap ? "Disable text wrapping" : "Wrap text",
    "aria-pressed": String(S.fileEditorWrap),
    title: S.fileEditorWrap ? "Disable text wrapping" : "Wrap text",
    onclick: toggleEditorWrap,
  },
  el("span", { class: "bot-power-track", "aria-hidden": "true" }, el("span", {})),
  el("span", { class: "bot-power-label" }, S.fileEditorWrap ? "On" : "Off"));
  const finishLeaving = () => {
    if (returnTo?.page === "configuration") {
      fileEscapeAction = null;
      navigate("configuration", { serverId: returnTo.serverId, fromOverview: returnTo.fromOverview, fromServers: returnTo.fromServers, fromConsole: returnTo.fromConsole });
      return;
    }
    pageFiles(main);
  };
  const leaveEditor = () => {
    if (ta.value === baseline) {
      finishLeaving();
      return;
    }
    confirmModal("Discard changes", `Discard unsaved changes to "${rel}"?`, "Discard", async () => finishLeaving());
  };
  fileEscapeAction = leaveEditor;
  main.append(
    el("div", { class: "toolbar" },
      el("h1", { class: "mono", style: "font-size:1rem" }, rel),
      fileRequest.scope.writable ? null : el("span", { class: "tag" }, "Read only"),
      el("div", { class: "spacer" }),
      el("span", { class: "file-editor-wrap-desktop-group" },
        el("span", { class: "file-editor-wrap-desktop-label" }, "Wrap text"),
        desktopWrapButton),
      mobileWrapButton,
      el("button", { class: "btn ghost", title: returnTo ? "Back to Configuration" : "Back to files", onclick: leaveEditor }, solarIcon("folder-open-linear"), "Back"),
      fileRequest.scope.writable ? el("button", { class: "btn primary", title: "Save file", onclick: async () => {
        try {
          await api(fileScopeEndpoint("/files/content", fileRequest.scope), { method: "POST", json: { path: fileRequest.scope.path, content: ta.value } });
          baseline = ta.value;
          toast("Saved (a .bonghos-backup copy of important files is kept)", "ok");
        } catch (e) { toast(e.message, "err"); }
      } }, solarIcon("diskette-linear"), "Save") : null),
    ta);
}

function fileCreateMenu(main, path, writable) {
  const menu = overflowActionsMenu("Create", [
    el("button", {
      class: "action-menu-item", type: "button", role: "menuitem",
      onclick: () => mkdirPrompt(main, path),
    }, solarIcon("folder-linear"), "New folder"),
    el("button", {
      class: "action-menu-item", type: "button", role: "menuitem",
      onclick: () => newFilePrompt(main, path),
    }, solarIcon("document-text-linear"), "New file"),
  ], "file-create-menu", el("span", { class: "file-create-plus", "aria-hidden": "true" }, "+"));
  const trigger = menu.querySelector(".action-menu-trigger");
  trigger.classList.remove("ghost");
  trigger.disabled = !writable;
  trigger.title = writable ? "Create" : "Create is available inside a project";
  return menu;
}

function newFilePrompt(main, path) {
  const inp = el("input", { placeholder: "filename.txt", autocomplete: "off" });
  modal("New file", [el("div", { class: "field-row" }, el("label", {}, "File name", inp))], [
    ["Cancel", "ghost", (c) => c()],
    ["Create", "primary", async (c) => {
      const name = inp.value.trim();
      if (!name) return toast("File name is required", "err");
      const scope = fileRequestScope(path);
      const target = (scope.path ? scope.path + "/" : "") + name;
      try {
        await api(fileScopeEndpoint("/files/create", scope), { method: "POST", json: { path: target } });
        c();
        openFile(main, (path ? path + "/" : "") + name);
      } catch (e) { toast(e.message, "err"); }
    }],
  ]);
}

function fileDestinationPicker(main, path, selectedPaths, operation) {
  const scope = fileRequestScope(path);
  if (!scope.writable || !scope.project || !selectedPaths.size) return;
  const sourceProject = scope.project;
  const sources = [...selectedPaths.keys()].map((rel) => fileRequestScope(rel).path);
  const sourceServersPath = projectServersPath(sourceProject);
  let destinationServersPath = sourceServersPath
    ? [sourceServersPath, scope.path].filter(Boolean).join("/")
    : "";
  let destinationProject = sourceServersPath ? sourceProject : null;
  let destination = sourceServersPath ? scope.path : "";
  const destinationLabel = el("span", { class: "mono" });
  const crumbs = el("div", { class: "breadcrumb file-picker-breadcrumb" });
  const folders = el("div", { class: "file-destination-list" });
  let confirmDestinationButton;
  let renderVersion = 0;
  const renderDestination = async (nextServersPath = "") => {
    const version = ++renderVersion;
    destinationServersPath = String(nextServersPath || "").replace(/^\/+|\/+$/g, "");
    const projectMatch = projectAtServersPath(destinationServersPath);
    destinationProject = projectMatch?.project || null;
    destination = projectMatch
      ? destinationServersPath.slice(projectMatch.directory.length).replace(/^\/+/, "")
      : "";
    if (confirmDestinationButton) {
      const sameFolder = destinationProject
        && String(destinationProject.id) === String(sourceProject.id)
        && destination === scope.path;
      confirmDestinationButton.disabled = !destinationProject || sameFolder;
      confirmDestinationButton.title = !destinationProject
        ? "Open a managed project to choose a destination"
        : (sameFolder ? "Choose a different destination folder" : `${operation === "copy" ? "Copy" : "Move"} here`);
    }
    destinationLabel.textContent = !destinationProject
      ? (destinationServersPath ? `Servers/${destinationServersPath} (read only)` : "Servers (read only)")
      : (destination ? `${destinationProject.display_name}/${destination}` : destinationProject.display_name);
    crumbs.innerHTML = "";
    crumbs.append(el("span", { onclick: () => renderDestination("") }, "Servers"));
    let acc = "";
    for (const part of destinationServersPath.split("/").filter(Boolean)) {
      acc += (acc ? "/" : "") + part;
      const target = acc;
      crumbs.append(" / ", el("span", { onclick: () => renderDestination(target) }, part));
    }
    folders.replaceChildren(el("p", { class: "muted" }, "Loading folders…"));
    try {
      const destinationScope = destinationProject
        ? { project: destinationProject, serversRoot: false }
        : { project: null, serversRoot: true };
      const endpoint = appendFileQuery(fileScopeEndpoint("/files", destinationScope), "path",
        destinationProject ? destination : destinationServersPath);
      const entries = await api(endpoint);
      if (version !== renderVersion) return;
      const rows = [];
      if (destinationServersPath) {
        const parent = destinationServersPath.split("/").filter(Boolean).slice(0, -1).join("/");
        rows.push(el("button", { class: "file-destination-row", type: "button", onclick: () => renderDestination(parent) },
          solarIcon("folder-linear"), el("span", { class: "mono" }, "..")));
      }
      for (const entry of entries.filter((item) => item.is_dir)) {
        const target = (destinationServersPath ? destinationServersPath + "/" : "") + entry.name;
        rows.push(el("button", { class: "file-destination-row", type: "button", onclick: () => renderDestination(target) },
          solarIcon("folder-linear"), el("span", { class: "mono" }, entry.name)));
      }
      folders.replaceChildren(...(rows.length ? rows : [el("p", { class: "muted" }, "No folders here.")]));
    } catch (error) {
      if (version !== renderVersion) return;
      folders.replaceChildren(el("p", { class: "error" }, error.message));
    }
  };
  const verb = operation === "copy" ? "Copy" : "Move";
  modal(`${verb} ${sources.length} selected item${sources.length === 1 ? "" : "s"}`, [
    el("p", { class: "muted" }, `${verb} to `, destinationLabel),
    crumbs,
    folders,
  ], [
    ["Cancel", "ghost", (c) => c()],
    [`${verb} here`, "primary", async (c) => {
      if (!destinationProject) return;
      try {
        await api(fileScopeEndpoint(`/files/${operation}`, scope), {
          method: "POST", json: {
            paths: sources, destination, destination_server_id: destinationProject.id,
          },
        });
        c();
        toast(`${sources.length} item${sources.length === 1 ? "" : "s"} ${operation === "copy" ? "copied" : "moved"}`, "ok");
        pageFiles(main, path, fileBrowseRoot);
      } catch (error) { toast(error.message, "err"); }
    }],
  ]);
  confirmDestinationButton = $("#modal-host .modal .actions .btn.primary");
  renderDestination(destinationServersPath);
}

function bulkDeleteEntries(main, path, selectedPaths) {
  const selected = [...selectedPaths.keys()];
  if (!selected.length) return;
  confirmModal("Delete selected items",
    `Delete ${selected.length} selected item${selected.length === 1 ? "" : "s"}? This cannot be undone.`,
    "Delete", async () => {
      try {
        for (const rel of selected) {
          const scope = fileRequestScope(rel);
          await api(fileScopeEndpoint("/files/delete", scope), {
            method: "POST", json: { path: scope.path, confirm: true },
          });
        }
        toast(`${selected.length} item${selected.length === 1 ? "" : "s"} deleted`, "ok");
        pageFiles(main, path, fileBrowseRoot);
      } catch (error) { toast(error.message, "err"); }
    });
}

function mkdirPrompt(main, path) {
  const inp = el("input", { placeholder: "folder-name" });
  modal("New folder", [el("div", { class: "field-row" }, inp)], [
    ["Cancel", "ghost", (c) => c()],
    ["Create", "primary", async (c) => {
      c();
      const scope = fileRequestScope(path);
      const target = (scope.path ? scope.path + "/" : "") + inp.value;
      try { await api(fileScopeEndpoint("/files/mkdir", scope), { method: "POST", json: { path: target } }); pageFiles(main, path, fileBrowseRoot); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function renameEntry(main, path, name) {
  const inp = el("input", { value: name });
  modal("Rename", [el("div", { class: "field-row" }, inp)], [
    ["Cancel", "ghost", (c) => c()],
    ["Rename", "primary", async (c) => {
      c();
      const scope = fileRequestScope(path);
      const from = (scope.path ? scope.path + "/" : "") + name;
      const to = (scope.path ? scope.path + "/" : "") + inp.value;
      try { await api(fileScopeEndpoint("/files/rename", scope), { method: "POST", json: { from, to } }); pageFiles(main, path, fileBrowseRoot); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function deleteEntry(main, path, name) {
  const rel = (path ? path + "/" : "") + name;
  confirmModal("Delete", `Delete ${rel}? This cannot be undone.`, "Delete", async () => {
    const scope = fileRequestScope(rel);
    try { await api(fileScopeEndpoint("/files/delete", scope), { method: "POST", json: { path: scope.path, confirm: true } }); pageFiles(main, path, fileBrowseRoot); }
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
  fileBrowseRoot = "project";
  filePath = String(path || "").split("/").filter(Boolean).slice(0, -1).join("/");
  S.pendingFileOpen = {
    path,
    returnTo: {
      page: "configuration",
      serverId: S.managedServerId,
      fromOverview: S.overviewReturn,
      fromServers: S.serverManagementReturn,
      fromConsole: S.consoleReturn,
    },
  };
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
      el("p", { class: "muted" }, "Upload a PNG, JPEG, or WebP image. This will be converted to a 64×64 PNG."),
      can("server.icon.manage") ? el("div", { class: "actions" }, changeButton, input) :
        el("p", { class: "hint" }, "You do not have permission to change this icon.")));
  card.resetPreview = () => preview.replaceChildren(serverCardIcon(server));
  return card;
}

async function pageConfiguration(main) {
  const d = await api(serverScopedPath("/configuration"));
  const inst = d.instance;
  main.innerHTML = "";

  const jvmEditable = !!(d.jvm && d.jvm.editable);
  const memoryInputAttrs = {
    disabled: jvmEditable ? null : "disabled",
    title: jvmEditable ? "" : "Open the detected JVM source to change this value",
  };
  const xms = el("input", { ...memoryInputAttrs, value: (d.jvm && d.jvm.xms) || inst.jvm_xms || "", placeholder: "e.g. 2G" });
  const xmx = el("input", { ...memoryInputAttrs, value: (d.jvm && d.jvm.xmx) || inst.jvm_xmx || "", placeholder: "e.g. 6G" });
  const scriptSel = el("select", {},
    ...(d.scripts || []).map((s) => el("option", { value: s.path, selected: s.path === inst.startup_script ? "" : null },
      `${s.path} (${s.modloader || "unknown"}, score ${s.score})`)));
  const javaSel = el("select", {},
    el("option", { value: "auto", selected: inst.java_selection === "auto" || !inst.java_selection ? "" : null }, "Automatic"),
    ...(d.java || []).map((j) => el("option", { value: j.path, selected: inst.java_selection === j.path ? "" : null },
      `${j.version} — ${j.path}`)));

  const props = d.properties || {};
  const commonProps = ["motd", "server-port", "max-players", "difficulty", "gamemode", "white-list", "pvp", "view-distance", "simulation-distance", "online-mode"];
  const numericPropertyRules = {
    "server-port": { min: "1", max: "65535" },
    "max-players": { min: "0" },
    "view-distance": { min: "0" },
    "simulation-distance": { min: "0" },
  };
  const propertyOptions = {
    difficulty: [["peaceful", "Peaceful"], ["easy", "Easy"], ["normal", "Normal"], ["hard", "Hard"]],
    gamemode: [["survival", "Survival"], ["creative", "Creative"], ["adventure", "Adventure"], ["spectator", "Spectator"]],
    "white-list": [["true", "True"], ["false", "False"]],
    pvp: [["true", "True"], ["false", "False"]],
    "online-mode": [["true", "True"], ["false", "False"]],
  };
  const propInputs = {};
  const propRows = commonProps.filter((k) => k in props).map((k) => {
    const current = esc(props[k]);
    let v;
    if (propertyOptions[k]) {
      const options = [...propertyOptions[k]];
      if (!options.some(([value]) => value === current)) options.unshift([current, current]);
      v = el("select", {}, ...options.map(([value, label]) =>
        el("option", { value, selected: value === current ? "" : null }, label)));
    } else if (numericPropertyRules[k]) {
      v = el("input", {
        type: "number", value: current, required: "", step: "1", inputmode: "numeric",
        ...numericPropertyRules[k],
      });
    } else {
      v = el("input", { value: current });
    }
    propInputs[k] = v;
    return el("div", { class: "field-row" }, el("label", {}, k, v));
  });

  const auto = el("input", { type: "checkbox" }); auto.checked = !!inst.autostart_enabled;
  const recover = el("input", { type: "checkbox" }); recover.checked = !!inst.recover_after_unclean_shutdown;
  const delay = el("input", { type: "number", min: "0", value: inst.boot_delay_seconds || 0, style: "width:110px" });
  const policy = el("select", {},
    ...["never", "on-failure", "always"].map((p) => el("option", { value: p, selected: p === (inst.restart_policy || "never") ? "" : null }, p)));

  const automationCard = el("div", { class: "card configuration-automation-card" },
    el("div", { class: "configuration-automation-controls" },
      el("div", { class: "field-row" }, el("label", { class: "check-row" }, auto, " Start this server when the machine boots")),
      el("div", { class: "field-row" }, el("label", { class: "check-row" }, recover, " Recover after unclean shutdown (power loss)")),
      el("div", { class: "field-row" }, el("label", {}, "Boot delay (seconds)", delay)),
      el("div", { class: "field-row" }, el("label", {}, "Crash restart policy", policy))),
    el("div", { class: "configuration-automation-help-column" },
      el("div", { class: "configuration-automation-help" },
        el("div", { class: "configuration-automation-help-group" },
          el("h3", {}, "Start this server when the machine boots"),
          el("p", { class: "muted" }, "Automatically launches this project after Bonghos starts and the boot delay finishes.")),
        el("div", { class: "configuration-automation-help-group" },
          el("h3", {}, "Recover after unclean shutdown"),
          el("p", { class: "muted" }, "Allows autostart after a power loss or another unclean shutdown. When disabled, Bonghos leaves the server stopped for review.")),
        el("div", { class: "configuration-automation-help-group" },
          el("h3", {}, "Boot delay"),
          el("p", { class: "muted" }, "Waits this many seconds after Bonghos starts before automatically launching this server.")),
        el("div", { class: "configuration-automation-help-group" },
          el("h3", {}, "Crash restart policy"),
          el("ul", { class: "configuration-automation-policy-list" },
            el("li", {}, el("strong", {}, "Never:"), " Leave the server stopped after a crash."),
            el("li", {}, el("strong", {}, "On failure:"), " Restart after an unexpected non-zero exit."),
            el("li", {}, el("strong", {}, "Always:"), " Keep automatic crash recovery enabled; requested and clean stops remain stopped.")),
          el("p", { class: "hint" }, "Crash-loop protection pauses automatic restarts after repeated rapid crashes.")))));

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
    const invalidProperty = Object.values(propInputs).find((input) => !input.checkValidity());
    if (invalidProperty) {
      invalidProperty.reportValidity();
      invalidProperty.focus();
      return;
    }
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

  const serverPropertiesCard = el("div", { class: "card server-properties-card" },
    can("server.files.manage")
      ? el("div", { class: "server-properties-card-actions" },
        el("button", { class: "btn ghost small", onclick: openServerProperties }, "Open server.properties"))
      : null,
    propRows.length
      ? el("div", { class: "server-properties-fields" }, ...propRows)
      : el("p", { class: "muted" }, "No server.properties found yet (it is created on first start)."));

  main.append(
    pageHeader("Configuration", projectContextSubtitle("Editing", inst, inst.id === S.activeId, false), [headerDiscard, headerSave], managedPageBackButton()),
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
        el("div", { class: "configuration-memory-fields" },
          el("div", { class: "field-row" }, el("label", {}, "Minimum (-Xms)", xms)),
          el("div", { class: "field-row" }, el("label", {}, "Maximum (-Xmx)", xmx)))),
      el("div", { class: "card" },
        el("h3", {}, "Startup"),
        el("div", { class: "field-row" }, el("label", {}, "Startup script", scriptSel)),
        el("div", { class: "field-row flow-section" }, el("label", {}, "Java installation", javaSel)))),
    el("h2", {}, "Server icon"),
    iconCard,
    el("h2", {}, "server.properties"),
    serverPropertiesCard,
    el("h2", {}, "Automation"),
    automationCard,
    bottomActions);

  updateActions();
}

// ----- backups --------------------------------------------------------------
async function pageBackups(main) {
  const [list, storage] = await Promise.all([api("/backups"), api("/backups/storage")]);
  main.innerHTML = "";
  const integrityIcon = (status) => {
    if (status !== "verified" && status !== "failed") return null;
    const failed = status === "failed";
    const label = failed ? "Integrity check failed" : "Integrity checked";
    return el("span", {
      class: `backup-integrity-icon${failed ? " is-error" : ""}`,
      title: label,
      "aria-label": label,
    }, solarIcon(failed ? "danger-triangle-linear" : "check-circle-linear"));
  };
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
      el("span", { class: "backup-id-line" },
        el("span", {}, b.backup_id),
        integrityIcon(b.verification_status)),
      el("span", { class: "mobile-only mobile-row-detail" }, `${b.backup_type.replace(/_/g, " ")} · ${fmtBytes(b.compressed_size)}`)),
    el("td", {}, b.backup_type.replace(/_/g, " ")),
    el("td", { class: "mobile-hide" }, b.consistency_mode + " / " + b.trigger_type),
    el("td", {}, fmtBytes(b.compressed_size)),
    el("td", { class: "mobile-hide" }, fmtTime(b.created_at)),
    el("td", { class: "table-actions" }, backupActions(b)));
  const tbody = el("tbody");
  let filterMode = "newest";
  const byNewest = (a, b) => (Date.parse(b.created_at) || 0) - (Date.parse(a.created_at) || 0);
  const draw = (query = "") => {
    let visible = (list || []).filter((backup) => recordMatchesSearch(backup, query, fmtBytes(backup.compressed_size), fmtTime(backup.created_at)));
    if (filterMode === "full") visible = visible.filter((backup) => backup.backup_type === "full_server");
    if (filterMode === "world") visible = visible.filter((backup) => String(backup.backup_type).includes("world"));
    if (filterMode === "configuration") visible = visible.filter((backup) => String(backup.backup_type).includes("configuration"));
    if (filterMode === "protected") visible = visible.filter((backup) => backup.protected);
    if (filterMode === "unprotected") visible = visible.filter((backup) => !backup.protected);
    visible.sort((a, b) => {
      if (filterMode === "oldest") return -byNewest(a, b);
      if (filterMode === "size-largest") return Number(b.compressed_size || 0) - Number(a.compressed_size || 0) || byNewest(a, b);
      if (filterMode === "size-smallest") return Number(a.compressed_size || 0) - Number(b.compressed_size || 0) || byNewest(a, b);
      return byNewest(a, b);
    });
    tbody.replaceChildren(...(visible.length
      ? visible.map(backupRow)
      : [el("tr", {}, el("td", { colspan: "6", class: "muted" }, (list || []).length ? "No matching backups." : "No backups yet."))]));
  };
  const search = pageSearchInput("backups", draw);
  const filter = pageFilterMenu("Filter backups", [
    ["newest", "Created: Newest"],
    ["oldest", "Created: Oldest"],
    ["size-largest", "Size: Largest"],
    ["size-smallest", "Size: Smallest"],
    ["full", "Full server only"],
    ["world", "World only"],
    ["configuration", "Configuration only"],
    ["protected", "Protected only"],
    ["unprotected", "Unprotected only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  main.append(
    pageHeader("Backups", "Verified archives, retention decisions, and restore controls. Online backups briefly pause world saving.", [
      el("div", { class: "page-search-filter-controls" }, search, filter),
      can("server.backups.create") ? mkBtn("world", "World backup") : null,
      can("server.backups.create") ? mkBtn("full", "Full backup") : null,
      can("server.backups.create") ? mkBtn("configuration", "Config backup") : null,
    ]),
    el("div", { class: "notice backup-storage-location" },
      el("strong", {}, storage.external ? "External backup storage" : "Backup storage"),
      el("span", { class: "mono" }, storage.path),
      storage.external ? el("span", { class: "muted" }, "Excluded from Bonghos disk size.") : null),
    el("div", { class: "progress hidden", id: "backup-progress" }, el("div", { style: "width:0%" })),
    el("div", { class: "table-wrap backups-table" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "ID"), el("th", {}, "Type"), el("th", { class: "mobile-hide" }, "Mode"), el("th", {}, "Size"), el("th", { class: "mobile-hide" }, "Created"), el("th", {}, ""))),
        tbody)),
    el("div", { class: "backup-action-help", "aria-label": "Backup actions explained" },
      el("div", { class: "backup-action-help-item" },
        el("strong", {}, "Restore"),
        el("p", { class: "muted" }, "Replaces the selected server data. The server must be stopped; Bonghos first makes an emergency backup.")),
      el("div", { class: "backup-action-help-item" },
        el("strong", {}, "Check"),
        el("p", { class: "muted" }, "Check integrity of this backup to ensure it is readable and unchanged.")),
      el("div", { class: "backup-action-help-item" },
        el("strong", {}, "Protect"),
        el("p", { class: "muted" }, "Blocks manual deletion and automatic cleanup. Restoring still works.")),
      el("div", { class: "backup-action-help-item" },
        el("strong", {}, "Delete"),
        el("p", { class: "muted" }, "Permanently removes an unprotected backup."))));
  draw();
}

function backupActionIcon(name) {
  if (name !== "shield-cross-linear" && name !== "file-magnifying-glass") return solarIcon(name);
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "icon");
  svg.setAttribute("viewBox", name === "shield-cross-linear" ? "0 0 24 24" : "0 0 256 256");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.innerHTML = name === "shield-cross-linear"
    ? `<path d="M0 0h24v24H0z" fill="none"/><g fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 10.417c0-3.198 0-4.797.378-5.335c.377-.537 1.88-1.052 4.887-2.081l.573-.196C10.405 2.268 11.188 2 12 2s1.595.268 3.162.805l.573.196c3.007 1.029 4.51 1.544 4.887 2.081C21 5.62 21 7.22 21 10.417v1.574c0 5.638-4.239 8.375-6.899 9.536C13.38 21.842 13.02 22 12 22s-1.38-.158-2.101-.473C7.239 20.365 3 17.63 3 11.991z"/><path stroke-linecap="round" d="m14.5 9.5l-5 5m0-5l5 5"/></g>`
    : `<path d="M0 0h256v256H0z" fill="none"/><path fill="currentColor" d="m213.66 82.34l-56-56A8 8 0 0 0 152 24H56a16 16 0 0 0-16 16v176a16 16 0 0 0 16 16h144a16 16 0 0 0 16-16V88a8 8 0 0 0-2.34-5.66M160 51.31L188.69 80H160ZM200 216H56V40h88v48a8 8 0 0 0 8 8h48zm-45.54-48.85a36.05 36.05 0 1 0-11.31 11.31l11.19 11.2a8 8 0 0 0 11.32-11.32ZM104 148a20 20 0 1 1 20 20a20 20 0 0 1-20-20"/>`;
  return svg;
}

function backupActions(backup) {
  const actions = [];
  if (can("server.backups.restore")) actions.push({
    slot: "restore", label: "Restore", icon: "archive-down-minimlistic-linear", run: () => restoreBackup(backup),
  });
  if (can("server.backups.create")) actions.push(
    { slot: "check", label: "Check", title: "Check integrity of this backup to ensure it is readable and unchanged", icon: "file-magnifying-glass", run: async () => {
      try {
        await api(`/backups/${backup.backup_id}/verify`, { method: "POST", json: {} });
        toast("Backup integrity check passed", "ok");
        renderPage();
      } catch (error) { toast(`Backup integrity check failed: ${error.message}`, "err"); }
    } },
    { slot: "protect", label: backup.protected ? "Unprotect" : "Protect", icon: backup.protected ? "shield-cross-linear" : "shield-check-linear", run: async () => {
      try { await api(`/backups/${backup.backup_id}/protect`, { method: "POST", json: { protected: !backup.protected } }); renderPage(); }
      catch (error) { toast(error.message, "err"); }
    } },
    { slot: "delete", label: "Delete", icon: "trash-bin-trash-linear", danger: true, run: () =>
      confirmModal("Delete backup", `Delete backup ${backup.backup_id}?` + (backup.protected ? " It is PROTECTED." : ""), "Delete", async () => {
        try { await api(`/backups/${backup.backup_id}`, { method: "DELETE" }); renderPage(); }
        catch (error) { toast(error.message, "err"); }
      }) });

  if (!actions.length) return el("span", { class: "muted" }, "View only");

  const desktop = el("div", { class: "desktop-row-actions backup-action-grid" },
    ...actions.map((action) => el("button", {
      class: `btn ${action.danger ? "danger" : "ghost"} backup-action-${action.slot}`,
      title: action.title,
      onclick: action.run,
    }, backupActionIcon(action.icon), action.label)));
  const mobile = overflowActionsMenu(`Actions for backup ${backup.backup_id}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", title: action.title, onclick: action.run,
    }, backupActionIcon(action.icon), action.label)), "mobile-row-actions");
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
    el("p", { class: "muted modal-control-note" },
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
    el("td", {}, s.name, s.enabled ? "" : el("span", { class: "tag inline-offset" }, "Disabled")),
    el("td", {}, s.action.replace(/_/g, " ")),
    el("td", { class: "mono" }, s.schedule_type + ": " + s.schedule_expression + " (" + (s.timezone || "UTC") + ")"),
    el("td", {}, fmtTime(s.next_run_at)),
    el("td", { class: "schedule-last-result" }, s.last_result || "—"),
    el("td", { class: "table-actions schedule-actions-cell" },
      can("server.schedules.manage") ? scheduleActions(s) : ""));
  const tbody = el("tbody");
  let filterMode = "next-run";
  const byName = (a, b) => String(a.name || "").localeCompare(String(b.name || ""), undefined, { sensitivity: "base" });
  const draw = (query = "") => {
    let visible = (list || []).filter((schedule) => recordMatchesSearch(schedule, query, fmtTime(schedule.next_run_at)));
    if (filterMode === "enabled") visible = visible.filter((schedule) => schedule.enabled);
    if (filterMode === "disabled") visible = visible.filter((schedule) => !schedule.enabled);
    if (filterMode === "result-success") visible = visible.filter((schedule) => String(schedule.last_result || "").toLowerCase() === "success");
    if (filterMode === "result-failed") visible = visible.filter((schedule) => String(schedule.last_result || "").toLowerCase() === "failed");
    if (filterMode === "result-skipped") visible = visible.filter((schedule) => String(schedule.last_result || "").toLowerCase() === "skipped");
    visible.sort((a, b) => {
      if (filterMode === "next-run" || filterMode === "next-run-latest") {
        const aTime = Date.parse(a.next_run_at);
        const bTime = Date.parse(b.next_run_at);
        if (Number.isFinite(aTime) !== Number.isFinite(bTime)) return Number.isFinite(aTime) ? -1 : 1;
        const difference = aTime - bTime;
        return (filterMode === "next-run-latest" ? -difference : difference) || byName(a, b);
      }
      if (filterMode === "name-desc") return -byName(a, b);
      if (filterMode === "action-asc") return String(a.action || "").localeCompare(String(b.action || ""), undefined, { sensitivity: "base" }) || byName(a, b);
      if (filterMode === "action-desc") return String(b.action || "").localeCompare(String(a.action || ""), undefined, { sensitivity: "base" }) || byName(a, b);
      return byName(a, b);
    });
    tbody.replaceChildren(...(visible.length
      ? visible.map(scheduleRow)
      : [el("tr", {}, el("td", { colspan: "6", class: "muted" }, (list || []).length ? "No matching schedules." : "No schedules yet."))]));
  };
  const search = pageSearchInput("schedules", draw);
  const filter = pageFilterMenu("Filter schedules", [
    ["next-run", "Next run: Soonest"],
    ["next-run-latest", "Next run: Latest"],
    ["name", "Name: A-Z"],
    ["name-desc", "Name: Z-A"],
    ["action-asc", "Action: A-Z"],
    ["action-desc", "Action: Z-A"],
    ["enabled", "Enabled only"],
    ["disabled", "Disabled only"],
    ["result-success", "Result: Success only"],
    ["result-failed", "Result: Failed only"],
    ["result-skipped", "Result: Skipped only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  main.append(
    pageHeader("Schedules", "Persistent Linux-host schedules with next run, last result, and manual run controls.", [
      el("div", { class: "page-search-filter-controls" }, search, filter),
      can("server.schedules.manage") ? el("button", { class: "btn primary", onclick: () => scheduleForm(null) }, "New schedule") : null,
    ]),
    el("div", { class: "table-wrap schedules-table" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Name"), el("th", {}, "Action"), el("th", {}, "When"), el("th", {}, "Next run"), el("th", { class: "schedule-last-result" }, "Last result"), el("th", {}, ""))),
        tbody)));
  draw();
}

function scheduleActions(schedule) {
  const actions = [
    { label: "Run now", icon: "play-linear", run: async () => {
      try { await api(`/schedules/${schedule.id}/run`, { method: "POST", json: {} }); toast("Running now", "ok"); }
      catch (error) { toast(error.message, "err"); }
    } },
    { label: "Edit", icon: "pen-new-square-linear", run: () => scheduleForm(schedule) },
    { label: "Duplicate", icon: "copy-linear", run: () => scheduleForm(schedule, true) },
    { label: "Delete", icon: "trash-bin-trash-linear", danger: true, run: () =>
      confirmModal("Delete schedule", `Delete schedule "${schedule.name}"?`, "Delete", async () => {
        try { await api(`/schedules/${schedule.id}`, { method: "DELETE" }); renderPage(); }
        catch (error) { toast(error.message, "err"); }
      }) },
  ];
  const desktop = el("div", { class: "desktop-row-actions schedule-row-actions schedule-action-grid" },
    ...actions.map((action) => el("button", {
      class: "btn " + (action.danger ? "danger" : "ghost"), onclick: action.run,
    }, action.label)));
  const mobile = overflowActionsMenu(`Actions for ${schedule.name}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
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

function scheduleForm(s, duplicate = false) {
  const editing = !!s && !duplicate;
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

  const duplicateName = `${String(s?.name || "Schedule").slice(0, 115).trimEnd()} copy`;
  const name = el("input", { value: duplicate ? duplicateName : (s?.name || ""), required: "", maxlength: "120" });
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
      const input = el("input", { type: "datetime-local", step: "1", value: saved.replace(" ", "T") });
      expressionHost.append(el("label", {}, "Date and time", input));
      readExpression = () => input.value.replace("T", " ");
    } else if (scheduleType === "hourly") {
      const [savedMinute = "0", savedSecond = "0"] = saved.split(":");
      const minute = el("input", { type: "number", min: "0", max: "59", step: "1", value: savedMinute || "0" });
      const second = el("input", { type: "number", min: "0", max: "59", step: "1", value: savedSecond || "0" });
      expressionHost.append(el("div", { class: "grid cols-2" },
        el("label", {}, "Minute of each hour", minute), el("label", {}, "Second", second)));
      readExpression = () => `${minute.value}:${String(Number(second.value) || 0).padStart(2, "0")}`;
    } else if (scheduleType === "daily") {
      const input = el("input", { type: "time", step: "1", value: saved || "04:00" });
      expressionHost.append(el("label", {}, "Time", input));
      readExpression = () => input.value;
    } else if (scheduleType === "weekly") {
      const [savedDay = "MON", savedTime = "04:00"] = saved.toUpperCase().split(/\s+/);
      const day = el("select", {}, ...[["MON", "Monday"], ["TUE", "Tuesday"], ["WED", "Wednesday"], ["THU", "Thursday"], ["FRI", "Friday"], ["SAT", "Saturday"], ["SUN", "Sunday"]]
        .map(([value, label]) => el("option", { value, selected: savedDay === value ? "" : null }, label)));
      const time = el("input", { type: "time", step: "1", value: savedTime || "04:00" });
      expressionHost.append(el("div", { class: "grid cols-2" }, el("label", {}, "Day", day), el("label", {}, "Time", time)));
      readExpression = () => `${day.value} ${time.value}`;
    } else if (scheduleType === "monthly") {
      const [savedDay = "1", savedTime = "04:00"] = saved.split(/\s+/);
      const day = el("input", { type: "number", min: "1", max: "31", step: "1", value: savedDay || "1" });
      const time = el("input", { type: "time", step: "1", value: savedTime || "04:00" });
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

  modal(editing ? "Edit schedule" : (duplicate ? "Duplicate schedule" : "New schedule"), [
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
        if (editing) await api(`/schedules/${s.id}`, { method: "PATCH", json: body });
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
  S.perfInternet = null;
  S.perfInternetHistory = [];
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
    ], overviewBackButton()),

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
          "Host physical memory, Java process RSS, and configured Java heap limits.", "server-2-linear")),
      el("div", { class: "performance-meter-grid" },
        performanceMeter("Machine memory", "host-memory"),
        performanceMeter("Java process memory (RSS / machine)", "allocated-memory")),
      el("div", { class: "grid cols-2 performance-domain-charts" },
        performanceChartPanel("Machine memory", "Physical memory used by the host", "performance-chart-host-memory"),
        performanceChartPanel("Java resident memory", "RSS includes heap and native memory; -Xmx limits only the heap", "performance-chart-rss"))),

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
      el("div", { class: "performance-storage-visual", id: "performance-storage-visual" })),

    el("section", { class: "performance-domain flow-section", "aria-labelledby": "performance-internet-title" },
      el("div", { class: "performance-section-heading has-action" },
        performanceSectionTitle("performance-internet-title", "Internet",
          "Checks follow the update interval while this page is open. Speed tests are manual.", "global-linear"),
        el("button", {
          class: "btn ghost icon-button performance-storage-refresh",
          id: "performance-internet-refresh",
          type: "button",
          title: "Refresh connectivity",
          "aria-label": "Refresh connectivity",
          onclick: () => refreshInternetConnectivity(true),
        }, solarIcon("storage-refresh"))),
      el("div", { class: "performance-domain-readouts performance-internet-readouts" },
        performanceReadout("Connectivity", "performance-internet-status", "Checking from the Bonghos host"),
        performanceReadout("Connection", "performance-internet-latency", "Average TCP connection time"),
        performanceReadout("DNS", "performance-internet-dns", "System resolver lookup time"),
        performanceReadout("HTTPS", "performance-internet-https", "Average diagnostic round trip"),
        performanceReadout("Reliability", "performance-internet-reliability", "Successful checks from the latest 10")),
      el("div", { class: "performance-internet-targets metric-note", id: "performance-internet-targets" }, "Checking Internet connectivity…"),
      el("div", { class: "grid cols-2 performance-domain-charts performance-internet-details" },
        performanceChartPanel("Connection latency", "Shared checks at the selected update interval", "performance-chart-internet-latency"),
        internetSpeedTestPanel())));

  syncPageSubscription("performance");
  updatePerformanceView(current || latestPerformanceSample());
  renderStorageVisual();
  renderInternetVisual();
  activatePendingPerformanceTarget();
  refreshPerformanceStorage();
  startPerformanceInternetPolling();
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
  if (S.page === "performance") startPerformanceInternetPolling();
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
  if (node) node.textContent = esc(value);
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
  const heapLimits = [];
  if (xms > 0) heapLimits.push(`${fmtBytes(xms)} min heap (-Xms)`);
  if (xmx > 0) heapLimits.push(`${fmtBytes(xmx)} max heap (-Xmx)`);
  updatePerformanceMeter("allocated-memory", rss, hostTotal,
    `${fmtBytes(rss)} resident · ${heapLimits.length ? heapLimits.join(" · ") : "configured heap limits not detected"}`);

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
        { label: "Max heap (-Xmx)", tone: "warning", value: (s) => Number(s.jvm_xmx_bytes) || configuredXmx, format: fmtBytes },
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
      centerTotal: diskUsed,
      centerLabel: "used",
      centerToggle: true,
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
      centerTotal: bonghosTotal,
      centerLabel: "used",
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

function internetSpeedTestPanel() {
  const allowed = can("server.performance.test");
  return el("div", { class: "card performance-chart-panel performance-speed-test-panel" },
    el("div", { class: "performance-chart-heading performance-speed-test-heading" },
      el("div", {},
        el("h3", {}, "Manual speed test"),
        el("p", { class: "metric-note" }, "Estimated WAN throughput between this host and Cloudflare.")),
      el("button", {
        class: "btn primary performance-speed-test-button",
        id: "performance-speed-test-button",
        type: "button",
        disabled: allowed ? null : "",
        title: allowed ? "Run speed test" : "Your role cannot run Internet speed tests",
        onclick: confirmInternetSpeedTest,
      }, solarIcon("play-linear"), "Test")),
    el("div", { class: "performance-speed-results" },
      performanceReadout("Download", "performance-speed-download", "Not tested"),
      performanceReadout("Upload", "performance-speed-upload", "Not tested")),
    el("div", { class: "performance-speed-test-detail metric-note", id: "performance-speed-test-detail" },
      allowed
        ? "Manual only · transfers up to about 53 MB · results are approximate"
        : "Requires the Test internet speed permission."));
}

function confirmInternetSpeedTest() {
  confirmModal(
    "Test Internet speed",
    "This test transfers up to about 53 MB through Cloudflare and may temporarily slow a running Minecraft server.",
    "Run test",
    runInternetSpeedTest,
    false,
  );
}

async function runInternetSpeedTest() {
  if (S.page !== "performance" || !can("server.performance.test")) return;
  stopPerformanceInternetPolling();
  cancelPerformanceSpeedTest();
  const controller = new AbortController();
  performanceSpeedAbort = controller;
  const button = $("#performance-speed-test-button");
  const detail = $("#performance-speed-test-detail");
  if (button) {
    button.disabled = true;
    button.classList.add("is-loading");
    button.replaceChildren(solarIcon("storage-refresh"), "Testing…");
  }
  if (detail) detail.textContent = "Testing download and upload throughput from the Bonghos host…";
  setNodeText("performance-speed-download", "Testing…");
  setNodeText("performance-speed-upload", "Waiting…");
  try {
    S.perfSpeedTest = await api("/metrics/internet/speed-test", {
      method: "POST", json: { confirm: true }, signal: controller.signal,
    });
    if (S.page !== "performance") return;
    renderInternetSpeedResult();
    toast("Internet speed test completed", "ok");
  } catch (error) {
    if (S.page !== "performance") return;
    setNodeText("performance-speed-download", "Failed");
    setNodeText("performance-speed-upload", "Failed");
    if (detail) detail.textContent = error.message;
    toast("Internet speed test failed: " + error.message, "err");
  } finally {
    if (performanceSpeedAbort === controller) performanceSpeedAbort = null;
    if (S.page !== "performance") return;
    const currentButton = $("#performance-speed-test-button");
    if (currentButton) {
      currentButton.disabled = !can("server.performance.test");
      currentButton.classList.remove("is-loading");
      currentButton.replaceChildren(solarIcon("play-linear"), "Test again");
    }
    startPerformanceInternetPolling();
  }
}

function cancelPerformanceSpeedTest() {
  if (!performanceSpeedAbort) return;
  performanceSpeedAbort.abort();
  performanceSpeedAbort = null;
}

function renderInternetSpeedResult(result = S.perfSpeedTest) {
  if (!result) return;
  setNodeText("performance-speed-download", formatMbps(result.download_mbps));
  setNodeText("performance-speed-upload", formatMbps(result.upload_mbps));
  setNodeText("performance-speed-download-note", `${fmtBytes(result.download_bytes)} transferred`);
  setNodeText("performance-speed-upload-note", `${fmtBytes(result.upload_bytes)} transferred`);
  const detail = $("#performance-speed-test-detail");
  if (detail) {
    detail.textContent = `${result.provider || "Remote edge"} · ${Number(result.latency_ms || 0).toFixed(1)} ms latency · ${formatDurationMilliseconds(result.duration_ms)} · ${fmtTime(result.tested_at)}`;
  }
}

function formatMbps(value) {
  const speed = Number(value);
  if (!Number.isFinite(speed) || speed < 0) return "—";
  const digits = speed >= 100 ? 0 : speed >= 10 ? 1 : 2;
  return `${speed.toFixed(digits)} Mbps`;
}

function formatDurationMilliseconds(value) {
  const milliseconds = Math.max(0, Number(value) || 0);
  return milliseconds >= 1000 ? `${(milliseconds / 1000).toFixed(1)} s` : `${Math.round(milliseconds)} ms`;
}

function startPerformanceInternetPolling() {
  stopPerformanceInternetPolling();
  if (S.page !== "performance") return;
  refreshInternetConnectivity();
  performanceInternetTimer = setInterval(
    refreshInternetConnectivity,
    S.perfIntervalSeconds * 1000,
  );
}

function stopPerformanceInternetPolling() {
  if (performanceInternetTimer) clearInterval(performanceInternetTimer);
  performanceInternetTimer = null;
  if (performanceInternetAbort) performanceInternetAbort.abort();
  performanceInternetAbort = null;
  performanceInternetRequest++;
}

async function refreshInternetConnectivity(manual = false) {
  if (S.page !== "performance") return;
  if (!manual && performanceInternetAbort) return;
  if (manual) stopPerformanceInternetPolling();
  const controller = manual ? null : new AbortController();
  if (controller) performanceInternetAbort = controller;
  const request = ++performanceInternetRequest;
  const button = $("#performance-internet-refresh");
  if (manual && button) {
    button.disabled = true;
    button.classList.add("is-loading");
  }
  try {
    const endpoint = manual
      ? "/metrics/internet/refresh"
      : `/metrics/internet?interval_seconds=${S.perfIntervalSeconds}`;
    const snapshot = await api(endpoint, manual ? { method: "POST" } : { signal: controller.signal });
    if (request !== performanceInternetRequest || S.page !== "performance") return;
    S.perfInternet = snapshot;
    const previous = S.perfInternetHistory[S.perfInternetHistory.length - 1];
    if (!previous || previous.collected_at !== snapshot.collected_at) {
      S.perfInternetHistory.push(snapshot);
      S.perfInternetHistory = S.perfInternetHistory.slice(-120);
    }
    renderInternetVisual();
  } catch (error) {
    if (request !== performanceInternetRequest || S.page !== "performance") return;
    if (manual) toast("Connectivity refresh failed: " + error.message, "err");
    const targets = $("#performance-internet-targets");
    if (targets && !S.perfInternet) targets.textContent = "Connectivity could not be checked.";
  } finally {
    if (controller && performanceInternetAbort === controller) performanceInternetAbort = null;
    if (request !== performanceInternetRequest || S.page !== "performance") return;
    const currentButton = $("#performance-internet-refresh");
    if (currentButton) {
      currentButton.disabled = false;
      currentButton.classList.remove("is-loading");
    }
    if (manual) startPerformanceInternetPolling();
  }
}

function renderInternetVisual(snapshot = S.perfInternet) {
  const chart = $("#performance-chart-internet-latency");
  if (!snapshot) {
    if (chart) chart.replaceChildren(el("div", { class: "performance-chart-empty" }, "Waiting for a connectivity check."));
    renderInternetSpeedResult();
    return;
  }
  const status = String(snapshot.status || "checking").toLowerCase();
  const connectionSuccessful = Number(snapshot.connection_successful_targets || 0);
  const connectionTotal = Number(snapshot.connection_total_targets || 0);
  const failureCount = Number(snapshot.consecutive_failures || 0);
  setNodeText("performance-internet-status", capitalizeFirst(status));
  setNodeText("performance-internet-status-note", status === "checking"
    ? "Waiting for the first TCP check"
    : connectionSuccessful === 0 && failureCount > 0 && status !== "offline"
      ? `Retrying · ${failureCount}/3 failed checks`
      : `${connectionSuccessful}/${connectionTotal} TCP targets reachable`);
  setNodeText("performance-internet-latency", connectionSuccessful ? `${Number(snapshot.connection_latency_ms || 0).toFixed(1)} ms` : "—");
  setNodeText("performance-internet-latency-note", "TCP connection time from the Bonghos host");

  const diagnosticsReady = Boolean(snapshot.diagnostics_collected_at);
  setNodeText("performance-internet-dns", diagnosticsReady
    ? snapshot.dns_ok ? `${Number(snapshot.dns_ms || 0).toFixed(1)} ms` : "Failed"
    : "Checking");
  setNodeText("performance-internet-dns-note", !diagnosticsReady
    ? "Waiting for DNS and HTTPS diagnostics"
    : snapshot.dns_ok ? "System resolver is responding" : "Check the host DNS configuration");
  const httpsSuccessful = Number(snapshot.successful_targets || 0);
  const httpsTotal = Number(snapshot.total_targets || 0);
  setNodeText("performance-internet-https", diagnosticsReady
    ? httpsSuccessful ? `${Number(snapshot.latency_ms || 0).toFixed(1)} ms` : "Failed"
    : "Checking");
  setNodeText("performance-internet-https-note", diagnosticsReady
    ? `${httpsSuccessful}/${httpsTotal} diagnostic targets reachable`
    : "Follows the selected update interval");

  const reliabilitySuccessful = Number(snapshot.reliability_successful || 0);
  const reliabilityTotal = Number(snapshot.reliability_total || 0);
  setNodeText("performance-internet-reliability", reliabilityTotal ? `${reliabilitySuccessful}/${reliabilityTotal}` : "—");
  setNodeText("performance-internet-reliability-note", reliabilityTotal
    ? `${Math.round(reliabilitySuccessful / reliabilityTotal * 100)}% reachable`
    : "Waiting for checks");

  const statusValue = $("#performance-internet-status");
  if (statusValue) statusValue.dataset.status = status;
  const targetDetail = $("#performance-internet-targets");
  if (targetDetail) {
    const connectionTargets = (snapshot.connection_targets || []).map((target) => target.reachable
      ? `${target.name} ${Number(target.latency_ms || 0).toFixed(1)} ms`
      : `${target.name} ${internetErrorLabel(target.error)}`);
    const httpsTargets = (snapshot.targets || []).map((target) => target.reachable
      ? `${target.name} ${Number(target.latency_ms || 0).toFixed(1)} ms`
      : `${target.name} ${internetErrorLabel(target.error)}`);
    const connectionText = connectionTargets.length ? `TCP: ${connectionTargets.join(" · ")}` : "TCP: checking";
    const diagnosticsText = diagnosticsReady
      ? `HTTPS: ${httpsTargets.join(" · ")} · Diagnostics ${fmtTime(snapshot.diagnostics_collected_at)}`
      : "HTTPS: checking";
    targetDetail.textContent = `${connectionText} · ${diagnosticsText}`;
  }
  if (chart) {
    const latencyHistory = S.perfInternetHistory.filter((entry) =>
      entry.connection_successful_targets > 0 && Number.isFinite(Number(entry.connection_latency_ms)));
    chart.replaceChildren(timeSeriesChart(latencyHistory, {
      label: "Connection latency history",
      min: 0,
      floorMax: 50,
      axisFormat: (value) => `${Math.round(value)} ms`,
      series: [{
        label: "Latency", tone: status === "offline" ? "warning" : "accent", area: true,
        value: (entry) => Number(entry.connection_latency_ms), format: (value) => `${value.toFixed(1)} ms`,
      }],
    }));
  }
  renderInternetSpeedResult();
}

function internetErrorLabel(code) {
  switch (code) {
    case "timeout": return "timed out";
    case "dns_failed": return "DNS failed";
    case "connection_failed": return "unreachable";
    default: return String(code || "failed").replace(/^http_/, "HTTP ");
  }
}

function storageDonutChart({ id = "", title, description, total, centerTotal = total, centerLabel = "total", centerToggle = false, segments, timestamp, emptyMessage }) {
  const attrs = { class: "card performance-storage-panel" };
  if (id) attrs.id = id;
  const heading = el("div", { class: "performance-chart-heading" },
    el("div", {}, el("h3", {}, title), el("p", { class: "metric-note" }, description)));
  if (total <= 0) return el("div", attrs, heading,
    el("div", { class: "performance-chart-empty" }, emptyMessage));

  const svg = svgElement("svg", { class: "performance-donut", viewBox: "0 0 240 240", role: "img", "aria-label": `${title} distribution` });
  const centerValue = el("strong", { class: "mono" });
  const centerCaption = el("span", {}, centerLabel);
  let centerControl = null;
  const detail = el("div", { class: "performance-donut-detail mono" }, `${fmtBytes(total)} total · ${fmtTime(timestamp)}`);
  let offset = 0;
  const entries = [];
  let activeEntry = null;
  const showSummary = () => {
    activeEntry = null;
    entries.forEach((entry) => {
      if (entry.circle) svg.append(entry.circle);
      entry.circle?.classList.remove("is-active");
      entry.row.classList.remove("is-active");
    });
    const showPercentage = centerToggle && performanceStorageShowPercentage;
    centerValue.textContent = showPercentage ? `${(centerTotal / total * 100).toFixed(1)}%` : fmtBytes(centerTotal);
    centerCaption.textContent = centerLabel;
    if (centerControl) {
      const action = showPercentage ? "Show used space" : "Show percentage used";
      centerControl.setAttribute("aria-label", action);
      centerControl.setAttribute("title", action);
      centerControl.setAttribute("aria-pressed", String(showPercentage));
    }
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
    centerCaption.textContent = entry.segment.label;
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
    row.addEventListener("pointerleave", showSummary);
    [row, circle].filter(Boolean).forEach((target) => {
      target.addEventListener("focus", () => showEntry(entry));
      target.addEventListener("blur", showSummary);
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
    else if (activeEntry && !entryHasVisibleFocus(activeEntry)) showSummary();
  });
  svg.addEventListener("pointerleave", () => {
    const focused = entries.find(entryHasVisibleFocus);
    if (focused) showEntry(focused);
    else showSummary();
  });
  if (centerToggle) {
    centerControl = el("button", {
      class: "performance-donut-center-toggle",
      type: "button",
      onclick: () => {
        performanceStorageShowPercentage = !performanceStorageShowPercentage;
        showSummary();
      },
    }, centerValue, centerCaption);
  }
  showSummary();
  return el("div", attrs, heading,
    el("div", { class: "performance-donut-layout" },
      el("div", { class: "performance-donut-plot" }, svg,
        el("div", { class: "performance-donut-center" }, centerControl || centerValue, centerControl ? null : centerCaption)),
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

function demoMetricSample(seconds) {
  const previous = latestPerformanceSample() || DEMO_METRICS[DEMO_METRICS.length - 1];
  const tick = Date.now() / 1000;
  return {
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
}

function syncDemoPerformanceStream() {
  if (demoOverviewTimer) clearInterval(demoOverviewTimer);
  if (demoPerformanceTimer) clearInterval(demoPerformanceTimer);
  demoOverviewTimer = null;
  demoPerformanceTimer = null;
  if (!DEMO_MODE) return;
  demoOverviewTimer = setInterval(() => {
    const sample = demoMetricSample(OVERVIEW_INTERVAL_SECONDS);
    updateSidebarLiveStats(sample);
    if (S.page !== "overview") return;
    appendPerformanceSample(sample);
    setUptimeBaseline(sample);
    updateLiveStats(sample);
  }, OVERVIEW_INTERVAL_SECONDS * 1000);
  if (S.page !== "performance") return;
  demoPerformanceTimer = setInterval(() => {
    const sample = demoMetricSample(S.perfIntervalSeconds);
    appendPerformanceSample(sample);
    setUptimeBaseline(sample);
    updateLiveStats(sample);
  }, S.perfIntervalSeconds * 1000);
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
  const summary = options.summary
    ? el("div", { class: "overview-sparkline-summary", "aria-hidden": "true" },
      el("div", { class: "metric-value" }, options.summary.value),
      el("div", { class: "metric-note" }, options.summary.note))
    : null;
  const plot = el("div", { class: "overview-sparkline-plot" }, svg, marker, tooltip, ...(summary ? [summary] : []));
  const wrapper = el("div", {
    class: "overview-sparkline", tabindex: "0",
    "aria-label": `${title} current ${fmt(points[points.length - 1].value)}. History from ${fmtTime(firstAt)} to ${fmtTime(lastAt)}, range ${axisFormat(min)} to ${axisFormat(max)}. Focus and use left or right arrow keys to inspect samples.`,
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
  const fallback = el("img", {
    class: "server-card-icon",
    src: "/server-placeholder.png",
    alt: "",
    width: 64,
    height: 64,
    loading: "lazy",
    decoding: "async",
  });
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
  const modifiedAt = server.updated_at || server.created_at;
  if (modifiedAt) items.push(el("span", { class: "server-game-version" },
    solarIcon("history-linear"), "Modified " + fmtTimeToMinute(modifiedAt)));
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

function overflowActionsMenu(label, items, className = "", triggerChildren = null) {
  if (!items.length) return null;
  const menu = el("div", { class: "action-menu", role: "menu", hidden: "" }, ...items);
  const triggerContent = triggerChildren === null
    ? [solarIcon("menu-dots-bold")]
    : (Array.isArray(triggerChildren) ? triggerChildren : [triggerChildren]);
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
  }, ...triggerContent);

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
        fileBrowseRoot = "project";
        fileEscapeAction = null;
        S.pendingFileOpen = null;
      }
      navigate(page, { serverId: server.id, fromServers: true });
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
    s2.id === S.activeId ? el("span", { class: "tag server-card-active-mobile" }, "Active") : null,
    el("div", { class: "server-card-body" },
      el("div", { class: "toolbar compact" },
        el("strong", {}, s2.display_name),
        s2.id === S.activeId ? el("span", { class: "tag server-card-active-desktop" }, "Active") : "",
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
  let filterMode = "name-asc";
  const byName = (a, b) => String(a.display_name || "").localeCompare(String(b.display_name || ""), undefined, { sensitivity: "base", numeric: true });
  const draw = (query = "") => {
    let visible = S.servers.filter((server) => recordMatchesSearch(server, query));
    if (filterMode === "active-only") visible = visible.filter((server) => server.id === S.activeId);
    if (filterMode === "non-active-only") visible = visible.filter((server) => server.id !== S.activeId);
    if (filterMode === "external-only") visible = visible.filter((server) => server.external_directory);
    if (filterMode === "managed-only") visible = visible.filter((server) => !server.external_directory);
    visible.sort((a, b) => {
      if (filterMode === "name-desc") return -byName(a, b);
      if (filterMode === "active-first") return Number(b.id === S.activeId) - Number(a.id === S.activeId) || byName(a, b);
      if (filterMode === "modified-newest") return (Date.parse(b.updated_at || b.created_at) || 0) - (Date.parse(a.updated_at || a.created_at) || 0) || byName(a, b);
      if (filterMode === "modified-oldest") return (Date.parse(a.updated_at || a.created_at) || 0) - (Date.parse(b.updated_at || b.created_at) || 0) || byName(a, b);
      if (filterMode === "game-version") return String(a.minecraft_version || "").localeCompare(String(b.minecraft_version || ""), undefined, { sensitivity: "base", numeric: true }) || byName(a, b);
      if (filterMode === "game-version-desc") return String(b.minecraft_version || "").localeCompare(String(a.minecraft_version || ""), undefined, { sensitivity: "base", numeric: true }) || byName(a, b);
      if (filterMode === "modloader") return String(a.modloader || "").localeCompare(String(b.modloader || ""), undefined, { sensitivity: "base" }) || String(a.modloader_version || "").localeCompare(String(b.modloader_version || ""), undefined, { numeric: true }) || byName(a, b);
      if (filterMode === "modloader-desc") return String(b.modloader || "").localeCompare(String(a.modloader || ""), undefined, { sensitivity: "base" }) || String(b.modloader_version || "").localeCompare(String(a.modloader_version || ""), undefined, { numeric: true }) || byName(a, b);
      return byName(a, b);
    });
    cardsHost.replaceChildren(...(visible.length
      ? visible.map(serverCard)
      : [el("p", { class: "muted" }, S.servers.length ? "No matching servers." : "No servers imported yet — use “Import server”.")]));
  };
  const search = pageSearchInput("servers", draw);
  const filter = pageFilterMenu("Filter servers", [
    ["name-asc", "Name: A-Z"],
    ["name-desc", "Name: Z-A"],
    ["active-first", "Active project first"],
    ["modified-newest", "Modified: Newest"],
    ["modified-oldest", "Modified: Oldest"],
    ["game-version", "Game version: Ascending"],
    ["game-version-desc", "Game version: Descending"],
    ["modloader", "Modloader: A-Z"],
    ["modloader-desc", "Modloader: Z-A"],
    ["active-only", "Active project only"],
    ["non-active-only", "Non-active projects only"],
    ["external-only", "External links only"],
    ["managed-only", "Bonghos storage only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  main.append(
    pageHeader("Servers", "Project inventory, active-project selection, and persistent import progress.", [
      el("div", { class: "page-search-filter-controls" }, search, filter),
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
  const fileInput = el("input", { type: "file", accept: ".zip,.tar,.gz,.tgz,.zst", style: "display:none" });
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
      name.value = f.name.replace(/\.(zip|tar|tgz|gz|zst)$/i, "").replace(/[._]+/g, " ").trim();
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
    el("label", {}, "Archive (.zip, .tar, .tar.gz, .tar.zst)", dropzone), fileInput);
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
const SECURITY_ACTIVITY_ACTION = /^(login_|passkey_|invitation_|role_|account_|sessions_|user_)/;
const isSecurityActivity = (event) => SECURITY_ACTIVITY_ACTION.test(String(event?.action || ""));

async function pageActivity(main) {
  const list = await api("/activity");
  main.innerHTML = "";
  const activityRow = (event) => el("tr", {},
    el("td", {}, fmtTime(event.at)), el("td", {}, event.username), el("td", {}, event.action.replace(/_/g, " ")),
    el("td", { class: "mono" }, event.target || ""), el("td", { class: "muted" }, event.detail || ""));
  const tbody = el("tbody");
  let filterMode = "newest";
  const byNewest = (a, b) => (Date.parse(b.at) || 0) - (Date.parse(a.at) || 0);
  const byText = (field, a, b) => String(a[field] || "").localeCompare(String(b[field] || ""), undefined, { sensitivity: "base", numeric: true });
  const draw = (query = "") => {
    let visible = (list || []).filter((event) => recordMatchesSearch(event, query, fmtTime(event.at), event.action.replace(/_/g, " ")));
    if (filterMode === "security") visible = visible.filter(isSecurityActivity);
    if (filterMode === "server-management") visible = visible.filter((event) => !isSecurityActivity(event));
    if (filterMode === "mine") visible = visible.filter((event) => String(event.username || "").toLowerCase() === String(S.me?.username || "").toLowerCase());
    visible.sort((a, b) => {
      if (filterMode === "oldest") return -byNewest(a, b);
      if (filterMode === "user-asc") return byText("username", a, b) || byNewest(a, b);
      if (filterMode === "user-desc") return -byText("username", a, b) || byNewest(a, b);
      if (filterMode === "action-asc") return byText("action", a, b) || byNewest(a, b);
      if (filterMode === "action-desc") return -byText("action", a, b) || byNewest(a, b);
      return byNewest(a, b);
    });
    tbody.replaceChildren(...(visible.length
      ? visible.map(activityRow)
      : [el("tr", {}, el("td", { colspan: "5", class: "muted" }, (list || []).length ? "No matching activity." : "No audit events recorded yet."))]));
  };
  const search = pageSearchInput("activity", draw);
  const filter = pageFilterMenu("Filter activity", [
    ["newest", "Date: Newest"],
    ["oldest", "Date: Oldest"],
    ["user-asc", "User: A-Z"],
    ["user-desc", "User: Z-A"],
    ["action-asc", "Action: A-Z"],
    ["action-desc", "Action: Z-A"],
    ["mine", "My activity only"],
    ["security", "Security only"],
    ["server-management", "Server management only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  main.append(
    pageHeader("Activity", "Audit trail of account and server-management actions.", [
      el("div", { class: "page-search-filter-controls" }, search, filter),
    ]),
    el("div", { class: "table-wrap" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "User"), el("th", {}, "Action"), el("th", {}, "Target"), el("th", {}, "Detail"))),
        tbody)));
  draw();
}

// ----- users -----------------------------------------------------------------
function groupedPermissionCatalog(catalog) {
  const groups = new Map();
  for (const definition of catalog || []) {
    if (!groups.has(definition.group)) groups.set(definition.group, []);
    groups.get(definition.group).push(definition);
  }
  return [...groups.entries()];
}

const ROLE_PERMISSION_DESCRIPTIONS = {
  owner: "Full access is permanent. Owner permissions cannot be changed.",
  admin: "Runs and manages Bonghos. Only an Owner can change this role.",
  member: "Operates the server with permissions granted by an Owner or authorized Admin.",
  viewer: "Read-only. Viewer can receive view permissions only.",
};
const ROLE_RANK = { owner: 4, admin: 3, member: 2, viewer: 1 };

async function manageRolePermissions() {
  let data;
  try { data = await api("/roles/permissions"); }
  catch (error) { return toast(error.message, "err"); }

  const roleOrder = ["owner", "admin", "member", "viewer"];
  let activeRole = roleOrder.find((role) => data.roles?.[role]?.editable) || "owner";
  const roleTabs = el("div", { class: "role-permission-tabs", role: "tablist", "aria-label": "Roles" });
  const permissionPane = el("div", { class: "role-permission-pane" });
  let saveButton = null;
  const drafts = Object.fromEntries(roleOrder.map((role) =>
    [role, new Set(data.roles?.[role]?.permissions || [])]));

  const samePermissionSelection = (left, right) => {
    if (left.size !== right.size) return false;
    return [...left].every((permission) => right.has(permission));
  };
  const roleIsDirty = (role) => !samePermissionSelection(
    drafts[role], new Set(data.roles?.[role]?.permissions || []));
  const captureActiveDraft = () => {
    const inputs = [...permissionPane.querySelectorAll("input[data-permission]")];
    if (!inputs.length) return;
    drafts[activeRole] = new Set(inputs.filter((input) => input.checked)
      .map((input) => input.dataset.permission));
  };
  const updateDraftState = () => {
    const roleInfo = data.roles[activeRole];
    const editable = !!roleInfo?.editable;
    const dirty = editable && roleIsDirty(activeRole);
    const stateTag = permissionPane.querySelector(".role-permission-state-tag");
    if (stateTag) stateTag.textContent = dirty
      ? "Unsaved"
      : (!editable ? "Fixed" : roleInfo.customized ? `Customized · r${roleInfo.revision}` : "Defaults");
    if (saveButton) {
      saveButton.disabled = !dirty;
      setButtonLabel(saveButton, `Save ${capitalizeFirst(activeRole)}`);
    }
  };

  const permissionInput = (permission) => permissionPane.querySelector(`input[data-permission="${CSS.escape(permission)}"]`);
  const applyDependencyChange = (changed, definitions) => {
    const byID = new Map(definitions.map((definition) => [definition.id, definition]));
    if (changed.checked) {
      const requirePermission = (permission, seen = new Set()) => {
        if (seen.has(permission)) return true;
        seen.add(permission);
        const input = permissionInput(permission);
        if (!input) return false;
        if (input.disabled && !input.checked) return false;
        for (const required of byID.get(permission)?.requires || []) {
          if (!requirePermission(required, seen)) return false;
        }
        input.checked = true;
        return true;
      };
      if (!requirePermission(changed.dataset.permission)) {
        changed.checked = false;
        toast("This permission requires access that cannot be granted by your account.", "err");
      }
      return;
    }
    const removeDependents = (permission, seen = new Set()) => {
      if (seen.has(permission)) return;
      seen.add(permission);
      for (const definition of definitions) {
        if (!(definition.requires || []).includes(permission)) continue;
        const input = permissionInput(definition.id);
        if (input && !input.disabled) input.checked = false;
        removeDependents(definition.id, seen);
      }
    };
    removeDependents(changed.dataset.permission);
  };

  const draw = () => {
    const roleInfo = data.roles[activeRole];
    const selected = drafts[activeRole] || new Set(roleInfo.permissions || []);
    const defaults = new Set(roleInfo.defaults || []);
    const editable = !!roleInfo.editable;
    const definitions = data.catalog || [];
    const groups = groupedPermissionCatalog(definitions);
    const canUseDefaults = S.me.role === "owner" || [...defaults].every((permission) =>
      can(permission) || selected.has(permission));
    roleTabs.querySelectorAll("button").forEach((button) => {
      const current = button.dataset.role === activeRole;
      button.classList.toggle("active", current);
      button.setAttribute("aria-selected", String(current));
    });
    permissionPane.replaceChildren(
      el("div", { class: "role-permission-heading" },
        el("div", {},
          el("div", { class: "role-permission-title-row" },
            el("h3", {}, capitalizeFirst(activeRole)),
            el("span", { class: "tag role-permission-state-tag" })),
          el("p", { class: "muted" }, ROLE_PERMISSION_DESCRIPTIONS[activeRole])),
        editable ? el("button", {
          class: "btn ghost role-defaults-button", type: "button",
          disabled: canUseDefaults ? null : "disabled",
          title: canUseDefaults ? "Restore the shipped permission choices" : "You do not hold every permission required by this role's defaults",
          onclick: () => {
          permissionPane.querySelectorAll("input[data-permission]").forEach((input) => {
            if (!input.disabled) input.checked = defaults.has(input.dataset.permission);
          });
          captureActiveDraft();
          updateDraftState();
          },
        }, "Use defaults") : el("span")),
      ...groups.map(([group, permissions]) => el("section", { class: "role-permission-group" },
        el("h4", {}, group),
        el("div", { class: "role-permission-options" }, ...permissions.map((definition) => {
          const permission = definition.id;
          const actorCanGrant = S.me.role === "owner" || can(permission) || selected.has(permission);
          const allowedForRole = (definition.assignable_roles || []).includes(activeRole);
          const disabled = !editable || !actorCanGrant || !allowedForRole;
          let disabledReason = "";
          if (!editable) disabledReason = "Owner permissions are fixed.";
          else if (!allowedForRole) disabledReason = `This permission cannot be assigned to ${capitalizeFirst(activeRole)}.`;
          else if (!actorCanGrant) disabledReason = "You cannot grant a permission you do not have.";
          const prerequisites = (definition.requires || []).map((required) =>
            definitions.find((candidate) => candidate.id === required)?.label || required);
          return el("label", {
            class: `check-row role-permission-option${disabled ? " is-disabled" : ""}`,
            title: disabledReason || null,
          },
            el("input", {
              type: "checkbox", checked: selected.has(permission) ? "" : null,
              disabled: disabled ? "" : null, "data-permission": permission,
              onchange(event) {
                applyDependencyChange(event.currentTarget, definitions);
                captureActiveDraft();
                updateDraftState();
              },
            }),
            el("span", {},
              el("strong", {}, definition.label),
              el("small", {}, definition.description),
              prerequisites.length ? el("small", { class: "role-permission-requires" }, `Requires: ${prerequisites.join(", ")}`) : null));
        })))));
    updateDraftState();
  };

  roleTabs.append(...roleOrder.map((role) => el("button", {
    class: "role-permission-tab", type: "button", role: "tab", "data-role": role,
    onclick: () => { captureActiveDraft(); activeRole = role; draw(); },
  }, capitalizeFirst(role))));

  modal("Role permissions", [
    el("p", { class: "role-permission-intro" }, "Choose what each role can do in Bonghos. Changes apply to every user with that role."),
    el("div", { class: "role-permission-manager" }, roleTabs, permissionPane),
    el("p", { class: "hint role-permission-footnote" }, "Drafts stay available while you switch roles. Closing this window discards unsaved drafts. Affected users reconnect only after a saved change."),
  ], [
    ["Close", "ghost", (close) => close()],
    ["Save permissions", "primary", async () => {
      if (!data.roles[activeRole]?.editable) return;
      captureActiveDraft();
      if (!roleIsDirty(activeRole)) return;
      saveButton.disabled = true;
      try {
        const permissions = [...drafts[activeRole]];
        const revision = data.roles[activeRole].revision;
        data = await api(`/roles/${activeRole}/permissions`, { method: "PUT", json: { permissions, revision } });
        drafts[activeRole] = new Set(data.roles[activeRole].permissions || []);
        toast(`${capitalizeFirst(activeRole)} permissions saved`, "ok");
        draw();
      } catch (error) {
        if (error.status === 409 || /changed elsewhere/i.test(error.message)) {
          try {
            data = await api("/roles/permissions");
            drafts[activeRole] = new Set(data.roles[activeRole].permissions || []);
          }
          catch { /* Keep the current view if refresh also fails. */ }
          toast("Role permissions changed elsewhere. The latest settings were loaded; review and save again.", "err");
          draw();
          return;
        }
        toast(error.message, "err");
        draw();
      }
    }],
  ], null, "role-permissions-modal");
  saveButton = [...document.querySelectorAll(".role-permissions-modal .actions .btn")]
    .find((button) => button.textContent.trim() === "Save permissions");
  draw();
}

async function pageUsers(main) {
  const users = await api("/users");
  main.innerHTML = "";
  const userRow = (u) => el("tr", {},
    el("td", {}, el("div", { class: "user-identity" },
      el("strong", {}, u.Username),
      el("span", { class: "mobile-only mobile-row-detail" }, `${capitalizeFirst(u.Role)} · ${u.Disabled ? "Disabled" : "Active"}`))),
    el("td", {}, el("span", { class: "tag" }, capitalizeFirst(u.Role))),
    el("td", {}, u.Disabled ? "Disabled" : "Active"),
    el("td", { class: "table-actions" }, userActions(u)));
  const tbody = el("tbody");
  let filterMode = "name-asc";
  const byName = (a, b) => String(a.Username || "").localeCompare(String(b.Username || ""), undefined, { sensitivity: "base" });
  const draw = (query = "") => {
    let visible = (users || []).filter((user) => recordMatchesSearch(user, query, user.Disabled ? "disabled" : "active"));
    if (filterMode === "active-only") visible = visible.filter((user) => !user.Disabled);
    if (filterMode === "disabled-only") visible = visible.filter((user) => user.Disabled);
    if (/^role-(owner|admin|member|viewer)$/.test(filterMode)) {
      visible = visible.filter((user) => String(user.Role || "").toLowerCase() === filterMode.slice(5));
    }
    visible.sort((a, b) => {
      if (filterMode === "role-owner-first") {
        const ownerOrder = Number(String(b.Role || "").toLowerCase() === "owner") - Number(String(a.Role || "").toLowerCase() === "owner");
        return ownerOrder || String(a.Role || "").localeCompare(String(b.Role || ""), undefined, { sensitivity: "base" }) || byName(a, b);
      }
      if (filterMode === "name-desc") return -byName(a, b);
      if (filterMode === "active-first") return Number(a.Disabled) - Number(b.Disabled) || byName(a, b);
      if (filterMode === "disabled-first") return Number(b.Disabled) - Number(a.Disabled) || byName(a, b);
      return byName(a, b);
    });
    tbody.replaceChildren(...(visible.length
      ? visible.map(userRow)
      : [el("tr", {}, el("td", { colspan: "4", class: "muted" }, (users || []).length ? "No matching users." : "No users returned by the API."))]));
  };
  const search = pageSearchInput("users", draw);
  const filter = pageFilterMenu("Filter users", [
    ["name-asc", "Username: A-Z"],
    ["name-desc", "Username: Z-A"],
    ["role-owner-first", "Role: Owner first"],
    ["active-first", "Status: Active first"],
    ["disabled-first", "Status: Disabled first"],
    ["active-only", "Active only"],
    ["disabled-only", "Disabled only"],
    ["role-owner", "Owner only"],
    ["role-admin", "Admin only"],
    ["role-member", "Member only"],
    ["role-viewer", "Viewer only"],
  ], (value) => { filterMode = value; draw(search.value.trim().toLowerCase()); }, filterMode);
  main.append(
    pageHeader("Users", "Accounts, roles, invitations, sessions, and final-Owner protection.", [
      el("div", { class: "page-search-filter-controls" }, search, filter),
      can("roles.manage") ? el("button", { class: "btn ghost", onclick: manageRolePermissions }, solarIcon("shield-keyhole-linear"), "Role permissions") : null,
      can("users.manage") ? el("button", { class: "btn primary", onclick: inviteUser }, "Invite user") : null,
    ]),
    el("div", { class: "table-wrap users-table" },
      el("table", {},
        el("thead", {}, el("tr", {}, el("th", {}, "Username"), el("th", {}, "Role"), el("th", {}, "Status"), el("th", {}, ""))),
        tbody)));
  draw();
}

function userActions(user) {
  if (user.ID === S.me.id) return el("span", { class: "muted" }, "you");
  if (!can("users.manage")) return el("span", { class: "muted" }, "View only");
  if (S.me.role !== "owner" && ROLE_RANK[S.me.role] <= ROLE_RANK[user.Role]) {
    return el("span", { class: "muted" }, "Protected");
  }
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
  const desktop = el("div", { class: "desktop-row-actions user-action-grid" },
    ...actions.map((action) => el("button", { class: "btn " + (action.danger ? "danger" : "ghost"), onclick: action.run }, action.label)));
  const mobile = overflowActionsMenu(`Actions for ${user.Username}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""), type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions" }, desktop, mobile);
}

function inviteUser() {
  const availableRoles = [["admin", "Admin"], ["member", "Member"], ["viewer", "Viewer"]]
    .filter(([value]) => S.me.role === "owner" || ROLE_RANK[value] < ROLE_RANK[S.me.role]);
  const role = el("select", {},
    ...availableRoles.map(([v, r]) => el("option", { value: v }, r)));
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
          el("p", { class: "muted modal-control-note" }, "Expires " + fmtTime(d.expires_at) + "."),
        ], [["Done", "primary", (c2) => c2()]]);
      } catch (e) { toast(e.message, "err"); }
    }]]);
}

function changeRole(u) {
  const availableRoles = [["owner", "Owner"], ["admin", "Admin"], ["member", "Member"], ["viewer", "Viewer"]]
    .filter(([value]) => S.me.role === "owner" || ROLE_RANK[value] < ROLE_RANK[S.me.role]);
  const role = el("select", {},
    ...availableRoles
      .map(([v, r]) => el("option", { value: v, selected: v === u.Role ? "" : null }, r)));
  modal("Change role", [el("div", { class: "field-row" }, el("label", {}, u.Username, role))], [
    ["Cancel", "ghost", (c) => c()],
    ["Save", "primary", async (c) => {
      c();
      try { await api(`/users/${u.ID}/role`, { method: "POST", json: { role: role.value } }); renderPage(); }
      catch (e) { toast(e.message, "err"); }
    }]]);
}

function accountHasLocalPasskey(passkeys) {
  const currentRP = location.hostname.toLowerCase();
  return passkeys.some((passkey) => String(passkey.rp_id || "").toLowerCase() === currentRP);
}

function verifyAccountAction(purpose, hasLocalPasskey, onVerified) {
  const password = el("input", {
    type: "password", autocomplete: "current-password", required: "",
    value: DEMO_MODE ? "demo-password" : "",
  });
  const code = el("input", {
    class: "account-reauth-code", autocomplete: "one-time-code", maxlength: "32", required: "",
    placeholder: "6-digit code or recovery code", value: DEMO_MODE ? "123456" : "",
  });
  const finish = async (close, actionToken) => {
    close();
    try { await onVerified(actionToken); }
    catch (error) { toast(error.message, "err"); }
  };
  const usePasskey = async (close) => {
    if (!passkeysSupported()) return toast("Passkeys require HTTPS or localhost.", "err");
    const button = document.activeElement;
    if (button instanceof HTMLButtonElement) button.disabled = true;
    try {
      const started = await api("/account/reauth/passkey/begin", { method: "POST", json: { purpose } });
      const credential = await navigator.credentials.get({
        publicKey: passkeyRequestOptions(started.options.publicKey),
      });
      const verified = await api(`/account/reauth/passkey/finish?flow=${encodeURIComponent(started.flow)}`, {
        method: "POST", json: passkeyCredentialJSON(credential),
      });
      await finish(close, verified.action_token);
    } catch (error) {
      toast(passkeyError(error, "Passkey verification failed."), "err");
    } finally {
      if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
    }
  };
  modal("Verify it’s you", [
    el("p", { class: "muted" }, "Sensitive account changes require a fresh identity check."),
    el("div", { class: "field-row" }, el("label", {}, "Current password", password)),
    el("div", { class: "field-row" }, el("label", {}, "Authenticator or recovery code", code),
      el("span", { class: "hint" }, "A recovery code is one-time and will be consumed.")),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ...(hasLocalPasskey && !DEMO_MODE
      ? [["Use passkey", "ghost", usePasskey]]
      : []),
    ["Verify", "primary", async (close) => {
      if (!password.value || !code.value.trim()) {
        toast("Enter your current password and authentication code.", "err");
        return;
      }
      const button = document.activeElement;
      if (button instanceof HTMLButtonElement) button.disabled = true;
      try {
        const verified = await api("/account/reauth/password", { method: "POST", json: {
          purpose, password: password.value, code: code.value.trim(),
        }});
        password.value = "";
        code.value = "";
        await finish(close, verified.action_token);
      } catch (error) {
        toast(error.message, "err");
      } finally {
        if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
      }
    }],
  ]);
}

function openPasswordChange(hasLocalPasskey) {
  verifyAccountAction("change_password", hasLocalPasskey, async (actionToken) => {
    const password = el("input", { type: "password", autocomplete: "new-password", minlength: "10", required: "" });
    const confirm = el("input", { type: "password", autocomplete: "new-password", minlength: "10", required: "" });
    modal("Change password", [
      el("p", { class: "muted" }, "Identity verified. Enter your new password. Every other signed-in device will be signed out."),
      el("div", { class: "field-row" }, el("label", {}, "New password", password)),
      el("div", { class: "field-row" }, el("label", {}, "Confirm new password", confirm)),
    ], [
      ["Cancel", "ghost", (close) => close()],
      ["Change password", "primary", async (close) => {
        if (password.value.length < 10) return toast("Password must be at least 10 characters.", "err");
        if (password.value !== confirm.value) return toast("Passwords do not match.", "err");
        const button = document.activeElement;
        if (button instanceof HTMLButtonElement) button.disabled = true;
        try {
          await api("/account/password", { method: "POST", json: {
            action_token: actionToken, new_password: password.value,
          }});
          password.value = "";
          confirm.value = "";
          close();
          toast("Password changed. Other sessions were signed out.", "ok");
          renderPage();
        } catch (error) {
          toast(error.message, "err");
        } finally {
          if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
        }
      }],
    ]);
  });
}

function recoveryCodesModal(title, codes, message) {
  const value = (codes || []).join("\n");
  modal(title, [
    el("div", { class: "alert danger" }, message || "Save these now. Bonghos cannot show these codes again."),
    el("div", { class: "activation-recovery-codes account-recovery-plaintext" },
      el("pre", { class: "mono" }, value),
      el("button", {
        class: "btn ghost small icon-button", type: "button", title: "Copy recovery codes",
        "aria-label": "Copy recovery codes", onclick: () => copyText(value, "Recovery codes copied"),
      }, solarIcon("copy-linear"))),
  ], [["Done", "primary", (close) => { close(); renderPage(); }]]);
}

function showTOTPEnrollment(setup) {
  const code = el("input", {
    class: "otp-input", inputmode: "numeric", pattern: "[0-9]{6}", maxlength: "6",
    autocomplete: "one-time-code", required: "", value: DEMO_MODE ? "123456" : "",
  });
  const otpWrap = el("div", { class: "otp-wrap account-totp-code", "aria-hidden": "true" },
    ...Array.from({ length: 6 }, () => el("span", {}, "")));
  const qrBox = el("div", { class: "qr-box hidden" });
  const svgOK = typeof setup.qr_svg === "string" &&
    setup.qr_svg.startsWith("<svg ") && !/<script|onload=|xlink:href/i.test(setup.qr_svg);
  if (svgOK) {
    qrBox.innerHTML = setup.qr_svg;
    qrBox.classList.remove("hidden");
  }
  const secretRow = el("div", { class: "activation-secret-row account-totp-secret" },
    el("code", { class: "activation-secret mono" }, setup.secret),
    el("button", {
      class: "btn ghost small icon-button", type: "button", title: "Copy secret key",
      "aria-label": "Copy secret key", onclick: () => copyText(setup.secret, "Secret key copied"),
    }, solarIcon("copy-linear")));
  modal("Replace authenticator", [
    el("p", { class: "muted" }, "Scan the new QR code, then enter its six-digit code. Your current authenticator remains active until this succeeds."),
    qrBox,
    el("details", { class: "activation-manual", open: svgOK ? null : "" },
      el("summary", {}, "Can’t scan the QR code?"), secretRow),
    el("label", { class: "otp-label account-totp-label" }, "New authenticator code", otpWrap, code),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Replace authenticator", "primary", async (close) => {
      if (!/^\d{6}$/.test(code.value)) {
        otpWrap.classList.add("error");
        return toast("Enter the six-digit code from the new authenticator.", "err");
      }
      const button = document.activeElement;
      if (button instanceof HTMLButtonElement) button.disabled = true;
      try {
        const result = await api("/account/totp/finish", { method: "POST", json: {
          setup_token: setup.setup_token, code: code.value,
        }});
        close();
        recoveryCodesModal("New recovery codes", result.recovery_codes,
          "Your authenticator was replaced and every previous recovery code was revoked. Save this new set now.");
      } catch (error) {
        otpWrap.classList.add("error");
        toast(error.message, "err");
      } finally {
        if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
      }
    }],
  ]);
  installOTPControl(code, otpWrap);
  syncOTPCells(code, otpWrap);
}

function openTOTPChange(hasLocalPasskey) {
  verifyAccountAction("change_totp", hasLocalPasskey, async (actionToken) => {
    const setup = await api("/account/totp/begin", { method: "POST", json: { action_token: actionToken } });
    showTOTPEnrollment(setup);
  });
}

function replaceRecoveryCodes(hasLocalPasskey, hasExistingCodes) {
  const title = hasExistingCodes ? "Replace recovery codes" : "Generate recovery codes";
  const message = hasExistingCodes
    ? "Every current recovery code will be revoked. Save the new set when it appears."
    : "Generate a set of one-time recovery codes and save it somewhere safe.";
  verifyAccountAction("regenerate_recovery_codes", hasLocalPasskey, async (actionToken) => {
    confirmModal(title, message, hasExistingCodes ? "Replace codes" : "Generate codes", async () => {
      try {
        const result = await api("/account/recovery-codes/regenerate", { method: "POST", json: { action_token: actionToken } });
        recoveryCodesModal("New recovery codes", result.recovery_codes);
      } catch (error) {
        toast(error.message, "err");
      }
    }, false);
  });
}

function accountCredentialsCard(hasLocalPasskey) {
  const row = (icon, title, note, actionLabel, action) => el("div", { class: "account-security-row" },
    el("span", { class: "account-security-icon", "aria-hidden": "true" }, solarIcon(icon)),
    el("div", { class: "account-security-copy" }, el("strong", {}, title), el("span", { class: "muted" }, note)),
    el("button", { class: "btn ghost small", onclick: action }, actionLabel));
  return el("div", { class: "card account-security-list" },
    row("lock-keyhole-linear", "Password", "Changing it signs out every other device.", "Change password", () => openPasswordChange(hasLocalPasskey)),
    row("shield-keyhole-linear", "Authenticator (TOTP)", "Replace it only after the new authenticator code is confirmed.", "Change authenticator", () => openTOTPChange(hasLocalPasskey)));
}

function recoveryCodeCard(items, hasLocalPasskey) {
  const available = items.filter((item) => !item.used_at).length;
  const used = items.length - available;
  const created = items.length ? items[0].created_at : null;
  return el("div", { class: "card recovery-code-card" },
    el("span", { class: "account-security-icon recovery-code-icon", "aria-hidden": "true" },
      solarIcon("shield-keyhole-linear")),
    el("div", { class: "recovery-code-summary" },
      el("strong", {}, items.length ? `${available} of ${items.length} available` : "No recovery codes"),
      el("span", { class: "muted" }, created
        ? `Generated ${fmtTime(created)}${used ? ` · ${used} used` : ""}`
        : "Generate a set before you need account recovery.")),
    el("button", {
      class: "btn ghost small",
      onclick: () => replaceRecoveryCodes(hasLocalPasskey, items.length > 0),
    }, solarIcon("restart-linear"), items.length ? "Replace codes" : "Generate codes"));
}

function openPasskeyEnrollment() {
  if (DEMO_MODE) {
    toast("Passkey enrollment requires a real Bonghos session over HTTPS or localhost.");
    return;
  }
  if (!passkeysSupported()) {
    toast("Passkeys require a supported browser over HTTPS or localhost.", "err");
    return;
  }
  const name = el("input", { maxlength: "80", placeholder: "Laptop, phone, or security key", autocomplete: "off" });
  const password = el("input", { type: "password", autocomplete: "current-password", required: "" });
  const code = el("input", {
    inputmode: "numeric", maxlength: "32", autocomplete: "one-time-code",
    placeholder: "6-digit code or recovery code", required: "",
  });
  modal("Add passkey", [
    el("div", { class: "passkey-enrollment-note" }, solarIcon("key-linear"),
      el("p", {}, "Your browser will let you choose this device, another device, or a USB/NFC security key.")),
    el("div", { class: "field-row" }, el("label", {}, "Passkey name (optional)", name)),
    el("div", { class: "field-row" }, el("label", {}, "Current password", password)),
    el("div", { class: "field-row" }, el("label", {}, "Authenticator or recovery code", code),
      el("span", { class: "hint" }, "Confirm your identity before adding a new sign-in method.")),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Add passkey", "primary", async (close) => {
      if (!password.value || !code.value.trim()) {
        toast("Enter your current password and authentication code.", "err");
        return;
      }
      const button = document.activeElement;
      if (button instanceof HTMLButtonElement) button.disabled = true;
      try {
        const started = await api("/passkeys/register/begin", { method: "POST", json: {
          name: name.value.trim(), password: password.value, code: code.value.trim(),
        }});
        password.value = "";
        code.value = "";
        const credential = await navigator.credentials.create({
          publicKey: passkeyCreationOptions(started.options.publicKey),
        });
        await api(`/passkeys/register/finish?flow=${encodeURIComponent(started.flow)}`, {
          method: "POST",
          json: passkeyCredentialJSON(credential),
        });
        close();
        toast("Passkey added.", "ok");
        renderPage();
      } catch (error) {
        toast(passkeyError(error, "Passkey enrollment failed."), "err");
      } finally {
        if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
      }
    }],
  ]);
}

function renamePasskey(passkey) {
  const currentName = passkey.name || "Passkey";
  const name = el("input", {
    value: currentName, maxlength: "80", required: "", autocomplete: "off",
    placeholder: "Passkey name",
  });
  modal("Rename passkey", [
    el("div", { class: "field-row" }, el("label", {}, "Passkey name", name),
      el("span", { class: "hint" }, "This changes only the label shown in Bonghos.")),
  ], [
    ["Cancel", "ghost", (close) => close()],
    ["Rename", "primary", async (close) => {
      const nextName = name.value.trim();
      if (!nextName) {
        toast("Passkey name is required.", "err");
        name.focus();
        return;
      }
      if (nextName === currentName) {
        close();
        return;
      }
      const button = document.activeElement;
      if (button instanceof HTMLButtonElement) button.disabled = true;
      try {
        await api(`/passkeys/${passkey.id}`, { method: "PATCH", json: { name: nextName } });
        close();
        toast("Passkey renamed.", "ok");
        renderPage();
      } catch (error) {
        toast(error.message, "err");
      } finally {
        if (button instanceof HTMLButtonElement && button.isConnected) button.disabled = false;
      }
    }],
  ]);
}

function removePasskey(passkey, hasLocalPasskey) {
  const name = passkey.name || "Passkey";
  verifyAccountAction("remove_passkey", hasLocalPasskey, async (actionToken) => {
    confirmModal("Remove passkey", `Remove “${name}”? Devices using it will no longer be able to sign in.`, "Remove", async () => {
      try {
        await api(`/passkeys/${passkey.id}`, { method: "DELETE", json: { action_token: actionToken } });
        toast("Passkey removed.", "ok");
        renderPage();
      } catch (error) { toast(error.message, "err"); }
    });
  });
}

function passkeyRowActions(passkey, hasLocalPasskey) {
  const actions = [
    { label: "Rename", icon: "pen-new-square-linear", run: () => renamePasskey(passkey) },
    { label: "Remove", icon: "trash-bin-trash-linear", danger: true, run: () => removePasskey(passkey, hasLocalPasskey) },
  ];
  const desktop = el("div", { class: "row-actions desktop-row-actions" },
    ...actions.map((action) => el("button", {
      class: "btn " + (action.danger ? "danger" : "ghost") + " small",
      type: "button", onclick: action.run,
    }, action.label)));
  const mobile = overflowActionsMenu(`Actions for ${passkey.name || "passkey"}`,
    actions.map((action) => el("button", {
      class: "action-menu-item" + (action.danger ? " danger" : ""),
      type: "button", role: "menuitem", onclick: action.run,
    }, solarIcon(action.icon), action.label)), "mobile-row-actions");
  return el("div", { class: "responsive-row-actions passkey-row-actions" }, desktop, mobile);
}

function securitySectionHead(title, description, action = null) {
  return el("div", { class: "security-section-head" },
    el("div", {}, el("h2", {}, title), description ? el("p", { class: "muted" }, description) : null),
    action);
}

function passkeyCard(passkeys, hasLocalPasskey) {
  const supported = DEMO_MODE || passkeysSupported();
  const currentRP = location.hostname.toLowerCase();
  const addButton = el("button", {
    class: "btn primary",
    disabled: supported ? null : "",
    title: supported ? "Add a passkey" : "Passkeys require HTTPS or localhost",
    onclick: openPasskeyEnrollment,
  }, "Add passkey");
  const content = passkeys.length
    ? el("div", { class: "card passkey-list" }, passkeys.map((passkey) => {
      const passkeyRP = String(passkey.rp_id || "").toLowerCase();
      const isCurrentRP = passkeyRP === currentRP;
      const kind = passkey.backed_up
        ? "Synced passkey"
        : passkey.backup_eligible ? "Sync available" : "Device or security key";
      return el("div", { class: "passkey-row" },
        el("span", { class: "passkey-row-icon", "aria-hidden": "true" }, solarIcon("key-linear")),
        el("div", { class: "passkey-row-copy" },
          el("div", { class: "passkey-row-title" },
            el("strong", {}, passkey.name || "Passkey"),
            el("span", { class: "tag passkey-kind" }, kind)),
          el("div", { class: "passkey-row-details muted" },
            el("span", { class: "passkey-row-origin" },
              isCurrentRP ? "This panel address" : `Added on ${passkey.rp_id || "another address"}`),
            !isCurrentRP ? el("span", { class: "tag passkey-other-origin" }, "Different address") : null,
            el("span", {}, `Added ${fmtTime(passkey.created_at)}`),
            el("span", {}, passkey.last_used_at ? `Last used ${fmtTime(passkey.last_used_at)}` : "Not used yet"))),
        passkeyRowActions(passkey, hasLocalPasskey));
    }))
    : el("div", { class: "card passkey-empty" },
      el("strong", {}, "No passkeys added"),
      el("p", { class: "muted" }, "Add one to sign in without entering your username, password, or TOTP code."));
  const availability = supported
    ? el("p", { class: "passkey-origin-note muted" }, solarIcon("lock-keyhole-linear"),
      el("span", {}, "Passkeys are tied to ", el("strong", {}, location.hostname), ". If you use Bonghos at another address, add a passkey there too."))
    : el("div", { class: "alert danger passkey-support-warning" },
      "Passkeys are unavailable here. Open Bonghos over HTTPS or localhost in a WebAuthn-compatible browser.");
  return [
    securitySectionHead("Passkeys", "Use this device, another device, or a FIDO2 security key.", addButton),
    availability,
    content,
  ];
}

async function pageSecurity(main) {
  const canViewActivity = can("activity.view");
  const canViewHost = can("host.view");
  const [activity, host, passkeys, recoveryCodes, turnstile] = await Promise.all([
    canViewActivity ? api("/activity").catch(() => []) : Promise.resolve([]),
    canViewHost ? api("/host").catch(() => null) : Promise.resolve(null),
    api("/passkeys").catch(() => []),
    api("/account/recovery-codes").catch(() => []),
    S.me?.role === "owner" ? api("/security/turnstile").catch(() => null) : Promise.resolve(null),
  ]);
  const hasLocalPasskey = accountHasLocalPasskey(passkeys || []);
  const securityActivity = (activity || []).filter(isSecurityActivity).slice(0, 8);
  const secureConnection = location.protocol === "https:";
  const bindAddress = String(host?.bind_address || "").trim();
  const localListener = /^(localhost|127(?:\.|$)|::1$)/i.test(bindAddress);
  const listenerAddress = host ? `${bindAddress || "0.0.0.0"}:${host.port}` : "";
  const sessionHours = Number(host?.session_hours);
  const postureItem = (icon, title, status, note, warning = false) => el("div", { class: "security-posture-item" },
    el("span", { class: "security-posture-icon", "aria-hidden": "true" }, solarIcon(icon)),
    el("div", { class: "security-posture-copy" }, el("strong", {}, title), el("span", { class: "muted" }, note)),
    el("span", { class: "tag security-posture-state" + (warning ? " is-warning" : "") }, status));
  const protectionItems = [
    postureItem("shield-check-linear", "Password and TOTP", "Required", "Remain available as a fallback sign-in method after you add a passkey."),
    postureItem("lock-keyhole-linear", "Password storage", "Argon2id", "Passwords are hashed and never stored as plaintext."),
    postureItem("shield-keyhole-linear", "Sign-in throttling", "Enabled", "Repeated failed sign-ins are temporarily rate limited."),
    postureItem("shield-keyhole-linear", "Session protection",
      Number.isFinite(sessionHours) ? `${sessionHours} hours` : "Protected",
      Number.isFinite(sessionHours)
        ? `Sessions expire after ${sessionHours} hours. HttpOnly, SameSite cookies and CSRF verification protect panel actions.`
        : "HttpOnly, SameSite cookies and CSRF verification protect panel actions."),
    postureItem(secureConnection ? "lock-keyhole-linear" : "danger-triangle-linear", "Current connection",
      secureConnection ? "HTTPS" : "HTTP",
      secureConnection ? "Traffic between this browser and the panel is encrypted." : "Traffic is not encrypted. Use HTTPS before exposing the panel.",
      !secureConnection),
  ];
  if (host) {
    protectionItems.push(postureItem("server-square-linear", "Panel listener", localListener ? "Local only" : "Network",
      `${listenerAddress}. ${localListener ? "A tunnel or reverse proxy may still provide external access." : "Restrict access with a firewall or trusted reverse proxy."}`,
      !localListener));
  }
  const protectionSection = el("section", {
    id: "account-protection-details",
    class: "security-protection-details hidden",
  },
    securitySectionHead("Account protection", "Security controls currently protecting your account and this panel."),
    el("div", { class: "card security-posture-card" },
      el("div", { class: "security-posture-list" }, protectionItems)));
  const protectionToggle = el("button", {
    class: "btn ghost small security-yapping-toggle",
    type: "button",
    "aria-controls": "account-protection-details",
    "aria-expanded": "false",
    onclick: (event) => {
      const button = event.currentTarget;
      const showing = protectionSection.classList.toggle("hidden") === false;
      button.setAttribute("aria-expanded", String(showing));
      button.textContent = showing ? "Hide yapping" : "Show yapping";
    },
  }, "Show yapping");
  main.innerHTML = "";
  main.append(
    pageHeader("Security", "Manage your sign-in methods and review the protections around your account."),
    securitySectionHead("Sign-in credentials", "Sensitive changes require your current password plus TOTP or a recovery code, or a user-verified passkey."),
    accountCredentialsCard(hasLocalPasskey),
    ...passkeyCard(passkeys || [], hasLocalPasskey),
    securitySectionHead("Recovery codes", "One-time fallback codes. Their plaintext is shown only when generated and cannot be viewed again."),
    recoveryCodeCard(recoveryCodes || [], hasLocalPasskey),
    ...(turnstile ? [turnstileSettingsSection(turnstile)] : []),
    ...(canViewActivity ? [securitySectionHead("Recent security activity", "Latest sign-in and account-management events.",
      el("button", { class: "btn ghost small", onclick: () => navigate("activity") }, solarIcon("history-linear"), "View all")),
    securityActivity.length
      ? el("div", { class: "table-wrap security-activity-table" },
        el("table", {},
          el("thead", {}, el("tr", {}, el("th", {}, "When"), el("th", {}, "Actor"), el("th", {}, "Event"), el("th", {}, "Target"), el("th", {}, "Address"))),
          el("tbody", {}, securityActivity.map((entry) => el("tr", {},
            el("td", {}, fmtTime(entry.at)),
            el("td", {}, entry.username || "System"),
            el("td", {}, String(entry.action || "").replace(/_/g, " ")),
            el("td", {}, entry.target || "Not available"),
            el("td", { class: "mono" }, entry.remote_addr || "Not available"))))))
      : el("div", { class: "card" }, el("p", { class: "muted" }, "No recent sign-in or account-management activity."))] : []),
    el("div", { class: "security-yapping-toggle-row" }, protectionToggle),
    protectionSection);
}

const BOT_EVENT_FIELDS = [
  ["notify_server_started", "Server started", "After Minecraft is fully ready"],
  ["notify_server_stopped", "Server stopping", "As soon as shutdown begins"],
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
        const testButton = button.closest(".bot-card")?.querySelector(".bot-test-button");
        if (testButton) {
          testButton.disabled = !next;
          testButton.title = next ? "Send a test notification" : "Turn on the bot to send a test notification";
        }
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

function botDestinations(bot) {
  let configured = [];
  if (Array.isArray(bot.destinations) && bot.destinations.length) {
    configured = bot.destinations.filter((destination) => destination && destination.id);
  } else if (bot.destination_id) {
    configured = [{ id: bot.destination_id, name: "", type: "" }];
  }
  const discovered = Array.isArray(bot?.discovered_destinations)
    ? bot.discovered_destinations.filter((destination) => destination && destination.id)
    : [];
  if (!discovered.length) return configured;
  return configured.map((destination) => {
    const match = bot.provider === "discord"
      ? discovered.find((container) => String(container.guild_id || container.id) === String(destination.guild_id || ""))
      : discovered.find((container) => String(container.id) === String(destination.id));
    return match ? { ...match, ...destination, discovered_at: destination.discovered_at || match.discovered_at } : destination;
  });
}

function botKnownContainers(bot) {
  const configured = botDestinations(bot);
  const discovered = Array.isArray(bot?.discovered_destinations)
    ? bot.discovered_destinations.filter((destination) => destination && destination.id)
    : [];
  if (bot?.provider === "telegram") {
    const byID = new Map(discovered.map((group) => [String(group.id), { ...group, configured: null }]));
    for (const destination of configured) {
      const key = String(destination.id);
      const current = byID.get(key) || {};
      byID.set(key, { ...current, ...destination, configured: destination });
    }
    return [...byID.values()];
  }
  const byGuild = new Map();
  for (const guild of discovered) {
    const guildID = String(guild.guild_id || guild.id);
    byGuild.set(guildID, [{ ...guild, guild_id: guildID, configured: null }]);
  }
  for (const destination of configured) {
    const guildID = String(destination.guild_id || "");
    const existing = byGuild.get(guildID);
    const base = existing?.[0] || {};
    const row = { ...base, ...destination, guild_id: guildID, configured: destination };
    if (existing?.[0]?.configured) existing.push(row);
    else byGuild.set(guildID, [row]);
  }
  return [...byGuild.values()].flat();
}

function botGroupInitials(destination) {
  const words = String(destination?.name || "").trim().split(/\s+/).filter(Boolean);
  if (!words.length) return "#";
  return (words.length > 1 ? words[0][0] + words[1][0] : words[0].slice(0, 2)).toUpperCase();
}

function botGroupAvatar(destination, bot = null) {
  const inline = String(destination?.photo_data_url || "");
  const local = String(destination?.photo_url || "");
  let source = /^data:image\/(?:png|jpeg|webp);base64,/i.test(inline) ? inline : "";
  if (!source && local.startsWith("/")) source = local;
  if (!source && bot?.id) {
    source = `/api/bots/${encodeURIComponent(bot.id)}/telegram/destinations/${encodeURIComponent(destination.id)}/photo`;
  }
  const avatar = el("span", { class: "bot-group-avatar", "aria-hidden": "true" }, botGroupInitials(destination));
  if (source) {
    avatar.append(el("img", {
      src: source, alt: "", loading: "lazy",
      onerror: (event) => event.currentTarget.remove(),
    }));
  }
  return avatar;
}

function discordServerAvatar(destination) {
  const guildID = String(destination?.guild_id || "");
  const icon = String(destination?.guild_icon || "");
  const source = /^\d{10,25}$/.test(guildID) && /^[A-Za-z0-9_]+$/.test(icon)
    ? `https://cdn.discordapp.com/icons/${guildID}/${icon}.png?size=64`
    : "";
  const avatar = el("span", { class: "bot-group-avatar", "aria-hidden": "true" }, botGroupInitials({ name: destination?.guild_name }));
  if (source) avatar.append(el("img", { src: source, alt: "", loading: "lazy", onerror: (event) => event.currentTarget.remove() }));
  return avatar;
}

async function botInviteURL(bot) {
  const result = await api(`/bots/${bot.id}/invite`);
  const invite = new URL(String(result?.url || ""));
  const allowed = bot.provider === "telegram"
    ? invite.protocol === "https:" && invite.hostname === "t.me"
    : invite.protocol === "https:" && invite.hostname === "discord.com";
  if (!allowed) throw new Error("The provider returned an invalid invite link");
  if (bot.provider === "telegram") {
    return `${invite.origin}${invite.pathname}?startgroup`;
  }
  return invite.href;
}

async function inviteBot(bot) {
  const popup = window.open("about:blank", "_blank");
  if (popup) popup.opener = null;
  try {
    const inviteURL = await botInviteURL(bot);
    if (popup) popup.location.replace(inviteURL);
    else if (!window.open(inviteURL, "_blank", "noopener,noreferrer")) {
      throw new Error("Allow pop-ups to open the bot invite");
    }
  } catch (error) {
    if (popup) popup.close();
    toast(error.message, "err");
  }
}

async function copyBotInvite(bot, button) {
  button.disabled = true;
  try {
    await copyText(await botInviteURL(bot), "Bot invite link copied");
  } catch (error) {
    toast(error.message, "err");
  } finally {
    button.disabled = false;
  }
}

function botInviteControls(bot, style = "primary") {
  const buttonClass = `btn ${style} small`;
  return el("span", { class: `bot-invite-group is-${style}` },
    el("button", { class: buttonClass, onclick: () => inviteBot(bot) }, "Invite Bot"),
    el("button", {
      class: buttonClass + " bot-invite-copy",
      type: "button",
      title: "Copy invite link",
      "aria-label": "Copy invite link",
      onclick: (event) => copyBotInvite(bot, event.currentTarget),
    }, solarIcon("copy-linear")));
}

function botCard(bot) {
  const destinations = botDestinations(bot);
  const destinationLabel = bot.provider === "telegram"
    ? `${destinations.length} of 3 groups`
    : `${destinations.length} of 3 channels`;
  return el("article", { class: "card bot-card" + (bot.enabled ? "" : " is-disabled") },
    el("div", { class: "bot-card-head" },
      botProviderMark(bot.provider),
      el("div", { class: "bot-card-identity" },
        el("strong", {}, bot.name),
        el("span", { class: "muted" }, botProviderName(bot.provider))),
      botPowerButton(bot)),
    el("div", { class: "bot-destination" },
      el("span", { class: "bot-destination-label" }, destinationLabel),
      el("div", { class: "bot-destination-values" }, destinations.length ? destinations.map((destination) =>
        el("span", { class: "bot-destination-value has-avatar" },
          bot.provider === "telegram" ? botGroupAvatar(destination, bot) : discordServerAvatar(destination),
          el("span", { class: "bot-destination-copy" },
            el("span", { class: "bot-destination-name" },
              bot.provider === "discord"
                ? el("strong", {}, destination.guild_name || "Discord server")
                : (destination.name ? el("strong", {}, destination.name) : null),
              bot.provider === "discord"
                ? el("small", {}, `#${String(destination.name || destination.id).replace(/^#/, "")}`)
                : (destination.thread_id ? el("small", {}, destination.thread_name || `Channel ${destination.thread_id}`) : null)),
            el("span", { class: "bot-destination-added" }, `Added on ${fmtTimeToMinute(destination.discovered_at || bot.created_at)}`)))) :
        el("span", { class: "muted" }, "Not configured"))),
    el("div", { class: "bot-events-title" }, "Notify when"),
    el("div", { class: "bot-event-grid" },
      ...BOT_EVENT_FIELDS.map(([field, label, note]) => botEventToggle(bot, field, label, note))),
    el("div", { class: "bot-card-actions" },
      el("button", { class: "btn ghost small bot-card-primary-action", onclick: () => botEditor(bot) }, "Edit"),
      el("button", {
        class: "btn ghost small bot-card-primary-action bot-test-button",
        disabled: bot.enabled ? null : "",
        title: bot.enabled ? "Send a test notification" : "Turn on the bot to send a test notification",
        onclick: async (event) => {
        const button = event.currentTarget;
        button.disabled = true;
        try {
          await api(`/bots/${bot.id}/test`, { method: "POST", json: {} });
          toast(`Test sent through ${bot.name}`, "ok");
        } catch (error) { toast(error.message, "err"); }
        finally { button.disabled = !bot.enabled; }
      } }, "Send test"),
      botInviteControls(bot),
      el("div", { class: "spacer" }),
      DEMO_DEBUG_BOTS ? null : el("button", { class: "btn danger small bot-card-primary-action", onclick: () => removeBot(bot) }, "Remove")));
}

function botEditor(existing = null, currentBots = []) {
  const name = el("input", { value: existing?.name || "", maxlength: "80", autocomplete: "off", placeholder: "Bot name" });
  const providerCounts = (currentBots || []).reduce((counts, bot) => {
    counts[bot.provider] = (counts[bot.provider] || 0) + 1;
    return counts;
  }, {});
  const availableProviders = existing
    ? [{ value: existing.provider, label: botProviderName(existing.provider) }]
    : [{ value: "telegram", label: "Telegram" }, { value: "discord", label: "Discord" }]
      .filter((option) => (providerCounts[option.value] || 0) < 2);
  if (!availableProviders.length) {
    toast("Up to two Telegram bots and two Discord bots can be connected", "err");
    return;
  }
  const provider = el("select", {}, ...availableProviders.map((option) =>
    el("option", { value: option.value }, option.label)));
  provider.value = existing?.provider || availableProviders[0].value;
  const token = el("input", {
    type: "password", autocomplete: "new-password", spellcheck: "false",
    placeholder: existing ? "Leave blank to keep the current token" : "Paste the bot token",
  });
  const tokenField = el("div", { class: "field-row" },
    el("label", {}, existing ? "New bot token (optional)" : "Bot token", token));
  const dnsServer = el("input", {
    value: existing?.dns_server || "", autocomplete: "off", spellcheck: "false",
    inputmode: "text", placeholder: "System DNS (for example, 1.1.1.1)",
  });
  const dnsServerField = el("div", { class: "field-row" },
    el("label", {}, "DNS server (optional)", dnsServer));
  const advancedFields = el("div", {
    id: "bot-advanced-fields", class: "bot-advanced-fields", "aria-hidden": "true",
  }, el("div", { class: "bot-advanced-fields-inner" }, tokenField, dnsServerField));
  advancedFields.inert = true;
  const advancedOpenToggle = el("button", {
    class: "bot-more-toggle", type: "button", "aria-expanded": "false", "aria-controls": "bot-advanced-fields",
  }, el("span", {}, "Show more"), solarIcon("alt-arrow-down-linear", "bot-more-icon"));
  const advancedCloseToggle = el("button", {
    class: "bot-more-toggle hidden", type: "button", "aria-expanded": "true", "aria-controls": "bot-advanced-fields",
  }, el("span", {}, "Show less"), solarIcon("alt-arrow-up-linear", "bot-more-icon"));
  const advancedDrawer = el("div", { class: "bot-advanced-drawer" },
    advancedOpenToggle, advancedFields, advancedCloseToggle);
  let modalResizeAnimation = null;
  const applyAdvancedState = (expanded) => {
    advancedOpenToggle.classList.toggle("hidden", expanded);
    advancedCloseToggle.classList.toggle("hidden", !expanded);
    advancedFields.classList.toggle("is-expanded", expanded);
    advancedFields.setAttribute("aria-hidden", String(!expanded));
    advancedFields.inert = !expanded;
  };
  const setAdvancedExpanded = (expanded) => {
    const dialog = advancedDrawer.closest(".modal");
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (!dialog || reducedMotion || typeof dialog.animate !== "function") {
      applyAdvancedState(expanded);
      return;
    }
    modalResizeAnimation?.cancel();
    const startHeight = dialog.getBoundingClientRect().height;
    advancedFields.classList.add("is-measuring");
    applyAdvancedState(expanded);
    const endHeight = dialog.getBoundingClientRect().height;
    applyAdvancedState(!expanded);
    void dialog.offsetHeight;
    advancedFields.classList.remove("is-measuring");
    applyAdvancedState(expanded);
    modalResizeAnimation = dialog.animate([
      { height: `${startHeight}px` },
      { height: `${endHeight}px` },
    ], { duration: 240, easing: "cubic-bezier(0.2, 0, 0, 1)" });
  };
  advancedOpenToggle.addEventListener("click", () => setAdvancedExpanded(true));
  advancedCloseToggle.addEventListener("click", () => setAdvancedExpanded(false));
  let configuredDestinations = existing ? botDestinations(existing) : [];
  let knownContainers = existing ? botKnownContainers(existing) : [];
  const groupList = el("div", { class: "bot-group-list" });
  const groupHeading = el("span");
  const groupInstruction = el("p", { class: "hint" });
  const findGroupsButton = el("button", { class: "btn ghost small", type: "button" }, "Refresh");
  let refreshTimer = null;
  let refreshInFlight = false;
  let submitted = false;

  const renderGroupChoices = () => {
    const telegram = provider.value === "telegram";
    groupList.innerHTML = "";
    groupHeading.textContent = `Destinations (${configuredDestinations.length}/3)`;
    groupInstruction.textContent = existing
      ? (telegram
        ? "Run /bonghos here in the destination topic."
        : "Run /bonghos here in the destination channel.")
      : (telegram
        ? "Save the bot, invite it, then run /bonghos here in the destination topic."
        : "Save the bot, invite it, then run /bonghos here in the destination channel.");
    if (!knownContainers.length) {
      groupList.append(el("div", { class: "bot-group-empty muted" },
        el("span", { class: "bot-group-empty-copy" }, existing ? `Bot has not joined any ${telegram ? "groups" : "servers"} yet.` : "Destinations are detected after the bot is saved and invited."),
        existing ? botInviteControls(existing, "ghost") : null));
      return;
    }
    const groups = [...knownContainers].sort((left, right) =>
      String(telegram ? left.name : left.guild_name).localeCompare(String(telegram ? right.name : right.guild_name), undefined, { sensitivity: "base" }));
    for (const group of groups) {
      const configured = group.configured;
      const topicLabel = !configured
        ? "Not configured"
        : (telegram
          ? (Number(configured.thread_id) > 1 ? (configured.thread_name || "Selected topic") : "General")
          : `#${String(configured.name || configured.id).replace(/^#/, "")}`);
      const row = el("div", { class: "bot-group-choice bot-group-connected" + (configured ? " is-selected" : " is-unconfigured") },
        telegram ? botGroupAvatar(group, existing) : discordServerAvatar(group),
        el("span", { class: "bot-group-copy" },
          el("strong", {}, telegram ? (group.name || group.id) : (group.guild_name || "Discord server")),
          el("small", {}, topicLabel)),
        el("small", { class: "bot-group-added" }, `Added on ${fmtTimeToMinute(group.discovered_at || existing?.created_at)}`));
      groupList.append(row);
    }
  };

  const refreshDestinations = async (announce = false) => {
    if (!existing) {
      if (announce) toast("Save the bot before connecting destinations", "err");
      return;
    }
    if (refreshInFlight) return;
    refreshInFlight = true;
    if (announce) findGroupsButton.disabled = true;
    try {
      const bots = await api("/bots");
      const refreshed = (Array.isArray(bots) ? bots : []).find((bot) => Number(bot.id) === Number(existing.id));
      if (refreshed) {
        Object.assign(existing, refreshed);
        configuredDestinations = botDestinations(refreshed);
        knownContainers = botKnownContainers(refreshed);
      }
      renderGroupChoices();
      if (announce) {
        const telegram = provider.value === "telegram";
        toast(knownContainers.length ? `${telegram ? "Groups" : "Servers"} refreshed` : `No ${telegram ? "groups" : "servers"} detected yet`, knownContainers.length ? "ok" : "");
      }
    } catch (error) {
      if (announce) toast(error.message, "err");
    } finally {
      refreshInFlight = false;
      if (announce) findGroupsButton.disabled = false;
    }
  };
  findGroupsButton.addEventListener("click", () => refreshDestinations(true));

  const telegramDestination = el("div", { class: "field-row bot-telegram-destinations" },
    el("div", { class: "bot-group-heading" },
      groupHeading,
      el("div", { class: "bot-group-heading-actions" }, findGroupsButton)),
    groupInstruction,
    groupList);
  const enabled = el("input", { type: "checkbox" });
  enabled.checked = existing ? !!existing.enabled : true;
  const eventInputs = {};
  const renderDestination = () => {
    telegramDestination.hidden = false;
    renderGroupChoices();
  };
  provider.disabled = !!existing;
  if (!existing) provider.addEventListener("change", renderDestination);
  renderGroupChoices();
  renderDestination();

  const notificationRows = existing ? [] : BOT_EVENT_FIELDS.map(([field, label, note]) => {
    const input = el("input", { type: "checkbox" });
    input.checked = existing ? !!existing[field] : true;
    eventInputs[field] = input;
    return el("label", { class: "bot-modal-event check-row" }, input,
      el("span", {}, el("strong", {}, label), el("small", {}, note)));
  });

  modal(existing ? "Edit bot" : "Add bot", [
    el("div", { class: "field-row" }, el("label", {}, "Name", name)),
    el("div", { class: "field-row" }, el("label", {}, "Provider", provider)),
    existing ? advancedDrawer : tokenField,
    existing ? null : dnsServerField,
    telegramDestination,
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
      const body = { name: name.value.trim() };
      if (!existing) {
        body.provider = provider.value;
        body.enabled = enabled.checked;
        BOT_EVENT_FIELDS.forEach(([field]) => { body[field] = eventInputs[field].checked; });
      }
      if (token.value.trim()) body.token = token.value.trim();
      body.dns_server = dnsServer.value.trim();
      try {
        if (existing) await api(`/bots/${existing.id}`, { method: "PATCH", json: body });
        else await api("/bots", { method: "POST", json: body });
        submitted = true;
        close();
        toast(existing ? "Bot updated" : "Bot added. Run /bonghos here to connect a destination.", "ok");
        await renderPage();
      } catch (error) { toast(error.message, "err"); }
    }],
  ], () => {
    if (refreshTimer) clearInterval(refreshTimer);
    if (existing && !submitted) void renderPage();
  });
  if (existing) {
    void refreshDestinations();
    refreshTimer = setInterval(() => void refreshDestinations(), 2000);
  }
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

function systemdServiceState(rawState, systemdAvailable) {
  if (!systemdAvailable) return { label: "Not managed", tone: "" };
  const state = String(rawState || "").trim().toLowerCase();
  if (/\bactivating\b|\bstarting\b/.test(state)) return { label: "Starting", tone: "starting" };
  if (/\bdeactivating\b|\bstopping\b/.test(state)) return { label: "Stopping", tone: "stopping" };
  if (/\bfailed\b|\bcrashed\b/.test(state)) return { label: "Failed", tone: "crashed" };
  if (/\binactive\b|\bdead\b|\bstopped\b/.test(state)) return { label: "Stopped", tone: "" };
  if (/\bactive\b|\brunning\b/.test(state)) return { label: "Active", tone: "running" };
  return { label: "Unknown", tone: "" };
}

function settingsStatusPill(label, tone = "") {
  return el("span", { class: `status-label ${tone}`.trim() },
    el("span", { class: "status-square", "aria-hidden": "true" }), label);
}

function settingsServiceCard(title, unit, description, rawState, systemdAvailable) {
  const state = systemdServiceState(rawState, systemdAvailable);
  return el("div", { class: "settings-service-card" },
    el("div", { class: "settings-service-card-head" },
      el("div", { class: "settings-service-name" },
        el("strong", {}, title),
        el("span", { class: "mono muted" }, unit)),
      settingsStatusPill(state.label, state.tone)),
    el("p", { class: "muted" }, description));
}

function playitState(config) {
  if (!config.enabled) return "Off";
  if (config.management_mode === "external") return "External agent";
  if (config.claim_pending) return "Waiting for approval";
  if (config.agent_online && config.public_address) return "Online";
  if (config.agent_phase === "unavailable") return "Agent unavailable";
  if (config.agent_error) return "Agent error";
  if (config.secret_configured && ["starting", "stopping", "stopped"].includes(config.agent_phase)) {
    return config.agent_phase === "starting" ? "Starting agent" : "Agent stopped";
  }
  if (config.secret_configured && config.agent_online) return "Agent ready";
  if (config.secret_configured) return "Starting agent";
  return "Setup required";
}

function playitSettingsSection(initialConfig) {
  const config = { account_mode: "account", management_mode: "none", local_port: 25565, detections: [], ...initialConfig };
  const section = el("section", { class: "settings-page-section", "aria-labelledby": "playit-settings-title" });
  let polling = false;
  let statusPolling = false;
  let claimAccountMode = config.account_mode;
  let editorRoot = null;
  let editorManagementMode = config.management_mode;
  let saveEditorChanges = null;
  let externalNoticeDismissed = false;

  const replaceConfig = (next = {}, overrides = {}) => {
    ["agent_id", "agent_name", "tunnel_id", "public_address", "claim_url", "notice", "agent_error",
      "agent_version", "tunnel_status", "guest_login_url", "account_status", "managed_state"]
      .forEach((key) => { delete config[key]; });
    Object.assign(config, next, overrides);
  };

  const savePreference = async (updates) => {
    const payload = {
      enabled: updates.enabled ?? config.enabled,
      account_mode: updates.account_mode ?? config.account_mode,
      management_mode: updates.management_mode ?? config.management_mode,
      public_address: updates.public_address ?? config.public_address ?? "",
      local_port: Number(updates.local_port ?? config.local_port ?? 25565),
    };
    const saved = await api("/playit", { method: "PUT", json: payload });
    replaceConfig(saved, updates);
    render();
    return saved;
  };

  const pollClaim = async (automatic = false) => {
    if (polling || !config.claim_pending) return;
    polling = true;
    try {
      while (config.claim_pending && document.body.contains(section)) {
        const result = await api("/playit/claim/poll", { method: "POST", json: {} });
        if (result.state === "complete") {
          replaceConfig(result.config || {}, { claim_pending: false, claim_url: "" });
          toast("Playit.gg agent linked", "ok");
          render();
          return;
        }
        if (result.state === "rejected") {
          config.claim_pending = false;
          toast("Playit.gg setup was not approved", "err");
          render();
          return;
        }
        if (!automatic) {
          toast("Waiting for approval on Playit.gg", "ok");
          return;
        }
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    } catch (error) {
      if (!automatic) toast(error.message, "err");
    } finally {
      polling = false;
    }
  };

  const beginClaim = async (button) => {
    if (button) button.disabled = true;
    try {
      const started = await api("/playit/claim", { method: "POST", json: { account_mode: claimAccountMode } });
      replaceConfig(started);
      render();
      return true;
    } catch (error) {
      toast(error.message, "err");
      if (button) button.disabled = false;
      return false;
    }
  };

  const pollAgent = async () => {
    if (statusPolling || !config.enabled || config.management_mode !== "bonghos" || !config.secret_configured) return;
    statusPolling = true;
    try {
      for (let attempt = 0; attempt < 60 && document.body.contains(section); attempt++) {
        const refreshed = await api("/playit/refresh", { method: "POST", json: {} });
        replaceConfig(refreshed);
        render();
        if (config.agent_online || config.agent_error || !config.enabled || config.management_mode !== "bonghos") return;
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }
    } catch (_) {
      // Manual Refresh remains available if a transient status request fails.
    } finally {
      statusPolling = false;
    }
  };

  const editorNotice = () => {
    const externalDetected = Array.isArray(config.detections)
      ? config.detections.find((item) => item.externally_managed && ["active", "running"].includes(item.state))
      : null;
    if (!config.daemon_available && externalDetected && !externalNoticeDismissed) {
      return el("div", { class: "notice playit-inline-notice playit-dismissible-notice" },
        el("span", { class: "playit-notice-copy" },
          `An external Playit${externalDetected.kind === "docker" ? " Docker" : ""} agent is running. Choose External agent to display its public address, or install playitd for Bonghos-managed tunnels. `,
          el("a", { href: "https://packages.playit.gg/", target: "_blank", rel: "noopener noreferrer" }, "Installation guide")),
        el("button", { class: "playit-notice-dismiss", type: "button", title: "Dismiss", "aria-label": "Dismiss notice", onclick: (event) => {
          externalNoticeDismissed = true;
          event.currentTarget.closest(".playit-dismissible-notice")?.remove();
        } }, solarIcon("close-linear")));
    }
    if (editorManagementMode === "external") return null;
    if (!config.daemon_available) {
      return el("div", { class: "notice playit-inline-notice" },
        el("span", {}, "Bonghos cannot create this tunnel because playitd is not installed on the host. "),
        el("a", { href: "https://packages.playit.gg/", target: "_blank", rel: "noopener noreferrer" }, "Installation guide"));
    }
    if (config.agent_error) return el("div", { class: "notice playit-inline-notice" }, config.agent_error);
    if (config.notice) return el("div", { class: "notice playit-inline-notice" }, config.notice);
    if (config.enabled && config.secret_configured && !config.agent_online) {
      return el("div", { class: "notice playit-inline-notice" }, "The agent is starting. Tunnel controls will unlock when it is ready.");
    }
    return null;
  };

  const renderEditor = () => {
    if (!editorRoot) return;
    saveEditorChanges = null;
    editorRoot.innerHTML = "";
    const notice = editorNotice();
    if (notice) editorRoot.append(notice);
    const managedChoice = editorManagementMode !== "external";
      const runningExternalAgent = Array.isArray(config.detections)
        ? config.detections.find((item) => item.externally_managed && ["active", "running"].includes(item.state))
        : null;
      const methodButtons = el("div", { class: "segmented-choice playit-choice" },
        el("button", { class: "btn" + (managedChoice ? " active" : ""), type: "button", onclick: async () => {
          if (config.management_mode === "bonghos") {
            editorManagementMode = "bonghos";
            renderEditor();
            return;
          }
          try {
            await savePreference({ enabled: config.enabled, management_mode: "bonghos" });
            editorManagementMode = "bonghos";
            render();
            toast("Bonghos-managed Playit agent selected", "ok");
          } catch (error) { toast(error.message, "err"); }
        } }, "Set up with Bonghos"),
        el("button", { class: "btn playit-external-agent-tab" + (!managedChoice ? " active" : ""), type: "button", onclick: () => {
          editorManagementMode = "external";
          renderEditor();
        } }, "External agent", runningExternalAgent
          ? el("span", { class: "playit-external-running", title: "External Playit service running", "aria-label": "External Playit service running" })
          : null));
      editorRoot.append(el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Agent"), el("p", { class: "muted" }, "Choose who runs the Playit agent.")),
        el("div", { class: "playit-form" }, methodButtons)));

      if (managedChoice) {
        const accountButtons = el("div", { class: "segmented-choice playit-choice" },
          el("button", { class: "btn" + (claimAccountMode === "account" ? " active" : ""), type: "button",
            onclick: () => { claimAccountMode = "account"; render(); } }, "Playit account"),
          el("button", { class: "btn" + (claimAccountMode === "guest" ? " active" : ""), type: "button",
            onclick: () => { claimAccountMode = "guest"; render(); } }, "Guest"));
        const controls = el("div", { class: "playit-form" }, accountButtons);
        if (config.claim_pending) {
          controls.append(el("p", { class: "muted" }, "Approve the agent on Playit.gg."),
            el("div", { class: "playit-actions" },
              el("a", { class: "btn primary", href: config.claim_url, target: "_blank", rel: "noopener noreferrer" }, "Open Playit.gg"),
              el("button", { class: "btn", type: "button", onclick: () => pollClaim(false) }, "Check status")));
          queueMicrotask(() => pollClaim(true));
        } else if (!config.secret_configured) {
          controls.append(el("p", { class: "muted" }, claimAccountMode === "guest"
            ? "Quick setup without a permanent account."
            : "Your Playit password never passes through Bonghos."),
            el("button", {
              class: "btn primary", type: "button", onclick: (event) => beginClaim(event.currentTarget),
              ...(config.daemon_available ? {} : { disabled: "disabled", title: "Install playitd first" }),
            }, "Connect Playit.gg"));
        } else {
          const agentName = el("input", {
            value: config.agent_name || "", maxlength: "100", placeholder: "Bonghos", spellcheck: "false",
          });
          const details = el("dl", { class: "kv compact" },
            el("dt", {}, "Account"), el("dd", {}, config.account_status || config.account_mode),
            el("dt", {}, "Agent name"), el("dd", {}, config.agent_name || "Managed in Playit"),
            el("dt", {}, "Agent ID"), el("dd", { class: "mono" }, config.agent_id || "Linked"),
            el("dt", {}, "Status"), el("dd", {}, String(config.agent_phase || "starting").replaceAll("_", " ")),
            config.agent_version ? [el("dt", {}, "Version"), el("dd", { class: "mono" }, config.agent_version)] : null,
            config.tunnel_id ? [el("dt", {}, "Tunnel"), el("dd", {}, config.tunnel_status || "pending")] : null,
            el("dt", {}, "Local port"), el("dd", { class: "mono" }, String(config.local_port || 25565)));
          if (config.guest_login_url) {
            controls.append(el("a", { class: "btn", href: config.guest_login_url, target: "_blank", rel: "noopener noreferrer" }, "Manage guest account"));
          } else if (config.account_mode === "guest") {
            controls.append(el("button", { class: "btn", type: "button", onclick: async (event) => {
              event.currentTarget.disabled = true;
              try {
                const result = await api("/playit/guest-login", { method: "POST", json: {} });
                config.guest_login_url = result.url;
                render();
              } catch (error) { toast(error.message, "err"); event.currentTarget.disabled = false; }
            } }, "Manage guest account"));
          }
          controls.append(details,
            el("div", { class: "playit-agent-name" },
              el("label", {}, "Agent name", agentName),
              el("button", { class: "btn playit-agent-name-save", type: "button", title: "Save agent name", "aria-label": "Save agent name", onclick: async (event) => {
                event.currentTarget.disabled = true;
                try {
                  replaceConfig(await api("/playit/agent", { method: "PUT", json: { name: agentName.value.trim() } }));
                  toast("Playit agent renamed", "ok");
                  render();
                } catch (error) { toast(error.message, "err"); event.currentTarget.disabled = false; }
              } }, solarIcon("diskette-linear"))),
            el("div", { class: "playit-actions playit-tunnel-actions" },
            el("button", { class: "btn primary", type: "button",
              ...(!config.enabled || !config.agent_online ? {
                disabled: "disabled",
                title: config.enabled ? "The managed agent must be ready first" : "Turn on Playit.gg to manage the tunnel",
            } : {}), onclick: async (event) => {
              event.currentTarget.disabled = true;
              try {
                const hadTunnel = !!config.tunnel_id;
                const missingTunnel = hadTunnel && config.tunnel_status === "missing";
                replaceConfig(await api("/playit/tunnel", { method: "POST", json: {} }));
                toast(missingTunnel ? "Playit tunnel recreated" : hadTunnel ? "Playit tunnel updated" : "Playit tunnel created", "ok");
                render();
              } catch (error) { toast(error.message, "err"); event.currentTarget.disabled = false; }
            } }, config.tunnel_status === "missing" ? "Recreate tunnel" : config.tunnel_id ? "Update tunnel" : "Create tunnel"),
            config.enabled && !config.agent_online && config.daemon_available && ["stopped", "error"].includes(config.agent_phase)
              ? el("button", { class: "btn", type: "button", onclick: async (event) => {
                event.currentTarget.disabled = true;
                try {
                  await savePreference({ enabled: true, management_mode: "bonghos" });
                  toast("Playit agent restart requested", "ok");
                } catch (error) { toast(error.message, "err"); event.currentTarget.disabled = false; }
              } }, "Restart agent") : null,
            el("button", { class: "btn", type: "button", onclick: async (event) => {
              event.currentTarget.disabled = true;
              try { replaceConfig(await api("/playit/refresh", { method: "POST", json: {} })); render(); }
              catch (error) { toast(error.message, "err"); event.currentTarget.disabled = false; }
            } }, "Refresh"),
            config.tunnel_id ? el("button", { class: "btn danger", type: "button", onclick: () => {
              editorRoot = null;
              confirmModal("Delete Playit tunnel",
                "This removes the public Playit address. It does not delete or stop the game server.",
                "Delete tunnel", async () => {
                  try {
                    replaceConfig(await api("/playit/tunnel", { method: "DELETE" }));
                    toast("Playit tunnel deleted", "ok");
                    render();
                  } catch (error) { toast(error.message, "err"); }
                });
            } }, "Delete tunnel") : null,
            el("button", { class: "btn", type: "button", onclick: () => {
              editorRoot = null;
              confirmModal("Relink Playit agent",
                config.tunnel_id
                  ? "Relinking replaces the current agent and deletes its existing Playit tunnel."
                  : "Relinking replaces the current Bonghos-managed Playit agent.",
                "Relink agent", async () => {
                  if (await beginClaim(null)) openEditor();
                }, false);
            } }, "Relink agent")));
        }
        editorRoot.append(el("div", { class: "settings-row" },
          el("div", {}, el("h3", {}, "Account"), el("p", { class: "muted" }, "Used when linking or relinking.")), controls));
      } else {
        const address = el("input", { value: config.public_address || "", placeholder: "example.gl.joinmc.link", spellcheck: "false" });
        const port = el("input", { type: "number", min: "1", max: "65535", value: String(config.local_port || 25565) });
        saveEditorChanges = async () => {
          const publicAddress = address.value.trim();
          const localPort = Number(port.value);
          if (config.management_mode === "external" && publicAddress === (config.public_address || "") && localPort === Number(config.local_port || 25565)) return false;
          await savePreference({ enabled: config.enabled, management_mode: "external", public_address: publicAddress, local_port: localPort });
          editorManagementMode = "external";
          return true;
        };
        editorRoot.append(el("div", { class: "settings-row playit-external-config" },
          el("div", {}, el("h3", {}, "External agent"),
            el("p", { class: "muted" }, "Enter its public IP or address. Bonghos only displays it and cannot directly manage the agent.")),
          el("div", { class: "playit-form" },
            el("label", {}, "Public address", address), el("label", {}, "Port", port))));
        if (Array.isArray(config.detections) && config.detections.length) {
          editorRoot.append(el("div", { class: "settings-row playit-detected-agents" },
            el("div", {}, el("h3", {}, "Detected agents"), el("p", { class: "muted" }, "Read-only.")),
            el("div", { class: "settings-services" }, ...config.detections.map((item) =>
              settingsServiceCard(item.name, item.kind,
                item.externally_managed ? "Managed outside Bonghos." : "Managed by Bonghos.", item.state, true)))));
        }
      }
  };

  const openEditor = () => {
    editorManagementMode = config.management_mode;
    externalNoticeDismissed = false;
    editorRoot = el("div", { class: "playit-editor" });
    renderEditor();
    modal("Playit.gg", [editorRoot], [["Done", "primary", async (close) => {
      try {
        const saved = saveEditorChanges ? await saveEditorChanges() : false;
        if (saved) toast("External Playit agent saved", "ok");
        close();
      } catch (error) { toast(error.message, "err"); }
    }]], () => { editorRoot = null; saveEditorChanges = null; }, "playit-editor-modal");
  };

  const render = () => {
    section.innerHTML = "";
    const power = el("button", {
      class: "bot-power connection-power" + (config.enabled ? " is-on" : ""), type: "button",
      "aria-pressed": String(!!config.enabled),
      "aria-label": `${config.enabled ? "Turn off" : "Turn on"} Playit.gg`,
      onclick: async () => {
        power.disabled = true;
        try {
          await savePreference({
            enabled: !config.enabled,
            management_mode: config.management_mode === "none" ? "bonghos" : config.management_mode,
          });
          toast(config.enabled ? "Playit.gg enabled" : "Playit.gg disabled", "ok");
        } catch (error) { toast(error.message, "err"); power.disabled = false; }
      },
    }, el("span", { class: "bot-power-track", "aria-hidden": "true" }, el("span", {})),
    el("span", { class: "bot-power-label" }, config.enabled ? "On" : "Off"));

    section.append(settingsSectionHeading(
      "playit-settings-title", "Playit.gg", "Public game-server access without router port forwarding."),
    el("div", { class: "card connection-settings-card" },
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Connection"), el("p", { class: "muted" }, playitState(config))),
        el("div", { class: "connection-status-line" },
          el("div", { class: "connection-actions" },
            el("button", { class: "btn", type: "button", onclick: openEditor }, "Edit"), power)))));

    if (editorRoot?.isConnected) renderEditor();
    if (config.enabled && config.management_mode === "bonghos" && config.secret_configured && !config.agent_online && !config.agent_error) {
      queueMicrotask(pollAgent);
    }
  };

  render();
  return section;
}

function botsSettingsSection(bots) {
  const providerCounts = bots.reduce((counts, bot) => {
    counts[bot.provider] = (counts[bot.provider] || 0) + 1;
    return counts;
  }, {});
  const canAddBot = bots.length < 4 && ["telegram", "discord"].some((provider) => (providerCounts[provider] || 0) < 2);
  return el("section", { class: "settings-page-section", "aria-labelledby": "bots-settings-title" },
    settingsSectionHeading("bots-settings-title", "Bots", "Send server and player activity to Telegram or Discord.",
      DEMO_DEBUG_BOTS
        ? el("span", { class: "muted mono" }, ".env.development")
        : canAddBot
          ? el("button", { class: "btn primary", onclick: () => botEditor(null, bots) }, "Add bot")
          : el("span", { class: "muted mono" }, "4/4 connected")),
    bots.length
      ? el("div", { class: "bot-grid" }, ...bots.map(botCard))
      : el("div", { class: "card bot-empty" }, solarIcon("send-square-linear"),
        el("strong", {}, DEMO_DEBUG_BOTS ? "No development bots configured" : "No notification bots yet"),
        el("p", { class: "muted" }, DEMO_DEBUG_BOTS
          ? "Add a Telegram or Discord credential pair to .env.development, then restart the development server."
          : "Add a Telegram or Discord bot to receive server alerts."),
        DEMO_DEBUG_BOTS ? null : el("button", { class: "btn", onclick: () => botEditor(null, bots) }, "Add bot")));
}

function turnstileSettingsSection(config) {
  let enabledValue = !!config.enabled;
  let savedConfig = { ...config };
  let saveTimer = null;
  const summaryStatus = el("p", { class: "muted" }, enabledValue ? "On" : "Off");
  const enabled = el("button", {
    class: "bot-power connection-power" + (enabledValue ? " is-on" : ""),
    type: "button",
    "aria-pressed": String(enabledValue),
    "aria-label": `${enabledValue ? "Turn off" : "Turn on"} login protection`,
    onclick: () => {
      enabledValue = !enabledValue;
      syncEnabled();
      scheduleSave();
    },
  }, el("span", { class: "bot-power-track", "aria-hidden": "true" }, el("span", {})),
  el("span", { class: "bot-power-label" }, enabledValue ? "On" : "Off"));
  const siteKey = el("input", {
    value: config.site_key || "", autocomplete: "off", spellcheck: "false",
    placeholder: "Turnstile site key",
  });
  const secretKey = el("input", {
    type: "password", autocomplete: "new-password", spellcheck: "false",
    placeholder: config.secret_configured ? "Configured — leave blank to keep" : "Turnstile secret key",
  });
  const status = el("p", { class: "hint turnstile-settings-status" },
    config.secret_configured ? "Secret key configured" : "Secret key not configured");

  function syncEnabled() {
    enabled.classList.toggle("is-on", enabledValue);
    enabled.setAttribute("aria-pressed", String(enabledValue));
    enabled.setAttribute("aria-label", `${enabledValue ? "Turn off" : "Turn on"} login protection`);
    enabled.querySelector(".bot-power-label").textContent = enabledValue ? "On" : "Off";
    summaryStatus.textContent = enabledValue ? "On" : "Off";
  }

  function scheduleSave() {
    clearTimeout(saveTimer);
    saveTimer = setTimeout(saveSettings, 0);
  }

  function scheduleCredentialSave() {
    const hasCompleteNewCredentials = !!siteKey.value.trim() && !!secretKey.value.trim();
    if (savedConfig.secret_configured || hasCompleteNewCredentials) scheduleSave();
  }

  async function saveSettings() {
    const previousEnabled = !!savedConfig.enabled;
    enabled.disabled = true;
    siteKey.disabled = true;
    secretKey.disabled = true;
    status.textContent = "Saving…";
    try {
      const payload = { enabled: enabledValue, site_key: siteKey.value.trim() };
      if (secretKey.value.trim()) payload.secret_key = secretKey.value.trim();
      const saved = await api("/security/turnstile", { method: "PUT", json: payload });
      savedConfig = { ...saved };
      enabledValue = !!saved.enabled;
      syncEnabled();
      secretKey.value = "";
      secretKey.placeholder = saved.secret_configured ? "Configured — leave blank to keep" : "Turnstile secret key";
      status.textContent = saved.secret_configured ? "Secret key configured" : "Secret key not configured";
      S.turnstile = saved.enabled && saved.secret_configured
        ? { enabled: true, site_key: saved.site_key }
        : { enabled: false };
      clearLoginTurnstileWidget();
      const message = saved.enabled !== previousEnabled
        ? `Login protection ${saved.enabled ? "enabled" : "disabled"}`
        : "Turnstile settings saved";
      toast(message, "ok");
    } catch (error) {
      enabledValue = !!savedConfig.enabled;
      siteKey.value = savedConfig.site_key || "";
      syncEnabled();
      status.textContent = error.message;
      toast(error.message, "err");
    } finally {
      enabled.disabled = false;
      siteKey.disabled = false;
      secretKey.disabled = false;
    }
  }

  siteKey.addEventListener("change", scheduleCredentialSave);
  secretKey.addEventListener("change", scheduleCredentialSave);

  const openEditor = () => modal("Cloudflare Turnstile", [
    el("div", { class: "turnstile-editor" },
      el("p", { class: "muted" }, "Managed browser checks without member emails."),
      el("div", { class: "notice turnstile-editor-notice" },
        "After enabling, keep this session open and test sign-in in a private window."),
      el("div", { class: "turnstile-settings-form" },
        el("label", {}, "Site key", siteKey),
        el("label", {}, "Secret key", secretKey),
        status)),
  ], [["Done", "primary", (close) => close()]], null, "turnstile-editor-modal");

  return el("section", {},
    securitySectionHead("Login protection", "Block automated sign-in attempts."),
    el("div", { class: "card connection-settings-card" },
      el("div", { class: "settings-row" },
        el("div", {},
          el("h3", {}, "Cloudflare Turnstile (CAPTCHA)"),
          summaryStatus),
        el("div", { class: "connection-status-line" },
          el("div", { class: "connection-actions" },
            el("button", { class: "btn", type: "button", onclick: openEditor }, "Edit"), enabled)))));
}

async function pageSettings(main) {
  const [version, rawPlayit, rawBots, host] = await Promise.all([
    api("/version").catch(() => ({ version: "unknown" })),
    can("playit.manage") ? api("/playit").catch(() => null) : Promise.resolve(null),
    can("bots.manage") ? api("/bots") : Promise.resolve([]),
    can("host.view") ? api("/host").catch(() => null) : Promise.resolve(null),
  ]);
  const bots = Array.isArray(rawBots) ? rawBots : [];
  const makeThemeButton = (value, label) => el("button", {
    class: "btn ghost" + (themeChoice() === value ? " active" : ""),
    "data-theme-choice": value,
    onclick: () => setTheme(value),
  }, label);
  main.innerHTML = "";
  main.append(
    pageHeader("Settings", "Appearance, connectivity, integrations, and installation details."),
    el("section", { class: "settings-page-section", "aria-labelledby": "general-settings-title" },
      settingsSectionHeading("general-settings-title", "General", "Appearance and local preferences."),
      el("div", { class: "card" },
      el("div", { class: "settings-row" },
        el("div", {}, el("h3", {}, "Theme"), el("p", { class: "muted theme-settings-description" }, "System follows the OS color scheme and reacts to changes.")),
        el("div", { class: "segmented-choice theme-choice" },
          makeThemeButton("system", "System"),
          makeThemeButton("dark", "Dark"),
          makeThemeButton("light", "Light"))))),
    ...(can("playit.manage") && rawPlayit ? [playitSettingsSection(rawPlayit)] : []),
    ...(can("bots.manage") ? [botsSettingsSection(bots)] : []),
    el("section", { class: "settings-page-section", "aria-labelledby": "about-settings-title" },
      settingsSectionHeading("about-settings-title", "About", "Installation details and service status."),
      el("div", { class: "card" },
      el("div", { class: "settings-row" },
        el("div", {},
          el("h3", {}, "Installation"),
          el("p", { class: "muted" }, "Version and local paths for this Bonghos installation.")),
        el("dl", { class: "kv" },
          el("dt", {}, "Version"), el("dd", { class: "mono" }, version.version),
          host ? el("dt", {}, "Listen address") : null,
          host ? el("dd", { class: "mono" }, `${host.bind_address}:${host.port}`) : null,
          host ? el("dt", {}, "Data directory") : null,
          host ? el("dd", { class: "mono" }, host.home) : null)),
      host ? el("div", { class: "settings-row" },
        el("div", {},
          el("h3", {}, "Services"),
          el("p", { class: "muted" }, host.systemd
            ? "The panel and Minecraft run as separate services."
            : "Bonghos is running without systemd user services.")),
        el("div", { class: "settings-services" },
          settingsServiceCard("Bonghos panel", "bonghos.service",
            "Web UI, API, notification bots, schedules, backups, and monitoring.",
            host.service_bonghos, host.systemd),
          settingsServiceCard("Minecraft server", "bonghos-minecraft.service",
            "Active server pack and Java process.",
            host.service_minecraft, host.systemd))) : null)),
    el("footer", { class: "settings-footer" },
      el("div", { class: "settings-footer-line" }, "Made by ",
        el("a", {
          class: "settings-footer-author",
          href: "https://github.com/Chansovisoth",
          target: "_blank",
          rel: "noopener noreferrer",
        },
          el("i", { class: "fa fa-github", "aria-hidden": "true" }),
          "Chansovisoth"),
        " · Bonghos © 2026."),
      el("div", { class: "settings-footer-line" },
        "Open-source software · ",
        el("a", {
          href: "https://github.com/Chansovisoth/Bonghos/blob/main/LICENSE",
          target: "_blank",
          rel: "noopener noreferrer",
        }, "GNU AGPL v3 or later"),
        " · No warranty · ",
        el("a", {
          href: "https://github.com/Chansovisoth/Bonghos",
          target: "_blank",
          rel: "noopener noreferrer",
        }, "Source"))));
}

// ---------------------------------------------------------------------------
// activation page (invited users land on /activate/<token>)
// ---------------------------------------------------------------------------
function authBrandLogo() {
  const symbol = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  symbol.setAttribute("class", "brand-symbol");
  symbol.setAttribute("viewBox", "0 0 24 24");
  symbol.setAttribute("aria-hidden", "true");
  const empty = document.createElementNS("http://www.w3.org/2000/svg", "path");
  empty.setAttribute("d", "M0 0h24v24H0z");
  empty.setAttribute("fill", "none");
  const mark = document.createElementNS("http://www.w3.org/2000/svg", "path");
  mark.setAttribute("fill", "currentColor");
  mark.setAttribute("d", "M23 11v2h-1v1h-7v1h-1v2h-1v1h-1v2h-1v1h-1v1H7v-3h1v-3h1v-2H5v1H4v1H3v1H1v-3h1v-4H1V7h2v1h1v1h1v1h4V8H8V5H7V2h3v1h1v1h1v2h1v1h1v2h1v1h7v1z");
  symbol.append(empty, mark);
  return el("div", { class: "brand brand-logo", "aria-label": "Bonghos" },
    el("span", { class: "brand-name" }, ">BONGHOS"), symbol);
}

async function invitationApi(path, opts = {}) {
  if (DEMO_MODE) return demoApi(path, opts);
  const method = opts.method || "GET";
  const headers = { ...(opts.headers || {}) };
  if (method !== "GET") {
    const csrfResponse = await fetch("/api/auth/csrf", { credentials: "same-origin" });
    const csrf = await csrfResponse.json();
    if (!csrfResponse.ok) throw new Error(csrf.error || "Could not start the invitation request.");
    headers["X-Bonghos-CSRF"] = csrf.csrf;
  }
  const request = { ...opts, method, headers, credentials: "same-origin" };
  if (opts.json !== undefined) {
    headers["Content-Type"] = "application/json";
    request.body = JSON.stringify(opts.json);
    delete request.json;
  }
  const response = await fetch("/api" + path, request);
  let data = null;
  try { data = await response.json(); } catch { /* empty response */ }
  if (!response.ok) throw new Error(data?.error || response.statusText || "Invitation request failed.");
  return data;
}

function base64URLToBytes(value) {
  const base64 = String(value || "").replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64 + "=".repeat((4 - base64.length % 4) % 4);
  const binary = atob(padded);
  return Uint8Array.from(binary, (char) => char.charCodeAt(0));
}

function bytesToBase64URL(value) {
  if (value === null || value === undefined) return null;
  const bytes = new Uint8Array(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function passkeyCreationOptions(json) {
  if (typeof PublicKeyCredential?.parseCreationOptionsFromJSON === "function") {
    return PublicKeyCredential.parseCreationOptionsFromJSON(json);
  }
  const options = { ...json, user: { ...json.user } };
  options.challenge = base64URLToBytes(json.challenge);
  options.user.id = base64URLToBytes(json.user.id);
  options.excludeCredentials = (json.excludeCredentials || []).map((item) => ({ ...item, id: base64URLToBytes(item.id) }));
  return options;
}

function passkeyRequestOptions(json) {
  if (typeof PublicKeyCredential?.parseRequestOptionsFromJSON === "function") {
    return PublicKeyCredential.parseRequestOptionsFromJSON(json);
  }
  const options = { ...json };
  options.challenge = base64URLToBytes(json.challenge);
  options.allowCredentials = (json.allowCredentials || []).map((item) => ({ ...item, id: base64URLToBytes(item.id) }));
  return options;
}

function passkeyCredentialJSON(credential) {
  if (typeof credential?.toJSON === "function") return credential.toJSON();
  const response = credential.response;
  const jsonResponse = {
    clientDataJSON: bytesToBase64URL(response.clientDataJSON),
  };
  if (response.attestationObject) {
    jsonResponse.attestationObject = bytesToBase64URL(response.attestationObject);
    jsonResponse.transports = typeof response.getTransports === "function" ? response.getTransports() : [];
  } else {
    jsonResponse.authenticatorData = bytesToBase64URL(response.authenticatorData);
    jsonResponse.signature = bytesToBase64URL(response.signature);
    jsonResponse.userHandle = bytesToBase64URL(response.userHandle);
  }
  return {
    id: credential.id,
    rawId: bytesToBase64URL(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment || undefined,
    response: jsonResponse,
    clientExtensionResults: credential.getClientExtensionResults(),
  };
}

function passkeysSupported() {
  return !!(window.isSecureContext && window.PublicKeyCredential && navigator.credentials);
}

function passkeyError(error, fallback) {
  if (error?.name === "NotAllowedError") return "The passkey request was cancelled or timed out.";
  if (error?.name === "InvalidStateError") return "That passkey is already registered here.";
  if (error?.name === "SecurityError") return "Passkeys require this panel to use HTTPS on the same hostname.";
  return error?.message || fallback;
}

async function activationFlow(token) {
  document.body.innerHTML = "";
  const wrap = el("div", { class: "login-wrap" });
  document.body.append(wrap, el("div", { id: "toast-host" }), el("div", { id: "modal-host" }));
  try {
    const info = await invitationApi(`/invitations/${token}`);
    const user = el("input", { autocomplete: "username", required: "", value: DEMO_MODE ? "invited-admin" : "" });
    const p1 = el("input", { type: "password", autocomplete: "new-password", minlength: "10", required: "", value: DEMO_MODE ? "demo-password" : "" });
    const p2 = el("input", { type: "password", autocomplete: "new-password", minlength: "10", required: "", value: DEMO_MODE ? "demo-password" : "" });
    const code = el("input", {
      id: "activation-code", class: "otp-input", inputmode: "numeric", pattern: "[0-9]{6}",
      maxlength: "6", autocomplete: "one-time-code", "aria-label": "Six-digit authenticator code", disabled: "",
    });
    const otpWrap = el("div", { class: "otp-wrap", "aria-hidden": "true" },
      ...Array.from({ length: 6 }, () => el("span")));
    const qrBox = el("div", { class: "qr-box hidden" });
    const secretBox = el("pre", { class: "activation-secret muted mono" });
    const manualSetup = el("details", { class: "activation-manual" },
      el("summary", {}, "Can't scan the QR code?"),
      el("div", { class: "activation-secret-row" },
        secretBox,
        el("button", {
          class: "btn ghost small icon-button", type: "button", title: "Copy authenticator secret",
          "aria-label": "Copy authenticator secret",
          onclick: () => secret && copyText(secret, "Authenticator secret copied"),
        }, solarIcon("copy-linear"))));
    const accountStep = el("div", { class: "activation-step" });
    const qrStep = el("div", { class: "activation-step hidden" });
    const verificationStep = el("div", { class: "activation-step hidden" });
    const verificationIntro = el("p", { class: "muted" });
    let secret = "";
    let currentStep = 1;
    const nextButton = el("button", { class: "btn primary", type: "submit" }, "Next");
    const showAccountStep = () => {
      currentStep = 1;
      secret = "";
      code.value = "";
      code.disabled = true;
      code.required = false;
      syncOTPCells(code, otpWrap);
      otpWrap.classList.remove("error", "focus");
      qrBox.replaceChildren();
      qrBox.classList.add("hidden");
      secretBox.textContent = "";
      manualSetup.open = false;
      accountStep.classList.remove("hidden");
      qrStep.classList.add("hidden");
      verificationStep.classList.add("hidden");
      setTimeout(() => user.focus(), 30);
    };
    const showOTPSetup = async () => {
      if (p1.value !== p2.value) {
        p2.setCustomValidity("Passwords do not match");
        p2.reportValidity();
        return;
      }
      p2.setCustomValidity("");
      nextButton.disabled = true;
      nextButton.textContent = "Preparing…";
      try {
        const d = await invitationApi(`/invitations/${token}/totp`, {
          method: "POST", json: { username: user.value },
        });
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
          qrBox.classList.remove("hidden");
        }
        secretBox.textContent = d.secret;
        manualSetup.open = !svgOK;
        currentStep = 2;
        accountStep.classList.add("hidden");
        qrStep.classList.remove("hidden");
      } catch (error) {
        secret = "";
        toast(error.message, "err");
      } finally {
        nextButton.disabled = false;
        nextButton.textContent = "Next";
      }
    };
    const showQRStep = () => {
      currentStep = 2;
      code.disabled = true;
      code.required = false;
      otpWrap.classList.remove("error", "focus");
      verificationStep.classList.add("hidden");
      qrStep.classList.remove("hidden");
    };
    const showVerificationStep = () => {
      currentStep = 3;
      code.disabled = false;
      code.required = true;
      code.value = DEMO_MODE ? "123456" : "";
      verificationIntro.textContent = `Enter the code for ${user.value.trim() || "your account"}.`;
      qrStep.classList.add("hidden");
      verificationStep.classList.remove("hidden");
      syncOTPCells(code, otpWrap);
      setTimeout(() => code.focus(), 30);
    };
    const form = el("form", { class: "login-card", onsubmit: async (e) => {
      e.preventDefault();
      if (currentStep === 1) return showOTPSetup();
      if (currentStep === 2) return showVerificationStep();
      if (!secret) return toast("Generate the authenticator secret first", "err");
      try {
        const d = await invitationApi(`/invitations/${token}/activate`, { method: "POST", json: {
          username: user.value, password: p1.value, totp_secret: secret, totp_code: code.value,
        } });
        const recoveryCodes = (d.recovery_codes || []).join("\n");
        form.innerHTML = "";
        form.append(
          authBrandLogo(),
          el("h1", { class: "activation-title" }, "Account created"),
          el("p", {}, "Store these one-time recovery codes safely:"),
          el("div", { class: "activation-recovery-codes" },
            el("pre", { class: "mono" }, recoveryCodes),
            el("button", {
              class: "btn ghost small icon-button", type: "button", title: "Copy recovery codes",
              "aria-label": "Copy recovery codes",
              onclick: () => recoveryCodes && copyText(recoveryCodes, "Recovery codes copied"),
            }, solarIcon("copy-linear"))),
          el("a", { class: "btn primary", href: DEMO_MODE ? "/?demo=login" : "/", style: "text-align:center" }, "Go to Login"));
      } catch (error) {
        otpWrap.classList.add("error");
        toast(error.message, "err");
      }
    } },
      authBrandLogo(),
      el("p", { class: "muted activation-intro" }, `${DEMO_MODE ? "Demo invitation · " : ""}You are joining as ${info.role}.`),
      accountStep,
      qrStep,
      verificationStep);
    accountStep.append(
      el("span", { class: "activation-step-kicker" }, "Step 1 of 3"),
      el("h1", { class: "activation-title" }, "Create your account"),
      el("label", {}, "Username", user),
      el("label", {}, "Password (min 10 chars)", p1),
      el("label", {}, "Confirm password", p2),
      nextButton);
    p1.addEventListener("input", () => p2.setCustomValidity(""));
    p2.addEventListener("input", () => p2.setCustomValidity(""));
    code.addEventListener("input", () => otpWrap.classList.remove("error"));
    qrStep.append(
      el("span", { class: "activation-step-kicker" }, "Step 2 of 3"),
      el("h1", { class: "activation-title" }, "Scan QR code"),
      el("p", { class: "muted" }, "Scan this code with your authenticator app."),
      qrBox,
      manualSetup,
      el("div", { class: "auth-actions" },
        el("button", { class: "btn ghost", type: "button", onclick: showAccountStep }, "Back"),
        el("button", { class: "btn primary", type: "submit" }, "Next")));
    verificationStep.append(
      el("span", { class: "activation-step-kicker" }, "Step 3 of 3"),
      el("h1", { class: "activation-title" }, "Enter authenticator code"),
      verificationIntro,
      otpWrap,
      code,
      el("p", { class: "hint" }, "Enter the six-digit code shown in your authenticator app."),
      el("div", { class: "auth-actions" },
        el("button", { class: "btn ghost", type: "button", onclick: showQRStep }, "Back"),
        el("button", { class: "btn primary", type: "submit" }, "Activate")));
    installOTPControl(code, otpWrap);
    wrap.append(form);
  } catch (e) {
    wrap.append(el("div", { class: "login-card" },
      authBrandLogo(),
      el("h1", { class: "activation-title" }, "Invitation unavailable"),
      el("p", { class: "error" }, e.message || "This invitation is invalid or expired.")));
  }
}

// ---------------------------------------------------------------------------
const activateMatch = location.pathname.match(/^\/activate\/([A-Za-z0-9_-]+)/);
if (DEMO_MODE && (DEMO_VIEW === "invite" || activateMatch)) activationFlow(DEMO_INVITE_TOKEN);
else if (activateMatch) activationFlow(activateMatch[1]);
else {
  installOTPControl();
  boot();
}
startConnectivityMonitor();
