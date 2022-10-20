package generator

import (
	"bytes"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

type schemaFetcher interface {
	GetDatabaseName() (dbName string, err error)
	GetTableNames() (tableNames []string, err error)
	GetFieldDescriptors(tableName string) ([]fieldDescriptor, error)
	QuoteIdentifier(identifier string) string
}

type fieldDescriptor struct {
	Name      string
	Type      string
	Size      int
	Unsigned  bool
	AllowNull bool
	Comment   string
}

func getSchemaFetcherFactory(driverName string) func(db *sql.DB) schemaFetcher {
	switch driverName {
	case "mysql":
		return newMySQLSchemaFetcher
	case "sqlite3":
		return newSQLite3SchemaFetcher
	case "postgres":
		return newPostgresSchemaFetcher
	default:
		_, _ = fmt.Fprintln(os.Stderr, "unsupported driver "+driverName)
		os.Exit(2)
		return nil
	}
}

var nonIdentifierRegexp = regexp.MustCompile(`\W`)

func ensureIdentifier(name string) string {
	result := nonIdentifierRegexp.ReplaceAllString(name, "_")
	if result == "" || (result[0] >= '0' && result[0] <= '9') {
		result = "_" + result
	}
	return result
}

func newBuffWithBaseHeader(dbName string) *bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString("// This file is generated by sqlmodel (https://github.com/Ficoto/sqlmodel)\n")
	buf.WriteString("// DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("package %s \n\n", ensureIdentifier(dbName)))
	return &buf
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func writeToFile(buffer *bytes.Buffer, outputFile string, force bool) error {
	exists, _ := pathExists(outputFile)
	if exists && !force {
		var override string
		fmt.Fprint(os.Stdout, fmt.Sprintf("file(%s) already exists，is overwritten(Y/N)? ", outputFile))
		fmt.Scanln(&override)

		if override != "Y" && override != "y" {
			fmt.Fprintln(os.Stdout, "skip"+outputFile)
			return nil
		}
	}

	f, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(buffer.String())

	return nil
}

func convertToExportedIdentifier(s string, forceCases []string) string {
	var words []string
	nextCharShouldBeUpperCase := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if nextCharShouldBeUpperCase {
				words = append(words, "")
				words[len(words)-1] += string(unicode.ToUpper(r))
				nextCharShouldBeUpperCase = false
			} else {
				words[len(words)-1] += string(r)
			}
		} else {
			nextCharShouldBeUpperCase = true
		}
	}
	result := ""
	for _, word := range words {
		for _, caseWord := range forceCases {
			if strings.EqualFold(word, caseWord) {
				word = caseWord
				break
			}
		}
		result += word
	}
	var firstRune rune
	for _, r := range result {
		firstRune = r
		break
	}
	if result == "" || !unicode.IsUpper(firstRune) {
		result = "E" + result
	}
	return result
}

func getType(fieldDescriptor fieldDescriptor) (goType string, err error) {
	switch strings.ToLower(fieldDescriptor.Type) {
	case "tinyint":
		goType = "int8"
	case "smallint":
		goType = "int16"
	case "int", "mediumint":
		goType = "int32"
	case "bigint", "integer":
		goType = "int64"
	case "float", "double", "decimal", "real":
		goType = "float64"
	case "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "enum", "json", "numeric", "character varying":
		goType = "string"
	case "datetime", "date", "time", "timestamp":
		goType = "time.Time"
	case "binary", "varbinary", "blob", "tinyblob", "mediumblob", "longblob":
		// TODO: use []byte ?
		goType = "string"
	case "geometry", "point", "linestring", "polygon", "multipoint", "multilinestring", "multipolygon", "geometrycollection":
		goType = "sqlingo.WellKnownBinary"
	case "bit":
		if fieldDescriptor.Size == 1 {
			goType = "bool"
		} else {
			goType = "string"
		}
	default:
		err = fmt.Errorf("unknown field type %s", fieldDescriptor.Type)
		return
	}
	if fieldDescriptor.Unsigned && strings.HasPrefix(goType, "int") {
		goType = "u" + goType
	}
	if fieldDescriptor.AllowNull {
		goType = "*" + goType
	}
	return
}

func getTag(fieldDescriptor fieldDescriptor, options options) string {
	var tag string
	switch options.tag {
	case "gorm":
		tag = fmt.Sprintf("`gorm:\"column:%s\"`", fieldDescriptor.Name)
	}
	return tag
}

func generateTable(schemaFetcher schemaFetcher, dbName, tableName string, options options) error {
	fieldDescriptors, err := schemaFetcher.GetFieldDescriptors(tableName)
	if err != nil {
		return err
	}

	className := convertToExportedIdentifier(tableName, options.forceCases)

	var (
		modeLinesBuf     bytes.Buffer
		isNeedImportTime bool
	)
	for _, fieldDescriptor := range fieldDescriptors {
		goName := convertToExportedIdentifier(fieldDescriptor.Name, options.forceCases)
		goType, err := getType(fieldDescriptor)
		if err != nil {
			return err
		}

		commentLine := ""
		if fieldDescriptor.Comment != "" {
			commentLine = "\t// " + strings.ReplaceAll(fieldDescriptor.Comment, "\n", " ") + "\n"
		}

		modeLinesBuf.WriteString(commentLine)
		modeLinesBuf.WriteString(fmt.Sprintf("\t%s %s %s\n", goName, goType, getTag(fieldDescriptor, options)))
		switch fieldDescriptor.Type {
		case "datetime", "date", "time", "timestamp":
			isNeedImportTime = true
		}
	}

	var buf = newBuffWithBaseHeader(dbName)
	if isNeedImportTime {
		buf.WriteString("import \"time\"\n\n")
	}
	buf.WriteString(fmt.Sprintf("type %s struct {\n", className))
	buf.WriteString(modeLinesBuf.String())
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func (m %s) TableName() string {\n", className))
	buf.WriteString(fmt.Sprintf("\treturn \"%s\"\n", tableName))
	buf.WriteString("}\n\n")
	err = writeToFile(buf, fmt.Sprintf("%s/%s.go", *outputPath, tableName), true)
	return err
}

var (
	outputPath         = flag.String("o", "", "file output path")
	databaseConnection = flag.String("dbc", "", "database connection")
	tables             = flag.String("t", "", "-t table1,table2,...")
	tag                = flag.String("tag", "", "-tag gorm")
	forcecases         = flag.String("forcecases", "", "-forcecases ID,IDs,HTML")
)

// Generate generates code for the given driverName.
func Generate(driverName string, exampleDataSourceName string) error {
	flag.Parse()
	if len(*outputPath) == 0 {
		printUsageAndExit(exampleDataSourceName)
	}
	if len(*databaseConnection) == 0 {
		printUsageAndExit(exampleDataSourceName)
	}
	var options options
	options.dataSourceName = *databaseConnection
	if len(*tables) != 0 {
		options.tableNames = strings.Split(*tables, ",")
	}
	if len(*tag) != 0 {
		options.tag = *tag
	}
	if len(*forcecases) != 0 {
		options.forceCases = strings.Split(*forcecases, ",")
	}

	db, err := sql.Open(driverName, options.dataSourceName)
	if err != nil {
		return err
	}

	schemaFetcherFactory := getSchemaFetcherFactory(driverName)
	schemaFetcher := schemaFetcherFactory(db)

	dbName, err := schemaFetcher.GetDatabaseName()
	if err != nil {
		return err
	}

	if dbName == "" {
		return errors.New("no database selected")
	}

	if len(options.tableNames) == 0 {
		options.tableNames, err = schemaFetcher.GetTableNames()
		if err != nil {
			return err
		}
	}

	for _, tableName := range options.tableNames {
		println("Generating", tableName)
		err = generateTable(schemaFetcher, dbName, tableName, options)
		if err != nil {
			return err
		}
	}

	return err
}
