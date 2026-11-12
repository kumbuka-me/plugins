"use strict";
(() => {
    const globals = globalThis;
    const assetBase = document.currentScript?.src ?? "";
    const alignmentDirective = /^%%[ \t]+kumbuka-align:[ \t]*(left|center|right)[ \t]*(?:\r?\n|$)/im;
    const interactiveDirective = /^%%[ \t]+kumbuka-interactive:[ \t]*(true|false)[ \t]*(?:\r?\n|$)/im;
    // load imports the plugin-bundled Mermaid runtime only when a diagram is rendered.
    async function load() {
        if (!globals.mermaid) {
            await new Promise((resolve, reject) => {
                const script = document.createElement("script");
                script.src = new URL("./mermaid.min.js", assetBase).href;
                script.onload = () => resolve();
                script.onerror = () => reject(new Error("Mermaid could not be loaded"));
                document.head.append(script);
            });
        }
        if (!globals.mermaid)
            throw new Error("Mermaid runtime is unavailable");
        return globals.mermaid;
    }
    // readAlignment returns the bounded presentation directive from an editable fence.
    function readAlignment(source) {
        return alignmentDirective.exec(source)?.[1].toLowerCase() ?? "center";
    }
    // readInteractive reports whether a fence must retain its live browser renderer.
    function readInteractive(source) {
        return interactiveDirective.exec(source)?.[1].toLowerCase() === "true";
    }
    // updateDirective replaces or clears an editor-owned Mermaid comment.
    function updateDirective(source, directive, replacement) {
        const normalized = source.replace(directive, "");
        return replacement ? `${replacement}\n${normalized}` : normalized;
    }
    // edit binds source changes to the standard isolated browser widget interface.
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
            source.value = updateDirective(source.value, alignmentDirective, alignment.value === "center"
                ? ""
                : `%% kumbuka-align: ${alignment.value}`);
            publish();
        });
        interactive.addEventListener("change", () => {
            source.value = updateDirective(source.value, interactiveDirective, interactive.checked ? "%% kumbuka-interactive: true" : "");
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
    // renderSource shares the exact Mermaid engine and appearance between live and build views.
    async function renderSource(root, source, theme, stableIDs = false) {
        const mermaid = await load();
        const diagram = document.createElement("div");
        diagram.className = "mermaid";
        diagram.textContent = source;
        diagram.style.textAlign = readAlignment(source);
        root.replaceChildren(diagram);
        const settings = {
            startOnLoad: false,
            theme: theme === "dark" ? "dark" : "default",
            securityLevel: "strict",
        };
        if (stableIDs) {
            settings.htmlLabels = false;
            settings.deterministicIds = true;
            settings.deterministicIDSeed = source;
            settings.flowchart = { htmlLabels: false };
        }
        mermaid.initialize(settings);
        await mermaid.run({ nodes: [diagram] });
        return diagram.querySelector("svg");
    }
    globals.kumbukaPlugin = {
        edit,
        async render(root, context) {
            await renderSource(root, context.source, context.theme);
        },
        // exportStatic returns null for interactive blocks and passive SVG for supported blocks.
        async exportStatic(root, context) {
            const code = root.querySelector("pre code.language-mermaid");
            if (!code || readInteractive(code.textContent ?? ""))
                return null;
            const source = code.textContent ?? "";
            const svg = await renderSource(root, source, context.theme, true);
            if (!svg)
                throw new Error("Mermaid did not produce an SVG");
            return {
                content: svg.outerHTML,
                alt: "Mermaid diagram",
                alignment: readAlignment(source),
            };
        },
    };
})();
