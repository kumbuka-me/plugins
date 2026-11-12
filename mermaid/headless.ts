// This bundle is evaluated inside an isolated, browser-free JavaScript runtime.
// The plugin owns Mermaid, diagram themes, and SVG generation.
import mermaid from "mermaid";

const host = globalThis as typeof globalThis & {
  kumbukaRenderArtifact?: (source: string, theme: string) => Promise<string>;
};

host.kumbukaRenderArtifact = async (source: string, theme: string) => {
  mermaid.initialize({
    startOnLoad: false,
    theme: theme === "dark" ? "dark" : "default",
    securityLevel: "strict",
    deterministicIds: true,
    deterministicIDSeed: source,
  });
  const { svg } = await mermaid.render("kumbuka-diagram", source);
  return svg;
};
