@echo off
set BUILD_NUMBER=%DATE:~-4%%DATE:~3,2%%DATE:~0,2%%TIME:~0,2%%TIME:~6,2%
set BUILD_NUMBER=%BUILD_NUMBER: =0%

go build -ldflags "-X main.BuildNumber=%BUILD_NUMBER%" -o launcher.exe .
