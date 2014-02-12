#!bin/sh
# depends on https://github.com/jimmyfrasche/txt
cat fixedscalers.go.input | txt -json fixedscalers.go.template > fixedscalers.go
go fmt fixedscalers.go
go run gen/hgen.go > hscalers_amd64.s && echo hscalers_amd64.s
