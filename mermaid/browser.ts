// The Mermaid runtime is generated from the pinned alpha and packaged with this plugin.
interface MermaidRuntime {
  initialize(options: {
    startOnLoad: boolean;
    theme: string;
    securityLevel: string;
  }): void;
  run(options: { nodes: HTMLElement[] }): Promise<void>;
}

interface EditorContext {
  source: string;
  onChange(source: string): void;
  onValidity?(state: { valid: boolean }): void;
}

interface RenderContext {
  source: string;
  theme: string;
}

const host = globalThis as typeof globalThis & {
  kumbukaPlugin?: {
    edit(root: HTMLElement, context: EditorContext): void;
    render(root: HTMLElement, context: RenderContext): Promise<void>;
  };
};

let loading: Promise<MermaidRuntime> | undefined;

// loadMermaid lazily initializes the renderer compiled into this script.
// Its static import specifier is resolved by esbuild at package build time.
function loadMermaid(): Promise<MermaidRuntime> {
  if (loading) return loading;

  loading = import("mermaid")
    .then((module) => {
      const runtime = module.default;
      if (
        typeof runtime?.initialize !== "function" ||
        typeof runtime.run !== "function"
      ) {
        throw new Error(
          "Mermaid bundle does not contain a compatible renderer",
        );
      }
      return runtime;
    })
    .catch((error: unknown) => {
      loading = undefined;
      throw new Error(
        `could not initialize Mermaid renderer: ${String(error)}`,
        { cause: error },
      );
    });

  return loading;
}

const interactiveDirective =
  /^%%[ \t]+kumbuka-interactive:[ \t]*true[ \t]*(?:\r?\n|$)/im;

const widthDirective =
  /^%%[ \t]+kumbuka-width:[ \t]*([1-9][0-9]{0,3})[ \t]*(?:\r?\n|$)/im;

const alignmentDirective =
  /^%%[ \t]+kumbuka-align:[ \t]*(left|center|right)[ \t]*(?:\r?\n|$)/im;

// readAlignment selects the requested presentation without exposing the host page.
function readAlignment(source: string): string {
  return alignmentDirective.exec(source)?.[1].toLowerCase() || "center";
}

// edit keeps the source and alignment inside the plugin-owned visual editor.
function edit(root: HTMLElement, context: EditorContext): void {
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

  const widthLabel = document.createElement("label");
  widthLabel.textContent = "Diagram width (px)";
  const width = document.createElement("input");
  width.type = "number";
  width.min = "1";
  width.max = "9999";
  width.placeholder = "Automatic";
  width.value = widthDirective.exec(context.source)?.[1] || "";
  widthLabel.append(width);

  const interactiveLabel = document.createElement("label");
  const interactive = document.createElement("input");
  interactive.type = "checkbox";
  interactive.checked = interactiveDirective.test(context.source);
  interactiveLabel.append(
    interactive,
    document.createTextNode(" Interactive diagram"),
  );

  const sourceLabel = document.createElement("label");
  sourceLabel.textContent = "Mermaid source";
  const source = document.createElement("textarea");
  source.rows = 12;
  source.value = context.source;
  sourceLabel.append(source);

  const publish = () => context.onChange(source.value);
  width.addEventListener("input", () => {
    context.onValidity?.({ valid: width.validity.valid });
    if (!width.validity.valid) return;
    const body = source.value.replace(widthDirective, "");
    source.value = width.value
      ? `%% kumbuka-width: ${width.valueAsNumber}\n${body}`
      : body;
    publish();
  });
  alignment.addEventListener("change", () => {
    const body = source.value.replace(alignmentDirective, "");
    source.value =
      alignment.value === "center"
        ? body
        : `%% kumbuka-align: ${alignment.value}\n${body}`;
    publish();
  });
  interactive.addEventListener("change", () => {
    const body = source.value.replace(interactiveDirective, "");
    source.value = interactive.checked
      ? `%% kumbuka-interactive: true\n${body}`
      : body;
    publish();
  });
  source.addEventListener("input", () => {
    width.value = widthDirective.exec(source.value)?.[1] || "";
    context.onValidity?.({ valid: true });
    alignment.value = readAlignment(source.value);
    interactive.checked = interactiveDirective.test(source.value);
    publish();
  });

  root.classList.add("mermaid-editor");
  root.replaceChildren(label, widthLabel, interactiveLabel, sourceLabel);
  context.onValidity?.({ valid: true });
}

// render runs the packaged browser renderer only in Kumbuka's isolated frame.
async function render(
  root: HTMLElement,
  context: RenderContext,
): Promise<void> {
  const mermaid = await loadMermaid();
  const diagram = document.createElement("div");
  diagram.className = "mermaid";
  diagram.textContent = context.source;
  diagram.style.textAlign = readAlignment(context.source);
  root.replaceChildren(diagram);

  mermaid.initialize({
    startOnLoad: false,
    theme: context.theme === "dark" ? "dark" : "default",
    securityLevel: "strict",
  });
  await mermaid.run({ nodes: [diagram] });
  const width = widthDirective.exec(context.source)?.[1];
  const svg = diagram.querySelector("svg");
  if (width && svg) {
    svg.style.width = `${width}px`;
    svg.style.maxWidth = "100%";
    svg.style.height = "auto";
  }
}

host.kumbukaPlugin = { edit, render };
