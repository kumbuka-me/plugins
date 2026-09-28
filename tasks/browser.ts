(() => {
  const statePrefix = "kumbuka-task-state__";
  const choiceIDPrefix = "kumbuka-task-choice-id__";
  const choiceActionPrefix = "kumbuka-task-choice-action__";
  const choiceColorPrefix = "kumbuka-task-choice-color__";
  const choiceCompletedPrefix = "kumbuka-task-choice-completed__";

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

  // classValue returns the suffix of the first class with the requested prefix.
  function classValue(element: HTMLElement, prefix: string): string {
    for (const name of element.classList) {
      if (name.startsWith(prefix)) return name.slice(prefix.length);
    }
    return "";
  }

  // parseChoice validates one sanitized host-rendered workflow choice.
  function parseChoice(element: HTMLElement): Choice {
    const id = classValue(element, choiceIDPrefix);
    const action = classValue(element, choiceActionPrefix);
    const color = classValue(element, choiceColorPrefix);
    const completed = classValue(element, choiceCompletedPrefix);
    if (!/^[a-z0-9][a-z0-9._-]{0,63}$/.test(id))
      throw new Error("Invalid task state ID");
    if (!/^task-[a-f0-9]{24}$/.test(action))
      throw new Error("Invalid task action");
    if (!/^[a-f0-9]{6}$/.test(color))
      throw new Error("Invalid task state color");
    if (completed !== "true" && completed !== "false")
      throw new Error("Invalid task completion marker");
    return {
      id,
      action,
      label: element.textContent?.trim() || id,
      color: `#${color}`,
      completed: completed === "true",
    };
  }

  function taskStateLabel(locale: string): string {
    return locale.toLowerCase().split("-")[0] === "de"
      ? "Aufgabenstatus"
      : "Task state";
  }

  // readableForeground chooses a readable foreground for a completed-state fill.
  function readableForeground(color: string): string {
    const red = Number.parseInt(color.slice(1, 3), 16);
    const green = Number.parseInt(color.slice(3, 5), 16);
    const blue = Number.parseInt(color.slice(5, 7), 16);
    const luminance = (red * 299 + green * 587 + blue * 114) / 1000;
    return luminance >= 150 ? "#111827" : "#ffffff";
  }

  // updateProgress refreshes the compact completed/total counter for the task list.
  function updateProgress(list: HTMLElement): void {
    const current = list.querySelector<HTMLElement>(
      ".kumbuka-task-progress-current",
    );
    if (!current) return;
    current.textContent = String(
      list.querySelectorAll(".task-item.kumbuka-task-completed").length,
    );
  }

  // applyChoice updates one row immediately while the host persists the command.
  function applyChoice(
    row: HTMLElement,
    control: HTMLElement,
    indicator: HTMLElement,
    choice: Choice,
  ): void {
    row.classList.toggle("kumbuka-task-completed", choice.completed);
    control.title = choice.label;
    indicator.style.borderColor = choice.color;
    indicator.style.backgroundColor = choice.completed
      ? choice.color
      : "transparent";
    indicator.style.color = choice.completed
      ? readableForeground(choice.color)
      : "transparent";
    indicator.textContent = choice.completed ? "✓" : "";
  }

  // upgradeTask turns one passive fallback row into a compact interactive row.
  function upgradeTask(
    row: HTMLElement,
    list: HTMLElement,
    locale: string,
  ): void {
    const metadata = row.querySelector<HTMLElement>(".kumbuka-task-meta");
    const box = row.querySelector<HTMLElement>(".kumbuka-task-box");
    const choiceElements = [
      ...row.querySelectorAll<HTMLElement>(".kumbuka-task-choice"),
    ];
    if (!metadata || !box || !choiceElements.length)
      throw new Error("Invalid task input");

    const state = classValue(metadata, statePrefix);
    if (!/^[a-z0-9][a-z0-9._-]{0,63}$/.test(state))
      throw new Error("Invalid task state");

    const choices = choiceElements.map(parseChoice);
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

    applyChoice(row, control, indicator, current);
    select.addEventListener("change", () => {
      const selected = choices.find((choice) => choice.action === select.value);
      if (!selected) return;
      applyChoice(row, control, indicator, selected);
      select.setAttribute(
        "aria-label",
        `${taskStateLabel(locale || "en")}: ${selected.label}`,
      );
      updateProgress(list);
    });

    control.append(indicator, select);
    box.replaceWith(control);
    row.querySelector(".kumbuka-task-state-label")?.remove();
    row.classList.remove("kumbuka-task-fallback");
    row.classList.add("task-item");
  }

  // renderTaskList upgrades one host-sanitized fallback task list.
  function renderTaskList(root: HTMLElement, context: BrowserContext): void {
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
