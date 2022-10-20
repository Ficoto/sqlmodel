# sqlmodel

[![Go Report Card](https://goreportcard.com/badge/github.com/Ficoto/sqlmodel)](https://goreportcard.com/report/github.com/Ficoto/sqlmodel)
[![MIT license](http://img.shields.io/badge/license-MIT-9d1f14)](http://opensource.org/licenses/MIT)

sqlmodel is a generating model tool in Go

### Install and use
```
$ go install github.com/Ficoto/sqlmodel/sqlm-gen-mysql
$ mkdir -p generated/sqlm
$ sqlm-gen-mysql -dbc root:123456@/database_name -o generated/sqlm
```