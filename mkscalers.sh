#!bin/sh
# depends on https://github.com/jimmyfrasche/txt
cat fixedscalers.go.input | txt -json fixedscalers.go.template > fixedscalers.go
go fmt fixedscalers.go
