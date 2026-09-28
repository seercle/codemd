#!/usr/bin/env bash
python3 -c 'import http.server,socketserver
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self): self.send_error(404)
    def log_message(self, *a): pass
socketserver.TCPServer(("127.0.0.1", 8099), H).serve_forever()' >/dev/null 2>&1 &

for _ in {1..100}; do
	(exec 3<>/dev/tcp/127.0.0.1/8099) 2>/dev/null && break
	sleep 0.05
done
