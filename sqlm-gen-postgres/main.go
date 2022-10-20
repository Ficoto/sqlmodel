package main

import (
	"github.com/Ficoto/sqlmodel/generator"
	_ "github.com/lib/pq"
)

func main() {
	err := generator.Generate("postgres", "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable")
	if err != nil {
		panic(err)
	}
}
