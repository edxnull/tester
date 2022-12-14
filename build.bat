@echo off

echo "Building *tester* app"
REM set GOOS=windows
REM set GOARCH=386
REM go build -gcflags="-N -l" main.go
REM go build ./
REM go fmt ./
REM go build -i -gcflags="-N -l" ./
REM go build -i -ldflags "-s -w -H windowsgui" ./

go build ./
tester.exe
echo "Finished...."

REM go build -i -ldflags "-H windowsgui"
REM go build -gcflags=m main.go
REM go tool pprof --alloc_space main.exe mem.prof
REM go tool pprof --alloc_objects main.exe mem.prof
REM --functions
REM --lines

REM go build -gcflags="-N -l" [.exe]
REM -N = no optimizations
REM -l = no inlining

REM I should create a .BAT file that accepts arguments like *build.bat* --windows --unix
REM go test -run=xxx -bench=. -benchmem
REM go test -bench=. -benchmem

REM pprof -http=":8081" [.exe] [.prof]
REM go test -bench=. -benchmem
REM go build -race
REM godoc -http=":8081"
REM http://127.0.0.1:8081/pkg/
REM godoc -http=6060
REM localhost:6060

REM BOUND CHECK ELIMINATION
REM go build -gcflags="-d=ssa/check_bce/debug=1" main.go

REM [!]
REM go build -ldflags="-s -w"
REM This can reduce golang binary size and *working set bytes*
