package services

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/utils"
)

type DBInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	SQLiteVer  string    `json:"sqlite_version"`
}

func GetDBInfo() (*DBInfo, error) {
	path := utils.DBPath // 假设你在 db/connection.go 中保存了原始路径
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var sqliteVer string
	_ = utils.DB.QueryRow("SELECT sqlite_version()").Scan(&sqliteVer)
	return &DBInfo{
		Path:       path,
		Size:       fi.Size(),
		CreatedAt:  fi.ModTime(), // SQLite 不记录创建时间
		ModifiedAt: fi.ModTime(),
		SQLiteVer:  sqliteVer,
	}, nil
}

func GetTables() ([]string, error) {
	rows, err := utils.DB.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables = make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}

	return tables, nil
}

type View struct {
	Name string `json:"name"`
	SQL  string `json:"sql"`
}

func GetViews() (*[]View, error) {
	rows, err := utils.DB.Query(`
		SELECT name, sql FROM sqlite_master WHERE type='view'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := make([]View, 0)
	for rows.Next() {
		var v View
		if err := rows.Scan(&v.Name, &v.SQL); err != nil {
			continue
		}
		views = append(views, v)
	}
	return &views, nil
}

// GetAllTriggers 查询所有触发器
func GetAllTriggers() ([]*models.Trigger, error) {
	query := `
        SELECT 
            name,
            tbl_name AS table_name,
            sql
        FROM sqlite_master 
        WHERE type = 'trigger'
          AND name NOT LIKE 'sqlite_%'
        ORDER BY name
    `
	rows, err := utils.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	triggers := make([]*models.Trigger, 0)
	for rows.Next() {
		var name, table, sql string
		if err := rows.Scan(&name, &table, &sql); err != nil {
			return nil, err
		}
		trigger := &models.Trigger{
			Name:       name,
			Table:      table,
			Definition: formatTriggerDefinition(sql),
			SQL:        sql,
		}
		// 解析类型和时机（INSERT/UPDATE/DELETE, BEFORE/AFTER）
		trigger.Type, trigger.Timing = parseTriggerTypeTiming(sql)
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

// GetTriggerByName 查询指定触发器
func GetTriggerByName(name string) (*models.Trigger, error) {
	var table, sqlStr string
	query := `SELECT tbl_name, sql FROM sqlite_master WHERE type = 'trigger' AND name = ?`
	err := utils.DB.QueryRow(query, name).Scan(&table, &sqlStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trigger not found: %s", name)
		}
		return nil, err
	}
	trigger := &models.Trigger{
		Name:       name,
		Table:      table,
		SQL:        sqlStr,
		Definition: formatTriggerDefinition(sqlStr),
	}
	trigger.Type, trigger.Timing = parseTriggerTypeTiming(sqlStr)
	return trigger, nil
}

// parseTriggerTypeTiming 从 SQL 中解析触发器类型和时机
// 简化处理，支持常见的 BEFORE/AFTER/INSTEAD OF + INSERT/UPDATE/DELETE
func parseTriggerTypeTiming(sql string) (action, timing string) {
	sql = strings.ToUpper(sql)
	if strings.Contains(sql, " INSTEAD OF ") {
		timing = "INSTEAD OF"
	} else if strings.Contains(sql, " BEFORE ") {
		timing = "BEFORE"
	} else {
		timing = "AFTER" // 默认
	}
	switch {
	case strings.Contains(sql, " INSERT "):
		action = "INSERT"
	case strings.Contains(sql, " UPDATE "):
		action = "UPDATE"
	case strings.Contains(sql, " DELETE "):
		action = "DELETE"
	default:
		action = "UNKNOWN"
	}
	return action, timing
}

// formatTriggerDefinition 简化 SQL 显示（用于前端展示）
func formatTriggerDefinition(sql string) string {
	// 去掉多余的空格，保留前 100 字符
	runes := []rune(sql)
	if len(runes) > 100 {
		return string(runes[:100]) + "..."
	}
	return sql
}

// QueryResult 查询结果
type QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	HasNext bool             `json:"has_next"`
}

// ExecuteQuery 执行带分页的 SQL 查询
// 支持 SELECT，自动处理 COUNT 和 LIMIT/OFFSET
func ExecuteQuery(sqlStr string, page, size int) (*QueryResult, error) {
	result := &QueryResult{
		Page: page,
	}
	// 获取列名（通过 EXPLAIN QUERY PLAN 或干跑查询）
	cols, err := getColumnsFromQuery(sqlStr)
	if err != nil {
		return nil, err
	}
	result.Columns = cols
	// 构造分页查询
	offset := (page - 1) * size
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", sqlStr, size+1, offset)
	rows, err := utils.DB.Query(paginatedSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// 检查是否有多余的一行（判断是否有下一页）
	rowCount := 0
	for rows.Next() && rowCount <= size {
		values, err := scanRow(rows, cols)
		if err != nil {
			continue
		}
		result.Rows = append(result.Rows, values)
		rowCount++
	}
	result.HasNext = rowCount > size
	if result.HasNext {
		result.Rows = result.Rows[:len(result.Rows)-1] // 去掉多余的
	}
	// 获取总数（仅对简单 SELECT 有效）
	result.Total = int64(len(result.Rows)) // 简化：先不执行 COUNT(*)
	// TODO: 可优化为：SELECT COUNT(*) FROM (original query) —— 但需判断是否聚合查询
	return result, nil
}

// getColumnsFromQuery 获取查询的列名
// 方法：执行一次干跑（带 LIMIT 0）
func getColumnsFromQuery(sqlStr string) ([]string, error) {
	rows, err := utils.DB.Query(fmt.Sprintf("SELECT * FROM (%s) AS t LIMIT 0", sqlStr))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows.Columns()
}

// scanRow 将一行数据扫描为 map[string]any
func scanRow(rows *sql.Rows, columns []string) (map[string]any, error) {
	values := make([]any, len(columns))
	valuePointers := make([]any, len(columns))
	for i := range values {
		valuePointers[i] = &values[i]
	}
	err := rows.Scan(valuePointers...)
	if err != nil {
		return nil, err
	}
	row := make(map[string]any)
	for i, col := range columns {
		val := values[i]
		// 处理 []byte -> string (如 BLOB)
		if b, ok := val.([]byte); ok {
			// 尝试转为字符串，避免显示为 base64
			row[col] = string(b)
		} else {
			row[col] = val
		}
	}
	return row, nil
}

// CreateSQLiteTable 根据请求创建表
func CreateSQLiteTable(req *models.CreateTableRequest) error {
	// 检查表名合法性（简单校验）
	if !IsValidIdentifier(req.TableName) {
		return fmt.Errorf("invalid table name: %s", req.TableName)
	}

	// Step 1: 检查表是否已存在
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name=? AND tbl_name=?)`
	err := utils.DB.QueryRow(query, req.TableName, req.TableName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}
	if exists {
		return fmt.Errorf("table %s already exists", req.TableName)
	}

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT
	)`, req.TableName)

	_, err = utils.DB.Exec(sql)
	return err
}

func DropSQLiteTable(tableName string) error {
	// 检查表名合法性（简单校验）
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	sql := fmt.Sprintf(`DROP TABLE "%s"`, tableName)

	_, err := utils.DB.Exec(sql)
	return err
}

// 导出查询数据
func ExportQuery(sql string, page, size int, fileType string, w io.Writer) error {
	// 获取列名（通过 EXPLAIN QUERY PLAN 或干跑查询）
	cols, err := getColumnsFromQuery(sql)
	if err != nil {
		return err
	}
	// 构造分页查询
	offset := (page - 1) * size
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", sql, size+1, offset)
	rows, err := utils.DB.Query(paginatedSQL)

	if err != nil {
		return err
	}
	switch fileType {
	case "json":
		return ExportToJSON(cols, rows, w)
	case "csv":
		return ExportToCSV(cols, rows, w)
	default:
		return fmt.Errorf("unsupported export format: %s", fileType)
	}
}

// 数据导出为 JSON
func ExportToJSON(columns []string, rows *sql.Rows, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}

	first := true
	values := make([]any, len(columns))
	valuePointers := make([]any, len(columns))
	for i := range values {
		valuePointers[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePointers...); err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		// 转 map[string]any
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			v := values[i]
			if b, ok := v.([]byte); ok {
				row[col] = string(b) // 转 string 避免 []uint8
			} else {
				row[col] = v
			}
		}

		if !first {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		first = false

		data, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("序列化行失败: %w", err)
		}
		// 去掉末尾 \n
		if len(data) > 0 && data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		_, err = w.Write(data)
		if err != nil {
			return fmt.Errorf("写入行失败: %w", err)
		}
	}

	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历结果时出错: %w", err)
	}

	return nil
}

// 数据导出为 CSV
func ExportToCSV(columns []string, rows *sql.Rows, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// 写入 UTF-8 BOM
	if _, err := w.Write([]byte("\xEF\xBB\xBF")); err != nil {
		return fmt.Errorf("写入BOM失败: %w", err)
	}

	// 写入表头
	if err := writer.Write(columns); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	// 预分配 record 缓冲区
	record := make([]string, len(columns))

	// 值指针（复用）
	values := make([]any, len(columns))
	valuePointers := make([]any, len(columns))
	for i := range valuePointers {
		valuePointers[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePointers...); err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		// 转换每一列
		for i, v := range values {
			switch val := v.(type) {
			case nil:
				record[i] = ""
			case []byte:
				record[i] = string(val)
			case string:
				record[i] = val
			case int64:
				record[i] = strconv.FormatInt(val, 10)
			case int:
				record[i] = strconv.FormatInt(int64(val), 10)
			case float64:
				record[i] = strconv.FormatFloat(val, 'g', -1, 64)
			case bool:
				record[i] = strconv.FormatBool(val)
			case time.Time:
				record[i] = val.Format("2006-01-02 15:04:05")
			default:
				record[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入数据行失败: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历结果时出错: %w", err)
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV刷新失败: %w", err)
	}

	return nil
}
