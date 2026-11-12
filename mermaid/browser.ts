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
  mermaid?: MermaidRuntime;
  kumbukaPlugin?: {
    edit(root: HTMLElement, context: EditorContext): void;
    render(root: HTMLElement, context: RenderContext): Promise<void>;
  };
};

const assetBase =
  document.currentScript instanceof HTMLScriptElement
    ? document.currentScript.src
    : undefined;
let loading: Promise<MermaidRuntime> | undefined;

// loadMermaid shares the one packaged renderer across subsequent frames in this document.
function loadMermaid(): Promise<MermaidRuntime> {
  if (host.mermaid) return Promise.resolve(host.mermaid);
  if (loading) return loading;
  loading = new Promise<MermaidRuntime>((resolve, reject) => {
    if (!assetBase) {
      reject(new Error("Mermaid plugin script URL is unavailable"));
      return;
    }
    const script = document.createElement("script");
    script.src = new URL("./mermaid.min.js", assetBase).href;
    script.onload = () => {
      if (host.mermaid) resolve(host.mermaid);
      else reject(new Error("Mermaid bundle did not register its runtime"));
    };
    script.onerror = () =>
      reject(new Error("Mermaid bundle could not be loaded"));
    document.head.append(script);
  }).catch((error: unknown) => {
    loading = undefined;
    throw error;
  });
  return loading;
}

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

  const sourceLabel = document.createElement("label");
  sourceLabel.textContent = "Mermaid source";
  const source = document.createElement("textarea");
  source.rows = 12;
  source.value = context.source;
  sourceLabel.append(source);

  const publish = () => context.onChange(source.value);
  alignment.addEventListener("change", () => {
    const body = source.value.replace(alignmentDirective, "");
    source.value =
      alignment.value === "center"
        ? body
        : `%% kumbuka-align: ${alignment.value}\n${body}`;
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
}

host.kumbukaPlugin = { edit, render };
