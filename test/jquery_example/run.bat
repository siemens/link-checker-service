@echo off
explorer http://localhost:8092
cd public
python3 -m http.server 8092
