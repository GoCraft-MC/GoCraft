"use strict";

editor.openGroup = name => {
  editor.selected = {type: "group", name};
  editor.show("group-editor");
  editor.element("group-name").textContent = name;
  const group = editor.document.groups[name];
  group.parents ||= [];
  group.permissions ||= {};
  editor.element("parents").value = group.parents.join(", ");

  const known = new Set(editor.commands.map(command => command.node));
  const commands = editor.element("commands");
  commands.replaceChildren();
  for (const command of editor.commands) {
    const row = document.createElement("div");
    row.className = "permission";
    row.dataset.search = `${command.command} ${command.node}`;
    const nameElement = document.createElement("span");
    nameElement.textContent = `/${command.command}`;
    const node = document.createElement("code");
    node.textContent = command.node;
    const select = document.createElement("select");
    for (const [value, label] of [["", "Unset"], ["true", "Allow"], ["false", "Deny"]]) {
      const option = document.createElement("option");
      option.value = value;
      option.textContent = label;
      select.append(option);
    }
    if (Object.hasOwn(group.permissions, command.node))
      select.value = String(group.permissions[command.node]);
    select.addEventListener("change", () => {
      if (select.value === "") delete group.permissions[command.node];
      else group.permissions[command.node] = select.value === "true";
    });
    row.append(nameElement, node, select);
    commands.append(row);
  }
  const custom = Object.fromEntries(Object.entries(group.permissions).filter(([node]) => !known.has(node)));
  editor.element("custom").value = editor.rulesText(custom);
  editor.renderSidebar();
};

editor.element("parents").addEventListener("input", event => {
  if (editor.selected?.type === "group")
    editor.document.groups[editor.selected.name].parents = editor.names(event.target.value);
});
editor.element("custom").addEventListener("change", event => {
  if (editor.selected?.type !== "group") return;
  try {
    const group = editor.document.groups[editor.selected.name];
    const known = new Set(editor.commands.map(command => command.node));
    for (const node of Object.keys(group.permissions)) if (!known.has(node)) delete group.permissions[node];
    Object.assign(group.permissions, editor.parseRules(event.target.value));
    editor.setStatus("Custom group permissions updated locally.");
  } catch (error) { editor.setStatus(error.message, true); }
});
editor.element("filter").addEventListener("input", event => {
  const query = event.target.value.toLowerCase();
  for (const row of editor.element("commands").children)
    row.hidden = !row.dataset.search.includes(query);
});
editor.element("delete-group").addEventListener("click", () => {
  const name = editor.selected?.name;
  if (!name || name === "default") return editor.setStatus("The default group cannot be deleted.", true);
  delete editor.document.groups[name];
  for (const group of Object.values(editor.document.groups))
    group.parents = (group.parents || []).filter(parent => parent !== name);
  for (const user of Object.values(editor.document.users))
    user.groups = (user.groups || []).filter(group => group !== name);
  editor.selected = null;
  editor.show("welcome");
  editor.renderSidebar();
});
