const assetBase = document.currentScript.src;
async function load() {
  if (globalThis.mermaid) return;
  await new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = new URL("./mermaid.min.js", assetBase).href;
    script.onload = resolve;
    script.onerror = () => reject(new Error("Mermaid could not be loaded"));
    document.head.append(script);
  });
}

// All DOM work is confined to the opaque sandbox. The host supplies only this
// diagram's text and presentation preferences, never page DOM or credentials.
const alignmentDirective = /^%%[ \t]+kumbuka-align:[ \t]*(left|center|right)[ \t]*(?:\r?\n|$)/i;
function readAlignment(source) {
  return alignmentDirective.exec(source)?.[1].toLowerCase() || "center";
}

// The plugin owns both its presentation directive and its editor controls.
function edit(root, context) {
  const label = document.createElement("label");
  label.textContent = "Diagram alignment";
  const alignment = document.createElement("select");
  for (const value of ["left", "center", "right"]) {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = value[0].toUpperCase() + value.slice(1);
    alignment.append(option);
  }
  alignment.value = readAlignment(context.source);
  label.append(alignment);
  const sourceLabel = document.createElement("label");
  sourceLabel.textContent = "Mermaid source";
  const source = document.createElement("textarea");
  source.rows = 12;
  source.value = context.source;
  sourceLabel.append(source);
  const publish = () => context.onChange(source.value);
  alignment.addEventListener("change", () => {
    const content = source.value.replace(alignmentDirective, "");
    source.value = alignment.value === "center" ? content : `%% kumbuka-align: ${alignment.value}\n${content}`;
    publish();
  });
  source.addEventListener("input", () => {
    alignment.value = readAlignment(source.value);
    publish();
  });
  root.classList.add("mermaid-editor");
  root.replaceChildren(label, sourceLabel);
  context.onValidity?.({ valid: true });
}

globalThis.kumbukaPlugin = {
  edit,
  async render(root, context) {
    await load();
    const diagram = document.createElement("div");
    diagram.className = "mermaid";
    diagram.textContent = context.source;
    diagram.style.textAlign = readAlignment(context.source);
    root.replaceChildren(diagram);
    globalThis.mermaid.initialize({
      startOnLoad: false,
      theme: context.theme === "dark" ? "dark" : "default",
      securityLevel: "strict",
    });
    await globalThis.mermaid.run({ nodes: [diagram] });
  },
};
