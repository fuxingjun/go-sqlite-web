package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
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

type columnPragma struct {
	CID          int            `db:"cid"`
	Name         string         `db:"name"`
	Type         string         `db:"type"`
	NotNull      int            `db:"notnull"`
	DefaultValue sql.NullString `db:"dflt_value"`
	PK           int            `db:"pk"`
	Hidden       int            `db:"hidden"`
}

// GetTableColumns 获取表的列信息，包括 Unique 和 AutoIncrement
func GetTableColumns(tableName string) ([]models.ColumnInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Step 1: 获取表的 CREATE TABLE 语句
	var ddl string
	err := utils.DB.Get(&ddl, "SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get table DDL: %w", err)
	}

	// Step 2: 使用 PRAGMA table_xinfo 获取列基本信息
	query := fmt.Sprintf("PRAGMA table_xinfo('%s')", tableName)
	rows, err := utils.DB.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table_xinfo: %w", err)
	}
	defer rows.Close()

	// Step 3: 获取 UNIQUE 列信息
	uniqueColumns, err := getUniqueColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique columns: %w", err)
	}

	var result []models.ColumnInfo
	for rows.Next() {
		var col columnPragma
		if err := rows.StructScan(&col); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		defaultVal := ""
		if col.DefaultValue.Valid {
			defaultVal = col.DefaultValue.String
		}

		// 判断是否为 AUTOINCREMENT
		isAutoIncrement := isColumnAutoIncrement(col.Name, ddl)

		// 判断是否为 UNIQUE（包括主键）
		isUnique := col.PK == 1 || uniqueColumns[col.Name]

		result = append(result, models.ColumnInfo{
			CID:           col.CID,
			Name:          col.Name,
			Type:          col.Type,
			Unique:        isUnique,
			NotNull:       col.NotNull == 1,
			Default:       defaultVal,
			Primary:       col.PK == 1,
			AutoIncrement: isAutoIncrement,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return result, nil
}

// getUniqueColumns 返回表中所有被 UNIQUE 约束覆盖的列（单列 UNIQUE）
func getUniqueColumns(tableName string) (map[string]bool, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	uniqueCols := make(map[string]bool)

	// 查询所有索引
	tableIndexes, err := GetTableIndexes(tableName)
	if err != nil {
		return nil, err
	}

	for _, index := range tableIndexes {
		if index.Unique {
			// 获取索引的列信息
			cols, err := getIndexColumns(index.Name)
			if err != nil {
				return nil, err
			}
			if len(cols) == 1 {
				uniqueCols[cols[0]] = true
			}
		}
	}

	return uniqueCols, nil
}

// 检查列是否为 AUTOINCREMENT
func isColumnAutoIncrement(columnName, createTableSQL string) bool {
	// 转换为大写进行比较
	sqlUpper := strings.ToUpper(createTableSQL)
	columnUpper := strings.ToUpper(columnName)

	// 查找列定义
	columnPattern := fmt.Sprintf(`\b%s\b[^,]*`, regexp.QuoteMeta(columnUpper))
	re := regexp.MustCompile(columnPattern)

	match := re.FindString(sqlUpper)
	if match == "" {
		return false
	}

	// 检查是否包含 AUTOINCREMENT 关键字
	return strings.Contains(match, "AUTOINCREMENT") || strings.Contains(match, "AUTO_INCREMENT")
}

type NewTableColumnSchema struct {
	Name          string `json:"name" validate:"required"`
	Type          string `json:"type" validate:"required"`
	NotNull       bool   `json:"notNull"`
	Default       string `json:"default,omitempty"`
	Primary       bool   `json:"pk,omitempty"`
	AutoIncrement bool   `json:"autoIncrement,omitempty"` // 允许不传, 默认 false
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
		if column.AutoIncrement && strings.ToUpper(column.Type) == "INTEGER" {
			sql += " AUTOINCREMENT"
		}
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

// indexListPragma 用于映射 PRAGMA index_list 返回的数据
type indexListPragma struct {
	Seq     int    `db:"seq"`
	Name    string `db:"name"`
	Unique  int    `db:"unique"`
	Origin  string `db:"origin"`
	Partial int    `db:"partial"`
}

// GetTableIndexes 获取指定表的所有索引及其列信息
func GetTableIndexes(tableName string) ([]models.IndexInfo, error) {
	if !IsValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// Step 1: 获取索引列表（index_list）
	query := fmt.Sprintf("PRAGMA index_list('%s')", tableName)
	var pragmas []indexListPragma
	if err := utils.DB.Select(&pragmas, query); err != nil {
		return nil, fmt.Errorf("failed to get index list: %w", err)
	}
	indexes := make([]models.IndexInfo, 0, len(pragmas))
	for _, p := range pragmas {
		// 获取索引的 SQL
		var sqlNull sql.NullString
		err := utils.DB.Get(&sqlNull, "SELECT sql FROM sqlite_master WHERE type='index' AND name=?", p.Name)

		var indexSQL string
		if err == nil && sqlNull.Valid {
			indexSQL = sqlNull.String
		}

		// 获取索引的列
		columns, err := getIndexColumns(p.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for index %s: %w", p.Name, err)
		}

		indexes = append(indexes, models.IndexInfo{
			Name:    p.Name,
			Unique:  p.Unique == 1,
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

// indexInfoPragma 用于映射 PRAGMA index_info 返回的数据
type indexInfoPragma struct {
	SeqNo      int    `db:"seqno"`
	Cid        int    `db:"cid"`
	ColumnName string `db:"name"`
}

// getIndexColumns 获取某个索引包含的列名
func getIndexColumns(indexName string) ([]string, error) {
	if !IsValidIdentifier(indexName) {
		return nil, fmt.Errorf("invalid index name: %s", indexName)
	}

	query := fmt.Sprintf("PRAGMA index_info('%s')", indexName)
	var pragmas []indexInfoPragma
	if err := utils.DB.Select(&pragmas, query); err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}

	// 提取列名
	columns := make([]string, len(pragmas))
	for i, p := range pragmas {
		columns[i] = p.ColumnName
	}

	return columns, nil
}

// triggerSchema 用于映射 sqlite_master 表中的触发器数据
type triggerSchema struct {
	Name string `db:"name"`
	SQL  string `db:"sql"`
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

	var triggers []triggerSchema
	if err := utils.DB.Select(&triggers, query, tableName); err != nil {
		return nil, fmt.Errorf("failed to get triggers: %w", err)
	}

	// 转换为业务模型
	result := make([]models.TriggerInfo, len(triggers))
	for i, t := range triggers {
		result[i] = models.TriggerInfo{
			Name: t.Name,
			SQL:  t.SQL,
		}
	}

	return result, nil
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
	// 2. 验证列是否存在
	cols, err := GetTableColumns(tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to get table columns: %w", err)
	}
	columnMap := make(map[string]models.ColumnInfo)
	for _, col := range cols {
		columnMap[col.Name] = col
	}

	columns := make([]string, 0, len(data))
	values := make([]string, 0, len(data))
	params := make(map[string]any)
	for k, v := range data {
		if !IsValidIdentifier(k) {
			return 0, fmt.Errorf("invalid column name: %s", k)
		}
		// 检查列是否存在
		if _, exists := columnMap[k]; !exists {
			return 0, fmt.Errorf("column not found: %s", k)
		}

		// 处理 nil 值
		if v == nil && columnMap[k].NotNull {
			return 0, fmt.Errorf("column %s cannot be null", k)
		}
		columns = append(columns, fmt.Sprintf(`"%s"`, k))
		values = append(values, ":"+k)
		params[k] = v
	}
	// 4. 执行 SQL
	query := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		tableName,
		strings.Join(columns, ", "),
		strings.Join(values, ", "),
	)

	result, err := utils.DB.NamedExec(query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to insert row: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// 如果有主键, 支持修改数据
func UpdateRow(tableName string, data map[string]any) (int64, error) {
	// 校验表名
	if !IsValidIdentifier(tableName) {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided for update")
	}
	// 查询表字段
	cols, err := GetTableColumns(tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to get table columns: %w", err)
	}
	// 用主键 WHERE 限制
	var pkCols []string
	colMap := make(map[string]models.ColumnInfo)
	for _, col := range cols {
		colMap[col.Name] = col
		if col.Primary {
			pkCols = append(pkCols, col.Name)
		}
	}
	if len(pkCols) == 0 {
		return 0, fmt.Errorf("table '%s' has no primary key, cannot update", tableName)
	}
	// 4. 验证数据
	for k := range data {
		if !IsValidIdentifier(k) {
			return 0, fmt.Errorf("invalid column name: %s", k)
		}
		if _, exists := colMap[k]; !exists {
			return 0, fmt.Errorf("column '%s' does not exist", k)
		}
	}

	// 5. 构建 UPDATE 语句
	var sets []string
	params := make(map[string]any)
	for k, v := range data {
		if _, exists := colMap[k]; exists {
			if !isPrimaryKey(k, pkCols) {
				sets = append(sets, fmt.Sprintf(`"%s" = :set_%s`, k, k))
				params["set_"+k] = v
			}
		}
	}
	// 构建 WHERE 子句
	var where []string
	for _, pk := range pkCols {
		val, ok := data[pk]
		if !ok {
			return 0, fmt.Errorf("primary key column '%s' must be provided", pk)
		}
		where = append(where, fmt.Sprintf(`"%s" = :where_%s`, pk, pk))
		params["where_"+pk] = val
	}
	if len(sets) == 0 {
		return 0, fmt.Errorf("no columns to update")
	}
	// 6. 执行更新
	query := fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE %s`,
		tableName,
		strings.Join(sets, ", "),
		strings.Join(where, " AND "),
	)
	result, err := utils.DB.NamedExec(query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to execute update: %w", err)
	}

	return result.RowsAffected()
}

// 辅助函数：判断是否为主键列
func isPrimaryKey(colName string, pkCols []string) bool {
	return slices.Contains(pkCols, colName)
}

// 如果有主键, 支持删除
func DeleteRow(tableName string, data map[string]string) (int64, error) {
	// 校验表名
	if !IsValidIdentifier(tableName) {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided for deletion")
	}
	// 查询表字段
	cols, err := GetTableColumns(tableName)
	if err != nil {
		return 0, fmt.Errorf("failed to get table columns: %w", err)
	}
	// 3. 找出主键列
	var pkCols []string
	colMap := make(map[string]bool)
	for _, col := range cols {
		colMap[col.Name] = true
		if col.Primary {
			pkCols = append(pkCols, col.Name)
		}
	}
	if len(pkCols) == 0 {
		return 0, fmt.Errorf("table '%s' has no primary key, cannot delete", tableName)
	}
	// 4. 构建 WHERE 子句和参数
	var where []string
	params := make(map[string]any)
	for _, pk := range pkCols {
		if !IsValidIdentifier(pk) {
			return 0, fmt.Errorf("invalid column name: %s", pk)
		}
		if !colMap[pk] {
			return 0, fmt.Errorf("column '%s' does not exist in table '%s'", pk, tableName)
		}
		val, ok := data[pk]
		if !ok {
			return 0, fmt.Errorf("primary key column '%s' must be provided for deletion", pk)
		}
		where = append(where, fmt.Sprintf(`"%s" = :pk_%s`, pk, pk))
		params["pk_"+pk] = val
	}

	// 5. 执行删除
	query := fmt.Sprintf(
		`DELETE FROM "%s" WHERE %s`,
		tableName,
		strings.Join(where, " AND "),
	)

	result, err := utils.DB.NamedExec(query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to execute delete: %w", err)
	}

	return result.RowsAffected()
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
	result := &QueryTableResult{
		Data: make([]map[string]any, 0),
	}
	// 统计总数：标识符不能参数化，需拼接
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tableName)
	if err := utils.DB.Get(&result.Total, countSQL); err != nil {
		return nil, fmt.Errorf("count failed: %w", err)
	}

	// Fetch data
	// 查询数据：表名拼接，limit/offset 用参数绑定
	dataSQL := fmt.Sprintf(`SELECT * FROM "%s" LIMIT ? OFFSET ?`, tableName)
	rows, err := utils.DB.Queryx(dataSQL, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		// 处理 []byte 类型
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}
		result.Data = append(result.Data, row)
	}

	return result, rows.Err()
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

// 上传文件导入数据JSON/CSV, 支持创建新列, 支持回滚控制
func ImportToTable(ctx context.Context, fileReader io.Reader, fileType, tableName string, createNewColumn, rollback bool) (*ImportResult, error) {
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
	// 开启事务
	tx, err := utils.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	// 仅在未显式提交/回滚时兜底回滚
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback()
		}
	}()
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
		if err := createNewColumns(tableName, columns); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("创建新列失败: %s", err.Error()))
			return result, err
		}
	}
	// 构建 INSERT 语句
	colNames := make([]string, len(columns))
	colParams := make([]string, len(columns))
	for i, col := range columns {
		colNames[i] = fmt.Sprintf(`"%s"`, col)
		colParams[i] = ":" + col
	}

	query := fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s)`,
		tableName,
		strings.Join(colNames, ","),
		strings.Join(colParams, ","),
	)

	// 执行批量插入
	failed := false
	for _, record := range records {
		_, err := tx.NamedExecContext(ctx, query, record)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, err.Error())
			// 限制一下error长度
			if len(result.Errors) > 5 {
				result.Errors = result.Errors[:5]
			}
			failed = true
		} else {
			result.SuccessCount++
		}
	}

	// 提交或回滚事务（由 rollback 参数控制）
	if failed {
		if rollback {
			if err := tx.Rollback(); err != nil {
				return result, fmt.Errorf("回滚事务失败: %w", err)
			}
			closed = true
			return result, fmt.Errorf("部分数据导入失败，已回滚")
		}
		// 不回滚：提交已成功的行
		if err := tx.Commit(); err != nil {
			return result, fmt.Errorf("提交事务失败: %w", err)
		}
		closed = true
		// 返回结果但不作为错误抛出，由调用方依据 FailedCount/Errors 展示
		return result, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	closed = true

	return result, fmt.Errorf("全部导入成功")
}

// 辅助函数：创建新列
func createNewColumns(tableName string, newColumns []string) error {
	// 获取现有列
	existingCols, err := GetTableColumns(tableName)
	if err != nil {
		return err
	}

	existingColMap := make(map[string]bool)
	for _, col := range existingCols {
		existingColMap[col.Name] = true
	}

	// 创建不存在的列
	for _, col := range newColumns {
		if !existingColMap[col] {
			err := NewTableColumn(tableName, NewTableColumnSchema{
				Name: col,
				Type: "TEXT",
			})
			if err != nil {
				return fmt.Errorf("创建列 %s 失败: %w", col, err)
			}
		}
	}
	return nil
}

// 导出表数据, 支持指定字段, 格式 JSON/CSV
func StreamExportTableData(ctx context.Context, tableName string, columns []string, fileType string, w io.Writer) error {
	// 参数验证
	if !IsValidIdentifier(tableName) {
		return fmt.Errorf("非法表名: %s", tableName)
	}
	for _, col := range columns {
		if !IsValidIdentifier(col) {
			return fmt.Errorf("非法列名: %s", col)
		}
	}
	// 构建查询
	query := fmt.Sprintf(
		`SELECT %s FROM "%s"`,
		strings.Join(quotedColumns(columns), ","),
		tableName,
	)
	// 使用 sqlx 查询
	rows, err := utils.DB.QueryxContext(ctx, query)
	if err != nil {
		return fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	// 根据格式进行流式导出
	switch fileType {
	case "json":
		return streamJSON(rows, columns, w)
	case "csv":
		return streamCSV(rows, columns, w)
	default:
		return fmt.Errorf("不支持的格式: %s", fileType)
	}
}

// 使用 sqlx 优化的 JSON 流式导出
func streamJSON(rows *sqlx.Rows, columns []string, w io.Writer) error {
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}

	isFirst := true
	for rows.Next() {
		// 使用 MapScan 直接扫描到 map
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 处理特殊类型
		for k, v := range row {
			switch val := v.(type) {
			case []byte:
				row[k] = string(val)
			}
		}

		// 只保留指定的列
		result := make(map[string]any)
		for _, col := range columns {
			if v, ok := row[col]; ok {
				result[col] = v
			}
		}

		if !isFirst {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		isFirst = false

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("编码 JSON 失败: %w", err)
		}
	}

	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}

	return rows.Err()
}

// 使用 sqlx 优化的 CSV 流式导出
func streamCSV(rows *sqlx.Rows, columns []string, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// 写入表头
	if err := writer.Write(columns); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	// 逐行处理数据
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 按列顺序构建记录
		record := make([]string, len(columns))
		for i, col := range columns {
			v, exists := row[col]
			if !exists {
				record[i] = ""
				continue
			}

			switch val := v.(type) {
			case nil:
				record[i] = ""
			case []byte:
				record[i] = string(val)
			default:
				record[i] = fmt.Sprintf("%v", val)
			}
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入数据行失败: %w", err)
		}
	}

	return rows.Err()
}

// 辅助函数：给列名添加引号
func quotedColumns(columns []string) []string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = fmt.Sprintf(`"%s"`, col)
	}
	return quoted
}
