// Draws a tree snapshot as SVG, and slides the keys from one snapshot to
// the next, so that a rotation looks like a rotation.

const GAP_X = 44; // between two keys of a binary tree
const GAP_Y = 70; // between two levels
const KEY_W = 30; // one key in a B-tree node
const PAD = 30;

const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);

// layout gives each key a position. A binary tree uses the order of the
// keys for x and the depth for y. A B-tree puts its leaves side by side,
// and each parent above the middle of its children.
export function layout(tree, multi = false) {
  const keys = new Map(); // key → {x, y, color, info, box}
  const edges = []; // binary: [parent key, child key]
  const boxes = []; // B-tree: {keys, children: [box index]}
  if (!tree) return { keys, edges, boxes, width: 0, height: 0, multi };
  let depthMax = 0;

  if (!multi) {
    let order = 0;
    const walk = (n, depth, parent) => {
      if (!n) return;
      walk(n.children?.[0], depth + 1, n.keys[0]);
      keys.set(n.keys[0], { x: PAD + order * GAP_X, y: PAD + depth * GAP_Y, color: n.color ?? "", info: n.info ?? "" });
      order++;
      depthMax = Math.max(depthMax, depth);
      if (parent !== undefined) edges.push([parent, n.keys[0]]);
      walk(n.children?.[1], depth + 1, n.keys[0]);
    };
    walk(tree, 0);
    return { keys, edges, boxes, width: PAD * 2 + Math.max(0, order - 1) * GAP_X, height: PAD * 2 + depthMax * GAP_Y, multi };
  }

  let cursor = PAD;
  const place = (n, depth) => {
    const index = boxes.length;
    const box = { keys: n.keys, children: [], center: 0, depth };
    boxes.push(box);
    depthMax = Math.max(depthMax, depth);
    const width = n.keys.length * KEY_W;
    if (!n.children?.length) {
      box.center = cursor + width / 2;
      cursor += width + 18;
    } else {
      for (const c of n.children) box.children.push(place(c, depth + 1));
      box.center = (boxes[box.children[0]].center + boxes[box.children.at(-1)].center) / 2;
    }
    n.keys.forEach((k, i) => {
      keys.set(k, { x: box.center - width / 2 + (i + 0.5) * KEY_W, y: PAD + depth * GAP_Y, color: "", info: "", box: index });
    });
    return index;
  };
  place(tree, 0);
  return { keys, edges, boxes, width: cursor - 18 + PAD, height: PAD * 2 + depthMax * GAP_Y, multi };
}

const ease = (t) => (t < 0.5 ? 2 * t * t : 1 - (-2 * t + 2) ** 2 / 2);

export class TreeDrawing {
  // multi draws B-tree nodes, with many keys in a box.
  constructor(container, { compact = false, multi = false } = {}) {
    this.container = container;
    this.compact = compact;
    this.multi = multi;
    this.current = null;
    this.frame = 0;
  }

  // show draws a snapshot. With a duration, the keys slide from where
  // they were.
  show(tree, marks = [], duration = 0) {
    const next = layout(tree, this.multi);
    const prev = this.current;
    this.current = next;
    cancelAnimationFrame(this.frame);
    if (!tree) {
      this.container.innerHTML = `<p class="empty">${this.compact ? "No keys yet." : "An empty clearing. Insert a key to plant the first tree."}</p>`;
      return;
    }
    const marked = new Set(marks);
    if (!prev || !duration) {
      this.render(next, next, 1, marked);
      return;
    }
    const start = performance.now();
    const tick = (now) => {
      const t = Math.min(1, (now - start) / duration);
      this.render(prev, next, ease(t), marked);
      if (t < 1) this.frame = requestAnimationFrame(tick);
    };
    this.frame = requestAnimationFrame(tick);
  }

  render(prev, next, t, marked) {
    const pos = new Map();
    for (const [k, p] of next.keys) {
      const from = prev.keys.get(k) ?? p;
      pos.set(k, { x: from.x + (p.x - from.x) * t, y: from.y + (p.y - from.y) * t, appear: prev.keys.has(k) ? 1 : t });
    }
    const width = Math.max(next.width, prev === next ? 0 : prev.width * (1 - t) + next.width * t, 120);
    const height = Math.max(next.height, 90);
    const parts = [];

    if (!next.multi) {
      for (const [a, b] of next.edges) {
        const p = pos.get(a);
        const q = pos.get(b);
        parts.push(`<line class="branch" x1="${p.x}" y1="${p.y}" x2="${q.x}" y2="${q.y}" opacity="${q.appear}"/>`);
      }
      for (const [k, info] of next.keys) {
        const p = pos.get(k);
        const cls = ["leaf", info.color, marked.has(k) ? "mark" : ""].join(" ");
        parts.push(`<g class="${cls}" transform="translate(${p.x} ${p.y}) scale(${0.3 + 0.7 * p.appear})">`
          + `<circle r="17"/><text>${esc(k)}</text>`
          + (info.info && !this.compact ? `<text class="info" y="29">${esc(info.info)}</text>` : "")
          + "</g>");
      }
    } else {
      // A box sits around the current positions of its keys.
      const geometry = next.boxes.map((box) => {
        const xs = box.keys.map((k) => pos.get(k).x);
        const y = pos.get(box.keys[0]).y;
        return { left: Math.min(...xs) - KEY_W / 2, right: Math.max(...xs) + KEY_W / 2, y };
      });
      next.boxes.forEach((box, i) => {
        box.children.forEach((c, slot) => {
          const g = geometry[i];
          const x = g.left + slot * KEY_W;
          const child = geometry[c];
          parts.push(`<line class="branch" x1="${x}" y1="${g.y + 14}" x2="${(child.left + child.right) / 2}" y2="${child.y - 14}"/>`);
        });
      });
      next.boxes.forEach((box, i) => {
        const g = geometry[i];
        const mark = box.keys.some((k) => marked.has(k)) ? " mark" : "";
        parts.push(`<rect class="plank${mark}" x="${g.left}" y="${g.y - 14}" width="${g.right - g.left}" height="28" rx="7"/>`);
        for (let s = 1; s < box.keys.length; s++) {
          parts.push(`<line class="slot" x1="${g.left + s * KEY_W}" y1="${g.y - 10}" x2="${g.left + s * KEY_W}" y2="${g.y + 10}"/>`);
        }
        for (const k of box.keys) {
          const p = pos.get(k);
          parts.push(`<text class="bkey${marked.has(k) ? " mark" : ""}" x="${p.x}" y="${p.y}">${esc(k)}</text>`);
        }
      });
    }
    this.container.innerHTML = `<svg viewBox="0 0 ${width} ${height + 20}" role="img" aria-label="A tree with ${next.keys.size} keys">${parts.join("")}</svg>`;
  }
}
