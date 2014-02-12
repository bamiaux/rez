#!bin/sh
# depends on https://github.com/jimmyfrasche/txt
cat fixedscalers.go.input | txt -json fixedscalers.go.template > fixedscalers.go
go fmt fixedscalers.go
go run gen/gen.go > scalers_amd64.s && echo scalers_amd64.s
