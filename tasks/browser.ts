(() => {
  const statePrefix = "kumbuka-task-state__";
  const choiceIDPrefix = "kumbuka-task-choice-id__";
  const choiceActionPrefix = "kumbuka-task-choice-action__";
  const choiceColorPrefix = "kumbuka-task-choice-color__";
  const choiceCompletedPrefix = "kumbuka-task-choice-completed__";

  type BrowserContext = {
    html: string;
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

  // renderTask replaces the passive sanitized fallback with an interactive workflow selector.
  function renderTask(root: HTMLElement, context: BrowserContext): void {
    const template = document.createElement("template");
    template.innerHTML = context.html;

    const fallback = template.content.querySelector<HTMLElement>(
      ".kumbuka-task-fallback",
    );
    const metadata =
      template.content.querySelector<HTMLElement>(".kumbuka-task-meta");
    const content = template.content.querySelector<HTMLElement>(
      ".kumbuka-task-content",
    );
    const choiceElements = [
      ...template.content.querySelectorAll<HTMLElement>(".kumbuka-task-choice"),
    ];
    if (!fallback || !metadata || !content || !choiceElements.length)
      throw new Error("Invalid task input");

    const state = classValue(metadata, statePrefix);
    if (!/^[a-z0-9][a-z0-9._-]{0,63}$/.test(state))
      throw new Error("Invalid task state");

    const choices = choiceElements.map(parseChoice);
    const current = choices.find((choice) => choice.id === state);
    if (!current) throw new Error("Unknown task state");

    const shell = document.createElement("div");
    shell.className = `task-shell${current.completed ? " task-shell-completed" : ""}`;

    const select = document.createElement("select");
    select.className = "task-state-select";
    select.dataset.kumbukaCommandModule = "page-details";
    select.setAttribute("aria-label", "Task state");
    select.style.borderColor = current.color;
    for (const choice of choices) {
      const option = document.createElement("option");
      option.value = choice.action;
      option.textContent = choice.label;
      option.selected = choice.id === state;
      select.append(option);
    }

    select.addEventListener("change", () => {
      const selected = choices.find((choice) => choice.action === select.value);
      if (!selected) return;
      shell.classList.toggle("task-shell-completed", selected.completed);
      select.style.borderColor = selected.color;
    });

    const renderedContent = content.cloneNode(true) as HTMLElement;
    renderedContent.querySelector(".kumbuka-task-state-label")?.remove();
    shell.append(select, renderedContent);
    root.className = "task-root";
    root.replaceChildren(shell);
  }

  (globalThis as unknown as { kumbukaPlugin: unknown }).kumbukaPlugin = {
    render(root: HTMLElement, context: BrowserContext) {
      renderTask(root, context);
    },
  };
})();
