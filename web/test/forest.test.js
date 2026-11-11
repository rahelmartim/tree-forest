// Tests the page in a headless Chrome, on the build in site/ (run
// "make web" first).

import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { resolve } from "node:path";
import { after, before, describe, test } from "node:test";
import { findChrome, launchChrome, serveApp } from "./browser.js";

const chromePath = findChrome();
const built = existsSync(resolve(import.meta.dirname, "..", "..", "site", "forest.wasm"));
const skip = !chromePath ? "Chrome is not installed" : !built ? "run make web first" : false;

describe("Tree Forest in the browser", { skip }, () => {
  let chrome;
  let app;
  let page;

  before(async () => {
    app = await serveApp();
    chrome = await launchChrome(chromePath);
    page = await chrome.newPage({ width: 1400, height: 1000 });
    await page.goto(app.url);
    await page.waitFor("!document.querySelector('#main').hidden", 30000);
  });

  after(async () => {
    page?.close();
    await chrome?.close();
    app?.server.close();
  });

  const text = (selector) => page.eval(`document.querySelector(${JSON.stringify(selector)})?.textContent ?? ""`);
  const stat = (name) =>
    page.eval(`[...document.querySelectorAll('#stats div')].find((d) => d.querySelector('dt').textContent === ${JSON.stringify(name)})?.querySelector('dd').textContent`);
  const tab = (name) => page.eval(`[...document.querySelectorAll('#tabs button')].find((b) => b.textContent === ${JSON.stringify(name)}).click(), true`);
  const fast = () => page.eval("document.querySelector('#speed').value = 10; true");
  // idle waits until the player shows the last step and nothing waits.
  // It also waits for the last slide of the drawing to end.
  const idle = async (timeout = 40000) => {
    await page.waitFor("(() => { const m = /^step (\\d+) of (\\d+)$/.exec(document.querySelector('#counter').textContent); return m && m[1] === m[2]; })()", timeout);
    await new Promise((r) => setTimeout(r, 700));
  };
  const op = async (name, key) => {
    await page.eval(`document.querySelector('#key').value = ${key}; true`);
    await page.eval(`[...document.querySelectorAll('#keyform button')].find((b) => b.textContent === ${JSON.stringify(name)}).click(), true`);
    await new Promise((r) => setTimeout(r, 50));
    await idle();
  };
  const flow = async (name) => {
    await page.clickText("#flows button", name);
    await new Promise((r) => setTimeout(r, 50));
    await idle(90000);
  };
  const svgKeys = () => page.eval("[...document.querySelectorAll('#canvas svg text:not(.info)')].map((t) => Number(t.textContent))");

  test("opens the forest on the AVL tree", async () => {
    assert.equal(await page.eval("document.querySelectorAll('#tabs button').length"), 9);
    assert.equal(await text("#tree-name"), "AVL tree");
    assert.match(await text("#canvas"), /empty clearing/);
  });

  test("an AVL tree rotates when it leans too far", async () => {
    await fast();
    for (const key of [1, 2, 3]) await op("Insert", key);
    assert.equal(await stat("Keys"), "3");
    assert.equal(await stat("Height"), "2");
    assert.equal(await stat("Rotations"), "1");
    assert.match(await text("#note"), /rotate left at 1: 2 goes up/);
    // 2 is the root now: the first key in the drawing.
    assert.equal((await svgKeys()).length, 3);
  });

  test("the player steps back and forward", async () => {
    await page.click("#prev");
    await page.click("#prev");
    assert.match(await text("#note"), /out of balance|grows as a new leaf|go right/);
    assert.match(await text("#counter"), /^step \d+ of \d+$/);
    await page.click("#next");
    await page.click("#next");
    assert.match(await text("#note"), /rotate left at 1/);
    await page.click("#play");
  });

  test("a search reports a missing key", async () => {
    await op("Search", 77);
    assert.match(await text("#note"), /77 is not in the tree/);
    await op("Delete", 2);
    assert.equal(await stat("Keys"), "2");
  });

  test("a red-black tree stays low with keys in order", async () => {
    await tab("Red-black");
    await fast();
    await flow("Grow 1 to 15 in order");
    assert.equal(await stat("Keys"), "15");
    assert.ok(Number(await stat("Height")) <= 6, `height ${await stat("Height")}`);
    assert.ok((await page.eval("document.querySelectorAll('#canvas .leaf.red').length")) > 0, "some nodes are red");
    assert.ok((await page.eval("document.querySelectorAll('#canvas .leaf.black').length")) > 0, "some nodes are black");
  });

  test("a plain BST of keys in order is a line", async () => {
    await tab("BST");
    await fast();
    await flow("Grow 1 to 15 in order");
    assert.equal(await stat("Height"), "15");
    assert.equal(await stat("Best height"), "4");
  });

  test("a B-tree splits full nodes, and its degree changes the tree", async () => {
    await tab("B-tree");
    await fast();
    await flow("Grow 1 to 15 in order");
    assert.equal(await stat("Keys"), "15");
    assert.ok(Number(await stat("Splits and merges")) >= 5);
    assert.ok((await page.eval("document.querySelectorAll('#canvas .plank').length")) > 3);
    await page.eval("document.querySelector('#degree').value = '3'; document.querySelector('#degree').dispatchEvent(new Event('change')); true");
    assert.match(await text("#canvas"), /empty clearing/);
    await flow("Grow 1 to 15 in order");
    assert.ok(Number(await stat("Height")) <= 3, `height ${await stat("Height")} with degree 3`);
  });

  test("heap sort empties the heap from the smallest key", async () => {
    await tab("Heap");
    await fast();
    await flow("Grow 15 random keys");
    assert.equal(await page.eval("document.querySelectorAll('#heap-array span').length"), 15);
    const root = await page.eval("Number(document.querySelector('#heap-array span').firstChild.textContent)");
    const all = await page.eval("[...document.querySelectorAll('#heap-array span')].map((s) => Number(s.firstChild.textContent))");
    assert.equal(root, Math.min(...all));
    await flow("Heap sort");
    assert.equal(await stat("Keys"), "0");
  });

  test("the splay tree moves a searched key to the root", async () => {
    await tab("Splay");
    await fast();
    await flow("Grow 15 random keys");
    const keys = await svgKeys();
    const key = keys[keys.length - 1];
    await op("Search", key);
    const root = await page.eval("(() => { const all = [...document.querySelectorAll('#canvas g.leaf')]; const top = all.sort((a, b) => a.transform.baseVal[0].matrix.f - b.transform.baseVal[0].matrix.f)[0]; return Number(top.querySelector('text').textContent); })()");
    assert.equal(root, key);
  });

  test("compare grows the same keys in every tree", async () => {
    await tab("Compare");
    await page.eval("document.querySelector('#cmp-count').value = 15; true");
    await page.click("#cmp-run");
    await page.waitFor("!document.querySelector('#cmp-table').hidden", 60000);
    const rows = await page.eval("[...document.querySelectorAll('#cmp-table tbody tr')].map((r) => [...r.children].map((c) => c.textContent))");
    assert.equal(rows.length, 7);
    const height = (name) => Number(rows.find((r) => r[0] === name)[1]);
    assert.equal(height("Binary search tree"), 15);
    assert.equal(height("AVL tree"), 4);
    assert.equal(height("Binary heap"), 4);
  });

  test("the benchmark times every tree", async () => {
    await tab("Benchmark");
    await page.eval("document.querySelector('#bench-keys').value = '1000'; true");
    await page.click("#bench-run");
    await page.waitFor("!document.querySelector('#bench-table').hidden", 60000);
    assert.equal(await page.eval("document.querySelectorAll('#bench-table tbody tr').length"), 7);
    assert.equal(await page.eval("document.querySelectorAll('#bench-charts .chart').length"), 4);
  });

  test("the address opens a tab", async () => {
    // A new query makes the browser load the page again; a new hash would not.
    await page.goto(`${app.url}?again=1#treap`);
    await page.waitFor("!document.querySelector('#main').hidden", 30000);
    assert.equal(await text("#tree-name"), "Treap");
  });

  test("the page had no JavaScript errors", () => {
    assert.deepEqual(page.errors, []);
  });
});
