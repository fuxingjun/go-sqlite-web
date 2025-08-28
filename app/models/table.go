package models

// CreateTableRequest 创建表的请求体
type CreateTableRequest struct {
	TableName string `json:"tableName" validate:"required"`
}

// TableColumn 表示一个列的定义
type TableColumn struct {
	Name    string  `json:"name" validate:"required"`
	Type    string  `json:"type" validate:"required,oneof=TEXT INTEGER REAL BLOB"` // 白名单类型
	Primary bool    `json:"primary,omitempty"`
	NotNull bool    `json:"not_null,omitempty"`
	Unique  bool    `json:"unique,omitempty"`
	Default *string `json:"default,omitempty"` // 支持 NULL 默认值
}

// TableInfo 表示一张表的完整结构信息
type TableInfo struct {
	Columns  []ColumnInfo  `json:"columns"`  // 字段信息
	Indexes  []IndexInfo   `json:"indexes"`  // 索引信息
	Triggers []TriggerInfo `json:"triggers"` // 触发器信息
}

// IndexInfo 表示索引信息
type IndexInfo struct {
	Name    string   `json:"name"`    // 索引名
	Unique  bool     `json:"unique"`  // 是否唯一
	SQL     string   `json:"sql"`     // 创建语句
	Columns []string `json:"columns"` // 索引包含的列（需额外查询）
}

// TriggerInfo 表示触发器信息
type TriggerInfo struct {
	Name string `json:"name"` // 触发器名
	SQL  string `json:"sql"`  // 创建语句
	// 可扩展：event, table, time, etc.
}

// ColumnInfo 表示一张表的结构信息
type ColumnInfo struct {
	CID     int    `json:"cid"`      // 列 ID（从 0 开始）
	Name    string `json:"name"`     // 列名
	Type    string `json:"type"`     // 数据类型
	NotNull bool   `json:"not_null"` // 是否非空
	Default string `json:"default"`  // 默认值（字符串形式）
	Primary bool   `json:"primary"`  // 是否为主键
}

// 导出表格数据参数
type ExportTableSchema struct {
	Columns  []string `json:"columns"`
	Page     int      `json:"page,omitempty"`
	Size     int      `json:"size,omitempty"`
	FileType string   `json:"fileType"` // "json" or "csv"
}
