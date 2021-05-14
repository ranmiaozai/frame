package frame

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	_ = iota
	SqlTypeSelect
	SqlTypeInsert
	SqlTypeUpdate
	SqlTypeReplace
	SqlTypeDelete
	SqlTypeInsertBatch
	SqlTypeUpdateBatch
	SqlTypeReplaceBatch
)

const RwTypeMaster = "m"
const RwTypeSlave = "s"

type Mysql struct {
	stmt          *sql.Stmt
	commitCon     *sql.Tx
	inTrans       bool
	transDepth    int
	lastErrorCode int
	//↓↓↓↓↓↓每次SQL拼接前都需要reset的属性↓↓↓↓↓↓//
	sqlType     int
	useDistinct bool
	useIgnore   bool
	//select count
	selectCountSql string
	//field
	fieldSql string
	//table
	tableSql string
	//join sql
	joinSql string
	//where
	whereSql    string
	whereParams []interface{}
	//group by
	groupBySql string
	//having
	havingSql    string
	havingParams []interface{}
	//order by
	orderBySql string
	//limit
	limit  int
	offset int
	page   int
	count  int
	//insert
	valuesSql string
	//insert batch
	valuesSqlArr []string
	//update
	updateSql string
	//update batch
	updateSqlArr         []string
	updateWhereSqlArr    []string
	updateWhereParamsArr []interface{}
	//insert,update
	params []interface{}
	//insert batch,update batch
	paramsArr []interface{}
	//update batch
	updateParamsArr [][]interface{}

	//上一次的SQL语句(参数)
	lastPreSql    string
	lastPreSqlArr []string
	lastParams    []interface{}
	//SQL执行后的影响行数
	affectedRows     int
	affectedRowsOnce int
	//上一次执行插入语句后返回的insert_id
	lastInsertId int
	//是否强制使用主库
	forceMaster bool
	//↑↑↑↑↑↑每次SQL拼接前都需要reset的属性↑↑↑↑↑↑//

	//连接闲置时间超时重连
	connectWaitTimeout int //连接超时时间,单位:秒
	connectTimeout     int //连接超时时间,单位:秒

	//当前操作 做临时变量用
	handleTemp string
	//begin exec time 开始执行时间
	BeginTime int

	DbGroup *dbGroup //数据库连接池
}

//没有使用单例 是因为协程间会共用 导致问题
//目前又无法获取协程id 无法做到同一协程间单例
func GetMysql(dbGroup string) *Mysql {
	DbGroup := openDB(dbGroup)
	return &Mysql{
		DbGroup:              DbGroup,
		stmt:                 nil,
		commitCon:            nil,
		inTrans:              false,
		transDepth:           0,
		lastErrorCode:        0,
		sqlType:              0,
		useDistinct:          false,
		useIgnore:            false,
		selectCountSql:       "",
		fieldSql:             "",
		tableSql:             "",
		joinSql:              "",
		whereSql:             "",
		whereParams:          make([]interface{}, 0),
		groupBySql:           "",
		havingSql:            "",
		havingParams:         make([]interface{}, 0),
		orderBySql:           "",
		limit:                0,
		offset:               -1,
		page:                 0,
		count:                0,
		valuesSql:            "",
		valuesSqlArr:         make([]string, 0),
		updateSql:            "",
		updateSqlArr:         make([]string, 0),
		updateWhereSqlArr:    make([]string, 0),
		updateWhereParamsArr: make([]interface{}, 0),
		params:               make([]interface{}, 0),
		paramsArr:            make([]interface{}, 0),
		updateParamsArr:      make([][]interface{}, 0),
		lastPreSql:           "",
		lastPreSqlArr:        make([]string, 0),
		lastParams:           make([]interface{}, 0),
		affectedRows:         0,
		affectedRowsOnce:     0,
		lastInsertId:         0,
		handleTemp:           "",
		BeginTime:            0,
		forceMaster:          false,
	}
}

func (mysql *Mysql) resetBefore() {
	mysql.affectedRows = 0
	mysql.affectedRowsOnce = 0
	mysql.lastInsertId = 0
}

func (mysql *Mysql) resetAfter() {
	mysql.sqlType = 0
	mysql.useDistinct = false
	mysql.useIgnore = false
	mysql.selectCountSql = ""
	mysql.fieldSql = ""
	mysql.tableSql = ""
	mysql.joinSql = ""
	mysql.whereSql = ""
	mysql.whereParams = make([]interface{}, 0)
	mysql.groupBySql = ""
	mysql.havingSql = ""
	mysql.orderBySql = ""
	mysql.offset = -1
	mysql.limit = 0
	mysql.page = 0
	mysql.count = 0
	mysql.valuesSql = ""
	mysql.valuesSqlArr = make([]string, 0)
	mysql.updateSql = ""
	mysql.updateSqlArr = make([]string, 0)
	mysql.updateWhereSqlArr = make([]string, 0)
	mysql.updateWhereParamsArr = make([]interface{}, 0)
	mysql.params = make([]interface{}, 0)
	mysql.paramsArr = make([]interface{}, 0)
	mysql.updateParamsArr = make([][]interface{}, 0)
	mysql.forceMaster = false
	mysql.lastPreSql = ""
	mysql.lastPreSqlArr = make([]string, 0)
	mysql.lastParams = make([]interface{}, 0)
	mysql.handleTemp = ""
	mysql.BeginTime = 0
}

func (mysql *Mysql) Reset() {
	mysql.resetBefore()
	mysql.resetAfter()
}

func (mysql *Mysql) escapeField(fieldName string) string {
	fieldName = strings.ReplaceAll(fieldName, "`", "")
	pos := strings.Index(fieldName, ".")
	if pos > 0 {
		table := string([]byte(fieldName)[0:pos])
		field := string([]byte(fieldName)[pos+1:])
		if field == "*" {
			return "`" + table + "`." + field
		} else {
			return "`" + table + "`.`" + field + "`"
		}
	} else {
		if fieldName == "*" {
			return "*"
		} else {
			return "`" + fieldName + "`"
		}
	}
}

func (mysql *Mysql) escapeTable(tableName string) string {
	return "`" + strings.ReplaceAll(strings.Trim(tableName, " "), "`", "") + "`"
}

func (mysql *Mysql) Select(field ...interface{}) Db {
	mysql.resetBefore()
	mysql.sqlType = SqlTypeSelect
	fieldArr := make([]string, 0)
	for _, v := range field {
		vType := reflect.TypeOf(v).String()
		switch vType {
		case "[]string":
			for _, val := range v.([]string) {
				fieldArr = append(fieldArr, strings.Split(val, ",")...)
			}
		default:
			fieldArr = append(fieldArr, strings.Split(v.(string), ",")...)
		}
	}
	fieldArrR := make([]string, 0)
	for _, v := range fieldArr {
		value := strings.Trim(v, " ")
		if strings.Index(value, " ") > 0 || strings.Index(value, "(") != -1 {
			fieldArrR = append(fieldArrR, value)
		} else {
			fieldArrR = append(fieldArrR, mysql.escapeField(value))
		}
	}
	if len(fieldArrR) > 0 {
		mysql.fieldSql = strings.Join(fieldArrR, ",")
	} else {
		mysql.fieldSql = "*"
	}
	return mysql
}

func (mysql *Mysql) SelectCount(field string, aliass ...string) Db {
	alias := "total"
	if len(aliass) > 0 {
		alias = aliass[0]
	}
	mysql.resetBefore()
	mysql.sqlType = SqlTypeSelect
	if field != "*" {
		field = mysql.escapeField(field)
	}
	mysql.selectCountSql = "SELECT COUNT(" + field + ") `" + alias + "`"
	return mysql
}

func (mysql *Mysql) Join(table string, condition string, alias ...string) Db {
	table = mysql.escapeTable(table)
	if len(alias) > 0 {
		table += " " + mysql.escapeTable(alias[0])
	}
	if mysql.joinSql != "" {
		mysql.joinSql += " JOIN " + table + " ON " + condition
	} else {
		mysql.joinSql = " JOIN " + table + " ON " + condition
	}
	return mysql
}

func (mysql *Mysql) LeftJoin(table string, condition string, alias ...string) Db {
	table = mysql.escapeTable(table)
	if len(alias) > 0 {
		table += " " + mysql.escapeTable(alias[0])
	}
	if mysql.joinSql != "" {
		mysql.joinSql += " LEFT JOIN " + table + " ON " + condition
	} else {
		mysql.joinSql = " LEFT JOIN " + table + " ON " + condition
	}
	return mysql
}

func (mysql *Mysql) RightJoin(table string, condition string, alias ...string) Db {
	table = mysql.escapeTable(table)
	if len(alias) > 0 {
		table += " " + mysql.escapeTable(alias[0])
	}
	if mysql.joinSql != "" {
		mysql.joinSql += " RIGHT JOIN " + table + " ON " + condition
	} else {
		mysql.joinSql = " RIGHT JOIN " + table + " ON " + condition
	}
	return mysql
}

func (mysql *Mysql) Insert(table string, info map[string]interface{}) Db {
	mysql.resetBefore()
	mysql.sqlType = SqlTypeInsert
	mysql.tableSql = mysql.escapeTable(table)
	params := make([]interface{}, 0)
	fields := make([]string, 0)
	valueSql := ""
	for k, v := range info {
		fields = append(fields, mysql.escapeField(k))
		params = append(params, v)
		valueSql += "?,"
	}
	mysql.fieldSql = strings.Join(fields, ",")
	mysql.valuesSql = "(" + strings.TrimRight(valueSql, ",") + ")"
	mysql.params = params
	return mysql
}

func (mysql *Mysql) InsertBatch(table string, data []map[string]interface{}, onceMaxCounts ...int) Db {
	onceMaxCount := 100
	if len(onceMaxCounts) > 0 {
		onceMaxCount = onceMaxCounts[0]
	}
	mysql.resetBefore()
	mysql.sqlType = SqlTypeInsertBatch
	if len(data) == 0 {
		return mysql
	}
	mysql.tableSql = mysql.escapeTable(table)
	fields := make([]string, 0)
	keys := make([]string, 0)
	valuesSegment := ""
	firstItem := data[0]
	for k := range firstItem {
		fields = append(fields, mysql.escapeField(k))
		keys = append(keys, k)
		valuesSegment += "?,"
	}
	mysql.fieldSql = strings.Join(fields, ",")
	valuesSegment = "(" + strings.TrimRight(valuesSegment, ",") + "),"
	if onceMaxCount <= 0 {
		onceMaxCount = 500
	}
	insertDataOnceArr := arrayChunk(data, onceMaxCount)
	for _, insertDataOnce := range insertDataOnceArr {
		insertDataCount := len(insertDataOnce)
		mysql.valuesSqlArr = append(mysql.valuesSqlArr, strings.TrimRight(strings.Repeat(valuesSegment, insertDataCount), ","))
		params := make([]interface{}, 0)
		for _, item := range insertDataOnce {
			for _, v := range keys {
				params = append(params, item[v])
			}
		}
		mysql.paramsArr = append(mysql.paramsArr, params)
	}
	return mysql
}

func (mysql *Mysql) Update(table string, info map[string]interface{}) Db {
	mysql.resetBefore()
	mysql.sqlType = SqlTypeUpdate
	mysql.tableSql = mysql.escapeTable(table)
	mysql.params = make([]interface{}, 0)
	updateSegment := make([]string, 0)
	for k, v := range info {
		if v == nil {
			updateSegment = append(updateSegment, k)
		} else {
			updateSegment = append(updateSegment, mysql.escapeField(k)+" =?")
			mysql.params = append(mysql.params, v)
		}
	}
	mysql.updateSql = strings.Join(updateSegment, ",")
	return mysql
}

func (mysql *Mysql) Replace(table string, info map[string]interface{}) Db {
	mysql.resetBefore()
	mysql.sqlType = SqlTypeReplace
	mysql.tableSql = mysql.escapeTable(table)
	field := make([]string, 0)
	for k, v := range info {
		field = append(field, mysql.escapeTable(k))
		mysql.params = append(mysql.params, v)
	}
	mysql.fieldSql = strings.Join(field, ",")
	mysql.valuesSql = "(" + strings.TrimRight(strings.Repeat("?,", len(info)), ",") + ")"
	return mysql
}

func (mysql *Mysql) ReplaceBatch(table string, data []map[string]interface{}, onceMaxCounts ...int) Db {
	onceMaxCount := 100
	if len(onceMaxCounts) > 0 {
		onceMaxCount = onceMaxCounts[0]
	}
	mysql.resetBefore()
	mysql.sqlType = SqlTypeReplaceBatch
	if len(data) <= 0 {
		return mysql
	}
	mysql.tableSql = mysql.escapeTable(table)
	firstItem := data[0]
	fields := make([]string, 0)
	for k := range firstItem {
		fields = append(fields, mysql.escapeField(k))
	}
	mysql.fieldSql = strings.Join(fields, ",")
	columnCount := len(firstItem)
	valuesSegment := "(" + strings.TrimRight(strings.Repeat("?,", columnCount), ",") + "),"
	if onceMaxCount <= 0 {
		onceMaxCount = 500
	}
	insertDataOnceArr := arrayChunk(data, onceMaxCount)
	for _, insertDataOnce := range insertDataOnceArr {
		insertDataCount := len(insertDataOnce)
		mysql.valuesSqlArr = append(mysql.valuesSqlArr, strings.TrimRight(strings.Repeat(valuesSegment, insertDataCount), ","))
		params := make([]interface{}, 0)
		for _, item := range insertDataOnce {
			for _, v := range item {
				params = append(params, v)
			}
		}
		mysql.paramsArr = append(mysql.paramsArr, params)
	}
	return mysql
}

func (mysql *Mysql) Delete(table string) Db {
	mysql.resetBefore()
	mysql.sqlType = SqlTypeDelete
	mysql.tableSql = mysql.escapeTable(table)
	return mysql
}

func (mysql *Mysql) Sql(preSql string, params ...interface{}) Db {
	mysql.resetBefore()
	mysql.lastPreSql = preSql
	mysql.lastParams = params
	//是否是插入数据的操作
	if strings.ToUpper(strings.Trim(preSql, " ")[:7]) == "INSERT " {
		mysql.sqlType = SqlTypeInsert
	}
	return mysql
}

func (mysql *Mysql) From(table string, alias ...string) Db {
	table = mysql.escapeTable(table)
	if len(alias) > 0 {
		table += " " + mysql.escapeTable(alias[0])
	}
	if mysql.tableSql != "" {
		mysql.tableSql += "," + table
	} else {
		mysql.tableSql = table
	}
	return mysql
}

func (mysql *Mysql) Distinct() Db {
	mysql.useDistinct = true
	return mysql
}

func (mysql *Mysql) Ignore() Db {
	mysql.useIgnore = true
	return mysql
}

func (mysql *Mysql) ForceMaster() Db {
	mysql.forceMaster = true
	return mysql
}

func (mysql *Mysql) condition(field string, value interface{}, op string) (string, []interface{}) {
	//$field '字段名' 或者 '字段名 运算符'
	field = strings.Trim(field, " ")
	if op == "" && strings.Index(field, " ") > 0 {
		arr := strings.SplitN(field, " ", 2)
		if len(arr) > 0 {
			field = arr[0]
		} else {
			field = ""
		}
		if len(arr) > 1 {
			op = strings.ToUpper(strings.Trim(arr[1], " "))
		} else {
			op = ""
		}
	}
	field = mysql.escapeField(field)
	conditionParams := make([]interface{}, 0)
	conditionSql := ""
	switch reflect.TypeOf(value).String() {
	case "[]interface {}":
	case "[]string":
	case "[]int":
		value1 := value.([]interface{})
		if op == "" {
			op = "IN"
		}
		switch op {
		case "IN":
			count := len(value1)
			conditionSql = field + " IN (" + strings.TrimRight(strings.Repeat("?,", count), ",") + ")"
			for _, item := range value1 {
				conditionParams = append(conditionParams, item)
			}
		case "NOT IN":
			count := len(value1)
			conditionSql = field + " NOT IN (" + strings.TrimRight(strings.Repeat("?,", count), ",") + ")"
			for _, item := range value1 {
				conditionParams = append(conditionParams, item)
			}
		default:
			panic("this op not support an array value")
		}
	default:
		if op == "" {
			op = "="
		}
		switch op {
		case "=", "!=", "<>", ">", ">=", "<", "<=", "LIKE", "NOT LIKE":
			conditionSql = field + " " + op + " ?"
			conditionParams = append(conditionParams, value)
		case "IS NULL", "IS NOT NULL":
			conditionSql = field + " " + op
		default:
			panic("this op just support an array value")
		}
	}
	return conditionSql, conditionParams
}

func (mysql *Mysql) Where(field string, value interface{}, ops ...string) Db {
	op := ""
	if len(ops) > 0 {
		op = ops[0]
	}
	whereSql, whereParams := mysql.condition(field, value, op)
	if mysql.whereSql != "" {
		if strings.LastIndex(mysql.whereSql, "(") == (len(mysql.whereSql) - 1) {
			mysql.whereSql += whereSql
		} else {
			mysql.whereSql += " AND " + whereSql
		}
	} else {
		mysql.whereSql = "WHERE " + whereSql
	}
	for _, v := range whereParams {
		mysql.whereParams = append(mysql.whereParams, v)
	}
	return mysql
}

func (mysql *Mysql) MultiWhere(conditions map[string]interface{}) Db {
	for k, v := range conditions {
		if v == nil {
			mysql.WhereSql(k)
		} else {
			mysql.Where(k, v, "")
		}
	}
	return mysql
}

func (mysql *Mysql) OrWhere(field string, value interface{}, ops ...string) Db {
	op := ""
	if len(ops) > 0 {
		op = ops[0]
	}
	whereSql, whereParams := mysql.condition(field, value, op)
	if mysql.whereSql != "" {
		if strings.LastIndex(mysql.whereSql, "(") == (len(mysql.whereSql) - 1) {
			mysql.whereSql += whereSql
		} else {
			mysql.whereSql += " OR " + whereSql
		}
	} else {
		mysql.whereSql = "WHERE " + whereSql
	}
	for _, v := range whereParams {
		mysql.whereParams = append(mysql.whereParams, v)
	}
	return mysql
}

func (mysql *Mysql) MultiOrWhere(conditions map[string]interface{}) Db {
	for k, v := range conditions {
		mysql.OrWhere(k, v, "")
	}
	return mysql
}

func (mysql *Mysql) WhereSql(whereSql string, paramss ...interface{}) Db {
	params := make([]interface{}, 0)
	if len(paramss) > 0 {
		paramssType := reflect.TypeOf(paramss[0]).String()
		switch paramssType {
		case "[]interface {}":
			params = paramss[0].([]interface{})
		default:
			params = paramss
		}
	}
	if mysql.whereSql != "" {
		if strings.LastIndex(mysql.whereSql, "(") == (len(mysql.whereSql) - 1) {
			mysql.whereSql += whereSql
		} else {
			mysql.whereSql += " AND " + whereSql
		}
	} else {
		mysql.whereSql += "WHERE " + whereSql
	}
	for _, v := range params {
		mysql.whereParams = append(mysql.whereParams, v)
	}
	return mysql
}

func (mysql *Mysql) Having(field string, value interface{}, ops ...string) Db {
	op := ""
	if len(ops) > 0 {
		op = ops[0]
	}
	havingSql, havingParams := mysql.condition(field, value, op)
	if mysql.havingSql != "" {
		if strings.LastIndex(mysql.havingSql, "(") == (len(mysql.havingSql) - 1) {
			mysql.havingSql += havingSql
		} else {
			mysql.havingSql += " AND " + havingSql
		}
	} else {
		mysql.havingSql = "HAVING " + havingSql
	}
	for _, v := range havingParams {
		mysql.havingParams = append(mysql.havingParams, v)
	}
	return mysql
}

func (mysql *Mysql) MultiHaving(conditions map[string]interface{}) Db {
	for k, v := range conditions {
		mysql.Having(k, v, "")
	}
	return mysql
}

func (mysql *Mysql) OrHaving(field string, value interface{}, ops ...string) Db {
	op := ""
	if len(ops) > 0 {
		op = ops[0]
	}
	havingSql, havingParams := mysql.condition(field, value, op)
	if mysql.havingSql != "" {
		if strings.LastIndex(mysql.havingSql, "(") == (len(mysql.havingSql) - 1) {
			mysql.havingSql += havingSql
		} else {
			mysql.havingSql += " OR " + havingSql
		}
	} else {
		mysql.havingSql = "HAVING " + havingSql
	}
	for _, v := range havingParams {
		mysql.havingParams = append(mysql.havingParams, v)
	}
	return mysql
}

func (mysql *Mysql) MultiOrHaving(conditions map[string]interface{}) Db {
	for k, v := range conditions {
		mysql.OrHaving(k, v, "")
	}
	return mysql
}

func (mysql *Mysql) HavingSql(havingSql string, paramss ...interface{}) Db {
	params := make([]interface{}, 0)
	if len(paramss) > 0 {
		paramssType := reflect.TypeOf(paramss[0]).String()
		switch paramssType {
		case "[]interface {}":
			params = paramss[0].([]interface{})
		default:
			params = paramss
		}
	}
	if mysql.havingSql != "" {
		if strings.LastIndex(mysql.havingSql, "(") == (len(mysql.havingSql) - 1) {
			mysql.havingSql += havingSql
		} else {
			mysql.havingSql += " AND " + havingSql
		}
	} else {
		mysql.havingSql += "HAVING " + havingSql
	}
	for _, v := range params {
		mysql.havingParams = append(mysql.havingParams, v)
	}
	return mysql
}

func (mysql *Mysql) BeginWhereGroup() Db {
	if mysql.whereSql != "" {
		mysql.whereSql += " AND ("
	} else {
		mysql.whereSql += " WHERE ("
	}
	return mysql
}

func (mysql *Mysql) BeginOrWhereGroup() Db {
	if mysql.whereSql != "" {
		mysql.whereSql += " OR ("
	} else {
		mysql.whereSql += " WHERE ("
	}
	return mysql
}

func (mysql *Mysql) EndWhereGroup() Db {
	if mysql.whereSql != "" {
		mysql.whereSql += ")"
	}
	return mysql
}

func (mysql *Mysql) BeginHavingGroup() Db {
	if mysql.havingSql != "" {
		mysql.havingSql += " AND ("
	} else {
		mysql.havingSql += " HAVING ("
	}
	return mysql
}

func (mysql *Mysql) BeginOrHavingGroup() Db {
	if mysql.havingSql != "" {
		mysql.havingSql += " OR ("
	} else {
		mysql.havingSql += " HAVING ("
	}
	return mysql
}

func (mysql *Mysql) EndHavingGroup() Db {
	if mysql.havingSql != "" {
		mysql.havingSql += ")"
	}
	return mysql
}

func (mysql *Mysql) GroupBy(field interface{}) Db {
	fieldArr := make([]string, 0)
	switch field.(type) {
	case string:
		fieldArr = strings.Split(field.(string), ",")
	default:
		fieldArr = field.([]string)
	}
	fieldArr1 := make([]string, 0)
	for _, v := range fieldArr {
		fieldArr1 = append(fieldArr1, mysql.escapeField(v))
	}
	if mysql.groupBySql != "" {
		mysql.groupBySql += "," + strings.Join(fieldArr, ",")
	} else {
		mysql.groupBySql = "GROUP BY " + strings.Join(fieldArr, ",")
	}
	return mysql
}

func (mysql *Mysql) OrderBy(field string) Db {
	if field == "" {
		return mysql
	}
	fieldArr := strings.Split(field, ",")
	fields := make([]string, 0)
	for _, v := range fieldArr {
		arr := strings.SplitN(v, " ", 2)
		orderField := mysql.escapeField(arr[0])
		if len(arr) > 1 && strings.ToUpper(arr[1]) == "DESC" {
			orderField += " DESC"
		}
		fields = append(fields, orderField)
	}
	if mysql.orderBySql != "" {
		mysql.orderBySql += "," + strings.Join(fields, ",")
	} else {
		mysql.orderBySql = "ORDER BY " + strings.Join(fields, ",")
	}
	return mysql
}

func (mysql *Mysql) Limit(count int) Db {
	mysql.limit = count
	return mysql
}

func (mysql *Mysql) OffSet(offset int) Db {
	mysql.offset = offset
	return mysql
}

func (mysql *Mysql) Page(page int) Db {
	if page <= 1 {
		page = 1
	}
	mysql.page = page
	return mysql
}

func (mysql *Mysql) Count(count int) Db {
	mysql.count = count
	return mysql
}

func (mysql *Mysql) getLimitSql() string {
	limitSql := ""
	if mysql.limit != 0 {
		if mysql.offset != -1 {
			limitSql += "LIMIT ?,?"
		} else {
			limitSql += "LIMIT ?"
		}
	} else if mysql.page > 0 && mysql.count > 0 {
		offset := 0
		if mysql.page > 1 {
			offset = (mysql.page - 1) * mysql.count
		}
		limitSql = "LIMIT ?,?"
		mysql.offset = offset
		mysql.limit = mysql.count
	}
	return limitSql
}

func (mysql *Mysql) getLimitParams() []int {
	limitParams := make([]int, 0)
	if mysql.limit > 0 {
		if mysql.offset > -1 {
			limitParams = append(limitParams, mysql.offset, mysql.limit)
		} else {
			limitParams = append(limitParams, mysql.limit)
		}
	}
	return limitParams
}

func (mysql *Mysql) getPrepareSql() string {
	if mysql.lastPreSql != "" {
		return mysql.lastPreSql
	}
	switch mysql.sqlType {
	case SqlTypeSelect:
		selectSql := ""
		if mysql.selectCountSql != "" {
			selectSql = mysql.selectCountSql
		} else {
			if mysql.useDistinct {
				selectSql = "SELECT DISTINCT " + mysql.fieldSql
			} else {
				selectSql = "SELECT " + mysql.fieldSql
			}
		}
		fromSql := "FROM " + mysql.tableSql
		sqlStr := selectSql + " " + fromSql
		if mysql.joinSql != "" {
			sqlStr += " " + mysql.joinSql
		}
		if mysql.whereSql != "" {
			sqlStr += " " + mysql.whereSql
		}
		if mysql.groupBySql != "" {
			sqlStr += " " + mysql.groupBySql
		}
		if mysql.havingSql != "" {
			sqlStr += " " + mysql.havingSql
		}
		if mysql.orderBySql != "" {
			sqlStr += " " + mysql.orderBySql
		}
		limitSql := mysql.getLimitSql()
		if limitSql != "" {
			sqlStr += " " + limitSql
		}
		mysql.lastPreSql = sqlStr
	case SqlTypeInsert:
		ignoreSql := ""
		if mysql.useIgnore {
			ignoreSql = "IGNORE "
		}
		mysql.lastPreSql = "INSERT " + ignoreSql + "INTO " + mysql.tableSql + " (" + mysql.fieldSql + ") VALUES " + mysql.valuesSql
	case SqlTypeInsertBatch:
		ignoreSql := ""
		if mysql.useIgnore {
			ignoreSql = "IGNORE "
		}
		for _, v := range mysql.valuesSqlArr {
			mysql.lastPreSqlArr = append(mysql.lastPreSqlArr, "INSERT "+ignoreSql+"INTO "+mysql.tableSql+" ("+mysql.fieldSql+") VALUES "+v)
		}
		mysql.lastPreSql = ""
	case SqlTypeUpdate:
		sqlStr := "UPDATE " + mysql.tableSql
		if mysql.joinSql != "" {
			sqlStr += " " + mysql.joinSql
		}
		sqlStr += " SET " + mysql.updateSql
		if mysql.whereSql != "" {
			sqlStr += " " + mysql.whereSql
		}
		if mysql.orderBySql != "" {
			sqlStr += " " + mysql.orderBySql
		}
		limitSql := mysql.getLimitSql()
		if limitSql != "" {
			sqlStr += " " + limitSql
		}
		mysql.lastPreSql = sqlStr
	case SqlTypeUpdateBatch:
		for key, updateSql := range mysql.updateSqlArr {
			sqlStr := "UPDATE " + mysql.tableSql
			if mysql.joinSql != "" {
				sqlStr += " " + mysql.joinSql
			}
			sqlStr += " SET " + updateSql + " " + mysql.updateWhereSqlArr[key]
			if mysql.whereSql != "" {
				sqlStr += " AND (" + string(mysql.whereSql[6:]) + ")"
			}
			if mysql.orderBySql != "" {
				sqlStr += " " + mysql.orderBySql
			}
			limitSql := mysql.getLimitSql()
			if limitSql != "" {
				sqlStr += " " + limitSql
			}
			mysql.lastPreSqlArr = append(mysql.lastPreSqlArr, sqlStr)
		}
		mysql.lastPreSql = ""
	case SqlTypeReplace:
		mysql.lastPreSql = "REPLACE INTO " + mysql.tableSql + " (" + mysql.fieldSql + ") VALUES " + mysql.valuesSql
	case SqlTypeReplaceBatch:
		for _, valueSql := range mysql.valuesSqlArr {
			mysql.lastPreSqlArr = append(mysql.lastPreSqlArr, "REPLACE INTO "+mysql.tableSql+" ("+mysql.fieldSql+") VALUES "+valueSql)
		}
		mysql.lastPreSql = ""
	case SqlTypeDelete:
		sqlStr := "DELETE FROM " + mysql.tableSql
		if mysql.whereSql != "" {
			sqlStr += " " + mysql.whereSql
		}
		if mysql.orderBySql != "" {
			sqlStr += " " + mysql.orderBySql
		}
		limitSql := mysql.getLimitSql()
		if limitSql != "" {
			sqlStr += " " + limitSql
		}
		mysql.lastPreSql = sqlStr
	default:
		mysql.lastPreSql = ""
	}
	return mysql.lastPreSql
}

func (mysql *Mysql) getParams() []interface{} {
	if len(mysql.lastParams) == 0 {
		switch mysql.sqlType {
		case SqlTypeSelect:
			mysql.lastParams = append(mysql.whereParams, mysql.havingParams...)
			for _, v := range mysql.getLimitParams() {
				mysql.lastParams = append(mysql.lastParams, v)
			}
		case SqlTypeInsert:
			mysql.lastParams = mysql.params
		case SqlTypeInsertBatch:
			mysql.lastParams = mysql.paramsArr
		case SqlTypeUpdate:
			mysql.lastParams = append(mysql.params, mysql.whereParams...)
			for _, v := range mysql.getLimitParams() {
				mysql.lastParams = append(mysql.lastParams, v)
			}
		case SqlTypeUpdateBatch:
			paramsArr := make([]interface{}, 0)
			for key, updateParams := range mysql.updateParamsArr {
				paramsArr = append(paramsArr, updateParams...)
				paramsArr = append(paramsArr, mysql.updateWhereParamsArr[key])
				paramsArr = append(paramsArr, mysql.whereParams...)
				for _, v := range mysql.getLimitParams() {
					paramsArr = append(paramsArr, v)
				}
			}
			mysql.lastParams = paramsArr
		case SqlTypeReplace:
			mysql.lastParams = mysql.params
		case SqlTypeReplaceBatch:
			mysql.lastParams = mysql.paramsArr
		case SqlTypeDelete:
			mysql.lastParams = mysql.whereParams
			for _, v := range mysql.getLimitParams() {
				mysql.lastParams = append(mysql.lastParams, v)
			}
		default:
			mysql.lastParams = make([]interface{}, 0)
		}
	}
	return mysql.lastParams
}

func (mysql *Mysql) getConn(rwType string) *sql.DB {
	if mysql.inTrans == true || mysql.forceMaster == true || rwType == RwTypeMaster {
		return mysql.DbGroup.Master
	}
	rand.Seed(time.Now().UnixNano())
	return mysql.DbGroup.Slaves[rand.Intn(len(mysql.DbGroup.Slaves))]
}

func (mysql *Mysql) pdoExecute(preSql string, params []interface{}, rwType string) interface{} {
	marker := "?"
	preSqlSegments := strings.Split(preSql, "?")
	paramCount := len(preSqlSegments) - 1
	actualPreSql := preSqlSegments[0]
	actualParams := make([]interface{}, 0)
	i := 1
	for _, val := range params {
		switch reflect.TypeOf(val).String() {
		case "[]interface {}":
			actualPreSql += strings.TrimRight(strings.Repeat(marker+",", len(val.([]interface{}))), ",") + preSqlSegments[i]
			for _, v := range val.([]interface{}) {
				actualParams = append(actualParams, v)
			}
		default:
			actualPreSql += marker + preSqlSegments[i]
			actualParams = append(actualParams, val)
		}
		if i > paramCount {
			break
		}
		i++
	}
	mysql.lastErrorCode = 0
	var stmt *sql.Stmt
	var err error
	mysql.BeginTime = int(time.Now().Unix())
	//前置操作
	if mysqlHandle != nil && mysqlHandle.beforeExecute != nil {
		mysqlHandle.beforeExecute(mysql)
	}
	if mysql.commitCon != nil {
		stmt, err = mysql.commitCon.Prepare(actualPreSql)
	} else {
		conn := mysql.getConn(rwType)
		stmt, err = conn.Prepare(actualPreSql)
	}
	if err != nil {
		//报错
		if mysqlHandle != nil && mysqlHandle.errExecute != nil {
			mysqlHandle.errExecute(mysql, err)
		}
		return nil
	}
	mysql.stmt = stmt
	if stmt == nil {
		//报错
		return nil
	}
	if mysql.handleTemp == "fetch" {
		row := stmt.QueryRow(actualParams...)
		return row
	} else if mysql.handleTemp == "fetchAll" {
		rows, err := stmt.Query(actualParams...)
		if err != nil {
			//报错
			if mysqlHandle != nil && mysqlHandle.errExecute != nil {
				mysqlHandle.errExecute(mysql, err)
			}
			return nil
		}
		return rows
	} else {
		result, err := mysql.stmt.Exec(actualParams...)
		if err != nil {
			//报错
			if mysqlHandle != nil && mysqlHandle.errExecute != nil {
				mysqlHandle.errExecute(mysql, err)
			}
			return nil
		}
		affectRows, _ := result.RowsAffected()
		mysql.affectedRowsOnce = int(affectRows)
		lastInsertId, _ := result.LastInsertId()
		mysql.lastInsertId = int(lastInsertId)
		//后置操作
		if mysqlHandle != nil && mysqlHandle.afterExecute != nil {
			mysqlHandle.afterExecute(mysql)
		}
		return result
	}
}

func (mysql *Mysql) GetSql() interface{} {
	defer mysql.resetAfter()
	if mysql.sqlType == SqlTypeInsertBatch || mysql.sqlType == SqlTypeUpdateBatch || mysql.sqlType == SqlTypeReplaceBatch {
		mysql.getPrepareSql()
		preSqlArr := mysql.lastPreSqlArr
		paramsArr := mysql.getParams()
		sqlArr := make([]string, 0)
		for key, preSql := range preSqlArr {
			preSqlSegments := strings.Split(preSql, "?")
			paramCount := len(preSqlSegments) - 1
			sqlBuffer := preSqlSegments[0]
			i := 1
			for _, val := range paramsArr[key].([]interface{}) {
				sqlBuffer += addSlashesParam(val) + preSqlSegments[i]
				if i >= paramCount {
					break
				}
				i++
			}
			sqlArr = append(sqlArr, sqlBuffer)
		}
		return sqlArr
	} else {
		preSqlSegments := strings.Split(mysql.getPrepareSql(), "?")
		paramCount := len(preSqlSegments) - 1
		result := preSqlSegments[0]
		i := 1
		for _, val := range mysql.getParams() {
			result += addSlashesParam(val) + preSqlSegments[i]
			if i > paramCount {
				break
			}
			i++
		}
		return result
	}
}

func (mysql *Mysql) Exec() (int, bool) {
	defer stmtClose(mysql)
	//执行"写"的SQL语句
	rwType := RwTypeMaster
	preSqlData := mysql.getPrepareSql()
	paramsData := mysql.getParams()
	mysql.affectedRows = 0
	if mysql.sqlType == SqlTypeInsertBatch || mysql.sqlType == SqlTypeUpdateBatch || mysql.sqlType == SqlTypeReplaceBatch {
		for key, preSql := range mysql.lastPreSqlArr {
			if res := mysql.pdoExecute(preSql, paramsData[key].([]interface{}), rwType); res == nil {
				mysql.affectedRows = mysql.affectedRowsOnce
				mysql.resetAfter()
				return mysql.affectedRows, false
			}
			if mysql.affectedRowsOnce > 0 {
				mysql.affectedRows += mysql.affectedRowsOnce
			}
		}
		mysql.resetAfter()
		return mysql.affectedRows, true
	} else {
		res := mysql.pdoExecute(preSqlData, paramsData, rwType)
		mysql.affectedRows = mysql.affectedRowsOnce
		defer mysql.resetAfter()
		if res == nil {
			return 0, false
		}
		//添加操作返回自增id
		if mysql.sqlType == SqlTypeInsert {
			if mysql.affectedRows == 1 {
				return mysql.lastInsertId, true
			}
		}
		return mysql.affectedRows, true
	}
}

func (mysql *Mysql) Fetch(res interface{}) (interface{}, error) {
	defer stmtClose(mysql)
	mysql.handleTemp = "fetch"
	execResult := mysql.pdoExecute(mysql.getPrepareSql(), mysql.getParams(), RwTypeSlave)
	if execResult == nil {
		return nil, DbHandleError
	}
	//后置操作
	if mysqlHandle != nil && mysqlHandle.afterExecute != nil {
		mysqlHandle.afterExecute(mysql)
	}
	mysql.resetAfter()

	s := reflect.ValueOf(res).Elem()
	length := s.NumField()
	oneRow := make([]interface{}, length)
	for i := 0; i < length; i++ {
		oneRow[i] = s.Field(i).Addr().Interface()
	}

	err := execResult.(*sql.Row).Scan(oneRow...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if mysqlHandle != nil && mysqlHandle.errExecute != nil {
			mysqlHandle.errExecute(mysql, err)
		}
		return nil, err
	}
	return s.Interface(), nil
}

func (mysql *Mysql) FetchAll(res interface{}) ([]interface{}, error) {
	defer stmtClose(mysql)
	defer mysql.resetAfter()
	mysql.handleTemp = "fetchAll"
	execResult := mysql.pdoExecute(mysql.getPrepareSql(), mysql.getParams(), RwTypeSlave)
	defer func() {
		if execResult != nil {
			_ = execResult.(*sql.Rows).Close()
		}
	}()
	if execResult == nil {
		return nil, DbHandleError
	}
	//后置操作
	if mysqlHandle != nil && mysqlHandle.afterExecute != nil {
		mysqlHandle.afterExecute(mysql)
	}

	s := reflect.ValueOf(res).Elem()
	length := s.NumField()
	oneRow := make([]interface{}, length)
	for i := 0; i < length; i++ {
		oneRow[i] = s.Field(i).Addr().Interface()
	}

	result := make([]interface{}, 0)
	for execResult.(*sql.Rows).Next() {
		err := execResult.(*sql.Rows).Scan(oneRow...)
		if err != nil {
			if mysqlHandle != nil && mysqlHandle.errExecute != nil {
				mysqlHandle.errExecute(mysql, err)
			}
			panic(err)
		}
		result = append(result, s.Interface())
	}

	return result, nil
}

func (mysql *Mysql) AffectedRows() int {
	if mysql.affectedRows <= 0 && mysql.affectedRowsOnce > 0 {
		mysql.affectedRows = mysql.affectedRowsOnce
	}
	return mysql.affectedRows
}

func (mysql *Mysql) GetLastInsertId() int {
	return mysql.lastInsertId
}

func (mysql *Mysql) BeginTrans() bool {
	if !mysql.inTrans {
		mysql.inTrans = true
	}
	mysql.transDepth++
	if mysql.transDepth > 1 {
		return true
	}
	mysql.lastErrorCode = 0
	tx, err := mysql.DbGroup.Master.Begin()
	if err != nil {
		if mysqlHandle != nil && mysqlHandle.errExecute != nil {
			mysqlHandle.errExecute(mysql, err)
		}
		return false
	}
	mysql.commitCon = tx
	return true
}

func (mysql *Mysql) CommitTrans() bool {
	if !mysql.inTrans {
		return true
	}
	if mysql.transDepth > 1 {
		mysql.transDepth--
		return true
	}
	mysql.lastErrorCode = 0
	if mysql.commitCon == nil {
		return true
	}
	err := mysql.commitCon.Commit()
	if err != nil {
		if mysqlHandle != nil && mysqlHandle.errExecute != nil {
			mysqlHandle.errExecute(mysql, err)
		}
		return false
	}
	mysql.commitCon = nil
	mysql.inTrans = false
	mysql.transDepth = 0
	return true
}

func (mysql *Mysql) RollbackTrans() bool {
	if !mysql.inTrans {
		return true
	}
	if mysql.transDepth > 1 {
		mysql.transDepth--
		return true
	}
	if mysql.commitCon == nil {
		return true
	}
	mysql.lastErrorCode = 0
	err := mysql.commitCon.Rollback()
	if err != nil {
		if mysqlHandle != nil && mysqlHandle.errExecute != nil {
			mysqlHandle.errExecute(mysql, err)
		}
		return false
	}
	mysql.commitCon = nil
	mysql.inTrans = false
	mysql.transDepth = 0
	return true
}

/**
几个注入mysql的方法
*/
type MySqlFun func(mysql *Mysql)
type MySqlErrorFun func(mysql *Mysql, err error)
type MySqlHandle struct {
	beforeExecute MySqlFun
	afterExecute  MySqlFun
	errExecute    MySqlErrorFun
}

var mysqlHandle *MySqlHandle
var mysqlHandleOnce sync.Once

func getMysqlHandle() *MySqlHandle {
	if mysqlHandle == nil {
		mysqlHandleOnce.Do(func() {
			mysqlHandle = &MySqlHandle{}
		})
	}
	return mysqlHandle
}
func SetMysqlBeforeExecute(f MySqlFun) {
	getMysqlHandle().beforeExecute = f
}
func SetMysqlAfterExecute(f MySqlFun) {
	getMysqlHandle().afterExecute = f
}
func SetMysqlErrorExecute(f MySqlErrorFun) {
	getMysqlHandle().errExecute = f
}

/**
一些用到的函数
*/
func stmtClose(mysql *Mysql) {
	if mysql.stmt != nil {
		_ = mysql.stmt.Close()
		mysql.stmt = nil
	}
}

func addSlashesParam(val interface{}) string {
	str := ""
	valType := reflect.TypeOf(val).String()
	switch valType {
	case "bool":
		if val.(bool) {
			str += "1"
		} else {
			str += "0"
		}
	case "string":
		str += "'" + addSlashes(val.(string)) + "'"
	case "[]interface {}":
		strIn := ""
		for _, item := range val.([]interface{}) {
			itemType := reflect.TypeOf(item).String()
			switch itemType {
			case "string":
				strIn += "'" + addSlashes(item.(string)) + "',"
			case "float64":
				strIn += strconv.FormatFloat(item.(float64), 'f', 6, 64)
			case "float32":
				strIn += strconv.FormatFloat(float64(item.(float32)), 'f', 6, 64)
			default:
				strIn += strconv.Itoa(item.(int))
			}
		}
		str += strings.TrimRight(strIn, ",")
	case "float64":
		str += strconv.FormatFloat(val.(float64), 'f', 6, 64)
	case "float32":
		str += strconv.FormatFloat(float64(val.(float32)), 'f', 6, 32)
	case "int", "int8", "int16", "int64", "int32", "uint", "uint8", "uint16", "uint32", "uint64":
		str += strconv.Itoa(val.(int))
	default:
		str += val.(string)
	}
	return str
}
