(() => {
  const choiceClassPrefix = "kumbuka-status-choice__";
  const toneClassPrefix = "kumbuka-status-";
  const tones = new Set([
    "gray",
    "blue",
    "green",
    "yellow",
    "orange",
    "red",
    "purple",
    "teal",
  ]);

  type BrowserContext = {
    html: string;
  };

  type StatusOption = {
    action: string;
    label: string;
    tone: string;
  };

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

  function statusTone(element: Element): string {
    for (const name of element.classList) {
      if (!name.startsWith(toneClassPrefix)) continue;
      const tone = name.slice(toneClassPrefix.length);
      if (tones.has(tone)) return tone;
    }
    return "gray";
  }

  function parseOptions(fallback: HTMLElement): StatusOption[] {
    const result: StatusOption[] = [];
    for (const name of fallback.classList) {
      if (!name.startsWith(choiceClassPrefix)) continue;
      const fields = name.slice(choiceClassPrefix.length).split("__");
      if (fields.length !== 3) continue;

      const [tone, action, encodedLabel] = fields;
      const label = decodeHex(encodedLabel || "");
      if (
        !tone ||
        !tones.has(tone) ||
        !action ||
        !/^[a-z0-9][a-z0-9._-]{0,127}$/.test(action) ||
        !label
      )
        continue;
      result.push({ action, label, tone });
    }
    return result;
  }

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
    select.className = `status-dropdown status-dropdown-${style} status-dropdown-${statusTone(badge)}`;
    select.setAttribute("aria-label", prefix ? `${prefix} status` : "Status");
    select.dataset.kumbukaCommandModule = "page-details";

    for (const option of options) {
      const item = document.createElement("option");
      item.value = option.action;
      item.textContent = option.label;
      item.selected = option.action === selected.action;
      item.dataset.tone = option.tone;
      select.append(item);
    }

    select.addEventListener("change", () => {
      for (const tone of tones)
        select.classList.remove(`status-dropdown-${tone}`);
      const tone = select.selectedOptions[0]?.dataset.tone || "gray";
      select.classList.add(`status-dropdown-${tone}`);
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
