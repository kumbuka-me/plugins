interface Route {
  label: string;
  target: string;
  line: number;
}
interface Step {
  id: string;
  title: string;
  routes: Route[];
}

function parseRunbook(source: string, validate = true): Step[] {
  const steps: Step[] = [];
  const positions = new Map<string, number>();
  function fail(line: number, message: string): never {
    throw new Error(`Line ${line}: ${message}`);
  }
  const lines = source.trimEnd().split(/\r?\n/);
  if (lines.length > 512)
    throw new Error(
      "Split runbooks longer than 512 lines into smaller procedures.",
    );
  for (const [index, raw] of lines.entries()) {
    const line = index + 1;
    if (!raw.trim()) continue;
    const text = raw.trim();
    const header = /^([a-z][a-z0-9_-]*):\s*(.*)$/i.exec(text);
    if (header && !/^\s/.test(raw)) {
      const [, id, title] = header;
      if (positions.has(id)) fail(line, `duplicate step "${id}".`);
      positions.set(id, steps.length);
      steps.push({ id, title, routes: [] });
      continue;
    }
    const route = /^(.*?)\s*->\s*([a-z][a-z0-9_-]*)$/i.exec(text);
    const step = steps.at(-1);
    if (!route || !step || !/^\s/.test(raw))
      fail(line, 'use "id: Step title" or an indented "Answer -> id".');
    step.routes.push({ label: route[1].trim(), target: route[2], line });
  }
  if (!steps.length) throw new Error("Add a starting step.");
  if (!validate) return steps;
  const reached = new Set([steps[0].id]);
  for (const [index, step] of steps.entries()) {
    if (!step.title.trim()) throw new Error(`Give step "${step.id}" a title.`);
    if (!reached.has(step.id))
      throw new Error(`No route reaches "${step.id}".`);
    const labels = new Set<string>();
    for (const route of step.routes) {
      const target = positions.get(route.target);
      if (target === undefined)
        fail(route.line, `unknown step "${route.target}".`);
      if (target <= index)
        fail(route.line, "routes must point to a later step.");
      if (step.routes.length > 1 && !route.label)
        fail(route.line, "label each choice.");
      if (labels.has(route.label))
        fail(route.line, "use a distinct choice label.");
      labels.add(route.label);
      reached.add(route.target);
    }
  }
  return steps;
}

interface Row {
  id: string;
  title: string;
  kind: "action" | "decision" | "end" | "choice";
  badge: string;
  targets: string[];
}

// Put a choice beside its destination unless that destination has multiple incoming routes.
function graphRows(steps: Step[]): Row[] {
  const incoming = new Map<string, Route[]>();
  for (const step of steps) {
    for (const route of step.routes) {
      const routes = incoming.get(route.target) ?? [];
      routes.push(route);
      incoming.set(route.target, routes);
    }
  }
  const rows: Row[] = [];
  for (const step of steps) {
    const routes = incoming.get(step.id);
    const row: Row = {
      id: step.id,
      title: step.title,
      kind:
        step.routes.length > 1
          ? "decision"
          : step.routes.length
            ? "action"
            : "end",
      badge: routes?.length === 1 ? routes[0].label : "",
      targets: [],
    };
    rows.push(row);
    for (const route of step.routes) {
      if (route.label && incoming.get(route.target)!.length > 1) {
        const id = `choice:${route.line}`;
        row.targets.push(id);
        rows.push({
          id,
          title: route.label,
          kind: "choice",
          badge: "",
          targets: [route.target],
        });
      } else {
        row.targets.push(route.target);
      }
    }
  }
  return rows;
}

const svgNS = "http://www.w3.org/2000/svg";
const laneX = (lane: number) => 12 + lane * 18;

// A lane holds its next destination until that row consumes it. Shared targets converge there.
function drawGraph(rows: Row[]): HTMLOListElement {
  const list = document.createElement("ol");
  list.className = "runbook";
  list.setAttribute("aria-label", "Decision runbook");
  const lanes: (string | null)[] = [];
  const titles = new Map(
    rows.map((row) => [
      row.id,
      row.badge ? row.badge + ": " + row.title : row.title,
    ]),
  );
  for (const row of rows) {
    const incoming = lanes.indexOf(row.id);
    const lane = Math.max(0, incoming);
    const x = laneX(lane);
    const svg = document.createElementNS(svgNS, "svg");
    svg.setAttribute("aria-hidden", "true");
    function path(d: string, style = "rail") {
      const line = document.createElementNS(svgNS, "path");
      line.setAttribute("d", d);
      line.setAttribute("class", style);
      svg.append(line);
    }
    // Draw the upper half from the previous row, then release all arriving lanes.
    for (const [i, target] of lanes.entries()) {
      if (!target) continue;
      const start = laneX(i);
      if (target === row.id) {
        path(`M ${start} 0 Q ${start} 18 ${x} 18`);
        lanes[i] = null;
      } else {
        path(`M ${start} 0 V 36`);
      }
    }
    for (const [index, target] of row.targets.entries()) {
      let next = index === 0 ? lane : lanes.indexOf(null);
      if (next < 0) next = lanes.length;
      lanes[next] = target;
      const end = laneX(next);
      path(`M ${x} 18 Q ${end} 18 ${end} 36`);
    }
    const width = laneX(Math.max(lanes.length - 1, lane)) + 18;
    if (row.badge || row.kind === "choice")
      path(`M ${x} 18 H ${width}`, "answer-link");
    if (row.kind === "decision") {
      path(`M ${x} 11 l 7 7 -7 7 -7 -7 Z`, "marker");
    } else {
      const dot = document.createElementNS(svgNS, "circle");
      dot.setAttribute("cx", String(x));
      dot.setAttribute("cy", "18");
      dot.setAttribute("r", row.kind === "choice" ? "3" : "5");
      dot.setAttribute("class", "marker");
      svg.append(dot);
    }

    svg.setAttribute("width", String(width));
    svg.setAttribute("height", "36");
    svg.setAttribute("viewBox", `0 0 ${width} 36`);
    svg.setAttribute("preserveAspectRatio", "none");
    const item = document.createElement("li");
    item.className = row.kind;
    const label = document.createElement("span");
    if (row.badge) {
      const badge = document.createElement("b");
      badge.className = "badge";
      badge.textContent = row.badge;
      label.append(badge);
    }
    label.append(row.title);
    if (row.targets.length) {
      const next = document.createElement("span");
      next.className = "sr-only";
      next.textContent =
        ". Next: " + row.targets.map((id) => titles.get(id)).join(" or ");
      label.append(next);
    }
    item.append(svg, label);
    list.append(item);
  }
  return list;
}

// serializeRunbook writes the plugin's fenced-code grammar from its structured editor state.
function serializeRunbook(steps: Step[]): string {
  return steps
    .flatMap((step) => [
      `${step.id}: ${step.title}`,
      ...step.routes.map(
        (route) =>
          `  ${route.label ? `${route.label} ` : ""}-> ${route.target}`,
      ),
    ])
    .join("\n");
}

function appendStep(steps: Step[]): Step {
  const ids = new Set(steps.map((step) => step.id));
  let number = steps.length + 1;
  while (ids.has(`step-${number}`)) number++;
  const step = { id: `step-${number}`, title: "New step", routes: [] };
  steps.push(step);
  return step;
}

// editRunbook mounts plugin-owned controls inside Kumbuka's isolated editor frame.
function editRunbook(
  root: HTMLElement,
  context: {
    source: string;
    theme: string;
    onChange: (value: string) => void;
    onValidity?: (state: { valid: boolean; message?: string }) => void;
  },
): void {
  root.dataset.theme = context.theme;
  root.classList.add("runbook-editor");
  const heading = document.createElement("h3");
  heading.textContent = "Runbook steps";
  const rows = document.createElement("div");
  rows.className = "runbook-editor-steps";
  const error = document.createElement("p");
  error.className = "runbook-editor-error";
  error.setAttribute("role", "status");
  const preview = document.createElement("div");
  preview.className = "runbook-editor-preview";
  let source = context.source;
  const addStep = document.createElement("button");
  addStep.type = "button";
  addStep.textContent = "+ Add step";
  const actions = document.createElement("div");
  actions.className = "runbook-editor-actions";
  actions.append(addStep);
  root.replaceChildren(heading, rows, actions, error, preview);

  const publish = (value: string) => {
    source = value;
    render();
    context.onChange(value);
  };

  const render = () => {
    rows.replaceChildren();
    preview.replaceChildren();
    error.textContent = "";
    let steps: Step[];
    try {
      steps = parseRunbook(source, false);
    } catch (reason) {
      error.textContent =
        reason instanceof Error ? reason.message : "Invalid runbook";
      error.textContent += " Edit this block in Markdown view to repair it.";
      context.onValidity?.({ valid: false, message: error.textContent });
      return;
    }
    try {
      parseRunbook(source);
      preview.append(drawGraph(graphRows(steps)));
      context.onValidity?.({ valid: true });
    } catch (reason) {
      error.textContent =
        reason instanceof Error ? reason.message : "Invalid runbook";
      context.onValidity?.({ valid: false, message: error.textContent });
    }
    for (const [index, step] of steps.entries()) {
      const card = document.createElement("fieldset");
      const legend = document.createElement("legend");
      legend.textContent = `${index + 1}. ${step.id}`;
      card.append(legend);
      const title = document.createElement("input");
      title.type = "text";
      title.value = step.title;
      title.setAttribute("aria-label", `Title for ${step.id}`);
      title.addEventListener("change", () => {
        steps[index].title = title.value;
        publish(serializeRunbook(steps));
      });
      card.append(title);
      for (const [routeIndex, route] of step.routes.entries()) {
        const routeRow = document.createElement("div");
        routeRow.className = "runbook-editor-route";
        const label = document.createElement("input");
        label.value = route.label;
        label.placeholder = "Choice label (optional)";
        label.setAttribute("aria-label", `Choice label from ${step.id}`);
        const target = document.createElement("select");
        target.setAttribute("aria-label", `Destination from ${step.id}`);
        for (const candidate of steps.slice(index + 1)) {
          const option = document.createElement("option");
          option.value = candidate.id;
          option.textContent = candidate.title;
          target.append(option);
        }
        target.value = route.target;
        const update = () => {
          steps[index].routes[routeIndex].label = label.value.trim();
          steps[index].routes[routeIndex].target = target.value;
          publish(serializeRunbook(steps));
        };
        label.addEventListener("change", update);
        target.addEventListener("change", update);
        routeRow.append(label, target);
        card.append(routeRow);
      }
      const addAnswer = document.createElement("button");
      addAnswer.type = "button";
      addAnswer.textContent = "Add answer";
      addAnswer.addEventListener("click", () => {
        const destination =
          steps
            .slice(index + 1)
            .find(
              (candidate) =>
                !step.routes.some((route) => route.target === candidate.id),
            ) ?? appendStep(steps);
        if (step.routes.length === 1 && !step.routes[0].label)
          step.routes[0].label = "Answer 1";
        let number = step.routes.length + 1;
        while (step.routes.some((route) => route.label === `Answer ${number}`))
          number++;
        step.routes.push({
          label: `Answer ${number}`,
          target: destination.id,
          line: 0,
        });
        publish(serializeRunbook(steps));
        const field = rows
          .querySelectorAll("fieldset")
          [index]?.querySelectorAll<HTMLInputElement>(
            ".runbook-editor-route input",
          );
        field?.[field.length - 1]?.select();
      });
      card.append(addAnswer);
      rows.append(card);
    }
  };

  addStep.addEventListener("click", () => {
    let steps: Step[];
    try {
      steps = parseRunbook(source);
    } catch {
      return;
    }
    const last = steps.at(-1)!;
    const next = appendStep(steps);
    last.routes.push({ label: "", target: next.id, line: 0 });
    publish(serializeRunbook(steps));
  });
  render();
}

Object.assign(globalThis, {
  kumbukaPlugin: {
    render(root: HTMLElement, context: { source: string; theme: string }) {
      root.dataset.theme = context.theme;
      try {
        root.replaceChildren(
          drawGraph(graphRows(parseRunbook(context.source))),
        );
      } catch (error) {
        const message = document.createElement("p");
        message.textContent =
          "Runbook: " +
          (error instanceof Error ? error.message : "Invalid graph.");
        root.replaceChildren(message);
      }
    },
    edit: editRunbook,
  },
});
