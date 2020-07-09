package frame

//数据库model 一般不直接对外提供服务
// 定义新的结构体  将TableTrait当成匿名属性继承使用
type TableTrait struct {
	DbType          string //数据库类型 默认 mysql
	DbGroup         string //数据库配置 如:db/main
	Table           string //数据表
	IsAutoIncrement bool   //是否自增,默认自增
	PrimaryKey      string //主键,默认id
	dbInstance      Db
}

func (tableTrait *TableTrait) Db() Db {
	if tableTrait.dbInstance == nil {
		if !tableTrait.IsAutoIncrement {
			tableTrait.IsAutoIncrement = true
		}
		if tableTrait.PrimaryKey == "" {
			tableTrait.PrimaryKey = "id"
		}
		if tableTrait.DbType == "" {
			tableTrait.DbType = "mysql"
		}
		if tableTrait.DbType == "mysql" {
			tableTrait.dbInstance = GetMysql(tableTrait.DbGroup)
		} else {
			panic(DbAllowError.Error() + ":" + tableTrait.DbType)
		}
	}
	return tableTrait.dbInstance
}

func (tableTrait *TableTrait) GetTable() string {
	return tableTrait.Table
}

func (tableTrait *TableTrait) Insert(info map[string]interface{}) (int, bool) {
	id, ok := tableTrait.Db().Insert(tableTrait.Table, info).Exec()
	if !ok {
		return 0, false
	}
	if tableTrait.IsAutoIncrement {
		if id > 0 {
			if oid, ok := info[tableTrait.PrimaryKey]; ok {
				return oid.(int), true
			} else {
				return id, true
			}
		}
	} else {
		if oid, ok := info[tableTrait.PrimaryKey]; ok {
			return oid.(int), true
		}
	}
	return 0, true
}

func (tableTrait *TableTrait) Update(id interface{}, info map[string]interface{}) (int, bool) {
	return tableTrait.Db().Where(tableTrait.PrimaryKey, id).Update(tableTrait.Table, info).Exec()
}

func (tableTrait *TableTrait) Delete(id interface{}) (int, bool) {
	return tableTrait.Db().Where(tableTrait.PrimaryKey, id).Delete(tableTrait.Table).Exec()
}

func (tableTrait *TableTrait) GetOne(id interface{}, res interface{}) (interface{}, error) {
	return tableTrait.Db().Select().Where(tableTrait.PrimaryKey, id).From(tableTrait.Table).Fetch(res)
}

func (tableTrait *TableTrait) GetMulti(idArr []interface{}, res interface{}) ([]interface{}, error) {
	return tableTrait.Db().Select().Where(tableTrait.PrimaryKey, idArr).From(tableTrait.Table).FetchAll(res)
}

func (tableTrait *TableTrait) TotalCount(where map[string]interface{}) (int, error) {
	whereSql := ""
	if sqlStr, ok := where["_sql"]; ok {
		whereSql = sqlStr.(string)
		delete(where, "_sql")
	}
	tableTrait.Db().MultiWhere(where)
	if whereSql != "" {
		tableTrait.Db().WhereSql(whereSql)
	}
	type Total struct {
		Total int
	}
	res, err := tableTrait.Db().SelectCount("*").From(tableTrait.Table).Fetch(&Total{})
	if err == nil {
		return res.(Total).Total, nil
	}
	return 0, err
}

func (tableTrait *TableTrait) Load(where map[string]interface{}, page int, pageItem int, order string, res interface{}) ([]interface{}, error) {
	whereSql := ""
	if sqlStr, ok := where["_sql"]; ok {
		whereSql = sqlStr.(string)
		delete(where, "_sql")
	}
	tableTrait.Db().MultiWhere(where)
	if whereSql != "" {
		tableTrait.Db().WhereSql(whereSql)
	}
	return tableTrait.Db().Page(page).Count(pageItem).OrderBy(order).Select("*").From(tableTrait.Table).FetchAll(res)
}

func (tableTrait *TableTrait) LoadAll(where map[string]interface{}, order string, res interface{}) ([]interface{}, error) {
	whereSql := ""
	if sqlStr, ok := where["_sql"]; ok {
		whereSql = sqlStr.(string)
		delete(where, "_sql")
	}
	tableTrait.Db().MultiWhere(where)
	if whereSql != "" {
		tableTrait.Db().WhereSql(whereSql)
	}
	return tableTrait.Db().OrderBy(order).Select("*").From(tableTrait.Table).FetchAll(res)
}

func (tableTrait *TableTrait) LoadOne(where map[string]interface{}, res interface{}) (interface{}, error) {
	whereSql := ""
	if sqlStr, ok := where["_sql"]; ok {
		whereSql = sqlStr.(string)
		delete(where, "_sql")
	}
	tableTrait.Db().MultiWhere(where)
	if whereSql != "" {
		tableTrait.Db().WhereSql(whereSql)
	}
	return tableTrait.Db().Select("*").Limit(1).From(tableTrait.Table).Fetch(res)
}
