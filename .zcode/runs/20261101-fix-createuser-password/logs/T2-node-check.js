// T2 附加验证（AC-2）：加载内嵌 HTML，用 DOM stub 执行页面脚本，
// 断言生成的配置 JSON 可被 JSON.parse 且含 images handler、url 指向本机 /api/search。
const fs = require('fs');
const assert = require('assert');
const html = fs.readFileSync(process.argv[2], 'utf8');
const m = html.match(/<script>([\s\S]*?)<\/script>/);
assert.ok(m, '未找到页面 <script> 块');
const els = {};
function el(id) {
  if (!els[id]) els[id] = { textContent: '', classList: { add() {}, remove() {} }, addEventListener() {}, style: {} };
  return els[id];
}
global.document = {
  getElementById: el,
  createElement: () => ({ style: {}, select() {} }),
  body: { appendChild() {}, removeChild() {} },
};
global.navigator = {};
global.location = { origin: 'http://localhost:8060' };
eval(m[1]);
const jsonText = el('config-json').textContent;
const cfg = JSON.parse(jsonText); // JSON.parse 必须成功
assert.ok(Array.isArray(cfg) && cfg.length === 1, '配置应为单元素数组');
const c = cfg[0];
assert.strictEqual(c.url, 'http://localhost:8060/api/search', 'url 应为 location.origin + /api/search');
assert.ok(c.data.images.handler.includes('env.images'), '应含 images handler');
assert.ok(c.data.options.handler.length > 0, '应含 options handler');
assert.ok(c.data.type.handler.includes("env.type === 'single'"), '应含 type handler');
assert.strictEqual(c.data.suggestion_title, '${suggestion_title}', '应含 suggestion_title 占位');
assert.ok(c.handler.includes('allAnswer'), '应含答案解析 handler');
// 复制按钮与导入步骤存在于 HTML
assert.ok(html.includes('复制配置'), '应含「复制配置」按钮');
assert.ok(html.includes('读取剪贴板'), '应含导入步骤说明');
console.log('NODE-CHECK OK: JSON.parse 通过，url=' + c.url + '，images handler=' + c.data.images.handler);
