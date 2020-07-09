package frame

// 数据库接口
type Db interface {
	Select(field ...interface{}) Db
	SelectCount(field string, alias ...string) Db
	Join(table string, condition string, alias ...string) Db
	LeftJoin(table string, condition string, alias ...string) Db
	RightJoin(table string, condition string, alias ...string) Db
	Insert(table string, info map[string]interface{}) Db
	InsertBatch(table string, data []map[string]interface{}, onceMaxCounts ...int) Db
	Update(table string, info map[string]interface{}) Db
	Replace(table string, info map[string]interface{}) Db
	ReplaceBatch(table string, data []map[string]interface{}, onceMaxCounts ...int) Db
	Delete(table string) Db
	Sql(preSql string, params ...interface{}) Db
	From(table string, alias ...string) Db
	Distinct() Db
	Ignore() Db
	ForceMaster() Db
	Where(field string, value interface{}, ops ...string) Db
	MultiWhere(conditions map[string]interface{}) Db
	OrWhere(field string, value interface{}, ops ...string) Db
	MultiOrWhere(conditions map[string]interface{}) Db
	WhereSql(whereSql string, params ...interface{}) Db
	Having(field string, value interface{}, ops ...string) Db
	MultiHaving(conditions map[string]interface{}) Db
	OrHaving(field string, value interface{}, ops ...string) Db
	MultiOrHaving(conditions map[string]interface{}) Db
	HavingSql(havingSql string, params ...interface{}) Db
	BeginWhereGroup() Db
	BeginOrWhereGroup() Db
	EndWhereGroup() Db
	BeginHavingGroup() Db
	BeginOrHavingGroup() Db
	EndHavingGroup() Db
	GroupBy(field interface{}) Db
	OrderBy(field string) Db
	Limit(count int) Db
	OffSet(count int) Db
	Page(count int) Db
	Count(count int) Db
	GetSql() interface{}
	Exec() (int, bool)
	Fetch(interface{}) (interface{}, error)
	FetchAll(interface{}) ([]interface{}, error)
	AffectedRows() int
	GetLastInsertId() int
	BeginTrans() bool
	CommitTrans() bool
	RollbackTrans() bool
}
