// Tree Forest. The trees run in Go (cmd/forest-wasm). This file plays
// their steps and holds the views.
import { TreeDrawing } from "./draw.js";

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];

function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === undefined || value === null || value === false) continue;
    if (key === "class") node.className = value;
    // The page policy blocks style attributes, so set the styles one by one.
    else if (key === "style") Object.assign(node.style, value);
    else if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value === true ? "" : String(value));
  }
  node.append(...children);
  return node;
}

/* Go */

async function loadGo() {
  await new Promise((resolve, reject) => {
    const script = el("script", { src: "wasm_exec.js" });
    script.onload = resolve;
    script.onerror = () => reject(new Error("cannot load wasm_exec.js"));
    document.head.append(script);
  });
  const go = new window.Go();
  const ready = new Promise((resolve) => window.addEventListener("forest-ready", resolve, { once: true }));
  const result = await WebAssembly.instantiateStreaming(fetch("forest.wasm"), go.importObject);
  go.run(result.instance);
  await ready;
}

function call(name, arg) {
  const reply = JSON.parse(window.forest[name](arg === undefined ? undefined : JSON.stringify(arg)));
  if (!reply.ok) throw new Error(reply.error);
  return reply.result;
}

/* Helpers */

const SHORT = { bst: "BST", avl: "AVL", redblack: "Red-black", splay: "Splay", treap: "Treap", btree: "B-tree", heap: "Heap" };
const MAX_KEYS = 63;
const fmt = (v, digits = 0) => (Number.isFinite(v) ? v.toLocaleString("en-US", { maximumFractionDigits: digits, minimumFractionDigits: digits }) : "–");
const ns = (v) => (v >= 1000 ? `${fmt(v / 1000, 2)} µs` : `${fmt(v, 0)} ns`);
const bestHeight = (n) => (n === 0 ? 0 : Math.ceil(Math.log2(n + 1)));

function keysOf(tree, out = []) {
  if (!tree) return out;
  out.push(...tree.keys);
  for (const c of tree.children ?? []) keysOf(c, out);
  return out;
}

function shuffle(list) {
  for (let i = list.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [list[i], list[j]] = [list[j], list[i]];
  }
  return list;
}

function randomKeys(n) {
  return shuffle(Array.from({ length: 99 }, (_, i) => i + 1)).slice(0, n);
}

// order makes the keys 1..n in a pattern, like trees.Order in Go.
function order(pattern, n) {
  const keys = Array.from({ length: n }, (_, i) => i + 1);
  if (pattern === "random") return shuffle(keys);
  if (pattern === "reversed") return keys.reverse();
  if (pattern === "zigzag") return keys.map((_, i) => (i % 2 === 0 ? i / 2 + 1 : n - (i - 1) / 2));
  return keys;
}

function stats(node, items) {
  node.replaceChildren(...items.map(([k, v]) => el("div", {}, el("dt", {}, k), el("dd", {}, v))));
}

/* The tree tabs */

let kinds = [];
const groves = {}; // kind → {id, steps, pos, pending, last, log}
let kind = "avl";
let drawing = null;
let playing = true;
let timer = 0;

const interval = () => 1500 / Number($("#speed").value);

function grove(k) {
  if (!groves[k]) {
    const degree = k === "btree" ? Number($("#degree").value) : 0;
    groves[k] = { id: call("create", { kind: k, degree }), steps: [], pos: -1, pending: [], last: null, log: [] };
  }
  return groves[k];
}

function flowsFor(k) {
  const g = () => grove(k);
  const current = () => keysOf(g().last?.tree ?? null);
  const list = [
    { name: "Grow 1 to 15 in order", hint: "the worst case of a plain BST", ops: () => Array.from({ length: 15 }, (_, i) => ["insert", i + 1]) },
    { name: "Grow 15 random keys", hint: "from 1 to 99", ops: () => randomKeys(15).map((key) => ["insert", key]) },
    { name: "Search for every key", hint: "in order", ops: () => current().sort((a, b) => a - b).map((key) => ["search", key]) },
    { name: "Cut every key, smallest first", hint: "", ops: () => current().sort((a, b) => a - b).map((key) => ["delete", key]) },
    { name: "Cut every key at random", hint: "", ops: () => shuffle(current()).map((key) => ["delete", key]) },
  ];
  if (k === "heap") {
    list.splice(3, 0, { name: "Heap sort", hint: "delete the root until the heap is empty", ops: () => current().map(() => ["delete", "root"]) });
  }
  if (k === "splay") {
    list.splice(3, 0, { name: "Search one key five times", hint: "it stays at the root", ops: () => {
      const keys = current();
      const key = keys[Math.floor(Math.random() * keys.length)];
      return key === undefined ? [] : Array.from({ length: 5 }, () => ["search", key]);
    } });
  }
  if (k === "btree") {
    list.splice(1, 0, { name: "Grow 1 to 40 in order", hint: "watch the nodes split", ops: () => Array.from({ length: 40 }, (_, i) => ["insert", i + 1]) });
  }
  return list;
}

function openTree(k) {
  kind = k;
  const info = kinds.find((x) => x.key === k);
  $("#tree-name").textContent = info.name;
  $("#tree-summary").textContent = info.summary;
  $("#tree-search").replaceChildren("Search: ", el("strong", {}, info.search), ". Restructures: ", el("strong", {}, info.restructures), ".");
  $("#degree-box").hidden = k !== "btree";
  $("#heap-array").hidden = k !== "heap";
  $("#flows").replaceChildren(
    ...flowsFor(k).map((f) => el("button", { type: "button", onclick: () => queue(f.ops()) }, f.name, f.hint ? el("small", {}, f.hint) : "")),
  );
  drawing = new TreeDrawing($("#canvas"), { multi: k === "btree" });
  const g = grove(k);
  const step = g.steps[g.pos];
  drawing.show(step?.tree ?? g.last?.tree ?? null, step?.mark ?? []);
  showNote(step);
  showStats();
  showLog();
  showArray(step?.tree ?? null, step?.mark ?? []);
  schedule();
}

function queue(ops) {
  const g = grove(kind);
  g.pending.push(...ops);
  playing = true;
  $("#play").textContent = "⏸";
  schedule(0);
}

// apply runs the next pending operation in Go, and adds its steps.
function apply(g) {
  let [op, key] = g.pending.shift();
  if (key === "root") key = g.last?.tree?.keys[0];
  if (key === undefined) return;
  if (op === "insert" && (g.last?.stats.size ?? 0) >= MAX_KEYS) {
    g.pending = [];
    g.steps.push({ kind: "missing", note: `The clearing is full: this page keeps at most ${MAX_KEYS} keys in a tree.`, tree: g.last.tree, mark: [] });
    return;
  }
  const res = call("apply", { id: g.id, op, key, record: true });
  // At high speed, skip the comparisons, so that long flows stay short.
  const fast = Number($("#speed").value) >= 8;
  let steps = res.steps.filter((s) => !fast || s.kind !== "compare");
  if (!steps.length || (op === "insert" && !res.done)) {
    const note = op === "insert" && !res.done ? `${key} is already in the tree` : op === "search" && res.done ? `${key} is here` : `${key} is not in the tree`;
    steps = [{ kind: res.done ? "found" : "missing", note, tree: res.tree, mark: res.done ? [key] : [] }];
  }
  g.steps.push(...steps);
  if (g.steps.length > 600) {
    const cut = g.steps.length - 600;
    g.steps.splice(0, cut);
    g.pos -= cut;
  }
  g.last = res;
  g.log.unshift(`${op} ${key}: ${res.done ? (op === "search" ? "found" : "done") : op === "insert" ? "already there" : "not found"} · ${res.stats.comparisons} comparisons`);
  g.log.length = Math.min(g.log.length, 40);
}

function schedule(delay = interval()) {
  clearTimeout(timer);
  if (!playing) return;
  timer = setTimeout(tick, delay);
}

function tick() {
  const g = grove(kind);
  if (g.pos >= g.steps.length - 1) {
    if (!g.pending.length) return; // idle
    apply(g);
    showLog();
  }
  if (g.pos < g.steps.length - 1) {
    g.pos++;
    showStep(g.steps[g.pos]);
  }
  schedule();
}

function showStep(step, animate = true) {
  drawing.show(step.tree, step.mark, animate ? Math.min(500, interval() * 0.8) : 0);
  showNote(step);
  showStats();
  showArray(step.tree, step.mark);
}

function showNote(step) {
  const g = grove(kind);
  if (!step) {
    $("#note").replaceChildren("Insert a key, or run a flow.");
  } else {
    $("#note").replaceChildren(el("span", { class: `kind ${step.kind}` }, step.kind), step.note);
  }
  $("#counter").textContent = g.steps.length ? `step ${g.pos + 1} of ${g.steps.length}${g.pending.length ? ` · ${g.pending.length} operations waiting` : ""}` : "";
}

function showStats() {
  const g = grove(kind);
  const s = g.last?.stats ?? { size: 0, height: 0, restructures: 0, comparisons: 0 };
  const info = kinds.find((x) => x.key === kind);
  stats($("#stats"), [
    ["Keys", fmt(s.size)],
    ["Height", fmt(s.height)],
    ["Best height", kind === "btree" ? "–" : fmt(bestHeight(s.size))],
    [info.restructures === "none" ? "Restructures" : info.restructures[0].toUpperCase() + info.restructures.slice(1), fmt(s.restructures)],
    ["Comparisons, last operation", fmt(s.comparisons)],
  ]);
}

function showLog() {
  $("#log").replaceChildren(...grove(kind).log.map((line) => el("li", {}, line)));
}

// showArray draws the array of a heap: the snapshot in level order.
function showArray(tree, marks) {
  if (kind !== "heap") return;
  const out = [];
  const queueOf = tree ? [tree] : [];
  while (queueOf.length) {
    const n = queueOf.shift();
    if (!n) continue;
    out.push(n.keys[0]);
    queueOf.push(...(n.children ?? []));
  }
  const marked = new Set(marks);
  $("#heap-array").replaceChildren(...out.map((k, i) => el("span", { class: marked.has(k) ? "mark" : "" }, String(k), el("small", {}, `[${i}]`))));
}

function bindTreeControls() {
  $("#keyform").addEventListener("submit", (event) => {
    event.preventDefault();
    const key = Math.max(0, Math.min(999, Math.round(Number($("#key").value)) || 0));
    $("#key").value = key;
    queue([[event.submitter?.value ?? "insert", key]]);
  });
  $("#random-key").addEventListener("click", () => {
    $("#key").value = 1 + Math.floor(Math.random() * 99);
  });
  $("#clear").addEventListener("click", () => {
    const g = grove(kind);
    g.pending = [];
    g.last = call("apply", { id: g.id, op: "clear", key: 0, record: false });
    g.steps = [];
    g.pos = -1;
    g.log = [];
    openTree(kind);
  });
  $("#degree").addEventListener("change", () => {
    delete groves.btree;
    openTree("btree");
  });
  $("#play").addEventListener("click", () => {
    playing = !playing;
    $("#play").textContent = playing ? "⏸" : "▶";
    $("#play").setAttribute("aria-label", playing ? "Pause" : "Play");
    schedule(0);
  });
  const move = (d) => {
    playing = false;
    $("#play").textContent = "▶";
    clearTimeout(timer);
    const g = grove(kind);
    if (d > 0 && g.pos >= g.steps.length - 1 && g.pending.length) apply(g);
    const pos = Math.max(0, Math.min(g.steps.length - 1, g.pos + d));
    if (pos === g.pos) return;
    g.pos = pos;
    showStep(g.steps[pos], d > 0);
    showLog();
  };
  $("#prev").addEventListener("click", () => move(-1));
  $("#next").addEventListener("click", () => move(1));
}

/* Compare */

const compare = {
  ids: {},
  drawings: {},

  init() {
    $("#groves").replaceChildren(
      ...kinds.map((k) => {
        const canvas = el("div", { class: "canvas" });
        this.drawings[k.key] = new TreeDrawing(canvas, { compact: true, multi: k.key === "btree" });
        this.drawings[k.key].show(null);
        return el("article", { class: "mini", "data-kind": k.key }, el("h3", {}, k.name, el("span", { class: "height" }, "")), canvas);
      }),
    );
    for (const k of kinds) this.ids[k.key] = call("create", { kind: k.key, degree: 2 });
    $("#cmp-run").addEventListener("click", () => this.run());
  },

  async run() {
    const button = $("#cmp-run");
    button.disabled = true;
    $("#cmp-table").hidden = true;
    const n = Math.max(3, Math.min(MAX_KEYS, Math.round(Number($("#cmp-count").value)) || 31));
    $("#cmp-count").value = n;
    const keys = order($("#cmp-pattern").value, n);
    const last = {};
    for (const k of kinds) last[k.key] = call("apply", { id: this.ids[k.key], op: "clear", key: 0, record: false });
    for (const key of keys) {
      for (const k of kinds) {
        last[k.key] = call("apply", { id: this.ids[k.key], op: "insert", key, record: false });
        this.drawings[k.key].show(last[k.key].tree, [key], 90);
        $(`.mini[data-kind="${k.key}"] .height`).textContent = `height ${last[k.key].stats.height}`;
      }
      await new Promise((r) => setTimeout(r, 110));
    }
    const rows = kinds.map((k) => {
      let comparisons = 0;
      for (const key of keys) comparisons += call("apply", { id: this.ids[k.key], op: "search", key, record: false }).stats.comparisons;
      const s = last[k.key].stats;
      return { name: k.name, height: s.height, restructures: s.restructures, label: k.restructures, comparisons: comparisons / keys.length };
    });
    // Splay trees change on every search: draw them as they are now.
    const splay = call("apply", { id: this.ids.splay, op: "search", key: keys[0], record: false });
    this.drawings.splay.show(splay.tree, [keys[0]], 200);
    $('.mini[data-kind="splay"] .height').textContent = `height ${splay.stats.height} after the searches`;
    $("#cmp-table tbody").replaceChildren(
      ...rows.map((r) =>
        el("tr", {},
          el("td", {}, r.name),
          el("td", { class: "num" }, String(r.height)),
          el("td", { class: "num" }, String(bestHeight(n))),
          el("td", { class: "num" }, r.label === "none" ? "–" : `${r.restructures} ${r.label}`),
          el("td", { class: "num" }, fmt(r.comparisons, 1)))),
    );
    $("#cmp-table").hidden = false;
    button.disabled = false;
  },
};

/* Benchmark */

const bench = {
  init() {
    $("#bench-run").addEventListener("click", () => this.run());
  },

  async run() {
    const button = $("#bench-run");
    button.disabled = true;
    const keys = Number($("#bench-keys").value);
    const pattern = $("#bench-pattern").value;
    $("#bench-progress").hidden = false;
    const bar = $("#bench-progress div");
    const rows = [];
    for (const [i, k] of kinds.entries()) {
      button.textContent = `Running ${SHORT[k.key]}…`;
      bar.style.width = `${(i / kinds.length) * 100}%`;
      await new Promise((r) => setTimeout(r, 30)); // let the page draw
      rows.push({ ...call("bench", { kind: k.key, keys, pattern }), label: k.restructures });
    }
    bar.style.width = "100%";
    $("#bench-table tbody").replaceChildren(
      ...rows.map((r) =>
        el("tr", {},
          el("td", {}, r.name),
          el("td", { class: "num" }, fmt(r.keys)),
          el("td", { class: "num" }, ns(r.insertNs)),
          el("td", { class: "num" }, ns(r.searchNs)),
          el("td", { class: "num" }, ns(r.deleteNs)),
          el("td", { class: "num" }, fmt(r.height)),
          el("td", { class: "num" }, fmt(r.comparisons, 1)),
          el("td", { class: "num" }, r.label === "none" ? "–" : fmt(r.restructures)))),
    );
    const chart = (title, value, label) => {
      const top = Math.max(...rows.map(value), 1e-9);
      return el("div", { class: "chart" },
        el("h3", {}, title),
        ...rows.map((r) =>
          el("div", { class: "bar-row" },
            el("span", {}, SHORT[r.kind]),
            el("div", {}, el("div", { class: "bar", style: { width: `${Math.max(1, (value(r) / top) * 100)}%` } })),
            el("span", { class: "value" }, label(r)))));
    };
    $("#bench-charts").replaceChildren(
      chart("Insert, for one key", (r) => r.insertNs, (r) => ns(r.insertNs)),
      chart("Search, for one key", (r) => r.searchNs, (r) => ns(r.searchNs)),
      chart("Delete, for one key", (r) => r.deleteNs, (r) => ns(r.deleteNs)),
      chart("Height", (r) => r.height, (r) => fmt(r.height)),
    );
    $("#bench-table").hidden = false;
    $("#bench-progress").hidden = true;
    button.disabled = false;
    button.textContent = "Run the benchmark";
  },
};

/* Tabs */

function selectTab(name) {
  $$("#tabs button").forEach((b) => b.setAttribute("aria-selected", String(b.dataset.tab === name)));
  $("#tree-view").hidden = name === "compare" || name === "bench";
  $("#compare-view").hidden = name !== "compare";
  $("#bench-view").hidden = name !== "bench";
  clearTimeout(timer);
  if (!$("#tree-view").hidden) openTree(name);
  history.replaceState(null, "", `#${name}`);
}

async function start() {
  try {
    await loadGo();
  } catch (error) {
    $("#loading").textContent = `The forest could not grow: ${error.message}`;
    return;
  }
  kinds = call("kinds");
  $("#tabs").replaceChildren(
    ...kinds.map((k) => el("button", { role: "tab", "data-tab": k.key, onclick: () => selectTab(k.key) }, SHORT[k.key])),
    el("span", { class: "sep" }),
    el("button", { role: "tab", "data-tab": "compare", onclick: () => selectTab("compare") }, "Compare"),
    el("button", { role: "tab", "data-tab": "bench", onclick: () => selectTab("bench") }, "Benchmark"),
  );
  $("#loading").hidden = true;
  $("#main").hidden = false;
  bindTreeControls();
  compare.init();
  bench.init();
  const first = location.hash.slice(1);
  selectTab([...kinds.map((k) => k.key), "compare", "bench"].includes(first) ? first : "avl");
}

start();
