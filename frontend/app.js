const state = {
  token: localStorage.getItem("crm_token") || "",
  user: JSON.parse(localStorage.getItem("crm_user") || "null"),
  socket: null,
};

const titles = {
  overview: ["Обзор системы", "Статус сервисов, быстрый демо-сценарий и realtime-события."],
  leads: ["Заявки и клиенты", "Проверка результата event-driven цепочки после создания заявки."],
  notifications: ["Уведомления", "События, созданные Notification Service после Kafka customer.created."],
  tools: ["Инструменты", "Быстрые ссылки на Swagger, Kafka UI, metrics и readiness."],
};

const services = [
  ["gatewayStatus", "/gateway/ready"],
  ["leadStatus", "/lead-service/ready"],
  ["customerStatus", "/customer-service/ready"],
  ["notificationStatus", "/notification-service/ready"],
];

document.addEventListener("DOMContentLoaded", () => {
  bindNavigation();
  bindActions();
  updateSession();
  refreshAll();
  if (state.token) {
    connectWebSocket();
  }
});

function bindNavigation() {
  document.querySelectorAll(".nav-item").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelectorAll(".nav-item").forEach((item) => item.classList.remove("active"));
      document.querySelectorAll(".view").forEach((view) => view.classList.remove("active"));
      button.classList.add("active");
      document.getElementById(button.dataset.view).classList.add("active");
      document.getElementById("viewTitle").textContent = titles[button.dataset.view][0];
      document.getElementById("viewSubtitle").textContent = titles[button.dataset.view][1];
    });
  });
}

function bindActions() {
  document.getElementById("refreshButton").addEventListener("click", refreshAll);
  document.getElementById("registerButton").addEventListener("click", register);
  document.getElementById("loginButton").addEventListener("click", login);
  document.getElementById("createLeadButton").addEventListener("click", createLead);
  document.getElementById("demoButton").addEventListener("click", runDemo);
  document.getElementById("reloadLeadsButton").addEventListener("click", loadLeadsAndCustomers);
  document.getElementById("reloadCustomersButton").addEventListener("click", loadLeadsAndCustomers);
  document.getElementById("reloadNotificationsButton").addEventListener("click", loadNotifications);
}

async function refreshAll() {
  await checkServices();
  if (state.token) {
    await Promise.allSettled([loadLeadsAndCustomers(), loadNotifications()]);
  }
}

async function checkServices() {
  await Promise.all(services.map(async ([elementId, url]) => {
    const badge = document.getElementById(elementId);
    badge.textContent = "checking";
    badge.className = "badge muted";
    try {
      const response = await fetch(url, { cache: "no-store" });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      badge.textContent = "ready";
      badge.className = "badge ok";
    } catch (error) {
      badge.textContent = "down";
      badge.className = "badge fail";
    }
  }));
}

async function register() {
  const body = authPayload();
  const data = await request("/auth/register", { method: "POST", body });
  setAuth(data);
  showResult("Registered", data);
}

async function login() {
  const body = authPayload();
  const data = await request("/auth/login", { method: "POST", body });
  setAuth(data);
  showResult("Logged in", data);
}

async function createLead() {
  ensureToken();
  const data = await request("/api/leads", {
    method: "POST",
    body: {
      name: value("leadNameInput"),
      email: value("leadEmailInput"),
      company: value("companyInput"),
      source: value("sourceInput"),
    },
  });
  showResult("Lead created", data);
  setTimeout(refreshAll, 1800);
}

async function runDemo() {
  if (!state.token) {
    const timestamp = Date.now();
    document.getElementById("emailInput").value = `manager+${timestamp}@example.com`;
    await register();
  }
  await createLead();
  await sleep(2500);
  await refreshAll();
}

async function loadLeadsAndCustomers() {
  ensureToken();
  const [leads, customers] = await Promise.all([
    request("/api/leads"),
    request("/api/customers"),
  ]);
  renderTable("leadsTable", leads.items || [], ["id", "name", "email", "company", "status", "source"]);
  renderTable("customersTable", customers.items || [], ["id", "lead_id", "name", "email", "company"]);
}

async function loadNotifications() {
  ensureToken();
  const data = await request("/api/notifications");
  renderTable("notificationsTable", data.items || [], ["id", "type", "customer_id", "message", "created_at"]);
}

async function request(url, options = {}) {
  const headers = { "Content-Type": "application/json" };
  if (state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }

  const response = await fetch(url, {
    method: options.method || "GET",
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  const text = await response.text();
  const data = text ? JSON.parse(text) : {};
  if (!response.ok) {
    throw new Error(data.error || `HTTP ${response.status}`);
  }
  return data;
}

function setAuth(data) {
  state.token = data.access_token;
  state.user = data.user;
  localStorage.setItem("crm_token", state.token);
  localStorage.setItem("crm_user", JSON.stringify(state.user));
  updateSession();
  connectWebSocket();
}

function updateSession() {
  const dot = document.getElementById("sessionState");
  const text = document.getElementById("sessionText");
  if (state.token && state.user) {
    dot.className = "status-dot ok";
    text.textContent = state.user.email;
    return;
  }
  dot.className = "status-dot warn";
  text.textContent = "Не авторизован";
}

function connectWebSocket() {
  if (state.socket) {
    state.socket.close();
  }
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsUrl = `${protocol}://${window.location.host}/ws?token=${encodeURIComponent(state.token)}`;
  const badge = document.getElementById("wsStatus");
  state.socket = new WebSocket(wsUrl);

  state.socket.addEventListener("open", () => {
    badge.textContent = "online";
    badge.className = "badge ok";
  });
  state.socket.addEventListener("message", (event) => {
    addEvent(event.data);
    refreshAll();
  });
  state.socket.addEventListener("close", () => {
    badge.textContent = "offline";
    badge.className = "badge muted";
  });
  state.socket.addEventListener("error", () => {
    badge.textContent = "error";
    badge.className = "badge fail";
  });
}

function addEvent(raw) {
  const feed = document.getElementById("eventFeed");
  const empty = feed.querySelector(".empty");
  if (empty) {
    empty.remove();
  }
  let parsed = raw;
  try {
    parsed = JSON.parse(raw);
  } catch (_error) {
    parsed = { type: "message", raw };
  }
  const item = document.createElement("div");
  item.className = "event-item";
  item.innerHTML = `<strong>${escapeHtml(parsed.type || "event")}</strong><code>${escapeHtml(JSON.stringify(parsed, null, 2))}</code>`;
  feed.prepend(item);
}

function renderTable(containerId, rows, columns) {
  const container = document.getElementById(containerId);
  if (!rows.length) {
    container.innerHTML = `<div class="empty">Нет данных.</div>`;
    return;
  }

  const header = columns.map((column) => `<th>${escapeHtml(column)}</th>`).join("");
  const body = rows.map((row) => {
    const cells = columns.map((column) => `<td class="truncate">${escapeHtml(formatCell(row[column]))}</td>`).join("");
    return `<tr>${cells}</tr>`;
  }).join("");
  container.innerHTML = `<table><thead><tr>${header}</tr></thead><tbody>${body}</tbody></table>`;
}

function authPayload() {
  return {
    email: value("emailInput"),
    password: value("passwordInput"),
  };
}

function ensureToken() {
  if (!state.token) {
    throw new Error("Сначала нажми Register или Login.");
  }
}

function value(id) {
  return document.getElementById(id).value.trim();
}

function showResult(title, data) {
  document.getElementById("resultBox").textContent = `${title}\n${JSON.stringify(data, null, 2)}`;
}

function formatCell(value) {
  if (value === null || value === undefined) {
    return "";
  }
  return String(value);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

window.addEventListener("unhandledrejection", (event) => {
  showResult("Error", { error: event.reason.message || String(event.reason) });
});

