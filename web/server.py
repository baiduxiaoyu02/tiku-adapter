#!/usr/bin/env python3
"""tiku-adapter 搜题代理服务器：解决 CORS 问题，同时提供搜题 UI。
用法：python server.py  →  http://localhost:8061
"""
import http.server, urllib.request, json, os, sys

PORT = 8061
TIKU_API = "http://localhost:8060/adapter-service/search"
WEB_DIR = os.path.dirname(os.path.abspath(__file__))

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/" or self.path == "/search.html":
            self._serve_file("search.html", "text/html; charset=utf-8")
        elif self.path.startswith("/api/search") or self.path == "/api/health":
            # GET 探活（OCS 题库状态检测可能用 GET）
            self._json({"ok": True, "service": "tiku-adapter-proxy"})
        else:
            self.send_error(404)

    def do_HEAD(self):
        # OCS 题库健康检查会发 HEAD 请求（带 ?t= 时间戳参数）
        if self.path.startswith("/api/search"):
            self.send_response(200)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", "0")
            self.end_headers()
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/api/search":
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length)
            try:
                req = urllib.request.Request(TIKU_API, data=body,
                    headers={"Content-Type": "application/json"})
                resp = urllib.request.urlopen(req, timeout=60)
                self.send_response(200)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.end_headers()
                self.wfile.write(resp.read())
            except urllib.error.HTTPError as e:
                self.send_response(e.code)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.end_headers()
                self.wfile.write(e.read())
            except Exception as e:
                self._json({"error": str(e)}, 502)
        else:
            self.send_error(404)

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def _serve_file(self, name, ctype):
        path = os.path.join(WEB_DIR, name)
        try:
            with open(path, "rb") as f:
                data = f.read()
            self.send_response(200)
            self.send_header("Content-Type", ctype)
            self.end_headers()
            self.wfile.write(data)
        except FileNotFoundError:
            self.send_error(404, f"{name} not found")

    def _json(self, obj, code=200):
        self.send_response(code)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.end_headers()
        self.wfile.write(json.dumps(obj, ensure_ascii=False).encode("utf-8"))

    def log_message(self, fmt, *args):
        pass  # quiet

if __name__ == "__main__":
    print(f"题库搜题服务启动: http://localhost:{PORT}")
    print(f"后端 API: {TIKU_API}")
    print(f"按 Ctrl+C 退出")
    try:
        http.server.HTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
    except KeyboardInterrupt:
        print("\n已退出")
