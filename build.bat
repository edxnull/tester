@echo off

echo "Building *tester* app"
REM set GOOS=windows
REM set GOARCH=386
go build main.go
echo "Finished...."

REM go build -gcflags=m main.go
REM go tool pprof --alloc_space main.exe mem.prof
REM go tool pprof --alloc_objects main.exe mem.prof
REM --functions
REM --lines

REM I should create a .BAT file that accepts arguments like *build.bat* --windows --unix
REM go test -run=xxx -bench=. -benchmem
REM go test -bench=. -benchmem

REM pprof -hhtp=":8081" [.exe] [.prof]
