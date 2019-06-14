@echo off

echo "Building debug_build *tester* app"
REM set GOOS=windows
REM set GOARCH=386
REM go build -gcflags="-N -l" main.go
REM go build ./
REM go fmt ./
REM go build -i ./
go build -i -gcflags="-m -N -l" ./
tester.exe
echo "Finished...."
