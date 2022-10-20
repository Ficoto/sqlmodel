package generator

import (
	"fmt"
	"os"
)

type options struct {
	dataSourceName string
	tableNames     []string
	tags           []string
	forceCases     []string
}

func printUsageAndExit(exampleDataSourceName string) {
	cmd := os.Args[0]
	_, _ = fmt.Fprintf(os.Stderr, `Usage:
	%s -o outpath -dbc databaseConnection [-t table1,table2,...] [-tag gorm,json,pg] [-forcecases ID,IDs,HTML]
Example:
	%s "%s"
`, cmd, cmd, fmt.Sprintf("-o ./ -d %s", exampleDataSourceName))
	os.Exit(1)
}
