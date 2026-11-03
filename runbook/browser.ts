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

function parseRunbook(source: string): Step[] {
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
    const header = /^([a-z][a-z0-9_-]*):\s*(.+)$/i.exec(text);
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
  const reached = new Set([steps[0].id]);
  for (const [index, step] of steps.entries()) {
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
    const width = laneX(Math.max(lanes.length - 1, lane)) + 12;
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
        const source = document.createElement("pre");
        source.textContent = context.source;
        root.replaceChildren(message, source);
      }
    },
  },
});
