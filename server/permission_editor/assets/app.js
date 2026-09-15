"use strict";

editor.renderSidebar = () => {
  const render = (target, names, type, open) => {
    target.replaceChildren();
    for (const name of names.sort()) {
      const button = document.createElement("button");
      button.textContent = name;
      button.className = editor.selected?.type === type && editor.selected.name === name ? "selected" : "";
      button.addEventListener("click", () => open(name));
      target.append(button);
    }
  };
  render(editor.element("groups"), Object.keys(editor.document.groups), "group", editor.openGroup);
  render(editor.element("users"), Object.keys(editor.document.users), "user", editor.openUser);
};

editor.openUser = name => {
  editor.selected = {type: "user", name};
  editor.show("user-editor");
  const user = editor.document.users[name];
  user.groups ||= [];
  user.permissions ||= {};
  editor.element("user-name").textContent = name;
  editor.element("user-groups").value = user.groups.join(", ");
  editor.element("user-permissions").value = editor.rulesText(user.permissions);
  editor.renderSidebar();
};

editor.element("user-groups").addEventListener("input", event => {
  if (editor.selected?.type === "user")
    editor.document.users[editor.selected.name].groups = editor.names(event.target.value);
});
editor.element("user-permissions").addEventListener("change", event => {
  if (editor.selected?.type !== "user") return;
  try {
    editor.document.users[editor.selected.name].permissions = editor.parseRules(event.target.value);
    editor.setStatus("User overrides updated locally.");
  } catch (error) { editor.setStatus(error.message, true); }
});
editor.element("delete-user").addEventListener("click", () => {
  if (!editor.selected || editor.selected.type !== "user") return;
  delete editor.document.users[editor.selected.name];
  editor.selected = null;
  editor.show("welcome");
  editor.renderSidebar();
});
editor.element("add-group").addEventListener("click", () => {
  const name = (prompt("New group name:") || "").trim().toLowerCase();
  if (!/^[a-z0-9_-]+$/.test(name)) return editor.setStatus("Use letters, numbers, _ or - for group names.", true);
  if (editor.document.groups[name]) return editor.setStatus("That group already exists.", true);
  editor.document.groups[name] = {parents: ["default"], permissions: {}};
  editor.openGroup(name);
});
editor.element("add-user").addEventListener("click", () => {
  const name = (prompt("Exact player name:") || "").trim().toLowerCase();
  if (!name) return;
  editor.document.users[name] ||= {groups: ["default"], permissions: {}};
  editor.openUser(name);
});

editor.element("save").addEventListener("click", async () => {
  try {
    editor.setStatus("Validating and saving edits…");
    const response = await fetch(`${location.pathname}/state`, {
      method: "PUT", headers: {"Content-Type": "application/json", "X-GoCraft-Editor": editor.token},
      body: JSON.stringify(editor.document),
    });
    const message = await response.text();
    if (!response.ok) throw new Error(message);
    editor.setStatus(message);
  } catch (error) { editor.setStatus(error.message, true); }
});

(async () => {
  try {
    const response = await fetch(`${location.pathname}/state`, {cache: "no-store"});
    if (!response.ok) throw new Error(await response.text());
    const state = await response.json();
    editor.document = state.document;
    editor.commands = state.commands;
    editor.renderSidebar();
    editor.setStatus(`Editor expires at ${new Date(state.expires).toLocaleString()}. Save here, then apply the link in game or console.`);
  } catch (error) { editor.setStatus(error.message, true); }
})();
