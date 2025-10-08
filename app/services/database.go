package services

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/utils"
	"github.com/jmoiron/sqlx"
)

type DBInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
	SQLiteVer  string    `json:"sqliteVersion"`
}

// GetFileCreationTime 返回文件的创建时间
func GetFileCreationTime(path string) (time.Time, error) {
	// linux 返回更新时间
	utils.GetLogger("").Debug("GetFileCreationTime", "os", runtime.GOOS, "path", path)
	if runtime.GOOS == "linux" {
		fi, err := os.Stat(path)
		if err != nil {
			return time.Time{}, err
		}
		return fi.ModTime(), nil
	}
	// windows 和 macOS 使用命令行获取创建时间
	return GetCreationTimeViaCommand(path)
}

func GetCreationTimeViaCommand(path string) (time.Time, error) {
	var cmd *exec.Cmd
	var stderr bytes.Buffer
	var creationTime time.Time
	switch filepath.Separator {
	case '\\':
		// Windows
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf(`[System.IO.File]::GetCreationTimeUtc('%s').ToString('yyyy-MM-ddTHH:mm:ss') + 'Z'`, path),
		)
	case '/', ':':
		// macOS (使用 stat 命令)
		cmd = exec.Command("stat", "-f", "%B", path)
	default:
		return time.Time{}, fmt.Errorf("unsupported operating system")
	}
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, fmt.Errorf("error executing command: %v, stderr: %s", err, stderr.String())
	}
	output := strings.TrimSpace(string(out))
	utils.GetLogger("").Debug("Command output", "output", output)
	// 解析输出的创建时间
	// 这里需要根据实际返回的格式进行格式化解析
	switch filepath.Separator {
	case '\\':
		// PowerShell的输出示例: 2025-09-14T09:14:23Z
		creationTime, err = time.Parse(time.RFC3339, output)
		if err != nil {
			return time.Time{}, fmt.Errorf("error parsing creation time from command output: %v", err)
		}
	default:
		// 解析为秒
		ts, err := strconv.ParseInt(output, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("时间解析失败: %w", err)
		}
		// 构造 time.Time(UTC 时间)
		creationTime = time.Unix(ts, 0).UTC()
	}
	return creationTime, nil
}

func GetDBInfo() (*DBInfo, error) {
	path := utils.DBPath // 假设你在 db/connection.go 中保存了原始路径
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 使用 sqlx.Get 简化版本查询
	var sqliteVer string
	err = utils.DB.Get(&sqliteVer, "SELECT sqlite_version()")
	if err != nil {
		return nil, fmt.Errorf("获取 SQLite 版本失败: %w", err)
	}

	ctime, _ := GetFileCreationTime(path)

	return &DBInfo{
		Path:       path,
		Size:       fi.Size(),
		CreatedAt:  ctime,
		ModifiedAt: fi.ModTime(),
		SQLiteVer:  sqliteVer,
	}, nil
}

type tableInfo struct {
	Name string `db:"name"`
}

func GetTables() ([]string, error) {
	// 使用 sqlx.Select 直接将结果映射到结构体切片
	var tables []tableInfo
	err := utils.DB.Select(&tables, `
        SELECT name 
        FROM sqlite_master 
        WHERE type='table' 
          AND name NOT LIKE 'sqlite_%'
    `)
	if err != nil {
		return nil, fmt.Errorf("获取表信息失败: %w", err)
	}

	// 提取表名
	result := make([]string, len(tables))
	for i, t := range tables {
		result[i] = t.Name
	}

	return result, nil
}

// View 结构添加 db 标签
type View struct {
	Name string `db:"name" json:"name"`
	SQL  string `db:"sql" json:"sql"`
}

func GetViews() (*[]View, error) {
	var views []View
	err := utils.DB.Select(&views, `
        SELECT name, sql 
        FROM sqlite_master 
        WHERE type='view'
        ORDER BY name
    `)
	if err != nil {
		return nil, fmt.Errorf("获取视图失败: %w", err)
	}
	return &views, nil
}

// triggerSchema 用于映射 sqlite_master 表中的触发器数据
type triggerSchemaItem struct {
	Name      string `db:"name"`
	TableName string `db:"table_name"`
	SQL       string `db:"sql"`
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
	var schemas []triggerSchemaItem
	if err := utils.DB.Select(&schemas, query); err != nil {
		return nil, fmt.Errorf("获取触发器失败: %w", err)
	}

	// 转换为业务模型
	triggers := make([]*models.Trigger, len(schemas))
	for i, s := range schemas {
		trigger := &models.Trigger{
			Name:       s.Name,
			Table:      s.TableName,
			Definition: formatTriggerDefinition(s.SQL),
			SQL:        s.SQL,
		}
		// 解析类型和时机
		trigger.Type, trigger.Timing = parseTriggerTypeTiming(s.SQL)
		triggers[i] = trigger
	}

	return triggers, nil
}

// triggerDetailSchema 用于映射单个触发器的查询结果
type triggerDetailSchema struct {
	TableName string `db:"tbl_name"`
	SQL       string `db:"sql"`
}

// GetTriggerByName 查询指定触发器
func GetTriggerByName(name string) (*models.Trigger, error) {
	if !IsValidIdentifier(name) {
		return nil, fmt.Errorf("非法触发器名称: %s", name)
	}

	var schema triggerDetailSchema
	query := `SELECT tbl_name, sql FROM sqlite_master WHERE type = 'trigger' AND name = ?`
	err := utils.DB.Get(&schema, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("触发器不存在: %s", name)
		}
		return nil, fmt.Errorf("查询触发器失败: %w", err)
	}

	trigger := &models.Trigger{
		Name:       name,
		Table:      schema.TableName,
		SQL:        schema.SQL,
		Definition: formatTriggerDefinition(schema.SQL),
	}
	trigger.Type, trigger.Timing = parseTriggerTypeTiming(schema.SQL)

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

type SQLResult struct {
	Type         string           `json:"type"` // query / exec
	Columns      []string         `json:"columns,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	Total        int64            `json:"total,omitempty"` // SELECT 总数
	HasNext      bool             `json:"hasNext,omitempty"`
	Page         int              `json:"page,omitempty"`
	Size         int              `json:"size,omitempty"`
	LastInsertId int64            `json:"lastInsertId,omitempty"` // INSERT
	Affected     int64            `json:"affected,omitempty"`     // UPDATE/DELETE
	Duration     float64          `json:"duration"`               // 执行毫秒
	Message      string           `json:"message,omitempty"`      // 提示信息（如 "1 row inserted"）
	Error        string           `json:"error,omitempty"`        // 错误信息
}

// SQL 语句类型正则表达式
var sqlPatterns = struct {
	selectStmt *regexp.Regexp
	insertStmt *regexp.Regexp
	updateStmt *regexp.Regexp
	deleteStmt *regexp.Regexp
}{
	selectStmt: regexp.MustCompile(`(?i)^\s*SELECT\s+`),
	insertStmt: regexp.MustCompile(`(?i)^\s*INSERT\s+`),
	updateStmt: regexp.MustCompile(`(?i)^\s*UPDATE\s+`),
	deleteStmt: regexp.MustCompile(`(?i)^\s*DELETE\s+`),
}

func classifySQL(sql string) string {
	switch {
	case sqlPatterns.selectStmt.MatchString(sql):
		return "SELECT"
	case sqlPatterns.insertStmt.MatchString(sql):
		return "INSERT"
	case sqlPatterns.updateStmt.MatchString(sql):
		return "UPDATE"
	case sqlPatterns.deleteStmt.MatchString(sql):
		return "DELETE"
	default:
		return "EXEC"
	}
}

// ExecuteSQL 执行任意 SQL 语句，适用于管理工具
func ExecuteSQL(sqlStr string, page, size int) *SQLResult {
	start := time.Now()
	result := &SQLResult{
		Duration: 0,
		Page:     page,
		Size:     size,
	}
	defer func() {
		result.Duration = float64(time.Since(start).Milliseconds())
	}()
	// 清理 SQL（去注释）
	sqlStr = cleanSQL(sqlStr)
	if sqlStr == "" {
		result.Error = "SQL is null"
		return result
	}
	// 判断类型
	stmtType := classifySQL(sqlStr)
	// 分页只对 SELECT 有效
	if stmtType == "SELECT" {
		return executeSelect(sqlStr, page, size, start)
	}
	// 其他类型：INSERT/UPDATE/DELETE/DDL
	return executeExec(sqlStr, stmtType, start)
}

type Pagination struct {
	Page int
	Size int
}

func getPagination(sql string) *Pagination {
	limitRe := regexp.MustCompile(`(?i)\s+LIMIT\s+(\d+)`)
	offsetRe := regexp.MustCompile(`(?i)\s+OFFSET\s+(\d+)`)
	var page, size int
	if limitMatch := limitRe.FindStringSubmatch(sql); len(limitMatch) == 2 {
		if l, err := strconv.Atoi(limitMatch[1]); err == nil && l > 0 {
			size = l
		}
	}
	if offsetMatch := offsetRe.FindStringSubmatch(sql); len(offsetMatch) == 2 {
		if o, err := strconv.Atoi(offsetMatch[1]); err == nil && o >= 0 {
			page = (o / size) + 1
		}
	}
	if size > 0 && page > 0 {
		return &Pagination{Page: page, Size: size}
	}
	return nil
}

func hasPagination(sql string) bool {
	limitRe := regexp.MustCompile(`(?i)\s+LIMIT\s+\d+`)
	offsetRe := regexp.MustCompile(`(?i)\s+OFFSET\s+\d+`)
	return limitRe.MatchString(sql) || offsetRe.MatchString(sql)
}

func executeSelect(sqlStr string, page, size int, start time.Time) *SQLResult {
	result := &SQLResult{
		Type:     "query",
		Page:     page,
		Size:     size,
		Duration: 0,
	}

	defer func() {
		result.Duration = float64(time.Since(start).Milliseconds())
	}()
	// 分页优先级: 1- 传的分页参数 2- SQL的分页参数 3- 默认分页
	pagination := getPagination(sqlStr)
	utils.GetLogger("").Debug("Pagination", "pagination", pagination)

	// 去掉分页参数, LIMIT, OFFSET 不区分大小写, 有可能只有 LIMIT 或者 OFFSET, 所以即时判断没有 pagination 参数也要去掉
	limitRe := regexp.MustCompile(`(?i)\s+LIMIT\s+\d+`)
	offsetRe := regexp.MustCompile(`(?i)\s+OFFSET\s+\d+`)
	sqlStr = limitRe.ReplaceAllString(sqlStr, "")
	sqlStr = offsetRe.ReplaceAllString(sqlStr, "")
	sqlStr = strings.TrimSpace(sqlStr)

	if page >= 1 && size >= 1 {
		result.Page = page
		result.Size = size
	} else {
		if pagination != nil {
			result.Size = pagination.Size
			result.Page = pagination.Page
		} else {
			// 默认分页
			result.Page = 1
			result.Size = 500
		}
	}

	// 获取总数
	if total, err := getCount(sqlStr); err == nil {
		result.Total = total
	}

	// 构建分页 SQL
	sqlStr = strings.TrimRight(sqlStr, ";")
	offset := (result.Page - 1) * result.Size
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", sqlStr, result.Size+1, offset)

	utils.GetLogger("").Debug("Executing paginated SQL", "sql", paginatedSQL)
	// 执行查询
	rows, err := utils.DB.Queryx(paginatedSQL)
	if err != nil {
		result.Error = fmt.Sprintf("execute failed: %v", err)
		return result
	}
	defer rows.Close()

	// 获取列名
	if cols, err := rows.Columns(); err == nil {
		result.Columns = cols
	} else {
		result.Error = fmt.Sprintf("get columns failed: %v", err)
		return result
	}

	// 使用 MapScan 扫描数据
	rowCount := 0
	for rows.Next() && rowCount <= result.Size {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			utils.GetLogger("").Error("scan row failed", "error", err)
			continue
		}
		// 处理 []byte 类型
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}
		result.Rows = append(result.Rows, row)
		rowCount++
	}

	result.HasNext = rowCount > result.Size
	if result.HasNext {
		result.Rows = result.Rows[:len(result.Rows)-1]
	}

	result.Message = fmt.Sprintf("%d rows returned", len(result.Rows))
	return result
}

func executeExec(sqlStr, stmtType string, start time.Time) *SQLResult {
	result := &SQLResult{
		Type:     "exec",
		Duration: 0,
	}

	defer func() {
		result.Duration = float64(time.Since(start).Milliseconds())
	}()

	res, err := utils.DB.Exec(sqlStr)
	if err != nil {
		result.Error = fmt.Sprintf("executed failed: %v", err)
		return result
	}

	// 获取影响行数
	affected, _ := res.RowsAffected()
	result.Affected = affected

	// 获取 LastInsertId（仅 INSERT）
	if stmtType == "INSERT" {
		if id, err := res.LastInsertId(); err == nil {
			result.LastInsertId = id
			result.Message = fmt.Sprintf("ins, ID=%d", id)
		} else {
			result.Message = fmt.Sprintf("inserted successfully, %d rows affected", affected)
		}
	} else {
		result.Message = fmt.Sprintf("executed successfully, %d rows affected", affected)
	}

	return result
}

func cleanSQL(sql string) string {
	sql = regexp.MustCompile(`--.*$`).ReplaceAllString(sql, "")
	sql = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(sql, "")
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")
	return strings.TrimSpace(sql)
}

// getCount 获取查询的总行数, 如果含有limit/offset, 去掉
func getCount(sql string) (int64, error) {
	if hasPagination(sql) {
		// 去掉 LIMIT 和 OFFSET
		sql = regexp.MustCompile(`(?i)\s+LIMIT\s+\d+`).ReplaceAllString(sql, "")
		sql = regexp.MustCompile(`(?i)\s+OFFSET\s+\d+`).ReplaceAllString(sql, "")
		sql = strings.TrimSpace(sql)
	}
	// 去掉封号
	sql = strings.TrimRight(sql, ";")
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS _count", strings.TrimRight(sql, ";"))
	utils.GetLogger("").Debug("Count SQL", "sql", countSQL)
	var total int64
	if err := utils.DB.Get(&total, countSQL); err != nil {
		return -1, fmt.Errorf("count failed: %w", err)
	}
	return total, nil
}

// getColumnsFromQuery 获取查询的列名
// 方法：执行一次干跑（带 LIMIT 0）
func getColumnsFromQuery(sqlStr string) ([]string, error) {
	rows, err := utils.DB.Queryx(fmt.Sprintf("SELECT * FROM (%s) AS t LIMIT 0", sqlStr))
	if err != nil {
		return nil, fmt.Errorf("dry run failed: %w", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get columns failed: %w", err)
	}
	return cols, nil
}

// CreateSQLiteTable 根据请求创建表
func CreateSQLiteTable(req *models.CreateTableRequest) error {
	// 检查表名合法性（简单校验）
	if !IsValidIdentifier(req.TableName) {
		return fmt.Errorf("invalid table name: %s", req.TableName)
	}

	// Step 1: 检查表是否已存在
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`
	err := utils.DB.Get(&exists, query, req.TableName)
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
		return fmt.Errorf("获取列名失败: %w", err)
	}
	// 构造分页查询
	offset := (page - 1) * size
	paginatedSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", sql, size+1, offset)

	// 使用 sqlx 查询
	rows, err := utils.DB.Queryx(paginatedSQL)
	if err != nil {
		return fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

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
func ExportToJSON(columns []string, rows *sqlx.Rows, w io.Writer) error {
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}

	first := true
	for rows.Next() {
		// 使用 MapScan 直接扫描到 map
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		// 处理特殊类型
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}

		// 只保留指定的列
		result := make(map[string]any)
		for _, col := range columns {
			if v, ok := row[col]; ok {
				result[col] = v
			}
		}

		if !first {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		first = false

		if err := json.NewEncoder(w).Encode(result); err != nil {
			return fmt.Errorf("序列化行失败: %w", err)
		}
	}

	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}

	return rows.Err()
}

// 数据导出为 CSV
func ExportToCSV(columns []string, rows *sqlx.Rows, w io.Writer) error {
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

	// 逐行处理数据
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}

		// 按列顺序构建记录
		record := make([]string, len(columns))
		for i, col := range columns {
			v, exists := row[col]
			if !exists {
				record[i] = ""
				continue
			}

			record[i] = formatValue(v)
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入数据行失败: %w", err)
		}
	}

	return rows.Err()
}

// 辅助函数：格式化值为字符串
func formatValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(val)
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	default:
		return fmt.Sprintf("%v", val)
	}
}
