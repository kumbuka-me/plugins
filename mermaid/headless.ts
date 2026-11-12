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
  // External images need an intrinsic width. Mermaid's inline max-width is
  // inside the SVG document and cannot constrain the enclosing HTML image.
  return svg.replace(/^(<svg\b[^>]*>)/, (root) => {
    const width = /\sviewBox="[^"]*?\s([\d.]+)\s[\d.]+"/.exec(root)?.[1];
    return width ? root.replace(' width="100%"', ` width="${width}"`) : root;
  });
};
