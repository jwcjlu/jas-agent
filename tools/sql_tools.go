package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"jas-agent/core"
	"strings"
)

// SQLConnection SQL连接配置
type SQLConnection struct {
	DB *sql.DB
}

// ListTablesTool 列出所有表
type ListTablesTool struct {
	conn *SQLConnection
}

func NewListTablesTool(conn *SQLConnection) *ListTablesTool {
	return &ListTablesTool{conn: conn}
}

func (t *ListTablesTool) Name() string {
	return "list_tables"
}

func (t *ListTablesTool) Description() string {
	return "列出数据库中的所有表。不需要参数。返回表名列表。"
}

func (t *ListTablesTool) Input() any {
	return nil
}

func (t *ListTablesTool) Type() core.ToolType {
	return core.Normal
}

func (t *ListTablesTool) Handler(ctx context.Context, input string) (string, error) {
	rows, err := t.conn.DB.QueryContext(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE()
	`)
	if err != nil {
		return "", fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return "", fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if len(tables) == 0 {
		return "No tables found in database", nil
	}

	return fmt.Sprintf("Tables: %s", strings.Join(tables, ", ")), nil
}

// TablesSchema 表结构信息
type TablesSchema struct {
	conn *SQLConnection
}

func NewTablesSchema(conn *SQLConnection) *TablesSchema {
	return &TablesSchema{conn: conn}
}

func (t *TablesSchema) Name() string {
	return "tables_schema"
}

func (t *TablesSchema) Description() string {
	return "获取指定表的结构信息（列名、数据类型等）。输入：表名（多个表用逗号分隔）。返回：表的详细结构。"
}

func (t *TablesSchema) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tables": map[string]interface{}{
				"type":        "string",
				"description": "表名，多个表用逗号分隔",
			},
		},
		"required": []string{"tables"},
	}
}

func (t *TablesSchema) Type() core.ToolType {
	return core.Normal
}

func (t *TablesSchema) Handler(ctx context.Context, input string) (string, error) {
	// 解析输入
	tableNames := strings.Split(strings.TrimSpace(input), ",")
	for i := range tableNames {
		tableNames[i] = strings.TrimSpace(tableNames[i])
	}

	var result strings.Builder
	for _, tableName := range tableNames {
		if tableName == "" {
			continue
		}

		rows, err := t.conn.DB.QueryContext(ctx, `
			SELECT column_name, column_type, is_nullable, column_key, column_default, extra
			FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ?
			ORDER BY ordinal_position
		`, tableName)
		if err != nil {
			return "", fmt.Errorf("failed to query schema for table %s: %w", tableName, err)
		}

		result.WriteString(fmt.Sprintf("\nTable: %s\n", tableName))
		result.WriteString("Columns:\n")

		hasColumns := false
		for rows.Next() {
			var columnName, columnType, isNullable, columnKey string
			var columnDefault, extra sql.NullString

			if err := rows.Scan(&columnName, &columnType, &isNullable, &columnKey, &columnDefault, &extra); err != nil {
				rows.Close()
				return "", fmt.Errorf("failed to scan column info: %w", err)
			}

			hasColumns = true
			result.WriteString(fmt.Sprintf("  - %s (%s)", columnName, columnType))
			if columnKey == "PRI" {
				result.WriteString(" [PRIMARY KEY]")
			}
			if isNullable == "NO" {
				result.WriteString(" [NOT NULL]")
			}
			if columnDefault.Valid {
				result.WriteString(fmt.Sprintf(" [DEFAULT: %s]", columnDefault.String))
			}
			if extra.Valid && extra.String != "" {
				result.WriteString(fmt.Sprintf(" [%s]", extra.String))
			}
			result.WriteString("\n")
		}
		rows.Close()

		if !hasColumns {
			result.WriteString(fmt.Sprintf("  Table '%s' not found or has no columns\n", tableName))
		}
	}

	return result.String(), nil
}

// ExecuteSQL 执行SQL查询
type ExecuteSQL struct {
	conn *SQLConnection
}

func NewExecuteSQL(conn *SQLConnection) *ExecuteSQL {
	return &ExecuteSQL{conn: conn}
}

func (e *ExecuteSQL) Name() string {
	return "execute_sql"
}

func (e *ExecuteSQL) Description() string {
	return "执行SQL查询并返回结果。输入：SQL查询语句。返回：查询结果（JSON格式）。仅支持SELECT语句。"
}

func (e *ExecuteSQL) Input() any {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sql": map[string]interface{}{
				"type":        "string",
				"description": "要执行的SQL查询语句",
			},
		},
		"required": []string{"sql"},
	}
}

func (e *ExecuteSQL) Type() core.ToolType {
	return core.Normal
}

func (e *ExecuteSQL) Handler(ctx context.Context, input string) (string, error) {
	// 安全检查：只允许 SELECT 语句
	sqlQuery := strings.TrimSpace(input)
	if !strings.HasPrefix(strings.ToUpper(sqlQuery), "SELECT") {
		return "", fmt.Errorf("only SELECT queries are allowed for security reasons")
	}

	rows, err := e.conn.DB.QueryContext(ctx, sqlQuery)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	// 构建结果
	var results []map[string]interface{}
	for rows.Next() {
		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		// 构建行数据
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 转换 []byte 为 string
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	// 转换为 JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	if len(results) == 0 {
		return "Query executed successfully but returned no results", nil
	}

	return fmt.Sprintf("Query returned %d rows:\n%s", len(results), string(jsonData)), nil
}

// RegisterSQLTools 注册所有SQL工具
func RegisterSQLTools(conn *SQLConnection) {
	tm := GetToolManager()
	tm.RegisterTool(NewListTablesTool(conn))
	tm.RegisterTool(NewTablesSchema(conn))
	tm.RegisterTool(NewExecuteSQL(conn))
}
