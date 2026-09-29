(() => {
  type BrowserContext = { html: string };

  const action = /^toggle-([0-9]+)-([01])$/;
  const actionPrefix = "checklist-action__";

  function actionValue(button: HTMLButtonElement): RegExpExecArray | null {
    const token = [...button.classList].find((name) =>
      name.startsWith(actionPrefix),
    );
    return action.exec(token?.slice(actionPrefix.length) || "");
  }

  function render(root: HTMLElement, context: BrowserContext): void {
    const template = document.createElement("template");
    template.innerHTML = context.html;
    const buttons = [
      ...template.content.querySelectorAll<HTMLButtonElement>(
        ".checklist-checkbox",
      ),
    ];
    if (!buttons.length) throw new Error("Invalid checklist input");

    for (const button of buttons) {
      const match = actionValue(button);
      if (!match) throw new Error("Invalid checklist action");
      button.dataset.kumbukaCommandModule = "commands";
      button.dataset.kumbukaCommandAction = match[0];
      button.setAttribute("role", "checkbox");
      button.setAttribute("aria-checked", String(match[2] === "1"));
      button.setAttribute(
        "aria-label",
        match[2] === "1" ? "Mark incomplete" : "Mark complete",
      );
      button.addEventListener("click", () => {
        const checked = match[2] === "1";
        button.classList.toggle("checked", !checked);
        button.setAttribute("aria-checked", String(!checked));
        button.textContent = "✓";
        button.disabled = true;
      });
    }

    root.replaceChildren(template.content);
  }

  (globalThis as unknown as { kumbukaPlugin: unknown }).kumbukaPlugin = {
    render,
  };
})();
