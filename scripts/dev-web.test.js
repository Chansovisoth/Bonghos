"use strict";

const assert = require("node:assert/strict");

const {
  parseEnv, configuredBots, providerRequest, discoverTelegramGroups,
  pollDevelopmentTelegramCommands, createDevServer,
  resolveDiscordApplication, discordCommandDefinition, registerDiscordCommands,
  handleDiscordInteraction, respondDiscordInteraction, startDevelopmentDiscordCommands,
  botInviteURL,
} = require("./dev-web.js");

const tests = [];

function test(name, run) {
  tests.push({ name, run });
}

test("parseEnv accepts the local development format", () => {
  const parsed = parseEnv(`
    # ignored
    BONGHOS_DEV_TELEGRAM_TOKEN="123456:token_value"
    BONGHOS_DEV_TELEGRAM_CHAT_ID=-1001234567890
  `);
  assert.equal(parsed.BONGHOS_DEV_TELEGRAM_TOKEN, "123456:token_value");
  assert.equal(parsed.BONGHOS_DEV_TELEGRAM_CHAT_ID, "-1001234567890");
});

test("configuredBots exposes one sanitized Telegram and Discord entry", () => {
  const bots = configuredBots({
    BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret",
    BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token",
  });
  assert.equal(bots.length, 2);
  const encodedPublic = JSON.stringify(bots.map((bot) => bot.public));
  assert.equal(encodedPublic.includes("telegram_secret"), false);
  assert.equal(encodedPublic.includes("discord.secret.token"), false);
  assert.deepEqual(bots.map((bot) => bot.provider), ["telegram", "discord"]);
  assert.equal(bots[0].public.destinations.length, 0);
  assert.equal(bots[1].public.destinations.length, 0);
});

test("configuredBots ignores legacy Telegram destination seeds", () => {
  const bots = configuredBots({
    BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret",
    BONGHOS_DEV_TELEGRAM_CHAT_ID: "-1001111111111,-1002222222222,-1003333333333",
  });
  assert.deepEqual(bots[0].destinations, []);
  assert.equal(bots[0].public.destination_id, "");
});

test("configuredBots allows an empty development environment", () => {
  assert.deepEqual(configuredBots({}), []);
});

test("configuredBots accepts a Telegram token before /bonghos here connects a group", () => {
  const bots = configuredBots({ BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret" });
  assert.equal(bots.length, 1);
  assert.deepEqual(bots[0].destinations, []);
});

test("configuredBots accepts a Discord token before /bonghos here connects a channel", () => {
  const bots = configuredBots({ BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token" });
  assert.equal(bots.length, 1);
  assert.deepEqual(bots[0].destinations, []);
});

test("development /bonghos here command connects one topic and advances its cursor", async () => {
  const bot = configuredBots({ BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret" })[0];
  bot.commandInitialized = true;
  const calls = [];
  const fakeFetch = async (url, options = {}) => {
    calls.push({ url, options });
    let result = {};
    if (url.includes("/getUpdates?")) result = [{
      update_id: 18,
      message: { text: "/bonghos here", message_thread_id: 42, from: { id: 7 },
        chat: { id: -1002, type: "supergroup", title: "Projects", is_forum: true } },
    }];
    else if (url.includes("/getChatMember?")) result = { status: "administrator" };
    return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result }) };
  };
  await pollDevelopmentTelegramCommands(bot, fakeFetch);
  assert.equal(bot.lastUpdateID, 18);
  assert.equal(bot.destinations.length, 1);
  assert.equal(bot.destinations[0].thread_id, 42);
  const acknowledgement = calls.find((call) => call.url.endsWith("/sendMessage"));
  assert.equal(JSON.parse(acknowledgement.options.body).message_thread_id, 42);
});

test("development relay shows Telegram membership before a destination is configured", async () => {
  const bot = configuredBots({ BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret" })[0];
  bot.commandInitialized = true;
  let updateCalls = 0;
  const fakeFetch = async (url) => {
    let result = {};
    if (url.includes("/getUpdates?")) {
      updateCalls += 1;
      result = updateCalls === 1
        ? [{ update_id: 30, my_chat_member: { chat: { id: -1007, type: "supergroup", title: "Operators" }, new_chat_member: { status: "administrator" } } }]
        : [{ update_id: 31, my_chat_member: { chat: { id: -1007, type: "supergroup", title: "Operators" }, new_chat_member: { status: "restricted", is_member: false } } }];
    } else if (url.includes("/getChat?")) {
      result = { id: -1007, type: "supergroup", title: "Operators" };
    }
    return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result }) };
  };
  await pollDevelopmentTelegramCommands(bot, fakeFetch);
  assert.equal(bot.destinations.length, 0);
  assert.equal(bot.public.discovered_destinations.length, 1);
  assert.equal(bot.public.discovered_destinations[0].name, "Operators");
  assert.match(bot.public.discovered_destinations[0].discovered_at, /^\d{4}-\d{2}-\d{2}T/);
  bot.destinations.push({ id: "-1007", name: "Operators", type: "supergroup" });
  bot.destination = "-1007";
  bot.public.destination_id = "-1007";
  bot.public.destinations = bot.destinations.map((destination) => ({ ...destination }));
  await pollDevelopmentTelegramCommands(bot, fakeFetch);
  assert.equal(bot.public.discovered_destinations.length, 0);
  assert.equal(bot.public.destinations.length, 0);
});

test("providerRequest sends Telegram token server-side only", async () => {
  const calls = [];
  const fakeFetch = async (url, options) => {
    calls.push({ url, options });
    return { ok: true, status: 200, text: async () => `{}` };
  };
  await providerRequest({
    provider: "telegram",
    token: "123456:telegram_secret",
    destination: "-1001234567890",
  }, "test message", fakeFetch, "-1001234567890", 42);
  assert.equal(calls.length, 1);
  assert.equal(JSON.parse(calls[0].options.body).chat_id, "-1001234567890");
  assert.equal(JSON.parse(calls[0].options.body).message_thread_id, 42);
  assert.equal(calls[0].options.body.includes("telegram_secret"), false);
});

test("Discord token resolution and command registration keep credentials server-side", async () => {
  const calls = [];
  const fakeFetch = async (url, options = {}) => {
    calls.push({ url, options });
    if (url.endsWith("/users/@me")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ id: "1536799744431755275", username: "Bonghos", bot: true }) };
    }
    return { ok: true, status: 200, text: async () => "[]" };
  };
  const resolved = await resolveDiscordApplication("discord.secret.token.value", fakeFetch);
  assert.equal(resolved.applicationID, "1536799744431755275");
  await registerDiscordCommands("discord.secret.token.value", resolved.applicationID, "123456789012345678", fakeFetch);
  const registration = calls[1];
  assert.match(registration.url, /\/guilds\/123456789012345678\/commands$/);
  assert.equal(registration.options.headers.Authorization, "Bot discord.secret.token.value");
  const definition = JSON.parse(registration.options.body)[0];
  assert.deepEqual(definition.options.map((option) => option.name), ["here", "where", "disconnect", "help"]);
  assert.equal(definition.default_member_permissions, "32");
  assert.equal(JSON.stringify(definition).includes("discord.secret"), false);
});

test("Discord identity and permission parsing fail closed", async () => {
  await assert.rejects(() => resolveDiscordApplication("discord.secret.token.value", async () => ({
    ok: true, status: 200,
    text: async () => JSON.stringify({ id: "1536799744431755275", username: "Not a bot", bot: false }),
  })), /bot application/);
  const bot = configuredBots({ BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token" })[0];
  const reply = handleDiscordInteraction(bot, {
    type: 2, guild_id: "123456789012345678", channel_id: "223456789012345678",
    member: { permissions: "not-a-number" }, channel: { name: "alerts" },
    data: { name: "bonghos", options: [{ type: 1, name: "here" }] },
  });
  assert.match(reply.content, /administrator/);
  assert.equal(bot.destinations.length, 0);
});

test("botInviteURL builds provider-standard links without exposing tokens", async () => {
  const telegram = configuredBots({ BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret" })[0];
  const telegramURL = await botInviteURL(telegram, async () => ({
    ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: { username: "bonghos_test_bot" } }),
  }));
  assert.equal(telegramURL, "https://t.me/bonghos_test_bot?startgroup&admin=manage_chat");

  const discord = configuredBots({ BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token" })[0];
  const discordURL = await botInviteURL(discord, async () => ({
    ok: true, status: 200, text: async () => JSON.stringify({ id: "1536799744431755275", username: "Bonghos", bot: true }),
  }));
  assert.match(discordURL, /client_id=1536799744431755275/);
  assert.match(discordURL, /permissions=274877910016/);
  assert.equal(discordURL.includes("discord.secret"), false);
});

test("Discord interaction handler enforces administrators and updates destinations", () => {
  const bot = configuredBots({ BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token" })[0];
  const base = {
    type: 2, guild_id: "123456789012345678", channel_id: "223456789012345678",
    data: { name: "bonghos", options: [{ type: 1, name: "here" }] },
    channel: { name: "alerts" },
  };
  const denied = handleDiscordInteraction(bot, { ...base, member: { permissions: "0" } });
  assert.match(denied.content, /administrator/);
  assert.equal(bot.destinations.length, 0);
  const deniedWhere = handleDiscordInteraction(bot, {
    ...base, data: { name: "bonghos", options: [{ type: 1, name: "where" }] }, member: { permissions: "0" },
  });
  assert.match(deniedWhere.content, /administrator/);
  const connected = handleDiscordInteraction(bot, { ...base, member: { permissions: "8" } });
  assert.match(connected.content, /will be sent/);
  assert.equal(bot.destinations[0].name, "alerts");
  const where = handleDiscordInteraction(bot, {
    ...base, data: { name: "bonghos", options: [{ type: 1, name: "where" }] }, member: { permissions: "32" },
  });
  assert.match(where.content, /configured/);
});

test("Discord gateway identifies, registers guild commands, and answers interactions", async () => {
  class FakeWebSocket {
    static OPEN = 1;
    static instances = [];
    constructor(url) {
      this.url = url;
      this.readyState = 1;
      this.listeners = new Map();
      this.sent = [];
      FakeWebSocket.instances.push(this);
    }
    addEventListener(name, listener) { this.listeners.set(name, listener); }
    send(value) { this.sent.push(JSON.parse(value)); }
    close() { this.readyState = 3; this.listeners.get("close")?.({}); }
    emit(payload) { this.listeners.get("message")?.({ data: JSON.stringify(payload) }); }
  }
  const calls = [];
  const fakeFetch = async (url, options = {}) => {
    calls.push({ url, options });
    if (url.endsWith("/users/@me")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ id: "1536799744431755275", username: "Bonghos", bot: true }) };
    }
    return { ok: true, status: 200, text: async () => "[]" };
  };
  const timers = [];
  const bot = configuredBots({ BONGHOS_DEV_DISCORD_TOKEN: "discord.secret.token" })[0];
  const stop = startDevelopmentDiscordCommands([bot], {
    fetchImpl: fakeFetch, WebSocketImpl: FakeWebSocket,
    setTimeoutImpl: (run) => { const timer = { run }; timers.push(timer); return timer; },
    clearTimeoutImpl: () => {}, random: () => 0.5, log: () => {},
  });
  await new Promise((resolve) => setImmediate(resolve));
  const socket = FakeWebSocket.instances[0];
  assert.ok(socket);
  assert.ok(calls.some((call) => /\/applications\/1536799744431755275\/commands$/.test(call.url)));
  socket.emit({ op: 10, d: { heartbeat_interval: 45000 } });
  assert.equal(socket.sent[0].op, 2);
  assert.equal(socket.sent[0].d.intents, 1);
  socket.emit({ op: 0, t: "READY", s: 1, d: {
    session_id: "session", resume_gateway_url: "wss://resume.discord.gg",
    application: { id: "1536799744431755275" }, guilds: [{ id: "123456789012345678" }],
  } });
  await new Promise((resolve) => setImmediate(resolve));
  assert.ok(calls.some((call) => call.url.includes("/guilds/123456789012345678/commands")));
  socket.emit({ op: 0, t: "GUILD_CREATE", s: 2, d: {
    id: "123456789012345678", name: "Bonghos Lab", icon: "guild-icon-hash",
  } });
  assert.equal(bot.destinations.length, 0);
  assert.equal(bot.public.discovered_destinations.length, 1);
  assert.equal(bot.public.discovered_destinations[0].guild_name, "Bonghos Lab");
  const discoveredAt = bot.public.discovered_destinations[0].discovered_at;
  assert.match(discoveredAt, /^\d{4}-\d{2}-\d{2}T/);
  socket.emit({ op: 0, t: "INTERACTION_CREATE", s: 3, d: {
    id: "323456789012345678", token: "interaction-token", type: 2,
    guild_id: "123456789012345678", channel_id: "223456789012345678",
    member: { permissions: "8" }, channel: { name: "alerts" },
    data: { name: "bonghos", options: [{ type: 1, name: "here" }] },
  } });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(bot.public.discovered_destinations[0].discovered_at, discoveredAt);
  const callback = calls.find((call) => call.url.includes("/interactions/323456789012345678/"));
  assert.ok(callback);
  assert.equal(JSON.parse(callback.options.body).data.flags, 64);
  assert.equal(bot.destinations[0].id, "223456789012345678");
  assert.equal(bot.destinations[0].guild_name, "Bonghos Lab");
  assert.equal(bot.destinations[0].guild_icon, "guild-icon-hash");
  socket.emit({ op: 0, t: "GUILD_DELETE", s: 4, d: {
    id: "123456789012345678", unavailable: false,
  } });
  assert.equal(bot.public.discovered_destinations.length, 0);
  assert.equal(bot.public.destinations.length, 0);
  stop();
});

test("discoverTelegramGroups returns unique non-private Telegram groups", async () => {
  const fakeFetch = async (url) => {
    if (url.endsWith("/getMe")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: { username: "bonghos_test_bot" } }) };
    }
    return {
      ok: true,
      status: 200,
      text: async () => JSON.stringify({
        ok: true,
        result: [
          { message: { chat: { id: 5, type: "private", username: "owner" } } },
          { message: { chat: { id: -1002, type: "supergroup", title: "Projects" } } },
          { message: { message_id: 8, message_thread_id: 7, is_topic_message: true, chat: { id: -1002, type: "supergroup", title: "Projects", is_forum: true }, reply_to_message: { forum_topic_created: { name: "Announcements" } } } },
          { message: { message_id: 9, message_thread_id: 7, is_topic_message: true, chat: { id: -1002, type: "supergroup", title: "Projects", is_forum: true }, reply_to_message: { forum_topic_created: { name: "Announcements" } } } },
          { message: { message_id: 10, message_thread_id: 8, is_topic_message: true, chat: { id: -1002, type: "supergroup", title: "Projects", is_forum: true }, reply_to_message: { forum_topic_created: { name: "Announcements\u200b" } } } },
          { my_chat_member: { chat: { id: -1001, type: "group", title: "Alerts" } } },
          { my_chat_member: { chat: { id: -1003, type: "group", title: "Removed" }, new_chat_member: { status: "left" } } },
          { edited_message: { chat: { id: -1002, type: "supergroup", title: "Projects" } } },
        ],
      }),
    };
  };
  const result = await discoverTelegramGroups({ token: "123456:telegram_secret" }, fakeFetch);
  assert.equal(result.bot_username, "bonghos_test_bot");
  assert.deepEqual(result.groups.map((group) => group.name), ["Alerts", "Projects"]);
  assert.deepEqual(result.groups[1].topics, [{ id: 8, name: "Announcements" }]);
});

test("development relay retains groups across discovery refreshes", async () => {
  let updateCalls = 0;
  const fakeFetch = async (url) => {
    if (url.endsWith("/getMe")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: { username: "bonghos_test_bot" } }) };
    }
    if (url.includes("/getUpdates?")) {
      updateCalls++;
      const id = updateCalls === 1 ? -1001 : -1002;
      const title = updateCalls === 1 ? "First" : "Second";
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: [
        { message: { chat: { id, type: "group", title } } },
      ] }) };
    }
    if (url.includes("/getChat?")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: {} }) };
    }
    throw new Error(`unexpected URL ${url}`);
  };
  const server = createDevServer({
    env: {
      BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret",
      BONGHOS_DEV_TELEGRAM_CHAT_ID: "-1001",
    },
    fetchImpl: fakeFetch,
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  try {
    const baseURL = `http://127.0.0.1:${server.address().port}`;
    const options = {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-Bonghos-Dev-Relay": "1" },
      body: "{}",
    };
    const first = await fetch(`${baseURL}/__dev/bots/1/telegram/discover`, options);
    assert.equal((await first.json()).groups.length, 1);
    const second = await fetch(`${baseURL}/__dev/bots/1/telegram/discover`, options);
    assert.equal((await second.json()).groups.length, 2);
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
});

test("discoverTelegramGroups returns a group profile without exposing the token", async () => {
  const token = "123456:telegram_photo_secret";
  const fakeFetch = async (url) => {
    if (url.endsWith("/getMe")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: { username: "bonghos_test_bot" } }) };
    }
    if (url.includes("/getUpdates?")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: [
        { message: { chat: { id: -1001, type: "group", title: "Alerts" } } },
      ] }) };
    }
    if (url.includes("/getChat?")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: {
        id: -1001, type: "group", title: "Alerts", photo: { small_file_id: "small-photo" },
      } }) };
    }
    if (url.includes("/getFile?")) {
      return { ok: true, status: 200, text: async () => JSON.stringify({ ok: true, result: { file_path: "photos/group.png" } }) };
    }
    if (url.includes("/file/bot")) {
      return {
        ok: true, status: 200,
        headers: { get: () => "image/png" },
        arrayBuffer: async () => Buffer.from("profile image"),
      };
    }
    throw new Error(`unexpected URL ${url}`);
  };
  const result = await discoverTelegramGroups({ token }, fakeFetch);
  assert.equal(result.groups[0].photo_file_id, "small-photo");
  assert.match(result.groups[0].photo_data_url, /^data:image\/png;base64,/);
  assert.equal(JSON.stringify(result).includes(token), false);
});

test("development relay returns sanitized bots and requires its private header", async () => {
  const providerCalls = [];
  const server = createDevServer({
    env: {
      BONGHOS_DEV_TELEGRAM_TOKEN: "123456:telegram_secret",
      BONGHOS_DEV_TELEGRAM_CHAT_ID: "-1001234567890",
    },
    fetchImpl: async (url, options) => {
      providerCalls.push({ url, options });
      return { ok: true, status: 200, text: async () => `{}` };
    },
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  try {
    const address = server.address();
    const baseURL = `http://127.0.0.1:${address.port}`;

    const refused = await fetch(`${baseURL}/__dev/bots`);
    assert.equal(refused.status, 403);

    const listed = await fetch(`${baseURL}/__dev/bots`, {
      headers: { "X-Bonghos-Dev-Relay": "1" },
    });
    assert.equal(listed.status, 200);
    const listedText = await listed.text();
    assert.equal(listedText.includes("telegram_secret"), false);
    assert.equal(JSON.parse(listedText)[0].destination_id, "");

    const patched = await fetch(`${baseURL}/__dev/bots/1`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Bonghos-Dev-Relay": "1",
      },
      body: JSON.stringify({
        destinations: [
          { id: "-1001234567890", name: "Projects", type: "supergroup" },
          { id: "-1009876543210", name: "Staff", type: "supergroup" },
        ],
      }),
    });
    assert.equal(patched.status, 200);
    assert.equal((await patched.json()).destinations.length, 2);

    const sent = await fetch(`${baseURL}/__dev/bots/1/test`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Bonghos-Dev-Relay": "1",
      },
      body: `{}`,
    });
    assert.equal(sent.status, 200);
    assert.equal(providerCalls.length, 2);

    const disabledPatch = await fetch(`${baseURL}/__dev/bots/1`, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-Bonghos-Dev-Relay": "1",
      },
      body: JSON.stringify({ enabled: false }),
    });
    assert.equal(disabledPatch.status, 200);
    const disabledTest = await fetch(`${baseURL}/__dev/bots/1/test`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Bonghos-Dev-Relay": "1",
      },
      body: `{}`,
    });
    assert.equal(disabledTest.status, 400);
    assert.match((await disabledTest.json()).error, /disabled/i);
    assert.equal(providerCalls.length, 2);
  } finally {
    await new Promise((resolve) => server.close(resolve));
  }
});

async function runTests() {
  let failures = 0;
  for (const entry of tests) {
    try {
      await entry.run();
      console.log(`ok - ${entry.name}`);
    } catch (error) {
      failures += 1;
      console.error(`not ok - ${entry.name}`);
      console.error(error);
    }
  }
  if (failures > 0) {
    throw new Error(`${failures} development relay test(s) failed`);
  }
}

runTests().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
