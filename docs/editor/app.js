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
    editor.setStatus("Saving edits…");
    const response = await fetch(`${editor.bytebinURL}/post`, {
      method: "POST",
      headers: {"Content-Type": "application/json; charset=utf-8"},
      body: JSON.stringify({type: "gocraft-permissions-save", document: editor.document}),
    });
    if (!response.ok) throw new Error(`Bytebin error: ${response.status}`);
    const {key} = await response.json();
    editor.setStatus(`Saved! Apply in-game or console: /gocraft applyedits ${key}`);
  } catch (error) { editor.setStatus(error.message, true); }
});

(async () => {
  if (!editor.token) {
    editor.setStatus("No key provided in URL. Open this page from the link given by /gocraft peditor.", true);
    return;
  }
  try {
    const response = await fetch(`${editor.bytebinURL}/${editor.token}`, {cache: "no-store"});
    if (response.status === 404) throw new Error("Permission editor session not found or expired.");
    if (!response.ok) throw new Error(`Failed to load: ${response.status}`);
    const payload = await response.json();
    if (!payload.document || !payload.commands) throw new Error("Invalid permission data from server.");
    editor.document = payload.document;
    editor.commands = payload.commands;
    editor.renderSidebar();
    editor.setStatus("Loaded. Edit permissions, then click Save — you will get a command to run in-game.");
  } catch (error) { editor.setStatus(error.message, true); }
})();
