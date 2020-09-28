@echo off

rem start the UI server
start link-checker-example-ui-win.exe
rem start the link checker service
start link-checker-service-win serve -o http://localhost:8092 -o http://localhost:8090
rem open the UI in the default browser
explorer http://localhost:8092
