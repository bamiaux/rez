#!bin/sh
# depends on https://github.com/jimmyfrasche/txt
cat scalers.go.input | txt -json scalers.go.template > scalers.go
go fmt scalers.go
