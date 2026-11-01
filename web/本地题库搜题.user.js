// ==UserScript==
// @name         本地题库搜题助手（GLM 5.2）
// @namespace    localhost-tiku
// @version      1.0
// @description  直接调用本地 tiku-adapter 题库搜题，自动填答案，不依赖 OCS 框架
// @author       tiku-adapter
// @match        *://*.chaoxing.com/*
// @match        *://*.zhihuishu.com/*
// @match        *://*.edu.cn/*
// @match        *://*.icve.com.cn/*
// @match        *://*.icourse163.org/*
// @grant        GM_xmlhttpRequest
// @grant        GM_addStyle
// @connect      localhost
// @connect      127.0.0.1
// @run-at       document-idle
// ==/UserScript==

(function () {
    'use strict';

    const TIKU_URL = 'http://localhost:8060/adapter-service/search';
    const TYPES = { single: 0, multiple: 1, judgement: 3, completion: 4 };

    // ========== 搜题 ==========
    function searchTiku(question, type, options) {
        return new Promise((resolve) => {
            GM_xmlhttpRequest({
                method: 'POST',
                url: TIKU_URL,
                headers: { 'Content-Type': 'application/json' },
                data: JSON.stringify({ question: question, options: options || [], type: type }),
                timeout: 30000,
                onload: (res) => {
                    try { resolve(JSON.parse(res.responseText)); }
                    catch (e) { resolve(null); }
                },
                onerror: () => resolve(null),
                ontimeout: () => resolve(null)
            });
        });
    }

    // ========== 提取题目（超星 + 智慧树通用）==========
    function extractQuestions() {
        const questions = [];
        // 超星学习通
        document.querySelectorAll('.TiMu').forEach((el, i) => {
            const titleEl = el.querySelector('.Zy_TItle .topicHtml, .mt5 .mark_name, .qtlabel');
            if (!titleEl) return;
            const title = titleEl.textContent.trim();
            if (!title) return;
            // 判断题型
            const hasCheckbox = el.querySelectorAll('input[type=checkbox]').length > 0;
            const hasRadio = el.querySelectorAll('input[type=radio]').length > 0;
            const type = hasCheckbox ? TYPES.multiple : (hasRadio ? TYPES.single : TYPES.completion);
            // 提取选项
            const optionEls = el.querySelectorAll('ul.Zy_ulTop li, .UxBasicInput, .answer-editor-row');
            const options = [];
            const inputs = [];
            optionEls.forEach((opt) => {
                const text = opt.textContent.replace(/^[A-Z]\s*/, '').trim();
                if (text) options.push(text);
                const input = opt.querySelector('input[type=radio], input[type=checkbox]');
                if (input) inputs.push(input);
            });
            questions.push({ el, title, type, options, inputs, index: i });
        });
        // 智慧树
        if (questions.length === 0) {
            document.querySelectorAll('.exam-box .single-item, .question-item').forEach((el, i) => {
                const titleEl = el.querySelector('.qt, .qt BK-tp, .question-title');
                if (!titleEl) return;
                const title = titleEl.textContent.trim();
                if (!title) return;
                const hasCheckbox = el.querySelectorAll('input[type=checkbox]').length > 0;
                const type = hasCheckbox ? TYPES.multiple : TYPES.single;
                const optionEls = el.querySelectorAll('.answer-editor-row, .option-item');
                const options = []; const inputs = [];
                optionEls.forEach((opt) => {
                    options.push(opt.textContent.trim());
                    const input = opt.querySelector('input');
                    if (input) inputs.push(input);
                });
                questions.push({ el, title, type, options, inputs, index: i });
            });
        }
        return questions;
    }

    // ========== 填答案 ==========
    function fillAnswer(q, answer) {
        if (!answer || !answer.answerKey || answer.answerKey.length === 0) {
            if (answer && answer.answerText) {
                // 填空题：填入文本
                const input = q.el.querySelector('input[type=text], textarea');
                if (input) { input.value = answer.answerText; input.dispatchEvent(new Event('input')); }
            }
            return false;
        }
        const keys = answer.answerKey; // ['A', 'B'] 或 ['A']
        const letterMap = { 'A': 0, 'B': 1, 'C': 2, 'D': 3, 'E': 4, 'F': 5, '对': 0, '错': 1, '正确': 0, '错误': 1 };
        let filled = 0;
        keys.forEach((key) => {
            const idx = letterMap[key.toUpperCase()] ?? letterMap[key];
            if (idx !== undefined && q.inputs[idx]) {
                q.inputs[idx].click();
                filled++;
            }
        });
        return filled > 0;
    }

    // ========== UI 面板 ==========
    GM_addStyle(`
        #tiku-panel{position:fixed;top:60px;right:20px;width:320px;background:#fff;border:2px solid #4f46e5;border-radius:10px;box-shadow:0 4px 20px rgba(0,0,0,0.15);z-index:999999;font-family:sans-serif;font-size:13px}
        #tiku-panel .header{background:#4f46e5;color:#fff;padding:10px 14px;border-radius:8px 8px 0 0;font-weight:600;display:flex;justify-content:space-between;align-items:center}
        #tiku-panel .body{padding:10px 14px;max-height:400px;overflow-y:auto}
        #tiku-panel .btn{background:#4f46e5;color:#fff;border:none;padding:8px 16px;border-radius:6px;cursor:pointer;font-size:13px;width:100%;margin:8px 0;font-weight:600}
        #tiku-panel .btn:hover{background:#4338ca}
        #tiku-panel .btn:disabled{background:#999}
        #tiku-panel .status{color:#666;margin:4px 0}
        #tiku-panel .item{padding:6px 0;border-bottom:1px solid #eee}
        #tiku-panel .item .q{color:#333;font-weight:500}
        #tiku-panel .item .a{color:#4f46e5;font-weight:600}
        #tiku-panel .item .err{color:#ef4444}
        #tiku-panel .minimize{cursor:pointer;font-size:16px}
        #tiku-panel.minimized .body{display:none}
    `);

    function createPanel() {
        const panel = document.createElement('div');
        panel.id = 'tiku-panel';
        panel.innerHTML = `
            <div class="header">
                <span>📚 本地题库搜题</span>
                <span class="minimize" id="tiku-min">−</span>
            </div>
            <div class="body">
                <div class="status" id="tiku-status">点击下方按钮开始搜题</div>
                <button class="btn" id="tiku-search">🔍 搜索本页所有题目</button>
                <div id="tiku-results"></div>
            </div>`;
        document.body.appendChild(panel);
        document.getElementById('tiku-min').onclick = () => panel.classList.toggle('minimized');
        document.getElementById('tiku-search').onclick = runSearch;
    }

    // ========== 主流程 ==========
    async function runSearch() {
        const btn = document.getElementById('tiku-search');
        const status = document.getElementById('tiku-status');
        const results = document.getElementById('tiku-results');
        btn.disabled = true;
        btn.textContent = '搜索中...';
        results.innerHTML = '';

        // 尝试进入 iframe
        let targetDoc = document;
        const iframes = document.querySelectorAll('iframe');
        for (const f of iframes) {
            try {
                if (f.contentDocument && f.contentDocument.querySelector('.TiMu, .exam-box')) {
                    targetDoc = f.contentDocument;
                    break;
                }
            } catch (e) { /* 跨域 iframe，跳过 */ }
        }

        // 用 targetDoc 查找题目
        const oldExtract = document.querySelectorAll;
        const questions = (function () {
            const qs = targetDoc.querySelectorAll('.TiMu');
            // fallback to all iframes recursively
            if (qs.length === 0) {
                // 尝试所有 iframe
                for (const f of document.querySelectorAll('iframe')) {
                    try {
                        if (f.contentDocument) {
                            const iq = f.contentDocument.querySelectorAll('.TiMu, .exam-box .single-item');
                            if (iq.length > 0) {
                                // 提取逻辑复用
                                return extractFromDoc(f.contentDocument);
                            }
                        }
                    } catch (e) {}
                }
            }
            return extractFromDoc(targetDoc);
        })();

        if (questions.length === 0) {
            status.textContent = '⚠️ 未找到题目。如果在 iframe 里，请手动进入作业页面。';
            status.className = 'status';
            btn.disabled = false;
            btn.textContent = '🔍 搜索本页所有题目';
            return;
        }

        status.textContent = `找到 ${questions.length} 道题，开始搜索...`;
        let hit = 0, miss = 0;

        for (let i = 0; i < questions.length; i++) {
            const q = questions[i];
            const shortTitle = q.title.length > 30 ? q.title.slice(0, 30) + '...' : q.title;
            const item = document.createElement('div');
            item.className = 'item';
            item.innerHTML = `<div class="q">${i + 1}. ${shortTitle}</div><div class="status">搜索中...</div>`;
            results.appendChild(item);

            const resp = await searchTiku(q.title, q.type, q.options);
            const a = resp && resp.answer ? resp.answer : null;

            if (a && (a.answerKey.length > 0 || a.answerText)) {
                const filled = fillAnswer(q, a);
                const ansText = a.answerKey.length > 0 ? a.answerKey.join('') : a.answerText;
                item.innerHTML = `<div class="q">${i + 1}. ${shortTitle}</div><div class="a">✅ 答案: ${ansText}${filled ? ' (已填入)' : ''}</div>`;
                hit++;
            } else {
                item.innerHTML = `<div class="q">${i + 1}. ${shortTitle}</div><div class="err">❌ 未找到答案</div>`;
                miss++;
            }
            status.textContent = `进度: ${i + 1}/${questions.length} | 命中: ${hit} | 未找到: ${miss}`;
        }

        status.textContent = `完成！命中 ${hit}/${questions.length}，未找到 ${miss}`;
        btn.disabled = false;
        btn.textContent = '🔍 重新搜索';
    }

    function extractFromDoc(doc) {
        const questions = [];
        doc.querySelectorAll('.TiMu').forEach((el, i) => {
            const titleEl = el.querySelector('.Zy_TItle .topicHtml, .mt5 .mark_name, .qtlabel');
            if (!titleEl) return;
            const title = titleEl.textContent.trim();
            if (!title) return;
            const hasCheckbox = el.querySelectorAll('input[type=checkbox]').length > 0;
            const hasRadio = el.querySelectorAll('input[type=radio]').length > 0;
            const type = hasCheckbox ? TYPES.multiple : (hasRadio ? TYPES.single : TYPES.completion);
            const optionEls = el.querySelectorAll('ul.Zy_ulTop li, .UxBasicInput');
            const options = []; const inputs = [];
            optionEls.forEach((opt) => {
                const text = opt.textContent.replace(/^[A-Z]\s*/, '').trim();
                if (text) options.push(text);
                const input = opt.querySelector('input[type=radio], input[type=checkbox]');
                if (input) inputs.push(input);
            });
            questions.push({ el, title, type, options, inputs, index: i });
        });
        // 智慧树 fallback
        if (questions.length === 0) {
            doc.querySelectorAll('.single-item, .question-item').forEach((el, i) => {
                const titleEl = el.querySelector('.qt, .question-title');
                if (!titleEl) return;
                const title = titleEl.textContent.trim();
                if (!title) return;
                const hasCheckbox = el.querySelectorAll('input[type=checkbox]').length > 0;
                const type = hasCheckbox ? TYPES.multiple : TYPES.single;
                const optionEls = el.querySelectorAll('.answer-editor-row, .option-item');
                const options = []; const inputs = [];
                optionEls.forEach((opt) => {
                    options.push(opt.textContent.trim());
                    const input = opt.querySelector('input');
                    if (input) inputs.push(input);
                });
                questions.push({ el, title, type, options, inputs, index: i });
            });
        }
        return questions;
    }

    // 等页面加载完
    setTimeout(createPanel, 2000);
})();
