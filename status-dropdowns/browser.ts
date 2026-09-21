(() => {
  const choiceClassPrefix = "kumbuka-status-choice__";
  const colorPattern = /^#[0-9a-f]{6}$/;

  type BrowserContext = {
    html: string;
  };

  type StatusOption = {
    action: string;
    label: string;
    color: string;
  };

  // decodeHex decodes UTF-8 metadata transported through sanitizer-safe class tokens.
  function decodeHex(value: string): string {
    if (
      value.length === 0 ||
      value.length % 2 !== 0 ||
      !/^[0-9a-f]+$/.test(value)
    )
      return "";

    const bytes = new Uint8Array(value.length / 2);
    for (let index = 0; index < bytes.length; index++)
      bytes[index] = Number.parseInt(value.slice(index * 2, index * 2 + 2), 16);

    try {
      return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      return "";
    }
  }

  // parseOptions extracts bounded status choices from sanitized fallback metadata.
  function parseOptions(fallback: HTMLElement): StatusOption[] {
    const result: StatusOption[] = [];
    for (const name of fallback.classList) {
      if (!name.startsWith(choiceClassPrefix)) continue;
      const fields = name.slice(choiceClassPrefix.length).split("__");
      if (fields.length !== 3) continue;

      const [encodedColor, action, encodedLabel] = fields;
      const color = decodeHex(encodedColor || "").toLowerCase();
      const label = decodeHex(encodedLabel || "");
      if (
        !colorPattern.test(color) ||
        !action ||
        !/^[a-z0-9][a-z0-9._-]{0,127}$/.test(action) ||
        !label
      )
        continue;
      result.push({ action, label, color });
    }
    return result;
  }

  // readableForeground chooses black or white text for a solid custom status color.
  function readableForeground(color: string): string {
    const red = Number.parseInt(color.slice(1, 3), 16);
    const green = Number.parseInt(color.slice(3, 5), 16);
    const blue = Number.parseInt(color.slice(5, 7), 16);
    const luminance = (red * 299 + green * 587 + blue * 114) / 1000;
    return luminance >= 150 ? "#111827" : "#ffffff";
  }

  // applyStatusColor updates the host-isolated dropdown presentation for one choice.
  function applyStatusColor(select: HTMLSelectElement, color: string): void {
    select.style.setProperty("--status-color", color);
    select.style.setProperty("--status-foreground", readableForeground(color));
  }

  // renderStatus replaces the passive sanitized fallback with one interactive native dropdown.
  function renderStatus(root: HTMLElement, context: BrowserContext): void {
    const template = document.createElement("template");
    template.innerHTML = context.html;

    const metadata = template.content.querySelector<HTMLElement>(
      ".kumbuka-status-options",
    );
    const badge =
      template.content.querySelector<HTMLElement>(".kumbuka-status");
    const prefix =
      template.content
        .querySelector<HTMLElement>(".kumbuka-status-prefix")
        ?.textContent?.trim() || "";
    const current =
      template.content
        .querySelector<HTMLElement>(".kumbuka-status-value")
        ?.textContent?.trim() || "";
    if (!metadata || !badge || !current)
      throw new Error("Invalid status input");

    const options = parseOptions(metadata);
    const selected = options.find((option) => option.label === current);
    if (!selected) throw new Error("Invalid status choices");

    const style = badge.classList.contains("kumbuka-status-outline")
      ? "outline"
      : "solid";

    const shell = document.createElement("label");
    shell.className = "status-dropdown-shell";

    if (prefix) {
      const label = document.createElement("span");
      label.className = "status-dropdown-prefix";
      label.textContent = prefix;
      label.setAttribute("aria-hidden", "true");
      shell.append(label);
    }

    const select = document.createElement("select");
    select.className = `status-dropdown status-dropdown-${style}`;
    select.setAttribute("aria-label", prefix ? `${prefix} status` : "Status");
    select.dataset.kumbukaCommandModule = "page-details";

    for (const option of options) {
      const item = document.createElement("option");
      item.value = option.action;
      item.textContent = option.label;
      item.selected = option.action === selected.action;
      item.dataset.color = option.color;
      select.append(item);
    }

    applyStatusColor(select, selected.color);
    select.addEventListener("change", () => {
      const color = select.selectedOptions[0]?.dataset.color || "#64748b";
      applyStatusColor(select, color);
    });

    shell.append(select);
    root.className = "status-dropdown-root";
    root.replaceChildren(shell);
  }

  (globalThis as unknown as { kumbukaPlugin: unknown }).kumbukaPlugin = {
    render(root: HTMLElement, context: BrowserContext) {
      renderStatus(root, context);
    },
  };
})();
