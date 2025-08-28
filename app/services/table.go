package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/utils"
	"github.com/gofiber/fiber/v2"
)

// GetTableDetail 获取表的完整结构：字段 + 索引 + 触发器
func GetTableInfo(tableName string) (*models.TableInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}
	detail := &models.TableInfo{}
	// 1. 获取字段信息
	cols, err := GetTableColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table info: %w", err)
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table '%s' not found or has no columns", tableName)
	}
	detail.Columns = cols
	// 2. 获取索引信息
	indexes, err := GetTableIndexes(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	detail.Indexes = indexes
	// 3. 获取触发器信息
	triggers, err := GetTableTriggers(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}
	detail.Triggers = triggers
	return detail, nil
}

// 查询表字段
func GetTableColumns(tableName string) ([]models.ColumnInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}
	query := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	rows, err := utils.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ColumnInfo
	for rows.Next() {
		var cid int
		var name, colType, defaultValue sql.NullString
		var notNull, pk int
		err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk)
		if err != nil {
			return nil, err
		}
		// 转换 NULL 值
		defaultVal := ""
		if defaultValue.Valid {
			defaultVal = defaultValue.String
		}
		result = append(result, models.ColumnInfo{
			CID:     cid,
			Name:    name.String,
			Type:    colType.String,
			NotNull: notNull == 1,
			Default: defaultVal,
			Primary: pk == 1,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type NewTableColumnSchema struct {
	Name    string `json:"name" validate:"required"`
	Type    string `json:"type" validate:"required"`
	NotNull bool   `json:"notNull"`
	Default string `json:"default"`
	Primary bool   `json:"pk"`
}

// 新建表字段
func NewTableColumn(tableName string, column NewTableColumnSchema) error {
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	if !IsValidIdentifier(column.Name) {
		return fmt.Errorf("invalid column name: %s", column.Name)
	}
	sql := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN \"%s\" %s", tableName, column.Name, column.Type)
	if column.NotNull {
		sql += " NOT NULL"
	}
	if column.Default != "" {
		sql += fmt.Sprintf(" DEFAULT '%s'", column.Default)
	}
	if column.Primary {
		sql += " PRIMARY KEY"
	}
	_, err := utils.DB.Exec(sql)
	return err
}

// 删除表字段
func DeleteTableColumn(tableName, columnName string) error {
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	if !IsValidIdentifier(columnName) {
		return fmt.Errorf("invalid column name: %s", columnName)
	}
	sql := fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN \"%s\"", tableName, columnName)
	_, err := utils.DB.Exec(sql)
	return err
}

// 表字段重命名
func RenameTableColumn(tableName, oldName, newName string) error {
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	if !IsValidIdentifier(oldName) || !IsValidIdentifier(newName) {
		return fmt.Errorf("invalid column name")
	}
	sql := fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\"", tableName, oldName, newName)
	_, err := utils.DB.Exec(sql)
	return err
}

// GetTableIndexes 获取指定表的所有索引及其列信息
func GetTableIndexes(tableName string) ([]models.IndexInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	var indexes []models.IndexInfo

	// Step 1: 获取索引列表（index_list）
	indexListQuery := fmt.Sprintf("PRAGMA index_list('%s')", tableName)
	rows, err := utils.DB.Query(indexListQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var seq, unique, partial int
		var indexName, origin string
		if err := rows.Scan(&seq, &indexName, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		// ✅ 安全读取可能为 NULL 的 sql 字段
		var sqlNull sql.NullString
		err := utils.DB.QueryRow("SELECT sql FROM sqlite_master WHERE type='index' AND name=?", indexName).Scan(&sqlNull)
		var indexSQL string
		if err == nil && sqlNull.Valid {
			indexSQL = sqlNull.String
		} else {
			// 索引无 SQL（如主键或唯一约束自动生成的索引）
			indexSQL = ""
		}
		columns, err := getIndexColumns(indexName)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, models.IndexInfo{
			Name:    indexName,
			Unique:  unique == 1,
			SQL:     indexSQL,
			Columns: columns,
		})
	}
	return indexes, nil
}

type NewTableIndexSchema struct {
	Name    string   `json:"name" validate:"required"`
	Columns []string `json:"columns" validate:"required"`
	Unique  bool     `json:"unique"`
}

// 新建表索引
func NewTableIndex(tableName string, index NewTableIndexSchema) error {
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	if !IsValidIdentifier(index.Name) {
		return fmt.Errorf("invalid index name: %s", index.Name)
	}
	for _, col := range index.Columns {
		if !IsValidIdentifier(col) {
			return fmt.Errorf("invalid column name: %s", col)
		}
	}
	var unique string
	if index.Unique {
		unique = "UNIQUE"
	}
	sql := fmt.Sprintf("CREATE %s INDEX \"%s\" ON \"%s\" (%s)", unique, index.Name, tableName, strings.Join(index.Columns, ", "))
	_, err := utils.DB.Exec(sql)
	return err
}

// 删除表索引
func DeleteTableIndex(tableName, indexName string) error {
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("invalid table name: %s", tableName)
	}
	if !IsValidIdentifier(indexName) {
		return fmt.Errorf("invalid index name: %s", indexName)
	}
	sql := fmt.Sprintf("DROP INDEX IF EXISTS \"%s\"", indexName)
	_, err := utils.DB.Exec(sql)
	return err
}

// getIndexColumns 获取某个索引包含的列名
func getIndexColumns(indexName string) ([]string, error) {
	if !IsValidIdentifier(indexName) {
		return nil, fmt.Errorf("invalid index name: %s", indexName)
	}

	query := fmt.Sprintf("PRAGMA index_info('%s')", indexName)
	rows, err := utils.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqNo, cid int
		var columnName string

		// index_info 返回：seqno, cid, name
		err := rows.Scan(&seqNo, &cid, &columnName)
		if err != nil {
			return nil, err
		}

		columns = append(columns, columnName)
	}

	return columns, nil
}

// GetTableTriggers 获取指定表的所有触发器
func GetTableTriggers(tableName string) ([]models.TriggerInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	query := `
        SELECT name, sql 
        FROM sqlite_master 
        WHERE type = 'trigger' 
          AND tbl_name = ? 
        ORDER BY name
    `

	rows, err := utils.DB.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []models.TriggerInfo
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			return nil, err
		}

		triggers = append(triggers, models.TriggerInfo{
			Name: name,
			SQL:  sql,
		})
	}

	return triggers, nil
}

// InsertRow 向指定表插入一行数据
func InsertRow(tableName string, data map[string]any) (int64, error) {
	// 校验表名
	if !IsValidIdentifier(tableName) {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided for insertion")
	}
	var columns []string
	var placeholders []string
	var args []any
	for k, v := range data {
		if !IsValidIdentifier(k) {
			return 0, fmt.Errorf("invalid column name: %s", k)
		}
		columns = append(columns, fmt.Sprintf(`"%s"`, k))
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}
	sql := fmt.Sprintf(
		"INSERT INTO \"%s\" (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	result, err := utils.DB.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// IsValidIdentifier 检查标识符是否合法（简单实现）
func IsValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
		if i == 0 && (c >= '0' && c <= '9') {
			return false // 不能以数字开头
		}
	}
	return true
}

type QueryTableResult struct {
	Data  []map[string]any
	Total int
}

func GetTableData(tableName string, limit, offset int) (*QueryTableResult, error) {
	// Count total
	var total int
	err := utils.DB.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Fetch data
	rows, err := utils.DB.Query(
		fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", tableName, limit, offset),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := scanRows(rows)
	if err != nil {
		return nil, err
	}

	return &QueryTableResult{Data: data, Total: total}, nil
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, _ := rows.Columns()
	var result []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))
		valuePointers := make([]any, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}
		if err := rows.Scan(valuePointers...); err != nil {
			continue
		}
		entry := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				entry[col] = string(b)
			} else {
				entry[col] = val
			}
		}
		result = append(result, entry)
	}
	return result, nil
}

// ParseJSON 解析为 []map[string]any
func ParseJSON(r io.Reader) ([]map[string]any, error) {
	var data []map[string]any
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// ParseCSV 解析为 []map[string]any
func ParseCSV(r io.Reader) ([]map[string]any, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}
	headers := records[0]
	var result []map[string]any
	for i := 1; i < len(records); i++ {
		row := make(map[string]any)
		for j, h := range headers {
			if j < len(records[i]) {
				row[h] = records[i][j]
			} else {
				row[h] = nil
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func parseFile(r io.Reader, fileType string) ([]map[string]any, error) {
	switch strings.ToLower(fileType) {
	case ".json":
		return ParseJSON(r)
	case ".csv":
		return ParseCSV(r)
	default:
		return nil, &fiber.Error{Code: 400, Message: "不支持的文件类型"}
	}
}

type ImportResult struct {
	TableName    string
	SuccessCount int
	FailedCount  int
	Errors       []string
}

// 上传文件导入数据JSON/CSV, 支持创建新列
func ImportToTable(ctx context.Context, fileReader io.Reader, fileType, tableName string, createNewColumn bool) (*ImportResult, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("非法表名: %s", tableName)
	}
	records, err := parseFile(fileReader, fileType)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &ImportResult{TableName: tableName}, nil
	}
	result := &ImportResult{
		TableName:    tableName,
		SuccessCount: 0,
		FailedCount:  0,
		Errors:       []string{},
	}
	tx, err := utils.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// 提取所有列名（取第一条记录的 key）
	var columns []string
	for k := range records[0] {
		if IsValidIdentifier(k) {
			columns = append(columns, k)
		} else {
			return nil, fmt.Errorf("非法列名: %s", k)
		}
	}
	if createNewColumn {
		// 获取现有列
		existingCols, err := GetTableColumns(tableName)
		if err != nil {
			return nil, err
		}
		existingColMap := make(map[string]bool)
		for _, col := range existingCols {
			existingColMap[col.Name] = true
		}
		// 创建不存在的列，全部使用 TEXT 类型
		for _, col := range columns {
			if !existingColMap[col] {
				err := NewTableColumn(tableName, NewTableColumnSchema{
					Name: col,
					Type: "TEXT",
				})
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("创建列 %s 失败: %s", col, err.Error()))
				}
			}
		}
	}

	// 构建 INSERT SQL
	// INSERT INTO table (a,b,c) VALUES (?, ?, ?)
	placeholders := strings.Repeat("?,", len(columns)-1) + "?"
	sqlStmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(columns, ","), placeholders)
	for _, record := range records {
		values := make([]any, 0, len(columns))
		for _, col := range columns {
			values = append(values, record[col])
		}
		_, err := tx.ExecContext(ctx, sqlStmt, values...)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.SuccessCount++
		}
	}
	// 只有全部成功才提交
	if result.FailedCount == 0 {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// 导出表数据, 支持指定字段, 格式 JSON/CSV
func StreamExportTableData(ctx context.Context, tableName string, columns []string, fileType string, w io.Writer) error {
	// 构建查询：全量导出（或可加 where）
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ","), tableName)

	rows, err := utils.DB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	// 根据格式初始化编码器
	switch fileType {
	case "json":
		return streamJSON(rows, columns, w)
	case "csv":
		return streamCSV(rows, columns, w)
	default:
		return fmt.Errorf("不支持的格式: %s", fileType)
	}
}

func streamJSON(rows *sql.Rows, columns []string, w io.Writer) error {
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}

	var isFirst = true
	for rows.Next() {
		vals := make([]any, len(columns))
		valuePointers := make([]any, len(columns))
		for i := range vals {
			valuePointers[i] = &vals[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			return err
		}

		row := make(map[string]any)
		for i, col := range columns {
			switch v := vals[i].(type) {
			case nil:
				row[col] = nil
			case []byte:
				row[col] = string(v) // 假设是 UTF-8 文本
			default:
				row[col] = v
			}
		}

		if !isFirst {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		isFirst = false

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false) // 可选
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}

	return rows.Err()
}

func streamCSV(rows *sql.Rows, columns []string, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// 写表头
	if err := writer.Write(columns); err != nil {
		return err
	}

	// 写数据行
	for rows.Next() {
		vals := make([]any, len(columns))
		valuePointers := make([]any, len(columns))
		for i := range vals {
			valuePointers[i] = &vals[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			return err
		}

		record := make([]string, len(columns))
		for i, v := range vals {
			switch x := v.(type) {
			case nil:
				record[i] = ""
			case []byte:
				record[i] = string(x)
			default:
				record[i] = fmt.Sprintf("%v", x)
			}
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return rows.Err()
}
