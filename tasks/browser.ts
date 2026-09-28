(() => {
  const statePrefix = "kumbuka-task-state__";
  const choicePrefix = "kumbuka-task-choice__";
  const styleID = "kumbuka-task-browser-styles";

  type BrowserContext = {
    html: string;
    locale: string;
  };

  type Choice = {
    id: string;
    action: string;
    label: string;
    color: string;
    completed: boolean;
  };

  const browserStyles = `
html,
body {
  margin: 0;
  padding: 0;
  background: transparent;
}

body,
#plugin-root,
.task-root {
  display: block;
  width: 100%;
}

.kumbuka-task-list {
  display: block;
  width: 100%;
  box-sizing: border-box;
  font-family: inherit;
}

.kumbuka-task-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.28rem;
  padding: 0 0.15rem 0.34rem;
  border-bottom: 1px solid color-mix(in srgb, currentColor 9%, transparent);
}

.kumbuka-task-list-title {
  font-size: 0.84rem;
  font-weight: 700;
  letter-spacing: 0.005em;
}

.kumbuka-task-progress {
  display: inline-flex;
  align-items: center;
  min-height: 1.35rem;
  padding: 0 0.42rem;
  border-radius: 999px;
  background: color-mix(in srgb, currentColor 6%, transparent);
  color: var(--text-secondary, #64748b);
  font-size: 0.69rem;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.kumbuka-task-items,
.kumbuka-task-node,
.kumbuka-task-children {
  display: block;
  min-width: 0;
}

.kumbuka-task-node {
  position: relative;
}

.kumbuka-task-node + .kumbuka-task-node {
  margin-top: 0.06rem;
}

.kumbuka-task-children {
  position: relative;
  margin-left: 0.67rem;
  padding-left: 0.73rem;
  border-left: 1px solid color-mix(in srgb, currentColor 13%, transparent);
}

.kumbuka-task-children > .kumbuka-task-node > .task-item::before {
  content: "";
  position: absolute;
  left: -0.77rem;
  top: 0.82rem;
  width: 0.56rem;
  border-top: 1px solid color-mix(in srgb, currentColor 13%, transparent);
}

.task-item {
  position: relative;
  display: grid;
  grid-template-columns: 1.35rem minmax(0, 1fr);
  align-items: start;
  gap: 0.44rem;
  box-sizing: border-box;
  min-width: 0;
  padding: 0.34rem 0.38rem;
  border-radius: 0.42rem;
}

.kumbuka-task-node:hover > .task-item {
  background: color-mix(in srgb, currentColor 3.5%, transparent);
}

.task-state-control {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.22rem;
  height: 1.22rem;
  margin-top: 0.08rem;
  border-radius: 999px;
  cursor: pointer;
}

.task-state-indicator {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 0.94rem;
  height: 0.94rem;
  box-sizing: border-box;
  border: 1.55px solid var(--task-state-color, currentColor);
  border-radius: 999px;
  background: transparent;
  color: transparent;
  font-size: 0.63rem;
  font-weight: 800;
  line-height: 1;
  pointer-events: none;
  transition:
    transform 100ms ease,
    background-color 120ms ease,
    border-color 120ms ease,
    color 120ms ease;
}

.task-state-control:hover .task-state-indicator {
  transform: scale(1.08);
}

.task-state-select {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  opacity: 0;
  cursor: pointer;
}

.task-state-control:focus-within .task-state-indicator {
  outline: 2px solid color-mix(in srgb, var(--task-state-color, currentColor) 32%, transparent);
  outline-offset: 2px;
}

.kumbuka-task-content {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.13rem;
}

.kumbuka-task-text {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 0.91rem;
  font-weight: 540;
  line-height: 1.34;
}

.kumbuka-task-details {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.26rem;
  min-height: 1.1rem;
  color: var(--text-secondary, #64748b);
  font-size: 0.69rem;
  line-height: 1.25;
}

.kumbuka-task-state-label,
.kumbuka-task-assignee {
  display: inline-flex;
  align-items: center;
  min-height: 1.08rem;
  box-sizing: border-box;
  border-radius: 999px;
  white-space: nowrap;
}

.kumbuka-task-state-label {
  gap: 0.24rem;
  padding: 0 0.34rem;
  border: 1px solid color-mix(in srgb, var(--task-state-color, currentColor) 28%, transparent);
  background: color-mix(in srgb, var(--task-state-color, currentColor) 7%, transparent);
  color: color-mix(in srgb, var(--task-state-color, currentColor) 76%, currentColor);
  font-weight: 650;
}

.kumbuka-task-state-label::before {
  content: "";
  width: 0.34rem;
  height: 0.34rem;
  border-radius: 999px;
  background: var(--task-state-color, currentColor);
}

.kumbuka-task-assignee {
  padding: 0 0.34rem;
  background: color-mix(in srgb, currentColor 6%, transparent);
  font-weight: 650;
}

.kumbuka-task-due {
  white-space: nowrap;
}

.kumbuka-task-completed .task-state-indicator {
  background: var(--task-state-color, currentColor);
  color: var(--task-state-foreground, #ffffff);
}

.kumbuka-task-completed .kumbuka-task-text {
  opacity: 0.56;
  text-decoration: line-through;
}

.kumbuka-task-completed .kumbuka-task-details {
  opacity: 0.78;
}
`;

  function ensureStyles(): void {
    if (document.getElementById(styleID)) return;
    const style = document.createElement("style");
    style.id = styleID;
    style.textContent = browserStyles;
    (document.head || document.documentElement).append(style);
  }

  function classValue(element: HTMLElement, prefix: string): string {
    for (const name of element.classList) {
      if (name.startsWith(prefix)) return name.slice(prefix.length);
    }
    return "";
  }

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

  function parseChoices(row: HTMLElement): Choice[] {
    const result: Choice[] = [];
    for (const className of row.classList) {
      if (!className.startsWith(choicePrefix)) continue;
      const fields = className.slice(choicePrefix.length).split("__");
      if (fields.length !== 5) continue;

      const [encodedID, action, color, completed, encodedLabel] = fields;
      const id = decodeHex(encodedID || "");
      const label = decodeHex(encodedLabel || "");
      if (
        !/^[a-z0-9][a-z0-9._-]{0,63}$/.test(id) ||
        !/^task-[a-f0-9]{24}$/.test(action || "") ||
        !/^[a-f0-9]{6}$/.test(color || "") ||
        (completed !== "0" && completed !== "1") ||
        !label
      )
        continue;

      result.push({
        id,
        action: action || "",
        label,
        color: `#${color}`,
        completed: completed === "1",
      });
    }
    return result;
  }

  function taskStateLabel(locale: string): string {
    return locale.toLowerCase().split("-")[0] === "de"
      ? "Aufgabenstatus"
      : "Task state";
  }

  function readableForeground(color: string): string {
    const red = Number.parseInt(color.slice(1, 3), 16);
    const green = Number.parseInt(color.slice(3, 5), 16);
    const blue = Number.parseInt(color.slice(5, 7), 16);
    const luminance = (red * 299 + green * 587 + blue * 114) / 1000;
    return luminance >= 150 ? "#111827" : "#ffffff";
  }

  function updateProgress(list: HTMLElement): void {
    const current = list.querySelector<HTMLElement>(
      ".kumbuka-task-progress-current",
    );
    if (!current) return;
    current.textContent = String(
      list.querySelectorAll(".task-item.kumbuka-task-completed").length,
    );
  }

  function applyChoice(
    row: HTMLElement,
    control: HTMLElement,
    indicator: HTMLElement,
    stateLabel: HTMLElement | null,
    choice: Choice,
  ): void {
    row.classList.toggle("kumbuka-task-completed", choice.completed);
    row.style.setProperty("--task-state-color", choice.color);
    row.style.setProperty(
      "--task-state-foreground",
      readableForeground(choice.color),
    );
    control.title = choice.label;
    indicator.textContent = choice.completed ? "✓" : "";
    if (stateLabel) stateLabel.textContent = choice.label;
  }

  function upgradeTask(
    row: HTMLElement,
    list: HTMLElement,
    locale: string,
  ): void {
    const box = row.querySelector<HTMLElement>(".kumbuka-task-box");
    if (!box) throw new Error("Invalid task input");

    const state = classValue(row, statePrefix);
    if (!/^[a-z0-9][a-z0-9._-]{0,63}$/.test(state))
      throw new Error("Invalid task state");

    const choices = parseChoices(row);
    const current = choices.find((choice) => choice.id === state);
    if (!current) throw new Error("Unknown task state");

    const control = document.createElement("label");
    control.className = "task-state-control";

    const indicator = document.createElement("span");
    indicator.className = "task-state-indicator";
    indicator.setAttribute("aria-hidden", "true");

    const select = document.createElement("select");
    select.className = "task-state-select";
    select.dataset.kumbukaCommandModule = "page-details";
    select.setAttribute(
      "aria-label",
      `${taskStateLabel(locale || "en")}: ${current.label}`,
    );

    for (const choice of choices) {
      const option = document.createElement("option");
      option.value = choice.action;
      option.textContent = choice.label;
      option.selected = choice.id === state;
      select.append(option);
    }

    const stateLabel = row.querySelector<HTMLElement>(
      ".kumbuka-task-state-label",
    );
    applyChoice(row, control, indicator, stateLabel, current);
    select.addEventListener("change", () => {
      const selected = choices.find((choice) => choice.action === select.value);
      if (!selected) return;
      applyChoice(row, control, indicator, stateLabel, selected);
      select.setAttribute(
        "aria-label",
        `${taskStateLabel(locale || "en")}: ${selected.label}`,
      );
      updateProgress(list);
    });

    control.append(indicator, select);
    box.replaceWith(control);
    row.classList.remove("kumbuka-task-fallback");
    row.classList.add("task-item");
  }

  function renderTaskList(root: HTMLElement, context: BrowserContext): void {
    ensureStyles();

    const template = document.createElement("template");
    template.innerHTML = context.html;

    const fallback =
      template.content.querySelector<HTMLElement>(".kumbuka-task-list");
    if (!fallback) throw new Error("Invalid task list input");

    const list = fallback.cloneNode(true) as HTMLElement;
    list.removeAttribute("data-kumbuka-fallback");
    const rows = [
      ...list.querySelectorAll<HTMLElement>(".kumbuka-task-fallback"),
    ];
    if (!rows.length) throw new Error("Invalid task list input");

    for (const row of rows) upgradeTask(row, list, context.locale || "en");
    updateProgress(list);

    root.className = "task-root";
    root.replaceChildren(list);
  }

  (globalThis as unknown as { kumbukaPlugin: unknown }).kumbukaPlugin = {
    render(root: HTMLElement, context: BrowserContext) {
      renderTaskList(root, context);
    },
  };
})();
