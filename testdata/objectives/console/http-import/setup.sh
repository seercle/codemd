#!/usr/bin/env bash
python3 -m http.server 8137 --bind 127.0.0.1 >/dev/null 2>&1 &

for _ in {1..100}; do
	(exec 3<>/dev/tcp/127.0.0.1/8137) 2>/dev/null && break
	sleep 0.05
done
