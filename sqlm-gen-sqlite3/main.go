package main

import (
	"github.com/Ficoto/sqlmodel/generator"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := generator.Generate("mysql", "./testdb.sqlite3")
	if err != nil {
		panic(err)
	}
}
