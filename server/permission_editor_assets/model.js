"use strict";

window.editor = {
  document: null,
  commands: [],
  selected: null,
  token: location.pathname.split("/").filter(Boolean).pop(),
};

editor.element = id => document.getElementById(id);
editor.names = text => [...new Set(text.split(",").map(value => value.trim().toLowerCase()).filter(Boolean))];
editor.rulesText = rules => Object.entries(rules || {})
  .sort(([left], [right]) => left.localeCompare(right))
  .map(([node, value]) => `${node}=${value}`).join("\n");
editor.parseRules = text => {
  const rules = {};
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (!line) continue;
    const match = /^([^\s=]+)=(true|false)$/i.exec(line);
    if (!match) throw new Error(`Invalid permission rule: ${line}`);
    rules[match[1].toLowerCase()] = match[2].toLowerCase() === "true";
  }
  return rules;
};
editor.setStatus = (message, error = false) => {
  const status = editor.element("status");
  status.textContent = message;
  status.className = error ? "error" : "ok";
};
editor.show = section => {
  for (const id of ["group-editor", "user-editor", "welcome"])
    editor.element(id).hidden = id !== section;
};
