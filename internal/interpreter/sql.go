package interpreter

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// SQLInterpreter executes SQL queries with a pure Go in-memory database
type SQLInterpreter struct {
	tables map[string]*sqlTable
}

type sqlTable struct {
	name    string
	columns []string
	rows    [][]interface{}
}

func (s *SQLInterpreter) Name() string {
	return "sql"
}

func (s *SQLInterpreter) Execute(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	s.tables = make(map[string]*sqlTable)

	// Remove single-line comments (-- style)
	code = regexp.MustCompile(`(?m)--.*$`).ReplaceAllString(code, "")
	// Remove multi-line comments (/* */ style)
	code = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(code, "")
	
	// Normalize whitespace
	code = strings.ReplaceAll(code, "\n", " ")
	code = regexp.MustCompile(`\s+`).ReplaceAllString(code, " ")

	// Split by semicolons
	statements := strings.Split(code, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := s.executeStatement(stmt, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "Error executing '%s': %v\n", stmt[:min(40, len(stmt))], err)
		}
	}

	return nil
}

func (s *SQLInterpreter) executeStatement(stmt string, stdout, stderr io.Writer) error {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return nil
	}

	upper := strings.ToUpper(stmt)

	// CREATE TABLE
	if strings.HasPrefix(upper, "CREATE TABLE") {
		return s.executeCreateTable(stmt, stdout)
	}

	// INSERT
	if strings.HasPrefix(upper, "INSERT") {
		return s.executeInsert(stmt, stdout)
	}

	// SELECT
	if strings.HasPrefix(upper, "SELECT") {
		return s.executeSelect(stmt, stdout)
	}

	// DROP TABLE
	if strings.HasPrefix(upper, "DROP TABLE") {
		return s.executeDropTable(stmt, stdout)
	}

	// UPDATE
	if strings.HasPrefix(upper, "UPDATE") {
		return s.executeUpdate(stmt, stdout)
	}

	// DELETE
	if strings.HasPrefix(upper, "DELETE") {
		return s.executeDelete(stmt, stdout)
	}

	return nil
}

func (s *SQLInterpreter) executeCreateTable(stmt string, stdout io.Writer) error {
	// CREATE TABLE users (id INTEGER, name TEXT)
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(\w+)\s*\(([^)]+)\)`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		fmt.Fprintf(stdout, "DEBUG: CREATE TABLE failed, stmt: %s\n", stmt[:min(50, len(stmt))])
		return fmt.Errorf("invalid CREATE TABLE syntax")
	}

	tableName := match[1]
	columnDefs := strings.Split(match[2], ",")

	var columns []string
	for _, col := range columnDefs {
		col = strings.TrimSpace(col)
		parts := strings.Fields(col)
		if len(parts) > 0 {
			columns = append(columns, strings.ToUpper(parts[0]))
		}
	}

	s.tables[tableName] = &sqlTable{
		name:    tableName,
		columns: columns,
		rows:    [][]interface{}{},
	}

	fmt.Fprintf(stdout, "Table '%s' created with columns: %v\n", tableName, columns)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *SQLInterpreter) executeInsert(stmt string, stdout io.Writer) error {
	// INSERT INTO users (id, name) VALUES (1, 'Alice')
	re := regexp.MustCompile(`INSERT\s+INTO\s+(\w+)\s*\(([^)]+)\)\s*VALUES\s*\(([^)]+)\)`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid INSERT syntax")
	}

	tableName := match[1]
	columns := strings.Split(match[2], ",")
	values := strings.Split(match[3], ",")

	table, ok := s.tables[tableName]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	row := make([]interface{}, len(table.columns))
	for i, col := range columns {
		colName := strings.TrimSpace(col)
		value := strings.TrimSpace(values[i])
		parsedValue := s.parseValue(value)

		// Find column index
		for j, tableCol := range table.columns {
			if strings.EqualFold(colName, tableCol) {
				row[j] = parsedValue
				break
			}
		}
	}

	table.rows = append(table.rows, row)
	fmt.Fprintf(stdout, "1 row inserted into '%s'\n", tableName)
	return nil
}

func (s *SQLInterpreter) executeSelect(stmt string, stdout io.Writer) error {
	// Check for JOIN
	if strings.Contains(strings.ToUpper(stmt), " JOIN ") {
		return s.executeJoinSelect(stmt, stdout)
	}

	// Simple SELECT (existing implementation)
	re := regexp.MustCompile(`(?i)SELECT\s+(.+?)\s+FROM\s+(\w+)(?:\s+WHERE\s+(.+?))?(?:\s+ORDER\s+BY\s+(\w+))?(?:\s+LIMIT\s+(\d+))?`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid SELECT syntax")
	}

	columns := strings.TrimSpace(match[1])
	tableName := match[2]
	whereClause := match[3]
	_ = match[4] // orderBy (not implemented yet)
	limit := match[5]

	table, ok := s.tables[tableName]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	// Determine which columns to select
	var selectCols []int
	var selectColNames []string
	if columns == "*" {
		for i := range table.columns {
			selectCols = append(selectCols, i)
			selectColNames = append(selectColNames, table.columns[i])
		}
	} else {
		for _, col := range strings.Split(columns, ",") {
			col = strings.TrimSpace(col)
			for i, tableCol := range table.columns {
				if strings.EqualFold(col, tableCol) {
					selectCols = append(selectCols, i)
					selectColNames = append(selectColNames, tableCol)
					break
				}
			}
		}
	}

	// Print header
	fmt.Fprintln(stdout, strings.Join(selectColNames, " | "))
	fmt.Fprintln(stdout, strings.Repeat("-+-", len(selectColNames)))

	// Filter and print rows
	limitNum := -1
	if limit != "" {
		limitNum, _ = strconv.Atoi(limit)
	}

	count := 0
	for _, row := range table.rows {
		// Apply WHERE clause
		if whereClause != "" && !s.evaluateWhere(whereClause, row, table.columns) {
			continue
		}

		// Print row
		var values []string
		for _, idx := range selectCols {
			values = append(values, fmt.Sprintf("%v", row[idx]))
		}
		fmt.Fprintln(stdout, strings.Join(values, " | "))

		count++
		if limitNum > 0 && count >= limitNum {
			break
		}
	}

	return nil
}

func (s *SQLInterpreter) executeJoinSelect(stmt string, stdout io.Writer) error {
	// SELECT users.name, posts.title FROM users JOIN posts ON users.id = posts.user_id
	re := regexp.MustCompile(`(?i)SELECT\s+(.+?)\s+FROM\s+(\w+)\s+(?:LEFT\s+|RIGHT\s+|INNER\s+)?JOIN\s+(\w+)\s+(?:ON\s+(.+?))?(?:\s+WHERE\s+(.+?))?$`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid JOIN syntax")
	}

	columns := strings.TrimSpace(match[1])
	table1Name := match[2]
	table2Name := match[3]
	onClause := match[4]
	whereClause := match[5]

	table1, ok := s.tables[table1Name]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", table1Name)
	}
	table2, ok := s.tables[table2Name]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", table2Name)
	}

	// Parse ON clause to get join columns
	var joinCol1, joinCol2 string
	if onClause != "" {
		parts := strings.Split(onClause, "=")
		if len(parts) == 2 {
			joinCol1 = strings.TrimSpace(parts[0])
			joinCol2 = strings.TrimSpace(parts[1])
		}
	}

	// Determine which columns to select
	var selectColSpecs [][2]string // [table, column]
	if columns == "*" {
		for _, col := range table1.columns {
			selectColSpecs = append(selectColSpecs, [2]string{table1Name, col})
		}
		for _, col := range table2.columns {
			selectColSpecs = append(selectColSpecs, [2]string{table2Name, col})
		}
	} else {
		for _, col := range strings.Split(columns, ",") {
			col = strings.TrimSpace(col)
			if strings.Contains(col, ".") {
				parts := strings.SplitN(col, ".", 2)
				selectColSpecs = append(selectColSpecs, [2]string{parts[0], parts[1]})
			}
		}
	}

	// Build header
	var header []string
	for _, spec := range selectColSpecs {
		header = append(header, spec[1])
	}
	fmt.Fprintln(stdout, strings.Join(header, " | "))
	fmt.Fprintln(stdout, strings.Repeat("-+-", len(header)))

	// Perform join
	for _, row1 := range table1.rows {
		for _, row2 := range table2.rows {
			// Check join condition
			if onClause != "" {
				val1 := s.getJoinColumnValue(row1, table1.columns, joinCol1)
				val2 := s.getJoinColumnValue(row2, table2.columns, joinCol2)
				if val1 != val2 {
					continue
				}
			}

			// Apply WHERE clause
			combinedRow := append(row1, row2...)
			combinedCols := append(table1.columns, table2.columns...)
			if whereClause != "" && !s.evaluateWhere(whereClause, combinedRow, combinedCols) {
				continue
			}

			// Build and print result row
			var values []string
			for _, spec := range selectColSpecs {
				var row []interface{}
				var cols []string
				if spec[0] == table1Name {
					row = row1
					cols = table1.columns
				} else {
					row = row2
					cols = table2.columns
				}
				for i, c := range cols {
					if strings.EqualFold(c, spec[1]) {
						values = append(values, fmt.Sprintf("%v", row[i]))
						break
					}
				}
			}
			if len(values) > 0 {
				fmt.Fprintln(stdout, strings.Join(values, " | "))
			}
		}
	}

	return nil
}

func (s *SQLInterpreter) getJoinColumnValue(row []interface{}, columns []string, colSpec string) interface{} {
	// Handle table.column format
	if strings.Contains(colSpec, ".") {
		parts := strings.SplitN(colSpec, ".", 2)
		colSpec = parts[1]
	}
	for i, c := range columns {
		if strings.EqualFold(c, colSpec) {
			return row[i]
		}
	}
	return nil
}

func (s *SQLInterpreter) executeDropTable(stmt string, stdout io.Writer) error {
	re := regexp.MustCompile(`DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(\w+)`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid DROP TABLE syntax")
	}

	tableName := match[1]
	if _, ok := s.tables[tableName]; ok {
		delete(s.tables, tableName)
		fmt.Fprintf(stdout, "Table '%s' dropped\n", tableName)
	} else {
		fmt.Fprintf(stdout, "Table '%s' does not exist\n", tableName)
	}
	return nil
}

func (s *SQLInterpreter) executeUpdate(stmt string, stdout io.Writer) error {
	// UPDATE users SET name = 'Bob' WHERE id = 1
	re := regexp.MustCompile(`UPDATE\s+(\w+)\s+SET\s+(.+?)(?:\s+WHERE\s+(.+?))?$`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid UPDATE syntax")
	}

	tableName := match[1]
	setClause := match[2]
	whereClause := match[3]

	table, ok := s.tables[tableName]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	// Parse SET clause
	setCols := make(map[string]interface{})
	for _, set := range strings.Split(setClause, ",") {
		parts := strings.SplitN(set, "=", 2)
		if len(parts) == 2 {
			col := strings.TrimSpace(parts[0])
			val := s.parseValue(strings.TrimSpace(parts[1]))
			setCols[col] = val
		}
	}

	// Update rows
	updated := 0
	for _, row := range table.rows {
		if whereClause == "" || s.evaluateWhere(whereClause, row, table.columns) {
			for i, col := range table.columns {
				if val, ok := setCols[col]; ok {
					row[i] = val
				}
			}
			updated++
		}
	}

	fmt.Fprintf(stdout, "%d row(s) updated\n", updated)
	return nil
}

func (s *SQLInterpreter) executeDelete(stmt string, stdout io.Writer) error {
	re := regexp.MustCompile(`DELETE\s+FROM\s+(\w+)(?:\s+WHERE\s+(.+?))?$`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return fmt.Errorf("invalid DELETE syntax")
	}

	tableName := match[1]
	whereClause := match[2]

	table, ok := s.tables[tableName]
	if !ok {
		return fmt.Errorf("table '%s' does not exist", tableName)
	}

	// Filter rows
	var newRows [][]interface{}
	deleted := 0
	for _, row := range table.rows {
		if whereClause == "" || !s.evaluateWhere(whereClause, row, table.columns) {
			newRows = append(newRows, row)
		} else {
			deleted++
		}
	}
	table.rows = newRows

	fmt.Fprintf(stdout, "%d row(s) deleted\n", deleted)
	return nil
}

func (s *SQLInterpreter) evaluateWhere(clause string, row []interface{}, columns []string) bool {
	// Simple WHERE clause evaluation
	clause = strings.TrimSpace(clause)

	// Handle AND (simplified - just take first condition)
	if idx := strings.Index(clause, " AND "); idx > 0 {
		clause = clause[:idx]
	}

	// Comparison operators
	operators := []string{">=", "<=", "<>", "!=", "=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(clause, op); idx > 0 {
			col := strings.TrimSpace(clause[:idx])
			val := strings.TrimSpace(clause[idx+len(op):])

			// Find column index
			colIdx := -1
			for i, c := range columns {
				if strings.EqualFold(col, c) {
					colIdx = i
					break
				}
			}
			if colIdx < 0 || colIdx >= len(row) {
				return false
			}

			rowVal := row[colIdx]
			expectedVal := s.parseValue(val)

			return s.compare(rowVal, op, expectedVal)
		}
	}

	return true
}

func (s *SQLInterpreter) compare(rowVal interface{}, op string, expectedVal interface{}) bool {
	switch a := rowVal.(type) {
	case int:
		if b, ok := expectedVal.(int); ok {
			switch op {
			case "=":
				return a == b
			case ">", "":
				return a > b
			case "<":
				return a < b
			case ">=":
				return a >= b
			case "<=":
				return a <= b
			case "<>", "!=":
				return a != b
			}
		}
	case string:
		if b, ok := expectedVal.(string); ok {
			switch op {
			case "=":
				return a == b
			case ">":
				return a > b
			case "<":
				return a < b
			case ">=":
				return a >= b
			case "<=":
				return a <= b
			case "<>", "!=":
				return a != b
			}
		}
	}
	return false
}

func (s *SQLInterpreter) parseValue(value string) interface{} {
	value = strings.TrimSpace(value)

	// String literal
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1]
	}

	// NULL
	if strings.EqualFold(value, "NULL") {
		return nil
	}

	// Integer
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	// Float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	return value
}
