"use strict";

const BYTEBIN = "https://bytebin.lucko.me";
const SESSION_KEY = new URLSearchParams(location.search).get("key") || "";

const state = {
  document: null,   // {groups: {name: {weight, prefix, parents, permissions}}, users: {name: {groups, permissions}}}
  commands: [],     // [{command, node, default_allowed}]
  selected: null,
  dragSrc: null,    // group name being dragged
};

const el = id => document.getElementById(id);

// ── Toast ─────────────────────────────────────────────────────────────────────

let toastTimer = null;
function showToast(msg, durationMs = 6000) {
  const t = el("toast");
  t.innerHTML = msg;
  t.classList.add("show");
  clearTimeout(toastTimer);
  if (durationMs > 0) toastTimer = setTimeout(() => t.classList.remove("show"), durationMs);
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

function groupsSorted() {
  return Object.keys(state.document.groups).sort((a, b) => {
    const wa = state.document.groups[a].weight ?? 0;
    const wb = state.document.groups[b].weight ?? 0;
    return wb - wa || a.localeCompare(b);
  });
}

function renderGroups() {
  const list = el("groupList");
  list.innerHTML = "";

  for (const name of groupsSorted()) {
    const group = state.document.groups[name];
    const parents = (group.parents || []).length;

    const btn = document.createElement("button");
    btn.className = "group-item" + (name === state.selected ? " active" : "");
    btn.dataset.group = name;
    btn.draggable = true;
    btn.innerHTML = `
      <span class="drag-handle" title="Drag to reorder">⠿</span>
      <span class="status-dot${name === "default" ? " muted" : ""}"></span>
      <span class="group-name"></span>
      <small></small>`;
    btn.querySelector(".group-name").textContent = name;
    btn.querySelector("small").textContent = parents ? `+${parents}` : "";
    btn.addEventListener("click", () => selectGroup(name));

    // ── drag-and-drop ──────────────────────────────────────────────────────
    btn.addEventListener("dragstart", e => {
      state.dragSrc = name;
      btn.classList.add("dragging");
      e.dataTransfer.effectAllowed = "move";
    });
    btn.addEventListener("dragend", () => {
      btn.classList.remove("dragging");
      list.querySelectorAll(".drag-over").forEach(el => el.classList.remove("drag-over"));
    });
    btn.addEventListener("dragover", e => {
      if (state.dragSrc && state.dragSrc !== name) {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        btn.classList.add("drag-over");
      }
    });
    btn.addEventListener("dragleave", () => btn.classList.remove("drag-over"));
    btn.addEventListener("drop", e => {
      e.preventDefault();
      btn.classList.remove("drag-over");
      if (!state.dragSrc || state.dragSrc === name) return;
      swapGroupWeights(state.dragSrc, name);
    });

    list.appendChild(btn);
  }
}

// Swap the weights of two groups so the dragged group takes the target's
// position in the hierarchy, then re-render the sidebar and metrics.
function swapGroupWeights(srcName, dstName) {
  const src = state.document.groups[srcName];
  const dst = state.document.groups[dstName];
  const tmp = src.weight ?? 0;
  src.weight = dst.weight ?? 0;
  dst.weight = tmp;
  renderGroups();
  if (state.selected === srcName || state.selected === dstName) updateMetrics();
}

// ── Group selection ───────────────────────────────────────────────────────────

function selectGroup(name) {
  state.selected = name;
  const group = state.document.groups[name];
  if (!group) return;

  el("groupTitle").textContent = name;
  el("groupAvatar").textContent = name[0].toUpperCase();

  const parentsInput = el("parentsInput");
  if (parentsInput) {
    parentsInput.value = (group.parents || []).join(", ");
    parentsInput.oninput = () => {
      group.parents = parentsInput.value
        .split(",").map(s => s.trim().toLowerCase()).filter(Boolean);
      renderInheritance(group.parents);
      renderGroups();
    };
  }

  const prefixInput = el("prefixInput");
  if (prefixInput) {
    prefixInput.value = group.prefix || "";
    prefixInput.oninput = () => {
      group.prefix = prefixInput.value;
    };
  }

  const weightInput = el("weightInput");
  if (weightInput) {
    weightInput.value = group.weight ?? 0;
    weightInput.oninput = () => {
      const w = parseInt(weightInput.value, 10);
      group.weight = isNaN(w) ? 0 : w;
      renderGroups();
    };
  }

  renderInheritance(group.parents || []);
  renderGroups();
  renderPermissions();
  updateMetrics();
}

function renderInheritance(parents) {
  const chain = el("inheritanceChain");
  if (!chain) return;
  chain.innerHTML = "";
  for (const p of parents) {
    const node = document.createElement("div");
    node.className = "inheritance-node";
    node.innerHTML = `${p} <span>Parent</span>`;
    chain.appendChild(node);
    const arrow = document.createElement("div");
    arrow.className = "arrow";
    arrow.textContent = "↓";
    chain.appendChild(arrow);
  }
  const cur = document.createElement("div");
  cur.className = "inheritance-node current";
  cur.innerHTML = `<span id="inheritCurrent">${state.selected || "—"}</span><span>Current</span>`;
  chain.appendChild(cur);
}

// ── Permission list ───────────────────────────────────────────────────────────

function renderPermissions() {
  const list = el("permissionList");
  list.innerHTML = "";
  const group = state.document.groups[state.selected];
  if (!group) return;

  const perms = group.permissions || {};
  const knownNodes = new Set(state.commands.map(c => c.node));

  for (const cmd of state.commands) {
    list.appendChild(buildRow(cmd.node, perms, cmd));
  }

  for (const node of Object.keys(perms).sort()) {
    if (knownNodes.has(node)) continue;
    list.appendChild(buildRow(node, perms, null));
  }

  applyFilter();
  updateMetrics();
}

function buildRow(node, perms, cmd) {
  const currentValue = Object.hasOwn(perms, node) ? String(perms[node]) : "";
  const defaultLabel = cmd
    ? (cmd.default_allowed ? "Unset — public" : "Unset — op only")
    : "Unset";

  const row = document.createElement("div");
  row.className = "permission-row";
  row.dataset.node = node;
  row.dataset.search = (cmd ? `${cmd.command} ` : "") + node;

  const info = document.createElement("div");
  const code = document.createElement("code");
  code.textContent = node;
  const desc = document.createElement("p");
  desc.textContent = cmd ? `/${cmd.command}` : "Custom node";
  info.append(code, desc);

  const select = document.createElement("select");
  select.className = "perm-select";
  for (const [val, label] of [["", defaultLabel], ["true", "Allow"], ["false", "Deny"]]) {
    const opt = document.createElement("option");
    opt.value = val;
    opt.textContent = label;
    select.appendChild(opt);
  }
  select.value = currentValue;
  styleSelect(select);

  select.addEventListener("change", () => {
    const group = state.document.groups[state.selected];
    if (!group) return;
    group.permissions ||= {};
    if (select.value === "") delete group.permissions[node];
    else group.permissions[node] = select.value === "true";
    styleSelect(select);
    updateMetrics();
  });

  row.append(info, select);
  return row;
}

function styleSelect(select) {
  select.style.borderColor =
    select.value === "true" ? "var(--success)" :
    select.value === "false" ? "var(--danger)" :
    "var(--border)";
}

// ── Filter ────────────────────────────────────────────────────────────────────

function applyFilter() {
  const query = el("searchPerm").value.trim().toLowerCase();
  const filter = el("categoryFilter").value;
  for (const row of el("permissionList").querySelectorAll(".permission-row:not(.empty-state)")) {
    const matchesText = !query || row.dataset.search.toLowerCase().includes(query);
    const sel = row.querySelector(".perm-select");
    const val = sel ? sel.value : "";
    const matchesFilter =
      filter === "all" ||
      (filter === "allowed" && val === "true") ||
      (filter === "denied" && val === "false") ||
      (filter === "unset" && val === "");
    row.hidden = !(matchesText && matchesFilter);
  }
}

el("searchPerm").addEventListener("input", applyFilter);
el("categoryFilter").addEventListener("change", applyFilter);

// ── Metrics ───────────────────────────────────────────────────────────────────

function updateMetrics() {
  const group = state.document && state.selected && state.document.groups[state.selected];
  if (!group) return;
  const perms = group.permissions || {};
  const members = Object.values(state.document.users || {})
    .filter(u => (u.groups || []).includes(state.selected)).length;
  el("metricPerms").textContent = Object.keys(perms).length;
  el("metricMembers").textContent = members;
  el("metricParents").textContent = (group.parents || []).length;
  el("metricCommands").textContent = state.commands.length;
}

// ── Save ──────────────────────────────────────────────────────────────────────

el("saveBtn").addEventListener("click", async () => {
  if (!state.document) return;
  const btn = el("saveBtn");
  btn.disabled = true;
  btn.textContent = "Saving…";
  try {
    const resp = await fetch(`${BYTEBIN}/post`, {
      method: "POST",
      headers: {"Content-Type": "application/json; charset=utf-8"},
      body: JSON.stringify({type: "gocraft-permissions-save", document: state.document}),
    });
    if (!resp.ok) throw new Error(`Bytebin error ${resp.status}`);
    const {key} = await resp.json();
    const command = `/gocraft applyedits ${key}`;
    el("applyCommand").textContent = command;
    el("applyDialog").showModal();
  } catch (err) {
    showToast(`Error saving: ${err.message}`);
  } finally {
    btn.disabled = false;
    btn.textContent = "Save changes";
  }
});

el("copyBtn").addEventListener("click", () => {
  const command = el("applyCommand").textContent;
  navigator.clipboard.writeText(command).then(() => {
    el("copyBtn").textContent = "✓ Copied!";
    setTimeout(() => { el("copyBtn").textContent = "⎘ Copy"; }, 2000);
  });
});

el("closeApplyBtn").addEventListener("click", () => el("applyDialog").close());

// ── Add group dialog ──────────────────────────────────────────────────────────

el("addGroupBtn").addEventListener("click", () => {
  el("newGroupName").value = "";
  el("groupDialog").showModal();
  el("newGroupName").focus();
});

el("createGroupBtn").addEventListener("click", e => {
  e.preventDefault();
  const name = el("newGroupName").value.trim().toLowerCase();
  if (!name || !/^[a-z0-9_-]+$/.test(name)) {
    el("newGroupName").setCustomValidity("Use letters, numbers, _ or - only.");
    el("newGroupName").reportValidity();
    return;
  }
  if (state.document.groups[name]) {
    el("newGroupName").setCustomValidity("Group already exists.");
    el("newGroupName").reportValidity();
    return;
  }
  el("newGroupName").setCustomValidity("");
  state.document.groups[name] = {weight: 0, prefix: "", parents: ["default"], permissions: {}};
  el("groupDialog").close();
  renderGroups();
  selectGroup(name);
});

// ── Add custom permission dialog ──────────────────────────────────────────────

el("addPermBtn").addEventListener("click", () => {
  el("newPermNode").value = "";
  el("permDialog").showModal();
  el("newPermNode").focus();
});

el("createPermBtn").addEventListener("click", e => {
  e.preventDefault();
  const node = el("newPermNode").value.trim().toLowerCase();
  if (!node) return;
  const group = state.document.groups[state.selected];
  if (!group) return;
  group.permissions ||= {};
  group.permissions[node] = el("newPermValue").value === "true";
  el("permDialog").close();
  renderPermissions();
});

// ── Init ──────────────────────────────────────────────────────────────────────

async function init() {
  if (!SESSION_KEY) {
    el("statusText").textContent = "No key";
    el("statusDot").style.background = "var(--danger)";
    showToast("No key in URL. Run <code>/gocraft peditor</code> in-game to get a link.", 0);
    return;
  }
  try {
    el("serverStatus").textContent = "Loading data…";
    const resp = await fetch(`${BYTEBIN}/${SESSION_KEY}`, {cache: "no-store"});
    if (resp.status === 404) throw new Error("Session not found or expired. Run /gocraft peditor again.");
    if (!resp.ok) throw new Error(`Failed to load data (${resp.status})`);

    const payload = await resp.json();
    if (payload.type !== "gocraft-permissions")
      throw new Error("Not a GoCraft permission editor link.");

    state.document = payload.document;
    state.commands = payload.commands || [];
    state.document.groups ||= {};
    state.document.users ||= {};

    el("serverStatus").textContent = "GoCraft Survival";
    el("statusDot").style.background = "var(--success)";
    el("statusText").textContent = "Connected";
    el("metricCommands").textContent = state.commands.length;

    renderGroups();
    const first = groupsSorted()[0];
    if (first) selectGroup(first);

  } catch (err) {
    el("statusDot").style.background = "var(--danger)";
    el("statusText").textContent = "Error";
    el("serverStatus").textContent = "Failed to load";
    showToast(`Error: ${err.message}`, 0);
  }
}

init();
