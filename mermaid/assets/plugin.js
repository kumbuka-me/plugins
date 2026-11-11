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

// Presentation directives are Mermaid comments, so they are preserved in Markdown.
const alignmentDirective = /^%%[ \t]+kumbuka-align:[ \t]*(left|center|right)[ \t]*(?:\r?\n|$)/im;
const interactiveDirective = /^%%[ \t]+kumbuka-interactive:[ \t]*(true|false)[ \t]*(?:\r?\n|$)/im;

function readAlignment(source) {
  return alignmentDirective.exec(source)?.[1].toLowerCase() || "center";
}

function readInteractive(source) {
  return interactiveDirective.exec(source)?.[1].toLowerCase() === "true";
}

function updateDirective(source, directive, replacement) {
  const normalized = source.replace(directive, "");
  return replacement ? `${replacement}\n${normalized}` : normalized;
}

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

  const interactiveLabel = document.createElement("label");
  const interactive = document.createElement("input");
  interactive.type = "checkbox";
  interactive.checked = readInteractive(context.source);
  interactiveLabel.append(interactive, " Interactive diagram");

  const sourceLabel = document.createElement("label");
  sourceLabel.textContent = "Mermaid source";
  const source = document.createElement("textarea");
  source.rows = 12;
  source.value = context.source;
  sourceLabel.append(source);

  const publish = () => context.onChange(source.value);
  alignment.addEventListener("change", () => {
    source.value = updateDirective(
      source.value,
      alignmentDirective,
      alignment.value === "center" ? "" : `%% kumbuka-align: ${alignment.value}`,
    );
    publish();
  });
  interactive.addEventListener("change", () => {
    source.value = updateDirective(
      source.value,
      interactiveDirective,
      interactive.checked ? "%% kumbuka-interactive: true" : "",
    );
    publish();
  });
  source.addEventListener("input", () => {
    alignment.value = readAlignment(source.value);
    interactive.checked = readInteractive(source.value);
    publish();
  });

  root.classList.add("mermaid-editor");
  root.replaceChildren(label, interactiveLabel, sourceLabel);
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
