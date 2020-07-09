package frame

import (
	"database/sql"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

type dbGroup struct {
	Master *sql.DB
	Slaves []*sql.DB
	Config *dbConfig
}
type dbConfig struct {
	Host   string
	Port   int
	DbName string
}

type dbHost struct {
	Type   string          `toml:"type"`
	Master *dbHostConfig   `toml:"master"`
	Slaves []*dbHostConfig `toml:"slaves"`
}
type dbHostConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	Username        string `toml:"username"`
	Password        string `toml:"password"`
	DbName          string `toml:"dbname"`
	Charset         string `toml:"charset"`
	MaxOpenConn     int    `toml:"max_open_conns"`
	MaxIdleConn     int    `toml:"max_idle_conns"`
	ConnMaxLifeTime int    `toml:"conns_max_lifetime"`
}

var dbGroupCache map[string]*dbGroup
var dbGroupCacheOnce sync.Once
var dbLock sync.Mutex

//获取数据库连接池
func openDB(dbGroups string) *dbGroup {
	if dbGroupCache == nil {
		dbGroupCacheOnce.Do(func() {
			dbGroupCache = make(map[string]*dbGroup)
		})
	}
	cache, ok := dbGroupCache[dbGroups]
	if ok {
		return cache
	}
	dbLock.Lock()
	defer dbLock.Unlock()
	dbHostConfig := &dbHost{}
	err := App().Env(dbGroups, dbHostConfig)
	if err != nil {
		panic(DbError.Error() + ":" + err.Error())
	}
	dbType := dbHostConfig.Type
	masterConfig := dbHostConfig.Master
	dbDriver := masterConfig.Username + ":" +
		masterConfig.Password + "@tcp(" +
		masterConfig.Host + ":" +
		strconv.Itoa(masterConfig.Port) + ")/" +
		masterConfig.DbName + "?charset=" +
		masterConfig.Charset
	master, err := sql.Open(dbType, dbDriver)
	if err != nil {
		msg := map[string]interface{}{
			"error": err.Error(),
			"stack": string(debug.Stack()),
		}
		App().Log.Error(msg, LogMysqlError)
		panic(DbError.Error() + ":" + dbGroups)
	}
	master.SetMaxOpenConns(masterConfig.MaxOpenConn)
	master.SetMaxIdleConns(masterConfig.MaxIdleConn)
	master.SetConnMaxLifetime(time.Duration(masterConfig.ConnMaxLifeTime) * time.Second)
	slavesConfig := dbHostConfig.Slaves
	slaves := make([]*sql.DB, 0)
	for _, v := range slavesConfig {
		bufferDriver := v.Username + ":" +
			v.Password + "@tcp(" +
			v.Host + ":" +
			strconv.Itoa(v.Port) + ")/" +
			v.DbName + "?charset=" +
			v.Charset
		slave, err := sql.Open(dbType, bufferDriver)
		if err != nil {
			msg := map[string]interface{}{
				"error": err.Error(),
				"stack": string(debug.Stack()),
			}
			App().Log.Error(msg, LogMysqlError)
			panic(DbError.Error() + ":" + dbGroups)
		}
		slave.SetMaxOpenConns(v.MaxOpenConn)
		slave.SetMaxIdleConns(v.MaxIdleConn)
		slave.SetConnMaxLifetime(time.Duration(v.ConnMaxLifeTime) * time.Second)
		slaves = append(slaves, slave)
	}
	config := &dbConfig{
		Host:   masterConfig.Host,
		Port:   masterConfig.Port,
		DbName: masterConfig.DbName,
	}
	dbGroupCache[dbGroups] = &dbGroup{Master: master, Slaves: slaves, Config: config}
	return dbGroupCache[dbGroups]
}

// 关闭数据库连接池
func closeDB(dbGroups ...string) {
	if dbGroupCache == nil {
		return
	}
	if len(dbGroups) > 0 {
		for _, v := range dbGroups {
			cache, ok := dbGroupCache[v]
			if ok {
				_ = cache.Master.Close()
				for _, v := range cache.Slaves {
					_ = v.Close()
				}
			}
		}
	} else {
		for _, cache := range dbGroupCache {
			_ = cache.Master.Close()
			for _, v := range cache.Slaves {
				_ = v.Close()
			}
		}
	}
	return
}
