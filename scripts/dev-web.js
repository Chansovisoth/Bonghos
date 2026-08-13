"use strict";

// Native local Web UI server with an opt-in notification relay. Provider
// tokens remain in the ignored .env.development file and are never returned
// to browser JavaScript. This helper is development-only and binds to loopback.

const fs = require("node:fs");
const http = require("node:http");
const path = require("node:path");

const REPO_ROOT = path.resolve(__dirname, "..");
const STATIC_ROOT = path.join(REPO_ROOT, "source", "web", "src");
const DEFAULT_ENV_PATH = path.join(REPO_ROOT, ".env.development");
const RELAY_HEADER = "x-bonghos-dev-relay";

function parseEnv(text) {
  const values = {};
  for (const rawLine of String(text).split(/\r?\n/)) {
    let line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    if (line.startsWith("export ")) line = line.slice(7).trim();
    const separator = line.indexOf("=");
    if (separator < 1) continue;
    const key = line.slice(0, separator).trim();
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) continue;
    let value = line.slice(separator + 1).trim();
    if (value.length >= 2 && ((value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'")))) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

function loadDevelopmentEnv(envPath = DEFAULT_ENV_PATH) {
  let fileValues = {};
  try {
    fileValues = parseEnv(fs.readFileSync(envPath, "utf8"));
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  return { ...fileValues, ...process.env };
}

function configuredBots(env) {
  const definitions = [
    {
      id: 1,
      provider: "telegram",
      name: env.BONGHOS_DEV_TELEGRAM_NAME || "Telegram development",
      token: String(env.BONGHOS_DEV_TELEGRAM_TOKEN || "").trim(),
    },
    {
      id: 2,
      provider: "discord",
      name: env.BONGHOS_DEV_DISCORD_NAME || "Discord development",
      token: String(env.BONGHOS_DEV_DISCORD_TOKEN || "").trim(),
    },
  ];

  const configured = [];
  for (const definition of definitions) {
    if (!definition.token) continue;
    if (/\s/.test(definition.token)) {
      throw new Error(`${definition.provider} token contains whitespace`);
    }
    // Provider destinations must be established by an administrator using
    // /bonghos here. Never seed the flow from a legacy environment ID.
    const destinationIDs = [];
    const destinations = destinationIDs.map((destinationID) => ({
      id: destinationID, name: destinationID, type: definition.provider === "telegram" ? "group" : "channel",
    }));
    configured.push({
      ...definition,
      destination: destinationIDs[0] || "",
      destinations,
      discoveredGroups: new Map(),
      public: {
        id: definition.id,
        name: definition.name,
        provider: definition.provider,
        destination_id: destinationIDs[0] || "",
        destinations: destinations.map((destination) => ({ ...destination })),
        discovered_destinations: [],
        enabled: true,
        notify_server_started: true,
        notify_server_stopped: true,
        notify_player_joined: true,
        notify_player_left: true,
        token_configured: true,
        development_env: true,
      },
    });
  }
  return configured;
}

async function telegramAPI(bot, method, params = {}, fetchImpl = fetch) {
  const query = new URLSearchParams(params);
  let response;
  try {
    response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/${method}${query.size ? `?${query}` : ""}`);
  } catch {
    throw new Error("telegram command request failed");
  }
  let payload = {};
  try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
  if (!response.ok || !payload.ok) throw new Error(`telegram command request returned HTTP ${response.status}`);
  return payload.result;
}

function telegramCommandName(text) {
  const fields = String(text || "").trim().split(/\s+/).filter(Boolean);
  const root = String(fields[0] || "").toLowerCase().replace(/^\//, "").split("@", 1)[0];
  if (root !== "bonghos") return "";
  const subcommand = String(fields[1] || "help").toLowerCase();
  return ["here", "disconnect", "where", "help"].includes(subcommand) ? subcommand : "help";
}

// Fetch a group's small profile photo and return it as a data URL, or "" when
// the group has no photo or it cannot be downloaded. A raw fetchImpl is passed
// so it works from both discovery and the /bonghos here command flow.
async function fetchTelegramGroupPhoto(bot, chatID, fetchImpl = fetch) {
  const api = async (method) => {
    const response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/${method}`);
    let payload = {};
    try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
    if (!response.ok || !payload.ok) throw new Error(`telegram returned HTTP ${response.status}`);
    return payload.result;
  };
  const fullChat = await api(`getChat?chat_id=${encodeURIComponent(chatID)}`);
  const fileID = String(fullChat?.photo?.small_file_id || "").trim();
  if (!fileID) return "";
  const file = await api(`getFile?file_id=${encodeURIComponent(fileID)}`);
  const filePath = String(file?.file_path || "").replace(/^\/+/, "");
  if (!filePath || filePath.split("/").some((part) => !part || part === "." || part === "..")) return "";
  const photoResponse = await fetchImpl(`https://api.telegram.org/file/bot${bot.token}/${filePath}`);
  if (!photoResponse.ok || typeof photoResponse.arrayBuffer !== "function") return "";
  const data = Buffer.from(await photoResponse.arrayBuffer());
  if (!data.length || data.length > 512 * 1024) return "";
  const declaredType = String(photoResponse.headers?.get?.("content-type") || "").split(";", 1)[0].toLowerCase();
  const contentType = ["image/jpeg", "image/png", "image/webp"].includes(declaredType)
    ? declaredType
    : (/\.png$/i.test(filePath) ? "image/png" : /\.webp$/i.test(filePath) ? "image/webp" : "image/jpeg");
  return `data:${contentType};base64,${data.toString("base64")}`;
}

function escapeTelegramHTML(value) {
  return String(value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// Resolve a human-readable forum topic name for the message's thread. Telegram
// includes the topic-creation service message on replies inside a topic, and
// prior discovery may already know the name; fall back to a numbered label.
function resolveTelegramTopicName(bot, message, chatID, threadID) {
  if (!threadID) return "General";
  const events = [
    message.forum_topic_created, message.forum_topic_edited,
    message.reply_to_message?.forum_topic_created, message.reply_to_message?.forum_topic_edited,
  ].filter(Boolean);
  const named = events.map((event) => String(event.name || "").trim()).filter(Boolean).at(-1);
  if (named) return named;
  const discovered = bot.discoveredGroups?.get(chatID)?.topics?.find((topic) => Number(topic.id) === threadID);
  if (discovered?.name) return discovered.name;
  return `Channel ${threadID}`;
}

async function pollDevelopmentTelegramCommands(bot, fetchImpl = fetch) {
  const updates = await telegramAPI(bot, "getUpdates", {
    allowed_updates: JSON.stringify(["message", "my_chat_member"]), limit: "100", timeout: "0",
    offset: String((Number(bot.lastUpdateID) || 0) + 1),
  }, fetchImpl);
	if (!bot.commandInitialized) {
	  for (const update of Array.isArray(updates) ? updates : []) {
		await rememberDevelopmentTelegramMembership(bot, update, fetchImpl);
		bot.lastUpdateID = Math.max(Number(bot.lastUpdateID) || 0, Number(update.update_id) || 0);
	  }
	  if (updates.length) bot.commandEmptyPolls = 0;
	  else bot.commandEmptyPolls = (Number(bot.commandEmptyPolls) || 0) + 1;
	  // Drain and confirm the old backlog before accepting commands. Two empty
	  // polls prevent a delayed historical update from reconnecting a stale topic.
	  bot.commandInitialized = bot.commandEmptyPolls >= 2;
	  return;
	}
  for (const update of Array.isArray(updates) ? updates : []) {
    bot.lastUpdateID = Math.max(Number(bot.lastUpdateID) || 0, Number(update.update_id) || 0);
    await rememberDevelopmentTelegramMembership(bot, update, fetchImpl);
    const message = update.message;
    if (["group", "supergroup"].includes(message?.chat?.type)) {
      await rememberDevelopmentTelegramGroup(bot, message.chat, fetchImpl);
    }
    const command = telegramCommandName(message?.text);
    if (!command || !["group", "supergroup"].includes(message?.chat?.type)) continue;
    const chatID = String(message.chat.id);
    const threadID = Number(message.message_thread_id) > 1 ? Number(message.message_thread_id) : 0;
    let administrator = message.sender_chat?.id === message.chat.id;
    if (!administrator && Number(message.from?.id) > 0) {
      const member = await telegramAPI(bot, "getChatMember", { chat_id: chatID, user_id: String(message.from.id) }, fetchImpl);
      administrator = ["creator", "administrator"].includes(member?.status);
    }
    const reply = (text) => providerRequest(bot, text, fetchImpl, chatID, threadID, { html: true });
    if (command === "help") {
      await reply("Bonghos commands:\n<code>/bonghos here</code> : Send notifications to this topic\n<code>/bonghos where</code> : Check this group's destination\n<code>/bonghos disconnect</code>: Stop notifications to this group\n\nOnly group administrators can change destinations.");
      continue;
    }
    if (!administrator) {
      await reply("Only a group administrator can configure Bonghos notifications.");
      continue;
    }
    const existingIndex = bot.destinations.findIndex((destination) => destination.id === chatID);
    if (command === "here") {
      if (existingIndex < 0 && bot.destinations.length >= 3) {
        await reply("Bonghos could not connect this destination: Telegram already has three connected groups");
        continue;
      }
      const destination = {
        id: chatID, name: String(message.chat.title || chatID), type: message.chat.type,
        forum: !!(message.chat.is_forum || threadID), thread_id: threadID,
        thread_name: resolveTelegramTopicName(bot, message, chatID, threadID),
        photo_data_url: String(bot.discoveredGroups?.get(chatID)?.photo_data_url || ""),
      };
      // Fetch the group photo so command-connected groups show an avatar in the
      // Web UI even when discovery was never run. A missing photo is not fatal.
      if (!destination.photo_data_url) {
        try { destination.photo_data_url = await fetchTelegramGroupPhoto(bot, chatID, fetchImpl); }
        catch { /* keep the initials fallback */ }
      }
      if (existingIndex >= 0) bot.destinations[existingIndex] = destination;
      else bot.destinations.push(destination);
      bot.destination = bot.destinations[0]?.id || "";
      bot.public.destination_id = bot.destination;
      bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
      await reply("Bonghos notifications will be sent here.");
    } else if (command === "disconnect") {
      bot.destinations = bot.destinations.filter((destination) => destination.id !== chatID);
      bot.destination = bot.destinations[0]?.id || "";
      bot.public.destination_id = bot.destination;
      bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
      await reply("Bonghos notifications are disconnected from this group.");
    } else {
      const destination = bot.destinations.find((value) => value.id === chatID);
      if (!destination) {
        await reply("This group is not connected. Run /bonghos here in the topic that should receive notifications.");
      } else {
        const topicName = escapeTelegramHTML(destination.thread_name
          || (Number(destination.thread_id) ? `Channel ${destination.thread_id}` : "General"));
        await reply(Number(destination.thread_id) === threadID
          ? `Bonghos notifications are configured for topic "${topicName}".`
          : `Bonghos notifications are configured for topic "${topicName}" in this group.`);
      }
    }
  }
}

function syncDevelopmentDiscoveries(bot) {
  bot.public.discovered_destinations = Array.from(bot.discoveredGroups?.values?.() || [])
    .map((value) => ({ ...value }))
    .sort((left, right) => String(left.name || left.guild_name || "").localeCompare(String(right.name || right.guild_name || "")));
}

async function rememberDevelopmentTelegramGroup(bot, chat, fetchImpl = fetch) {
  if (!["group", "supergroup"].includes(chat?.type)) return;
  if (!bot.discoveredGroups) bot.discoveredGroups = new Map();
  const id = String(chat.id);
  const previous = bot.discoveredGroups.get(id) || {};
  const group = {
    ...previous, id, name: String(chat.title || previous.name || id), type: chat.type,
    forum: !!(chat.is_forum || previous.forum), topics: previous.topics || [],
    discovered_at: previous.discovered_at || new Date().toISOString(),
  };
  if (!group.photo_data_url) {
    try { group.photo_data_url = await fetchTelegramGroupPhoto(bot, id, fetchImpl); }
    catch { /* initials remain available when Telegram has no group photo */ }
  }
  bot.discoveredGroups.set(id, group);
  syncDevelopmentDiscoveries(bot);
}

async function rememberDevelopmentTelegramMembership(bot, update, fetchImpl = fetch) {
  const membership = update?.my_chat_member;
  if (!membership || !["group", "supergroup"].includes(membership.chat?.type)) return;
  const id = String(membership.chat.id);
  const status = String(membership.new_chat_member?.status || "").toLowerCase();
  if (["left", "kicked"].includes(status) ||
    (status === "restricted" && membership.new_chat_member?.is_member === false)) {
    bot.discoveredGroups?.delete(id);
    bot.destinations = bot.destinations.filter((destination) => String(destination.id) !== id);
    bot.destination = bot.destinations[0]?.id || "";
    bot.public.destination_id = bot.destination;
    bot.public.destinations = bot.destinations.map((destination) => ({ ...destination }));
    syncDevelopmentDiscoveries(bot);
    return;
  }
  await rememberDevelopmentTelegramGroup(bot, membership.chat, fetchImpl);
}

function startDevelopmentTelegramCommands(bots, fetchImpl = fetch) {
  const telegram = bots.find((bot) => bot.provider === "telegram");
  if (!telegram) return () => {};
  let running = false;
  const poll = async () => {
    if (running) return;
    running = true;
    try { await pollDevelopmentTelegramCommands(telegram, fetchImpl); }
    catch (error) { console.warn(`Telegram command polling: ${error.message}`); }
    finally { running = false; }
  };
  void poll();
  const timer = setInterval(poll, 3000);
  return () => clearInterval(timer);
}

async function providerRequest(bot, message, fetchImpl = fetch, destination = bot.destination, threadID = 0, options = {}) {
  let response;
  try {
    if (bot.provider === "telegram") {
      if (!/^[A-Za-z0-9:_-]+$/.test(bot.token)) throw new Error("Telegram token has an invalid format");
      response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/sendMessage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          chat_id: destination, text: message,
          ...(options.html ? { parse_mode: "HTML" } : {}),
          ...(Number(threadID) > 0 ? { message_thread_id: Number(threadID) } : {}),
        }),
      });
    } else {
      response = await fetchImpl(`https://discord.com/api/v10/channels/${destination}/messages`, {
        method: "POST",
        headers: {
          "Authorization": `Bot ${bot.token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content: message, allowed_mentions: { parse: [] } }),
      });
    }
  } catch {
    throw new Error(`${bot.provider} request failed`);
  }

  const responseText = await response.text();
  if (!response.ok) {
    let detail = "";
    try {
      const payload = JSON.parse(responseText);
      detail = String(payload.description || payload.message || "").slice(0, 240);
    } catch {
      // Provider returned a non-JSON error. Do not reflect arbitrary content.
    }
    throw new Error(`${bot.provider} returned HTTP ${response.status}${detail ? `: ${detail}` : ""}`);
  }
}

function uniqueTelegramTopics(values) {
  const byID = new Map();
  for (const topic of Array.isArray(values) ? values : []) {
    const id = Number(topic?.id) || 0;
    if (id <= 1) continue;
    const name = String(topic.name || "").normalize("NFKC")
      .replace(/[\u200B-\u200D\u2060\uFEFF]/gu, "")
      .trim().replace(/\s+/gu, " ") || `Channel ${id}`;
    byID.set(id, { id, name });
  }
  const byName = new Map();
  for (const topic of byID.values()) {
    const key = topic.name.normalize("NFKC")
      .replace(/[\u200B-\u200D\u2060\uFEFF]/gu, "")
      .trim().replace(/\s+/gu, " ").toLocaleLowerCase();
    const current = byName.get(key);
    if (!current || topic.id > current.id) byName.set(key, topic);
  }
  return Array.from(byName.values()).sort((left, right) => left.name.localeCompare(right.name));
}

async function discoverTelegramGroups(bot, fetchImpl = fetch) {
  const request = async (method) => {
    let response;
    try {
      response = await fetchImpl(`https://api.telegram.org/bot${bot.token}/${method}`);
    } catch {
      throw new Error("telegram discovery request failed");
    }
    let payload = {};
    try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
    if (!response.ok || !payload.ok) {
      const detail = String(payload.description || "").slice(0, 200);
      throw new Error(`telegram returned HTTP ${response.status}${detail ? `: ${detail}` : ""}`);
    }
    return payload.result;
  };
  const me = await request("getMe");
  const updates = await request(`getUpdates?allowed_updates=${encodeURIComponent(JSON.stringify([
    "message", "edited_message", "channel_post", "edited_channel_post", "my_chat_member",
  ]))}&limit=100&timeout=0`);
  const byID = new Map();
  const topicsByChat = new Map();
  const add = (chat) => {
    if (!chat || !["group", "supergroup"].includes(chat.type)) return;
    const id = String(chat.id);
    const current = byID.get(id) || {};
    byID.set(id, {
      ...current, id, name: chat.title || (chat.username ? `@${chat.username}` : id),
      type: chat.type, forum: !!(current.forum || chat.is_forum),
    });
  };
  const addMessage = (message) => {
    if (!message) return;
    add(message.chat);
    if (!message.chat || !["group", "supergroup"].includes(message.chat.type)) return;
    let threadID = Number(message.message_thread_id) || 0;
    if (threadID <= 1 && message.forum_topic_created) threadID = Number(message.message_id) || 0;
    if (threadID <= 1 || (!message.is_topic_message && !message.forum_topic_created && !message.forum_topic_edited)) return;
    const chatID = String(message.chat.id);
    if (!topicsByChat.has(chatID)) topicsByChat.set(chatID, new Map());
    const topicMap = topicsByChat.get(chatID);
    const events = [
      message.forum_topic_created, message.forum_topic_edited,
      message.reply_to_message?.forum_topic_created, message.reply_to_message?.forum_topic_edited,
    ].filter(Boolean);
    const name = events.map((event) => String(event.name || "").trim()).filter(Boolean).at(-1)
      || topicMap.get(threadID)?.name || `Channel ${threadID}`;
    topicMap.set(threadID, { id: threadID, name });
    byID.get(chatID).forum = true;
  };
  for (const update of Array.isArray(updates) ? updates : []) {
    addMessage(update.message);
    addMessage(update.edited_message);
    addMessage(update.channel_post);
    addMessage(update.edited_channel_post);
    const membership = update.my_chat_member;
    const status = String(membership?.new_chat_member?.status || "").toLowerCase();
    if (["left", "kicked"].includes(status) ||
      (status === "restricted" && membership?.new_chat_member?.is_member === false)) {
      byID.delete(String(membership.chat?.id));
    } else {
      add(membership?.chat);
    }
  }
  const groups = Array.from(byID.values()).map((group) => ({
    ...group,
    topics: uniqueTelegramTopics(Array.from(topicsByChat.get(group.id)?.values() || [])),
  })).sort((left, right) => left.name.localeCompare(right.name));
  for (const group of groups.slice(0, 20)) {
    try {
      const fullChat = await request(`getChat?chat_id=${encodeURIComponent(group.id)}`);
      group.forum = !!(group.forum || fullChat?.is_forum);
      const fileID = String(fullChat?.photo?.small_file_id || "").trim();
      if (!fileID) continue;
      group.photo_file_id = fileID;
      const file = await request(`getFile?file_id=${encodeURIComponent(fileID)}`);
      const filePath = String(file?.file_path || "").replace(/^\/+/, "");
      if (!filePath || filePath.split("/").some((part) => !part || part === "." || part === "..")) continue;
      const photoResponse = await fetchImpl(`https://api.telegram.org/file/bot${bot.token}/${filePath}`);
      if (!photoResponse.ok || typeof photoResponse.arrayBuffer !== "function") continue;
      const data = Buffer.from(await photoResponse.arrayBuffer());
      if (!data.length || data.length > 512 * 1024) continue;
      const declaredType = String(photoResponse.headers?.get?.("content-type") || "").split(";", 1)[0].toLowerCase();
      const contentType = ["image/jpeg", "image/png", "image/webp"].includes(declaredType)
        ? declaredType
        : (/\.png$/i.test(filePath) ? "image/png" : /\.webp$/i.test(filePath) ? "image/webp" : "image/jpeg");
      group.photo_data_url = `data:${contentType};base64,${data.toString("base64")}`;
    } catch {
      // A missing group photo must not prevent selecting that group.
    }
  }
  return { bot_username: me?.username || "", groups };
}

// ---------------------------------------------------------------------------
// Discord slash commands
//
// Unlike Telegram (HTTP long-poll of getUpdates), Discord delivers slash-command
// interactions only over a gateway websocket or a public HTTPS endpoint. This
// block resolves the application ID, registers /bonghos, handles interactions,
// and maintains the outbound Gateway connection used by local development.
// ---------------------------------------------------------------------------

const DISCORD_API = "https://discord.com/api/v10";

// Resolve a Discord bot's application ID and username from its token. For a bot
// user, GET /users/@me returns id === application_id, so the user never has to
// paste the application ID separately. Doubles as token validation.
async function resolveDiscordApplication(token, fetchImpl = fetch) {
  token = String(token || "").trim();
  if (!/^[A-Za-z0-9._-]{20,}$/.test(token)) throw new Error("Discord bot token is invalid");
  let response;
  try {
    response = await fetchImpl(`${DISCORD_API}/users/@me`, {
      headers: { "Authorization": `Bot ${token}`, "User-Agent": "Bonghos/notification-bot" },
    });
  } catch {
    throw new Error("Discord request failed");
  }
  let payload = {};
  try { payload = JSON.parse(await response.text()); } catch { /* handled below */ }
  if (!response.ok) throw new Error(`Discord returned HTTP ${response.status}`);
  const applicationID = String(payload.id || "").trim();
  if (!/^\d{10,25}$/.test(applicationID)) throw new Error("Discord did not return an application ID");
  if (!payload.bot) throw new Error("Discord token does not belong to a bot application");
  const username = String(payload.username || "").trim();
  return { applicationID, username, bot: !!payload.bot };
}

async function botInviteURL(bot, fetchImpl = fetch) {
  if (bot.provider === "telegram") {
    const me = await telegramAPI(bot, "getMe", {}, fetchImpl);
    const username = String(me?.username || "").replace(/^@/, "").trim();
    if (!/^[A-Za-z0-9_]{5,}$/.test(username)) throw new Error("Telegram bot username is unavailable");
    return `https://t.me/${username}?startgroup`;
  }
  if (bot.provider === "discord") {
    const application = await resolveDiscordApplication(bot.token, fetchImpl);
    const query = new URLSearchParams({
      client_id: application.applicationID,
      scope: "bot applications.commands",
      permissions: "274877910016",
      integration_type: "0",
    });
    return `https://discord.com/oauth2/authorize?${query}`;
  }
  throw new Error("unsupported notification provider");
}

// The /bonghos command definition, mirroring the Telegram subcommands.
// Manage Server is the default command permission. The handler repeats the
// check server-side and also accepts Discord administrators.
function discordCommandDefinition() {
  return {
    name: "bonghos",
    description: "Configure Bonghos notifications for this channel",
    type: 1, // CHAT_INPUT
    dm_permission: false,
    default_member_permissions: "32",
    options: [
      { type: 1, name: "here", description: "Send Bonghos notifications to this channel" },
      { type: 1, name: "where", description: "Check this channel's Bonghos notification status" },
      { type: 1, name: "disconnect", description: "Stop Bonghos notifications in this channel" },
      { type: 1, name: "help", description: "Show Bonghos notification commands" },
    ],
  };
}

// Register /bonghos. Guild-scoped registration is instant and ideal for
// development; global registration (no guildID) can take up to an hour to
// propagate. Requires the bot to have been invited with applications.commands.
async function registerDiscordCommands(token, applicationID, guildID, fetchImpl = fetch) {
  applicationID = String(applicationID || "").trim();
  if (!/^\d{10,25}$/.test(applicationID)) throw new Error("Discord application ID is invalid");
  const scope = guildID
    ? `applications/${applicationID}/guilds/${encodeURIComponent(String(guildID))}/commands`
    : `applications/${applicationID}/commands`;
  let response;
  try {
    response = await fetchImpl(`${DISCORD_API}/${scope}`, {
      method: "PUT",
      headers: {
        "Authorization": `Bot ${token}`,
        "Content-Type": "application/json",
        "User-Agent": "Bonghos/notification-bot",
      },
      body: JSON.stringify([discordCommandDefinition()]),
    });
  } catch {
    throw new Error("Discord command registration failed");
  }
  const text = await response.text();
  if (!response.ok) {
    let detail = "";
    try { detail = String(JSON.parse(text)?.message || "").slice(0, 200); } catch { /* ignore */ }
    throw new Error(`Discord command registration returned HTTP ${response.status}${detail ? `: ${detail}` : ""}`);
  }
  try { return JSON.parse(text); } catch { return []; }
}

// True when the interaction member has the Administrator or Manage Guild bit.
// Discord sends member.permissions as a decimal string bitfield.
function discordMemberIsAdministrator(interaction) {
  let bits;
  try { bits = BigInt(String(interaction?.member?.permissions || "0") || "0"); }
  catch { return false; }
  const ADMINISTRATOR = 1n << 3n;
  const MANAGE_GUILD = 1n << 5n;
  return (bits & ADMINISTRATOR) === ADMINISTRATOR || (bits & MANAGE_GUILD) === MANAGE_GUILD;
}

function discordSubcommand(interaction) {
  const option = (interaction?.data?.options || []).find((entry) => entry?.type === 1);
  const name = String(option?.name || "").toLowerCase();
  return ["here", "where", "disconnect"].includes(name) ? name : "help";
}

// Pure handler: takes a bot record and a Discord INTERACTION_CREATE payload,
// mutates the bot's destinations, and returns the reply content string. The
// caller is responsible for delivering the reply over whatever transport it has.
// Ephemeral replies (flags 64) keep configuration chatter private to the admin.
function handleDiscordInteraction(bot, interaction) {
  // type 1 == PING (gateway/endpoint liveness), type 2 == APPLICATION_COMMAND.
  if (interaction?.type !== 2 || interaction?.data?.name !== "bonghos") return null;
  if (!interaction.guild_id || !interaction.channel_id) {
    return { content: "Bonghos notifications can only be configured inside a server channel.", flags: 64 };
  }
  const guildID = String(interaction.guild_id);
  if (interaction.guild && guildID) {
    const previous = bot.discoveredGroups.get(guildID) || {};
    bot.discoveredGroups.set(guildID, {
      ...previous,
      id: guildID, name: String(interaction.guild.name || guildID), type: "guild",
      guild_id: guildID, guild_name: String(interaction.guild.name || guildID),
      guild_icon: String(interaction.guild.icon || ""),
      discovered_at: previous.discovered_at || new Date().toISOString(),
    });
    syncDevelopmentDiscoveries(bot);
  }
  const channelID = String(interaction.channel_id);
  const command = discordSubcommand(interaction);
  if (!discordMemberIsAdministrator(interaction)) {
    return { content: "Only a server administrator can use Bonghos configuration commands.", flags: 64 };
  }
  if (command === "help") {
    return { content: "Bonghos commands:\n`/bonghos here`: Send notifications to this channel\n`/bonghos where`: Check this channel's status\n`/bonghos disconnect`: Stop notifications here", flags: 64 };
  }
  const existingIndex = bot.destinations.findIndex((destination) => destination.id === channelID);
  if (command === "here") {
    if (existingIndex < 0 && bot.destinations.length >= 3) {
      return { content: "Bonghos could not connect this channel: Discord already has three connected channels.", flags: 64 };
    }
    const channelName = String(interaction.channel?.name || "").trim();
    const destination = {
      id: channelID, name: channelName || `Channel ${channelID}`, type: "channel",
      guild_id: String(interaction.guild_id),
      guild_name: String(interaction.guild?.name || "").trim(),
      guild_icon: String(interaction.guild?.icon || "").trim(),
    };
    if (existingIndex >= 0) bot.destinations[existingIndex] = destination;
    else bot.destinations.push(destination);
    bot.destination = bot.destinations[0]?.id || "";
    bot.public.destination_id = bot.destination;
    bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
    return { content: "Bonghos notifications will be sent to this channel.", flags: 64 };
  }
  if (command === "disconnect") {
    bot.destinations = bot.destinations.filter((destination) => destination.id !== channelID);
    bot.destination = bot.destinations[0]?.id || "";
    bot.public.destination_id = bot.destination;
    bot.public.destinations = bot.destinations.map((value) => ({ ...value }));
    return { content: "Bonghos notifications are disconnected from this channel.", flags: 64 };
  }
  // where
  const destination = bot.destinations.find((value) => value.id === channelID);
  return {
    content: destination
      ? "Bonghos notifications are configured for this channel."
      : "This channel is not connected. Run /bonghos here in the channel that should receive notifications.",
    flags: 64,
  };
}

async function respondDiscordInteraction(interaction, reply, fetchImpl = fetch) {
  const interactionID = String(interaction?.id || "");
  const interactionToken = String(interaction?.token || "");
  if (!/^\d{10,25}$/.test(interactionID) || !interactionToken) {
    throw new Error("Discord interaction response is invalid");
  }
  let response;
  try {
    response = await fetchImpl(`${DISCORD_API}/interactions/${encodeURIComponent(interactionID)}/${encodeURIComponent(interactionToken)}/callback`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "User-Agent": "Bonghos/notification-bot" },
      body: JSON.stringify({ type: 4, data: reply }),
    });
  } catch {
    throw new Error("Discord interaction response failed");
  }
  if (!response.ok) {
    try { await response.text(); } catch { /* discard provider response */ }
    throw new Error(`Discord interaction response returned HTTP ${response.status}`);
  }
}

// Connect to Discord's outbound Gateway so loopback-only development can
// receive slash-command interactions without exposing a public HTTPS endpoint.
// The returned stop function closes the socket and cancels reconnects.
function startDevelopmentDiscordCommands(bots, options = {}) {
  const bot = bots.find((candidate) => candidate.provider === "discord");
  if (!bot) return () => {};
  const fetchImpl = options.fetchImpl || fetch;
  const WebSocketImpl = options.WebSocketImpl || globalThis.WebSocket;
  const log = options.log || ((message) => console.warn(message));
  const setTimer = options.setTimeoutImpl || setTimeout;
  const clearTimer = options.clearTimeoutImpl || clearTimeout;
  const random = options.random || Math.random;
  if (typeof WebSocketImpl !== "function") {
    log("Discord gateway unavailable: this Node version has no WebSocket client");
    return () => {};
  }

  let socket = null;
  let stopped = false;
  let reconnectTimer = null;
  let heartbeatTimer = null;
  let heartbeatInterval = 0;
  let heartbeatAcknowledged = true;
  let sequence = null;
  let sessionID = "";
  let resumeGatewayURL = "";
  let applicationID = "";
  let reconnectAttempt = 0;
  const registeredGuilds = new Set();
  const guilds = new Map();

  const clearTimers = () => {
    if (reconnectTimer) clearTimer(reconnectTimer);
    if (heartbeatTimer) clearTimer(heartbeatTimer);
    reconnectTimer = null;
    heartbeatTimer = null;
  };
  const send = (payload) => {
    if (socket?.readyState === WebSocketImpl.OPEN || socket?.readyState === 1) {
      socket.send(JSON.stringify(payload));
    }
  };
  const heartbeat = () => {
    if (stopped || !socket) return;
    if (!heartbeatAcknowledged) {
      try { socket.close(4000, "heartbeat timeout"); } catch { /* reconnect on close */ }
      return;
    }
    heartbeatAcknowledged = false;
    send({ op: 1, d: sequence });
    heartbeatTimer = setTimer(heartbeat, heartbeatInterval);
  };
  const startHeartbeat = (interval) => {
    heartbeatInterval = Math.max(1000, Number(interval) || 45000);
    heartbeatAcknowledged = true;
    if (heartbeatTimer) clearTimer(heartbeatTimer);
    heartbeatTimer = setTimer(heartbeat, Math.floor(random() * heartbeatInterval));
  };
  const registerGuild = async (guildID) => {
    guildID = String(guildID || "");
    if (!/^\d{10,25}$/.test(guildID) || registeredGuilds.has(guildID)) return;
    try {
      await registerDiscordCommands(bot.token, applicationID, guildID, fetchImpl);
      registeredGuilds.add(guildID);
      bot.public.registered_guild_count = registeredGuilds.size;
    } catch (error) {
      log(`Discord command registration for guild failed: ${error.message}`);
    }
  };
  const scheduleReconnect = () => {
    if (stopped || reconnectTimer) return;
    const delay = Math.min(30000, 1000 * (2 ** Math.min(reconnectAttempt, 5)));
    reconnectAttempt += 1;
    reconnectTimer = setTimer(() => {
      reconnectTimer = null;
      connect();
    }, delay);
  };
  const connect = () => {
    if (stopped) return;
    const gateway = (resumeGatewayURL || "wss://gateway.discord.gg").replace(/\/+$/, "");
    try {
      socket = new WebSocketImpl(`${gateway}/?v=10&encoding=json`);
    } catch {
      log("Discord gateway connection failed");
      scheduleReconnect();
      return;
    }
    socket.addEventListener("message", (event) => {
      let packet;
      try { packet = JSON.parse(String(event.data)); } catch { return; }
      if (Number.isInteger(packet.s)) sequence = packet.s;
      if (packet.op === 10) {
        startHeartbeat(packet.d?.heartbeat_interval);
        if (sessionID && sequence !== null) {
          send({ op: 6, d: { token: bot.token, session_id: sessionID, seq: sequence } });
        } else {
          send({ op: 2, d: {
            token: bot.token,
            intents: 1,
            properties: { os: process.platform, browser: "bonghos", device: "bonghos" },
          } });
        }
        return;
      }
      if (packet.op === 1) {
        send({ op: 1, d: sequence });
        return;
      }
      if (packet.op === 11) {
        heartbeatAcknowledged = true;
        return;
      }
      if (packet.op === 7) {
        try { socket.close(4000, "server reconnect"); } catch { scheduleReconnect(); }
        return;
      }
      if (packet.op === 9) {
        if (!packet.d) {
          sessionID = "";
          sequence = null;
          resumeGatewayURL = "";
        }
        try { socket.close(4000, "invalid session"); } catch { scheduleReconnect(); }
        return;
      }
      if (packet.op !== 0) return;
      if (packet.t === "READY") {
        reconnectAttempt = 0;
        bot.public.gateway_connected = true;
        sessionID = String(packet.d?.session_id || "");
        resumeGatewayURL = String(packet.d?.resume_gateway_url || "");
        const readyApplicationID = String(packet.d?.application?.id || packet.d?.user?.id || "");
        if (/^\d{10,25}$/.test(readyApplicationID)) applicationID = readyApplicationID;
        for (const guild of Array.isArray(packet.d?.guilds) ? packet.d.guilds : []) void registerGuild(guild.id);
      } else if (packet.t === "GUILD_CREATE") {
        const guildID = String(packet.d?.id || "");
        if (guildID) {
          const guild = {
            id: guildID,
            name: String(packet.d?.name || ""),
            icon: String(packet.d?.icon || ""),
          };
          guilds.set(guildID, guild);
          const previous = bot.discoveredGroups.get(guildID) || {};
          bot.discoveredGroups.set(guildID, {
            ...previous,
            id: guildID, name: guild.name, type: "guild",
            guild_id: guildID, guild_name: guild.name, guild_icon: guild.icon,
            discovered_at: previous.discovered_at || new Date().toISOString(),
          });
          syncDevelopmentDiscoveries(bot);
        }
        void registerGuild(packet.d?.id);
      } else if (packet.t === "GUILD_DELETE" && !packet.d?.unavailable) {
        const guildID = String(packet.d?.id || "");
        guilds.delete(guildID);
        bot.discoveredGroups.delete(guildID);
        bot.destinations = bot.destinations.filter((destination) => String(destination.guild_id || "") !== guildID);
        bot.destination = bot.destinations[0]?.id || "";
        bot.public.destination_id = bot.destination;
        bot.public.destinations = bot.destinations.map((destination) => ({ ...destination }));
        syncDevelopmentDiscoveries(bot);
      } else if (packet.t === "INTERACTION_CREATE") {
        const guild = guilds.get(String(packet.d?.guild_id || ""));
        if (guild) packet.d.guild = guild;
        const reply = handleDiscordInteraction(bot, packet.d);
        if (reply) void respondDiscordInteraction(packet.d, reply, fetchImpl)
          .catch((error) => log(`Discord interaction reply failed: ${error.message}`));
      }
    });
    socket.addEventListener("close", () => {
      bot.public.gateway_connected = false;
      if (heartbeatTimer) clearTimer(heartbeatTimer);
      heartbeatTimer = null;
      socket = null;
      scheduleReconnect();
    });
    socket.addEventListener("error", () => {
      // The close event owns retry scheduling. Never stringify the event;
      // provider errors may contain the credential-bearing gateway state.
    });
  };

  void resolveDiscordApplication(bot.token, fetchImpl).then((application) => {
    if (stopped) return;
    applicationID = application.applicationID;
    bot.public.provider_username = application.username;
    bot.public.gateway_connected = false;
    bot.public.registered_guild_count = 0;
    connect();
    void registerDiscordCommands(bot.token, applicationID, "", fetchImpl).then(() => {
      bot.public.global_command_registered = true;
    }).catch((error) => {
      bot.public.global_command_registered = false;
      log(`Discord global command registration failed: ${error.message}`);
    });
  }).catch((error) => log(`Discord setup failed: ${error.message}`));

  return () => {
    stopped = true;
    clearTimers();
    if (socket) {
      try { socket.close(1000, "development relay stopped"); } catch { /* already closed */ }
    }
    socket = null;
  };
}

function jsonResponse(response, status, payload) {
  const body = JSON.stringify(payload);
  response.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Content-Length": Buffer.byteLength(body),
    "Cache-Control": "no-store",
    "X-Content-Type-Options": "nosniff",
  });
  response.end(body);
}

function readJSON(request, limit = 64 * 1024) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    request.on("data", (chunk) => {
      size += chunk.length;
      if (size > limit) {
        reject(new Error("request body is too large"));
        request.destroy();
        return;
      }
      chunks.push(chunk);
    });
    request.on("end", () => {
      try {
        resolve(chunks.length ? JSON.parse(Buffer.concat(chunks).toString("utf8")) : {});
      } catch {
        reject(new Error("invalid JSON request"));
      }
    });
    request.on("error", reject);
  });
}

function contentType(filePath) {
  switch (path.extname(filePath).toLowerCase()) {
  case ".html": return "text/html; charset=utf-8";
  case ".js": return "text/javascript; charset=utf-8";
  case ".css": return "text/css; charset=utf-8";
  case ".png": return "image/png";
  case ".svg": return "image/svg+xml";
  case ".txt": return "text/plain; charset=utf-8";
  default: return "application/octet-stream";
  }
}

function createDevServer(options = {}) {
  const env = options.env || loadDevelopmentEnv(options.envPath);
  const bots = options.bots || configuredBots(env);
  const botByID = new Map(bots.map((bot) => [bot.id, bot]));
  const fetchImpl = options.fetchImpl || fetch;
  const staticRoot = options.staticRoot || STATIC_ROOT;

  return http.createServer(async (request, response) => {
    const requestURL = new URL(request.url, "http://127.0.0.1");
    if (requestURL.pathname.startsWith("/__dev/bots")) {
      if (request.headers[RELAY_HEADER] !== "1") {
        jsonResponse(response, 403, { error: "development relay header required" });
        return;
      }
      try {
        if (request.method === "GET" && requestURL.pathname === "/__dev/bots") {
          jsonResponse(response, 200, bots.map((bot) => ({ ...bot.public })));
          return;
        }
        const newDiscovery = requestURL.pathname === "/__dev/bots/telegram/discover";
        const existingDiscovery = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)\/telegram\/discover$/);
        if (request.method === "POST" && (newDiscovery || existingDiscovery)) {
          await readJSON(request);
          const bot = existingDiscovery
            ? botByID.get(Number(existingDiscovery[1]))
            : bots.find((candidate) => candidate.provider === "telegram");
          if (!bot || bot.provider !== "telegram") throw new Error("Telegram development bot is not configured");
          const discovery = await discoverTelegramGroups(bot, fetchImpl);
          if (!bot.discoveredGroups) bot.discoveredGroups = new Map();
          for (const group of discovery.groups) {
            const previous = bot.discoveredGroups.get(group.id) || {};
            const topics = uniqueTelegramTopics([...(previous.topics || []), ...(group.topics || [])]);
            bot.discoveredGroups.set(group.id, {
              ...previous, ...group,
              photo_file_id: group.photo_file_id || previous.photo_file_id || "",
              photo_data_url: group.photo_data_url || previous.photo_data_url || "",
              forum: !!(group.forum || previous.forum || topics.length),
              discovered_at: previous.discovered_at || new Date().toISOString(),
              topics,
            });
          }
          discovery.groups = Array.from(bot.discoveredGroups.values())
            .sort((left, right) => left.name.localeCompare(right.name));
          syncDevelopmentDiscoveries(bot);
          bot.public.destinations = bot.public.destinations.map((destination) => ({
            ...destination,
            ...(bot.discoveredGroups.get(destination.id) || {}),
          }));
          jsonResponse(response, 200, discovery);
          return;
        }
        const testMatch = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)\/test$/);
        if (request.method === "POST" && testMatch) {
          const bot = botByID.get(Number(testMatch[1]));
          if (!bot) throw new Error("development bot is not configured");
          if (!bot.public.enabled) throw new Error("notification bot is disabled");
          await readJSON(request);
          for (const destination of bot.destinations) {
            await providerRequest(bot, `Bonghos local development test\n${bot.name} is connected.`, fetchImpl, destination.id, destination.thread_id);
          }
          jsonResponse(response, 200, { ok: true });
          return;
        }
        const inviteMatch = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)\/invite$/);
        if (request.method === "GET" && inviteMatch) {
          const bot = botByID.get(Number(inviteMatch[1]));
          if (!bot) throw new Error("development bot is not configured");
          jsonResponse(response, 200, { url: await botInviteURL(bot, fetchImpl) });
          return;
        }
        const patchMatch = requestURL.pathname.match(/^\/__dev\/bots\/(\d+)$/);
        if (request.method === "PATCH" && patchMatch) {
          const bot = botByID.get(Number(patchMatch[1]));
          if (!bot) throw new Error("development bot is not configured");
          const patch = await readJSON(request);
          for (const field of ["enabled", "notify_server_started", "notify_server_stopped", "notify_player_joined", "notify_player_left"]) {
            if (typeof patch[field] === "boolean") bot.public[field] = patch[field];
          }
          if (typeof patch.name === "string" && patch.name.trim()) {
            bot.name = patch.name.trim().slice(0, 80);
            bot.public.name = bot.name;
          }
          if (Array.isArray(patch.destinations)) {
            if (bot.provider !== "telegram" || patch.destinations.length < 1 || patch.destinations.length > 3) {
              throw new Error("Telegram requires between 1 and 3 destination groups");
            }
            const destinations = patch.destinations.map((destination) => {
              const id = String(destination.id || "").trim();
              const discovered = bot.discoveredGroups?.get(id) || {};
              return {
                id,
                name: String(destination.name || discovered.name || destination.id || "").trim(),
                type: String(destination.type || discovered.type || "group").trim(),
                photo_file_id: String(destination.photo_file_id || discovered.photo_file_id || "").trim(),
                photo_data_url: String(discovered.photo_data_url || ""),
                forum: !!(destination.forum || discovered.forum),
                thread_id: Number(destination.thread_id) || 0,
                thread_name: String(destination.thread_name || "").trim(),
                topics: Array.isArray(discovered.topics) ? discovered.topics.map((topic) => ({ ...topic })) : [],
              };
            });
            if (destinations.some((destination) => !/^(?:-?\d+|@[A-Za-z0-9_]{5,})$/.test(destination.id))) {
              throw new Error("Telegram chat IDs must be numeric or public @channel usernames");
            }
            if (new Set(destinations.map((destination) => destination.id)).size !== destinations.length) {
              throw new Error("Telegram destination groups must be unique");
            }
            bot.destinations = destinations;
            bot.destination = destinations[0].id;
            bot.public.destinations = destinations.map((destination) => ({ ...destination }));
            bot.public.destination_id = destinations[0].id;
          }
          jsonResponse(response, 200, { ...bot.public });
          return;
        }
        jsonResponse(response, 405, { error: "edit .env.development and restart the development relay" });
      } catch (error) {
        jsonResponse(response, 400, { error: error.message || "development relay request failed" });
      }
      return;
    }

    if (request.method !== "GET" && request.method !== "HEAD") {
      response.writeHead(405, { "Content-Type": "text/plain; charset=utf-8" });
      response.end("Method not allowed");
      return;
    }
    let relativePath;
    try {
      relativePath = decodeURIComponent(requestURL.pathname);
    } catch {
      response.writeHead(400, { "Content-Type": "text/plain; charset=utf-8" });
      response.end("Invalid path");
      return;
    }
    if (relativePath === "/") relativePath = "/index.html";
    const filePath = path.resolve(staticRoot, "." + relativePath);
    const rootPrefix = path.resolve(staticRoot) + path.sep;
    if (!filePath.startsWith(rootPrefix)) {
      response.writeHead(404);
      response.end();
      return;
    }
    fs.readFile(filePath, (error, data) => {
      if (error) {
        response.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
        response.end("Not found");
        return;
      }
      response.writeHead(200, {
        "Content-Type": contentType(filePath),
        "Content-Length": data.length,
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      });
      if (request.method === "HEAD") response.end();
      else response.end(data);
    });
  });
}

if (require.main === module) {
  try {
    const env = loadDevelopmentEnv();
    const port = Number.parseInt(env.BONGHOS_DEV_PORT || "8000", 10);
    if (!Number.isInteger(port) || port < 1024 || port > 65535) {
      throw new Error("BONGHOS_DEV_PORT must be between 1024 and 65535");
    }
    const bots = configuredBots(env);
    const server = createDevServer({ env, bots });
    let stopTelegramCommands = () => {};
    let stopDiscordCommands = () => {};
    server.on("close", () => {
      stopTelegramCommands();
      stopDiscordCommands();
    });
    server.on("error", (error) => {
      stopTelegramCommands();
      stopDiscordCommands();
      if (error.code === "EADDRINUSE") {
        console.error(`Development server failed: port ${port} is already in use. Stop the existing server or set BONGHOS_DEV_PORT to another port.`);
      } else {
        console.error(`Development server failed: ${error.message}`);
      }
      process.exitCode = 1;
    });
    server.listen(port, "127.0.0.1", () => {
      stopTelegramCommands = startDevelopmentTelegramCommands(bots);
      stopDiscordCommands = startDevelopmentDiscordCommands(bots);
      const providers = bots.map((bot) => bot.provider).join(" and ") || "no providers";
      console.log(`Bonghos development Web UI: http://127.0.0.1:${port}/?demo&debug-bots`);
      console.log(`Configured from .env.development: ${providers}`);
      if (!bots.length) console.log("Add one Telegram and/or one Discord credential pair, then restart this command.");
    });
  } catch (error) {
    console.error(`Development server failed: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = {
  parseEnv, configuredBots, providerRequest, discoverTelegramGroups,
  pollDevelopmentTelegramCommands, createDevServer,
  resolveDiscordApplication, discordCommandDefinition, registerDiscordCommands,
  handleDiscordInteraction, respondDiscordInteraction, startDevelopmentDiscordCommands, botInviteURL,
};
