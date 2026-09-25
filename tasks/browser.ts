(() => {
  const actionPrefix = "kumbuka-task-action__";
  const statePrefix = "kumbuka-task-state__";

  type BrowserContext = {
    html: string;
  };

  // classValue returns the suffix of the first class with the requested prefix.
  function classValue(element: HTMLElement, prefix: string): string {
    for (const name of element.classList) {
      if (name.startsWith(prefix)) return name.slice(prefix.length);
    }
    return "";
  }

  // renderTask replaces the passive sanitized fallback with an interactive checkbox.
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
    if (!fallback || !metadata || !content)
      throw new Error("Invalid task input");

    const action = classValue(metadata, actionPrefix);
    const state = classValue(metadata, statePrefix);
    if (!/^[a-z0-9][a-z0-9._-]{0,127}$/.test(action))
      throw new Error("Invalid task action");
    if (state !== "open" && state !== "done")
      throw new Error("Invalid task state");

    const shell = document.createElement("label");
    shell.className = `task-shell${state === "done" ? " task-shell-done" : ""}`;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.className = "task-checkbox";
    checkbox.checked = state === "done";
    checkbox.setAttribute("aria-label", "Toggle task completion");

    const command = document.createElement("select");
    command.className = "task-command";
    command.tabIndex = -1;
    command.setAttribute("aria-hidden", "true");
    command.dataset.kumbukaCommandModule = "page-details";
    const option = document.createElement("option");
    option.value = action;
    option.textContent = action;
    option.selected = true;
    command.append(option);

    checkbox.addEventListener("change", () => {
      shell.classList.toggle("task-shell-done", checkbox.checked);
      command.dispatchEvent(new Event("change", { bubbles: true }));
    });

    shell.append(checkbox, content.cloneNode(true), command);
    root.className = "task-root";
    root.replaceChildren(shell);
  }

  (globalThis as unknown as { kumbukaPlugin: unknown }).kumbukaPlugin = {
    render(root: HTMLElement, context: BrowserContext) {
      renderTask(root, context);
    },
  };
})();
