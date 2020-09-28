@echo off
explorer http://localhost:8092
cd public
python -m http.server 8092
