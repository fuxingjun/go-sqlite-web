## 功能

### 1. 主页
- [ ] 数据库信息
- [ ] 查询所有表
- [ ] 创建表
- [ ] 执行SQL
- [ ] 导出查询结果 json/CSV

### 2. 操作表
- [ ] 删除表,导入数据,导出数据
- [ ] 查询表创建语句
- [ ] 表字段增删改查
- [ ] 表索引查询,增加,删除
- [ ] 查看表数据
- [ ] 执行SQL
- [ ] 插入数据
- [ ] 导入,导出表数据json/CSV



### 1. 主页与登录
|  路径   | 方法  | 功能 |
|  ----  | ----  | ---- |
| /  | GET | 首页，展示数据库基本信息（文件名、大小、表列表等）|
| /login/  | GET/POST | 登录页，支持密码认证 |
| /logout/  |	GET |	清除 session，退出登录  |
> 注：如果设置了密码，则所有非 /login、/static 请求都会被拦截跳转。

### 2. 数据库元信息接口（通用）
|  路径 |	方法 |	功能 |
|  ----  | ----  | ---- |
|  /query/ |	GET/POST |	执行任意 SQL 查询，支持分页、排序、导出 |
|  /create-table/ |	POST |	创建新表（跳转到导入页面） |
|  /export/ |	POST |	导出整个数据库或查询结果为 JSON/CSV |

### 3. 表级操作（需 {table} 参数）
#### 3.1 表结构相关
| 路径 |	方法 |	功能 |
|  ----  | ----  | ---- |
| /{table}/ |	GET	| 查看表结构：列、索引、外键、触发器、建表语句 |
| /{table}/add-column/ |	GET/POST |	添加字段 |
| /{table}/drop-column/ |	GET/POST |	删除字段 |
| /{table}/rename-column/ |	GET/POST |	重命名字段 |
| /{table}/add-index/ |	GET/POST |	添加索引（可唯一） |
| /{table}/drop-index/ |	GET/POST |	删除索引 |
| /{table}/drop-trigger/ |	GET/POST |	删除触发器 |
| /{table}/drop/ |	GET/POST |	删除表或视图 |

#### 3.2 表内容相关
| 路径 |	方法 |	功能 |
|  ----  | ----  | ---- |
| /{table}/content/ |	GET |	查看表数据，分页展示 |
| /{table}/insert/ |	GET/POST |	插入新行 |
| /{table}/update/<b64:pk>/ |	GET/POST |	根据主键更新行 |
| /{table}/delete/<b64:pk>/ |	GET/POST |	根据主键删除行 |

#### 3.3 查询与导入导出
| 路径 |	方法 |	功能 |
|  ----  | ----  | ---- |
| /{table}/query/ |	GET/POST |	对该表执行自定义 SQL 查询 |
| /{table}/export/ |	GET/POST |	导出指定列数据为 JSON/CSV |
| /{table}/import/ |	GET/POST |	从 CSV/JSON 文件导入数据 |