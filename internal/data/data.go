package data

import (
	"fmt"
	"time"

	"jas-agent/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gLogger "gorm.io/gorm/logger"
)

// ProviderSet 定义 data 层依赖注入集合
var ProviderSet = wire.NewSet(
	NewDB,
	NewData,
	NewAgentRepo,
	NewMCPRepo,
	NewKnowledgeBaseRepo,
	NewDocumentRepo,
)

// Data 聚合数据访问资源。
type Data struct {
	db  *gorm.DB
	log *log.Helper
}

// NewDB 根据配置初始化 GORM 数据库。
func NewDB(c *conf.Data, logger log.Logger) (*gorm.DB, error) {
	if c == nil || c.Database == nil || c.Database.Source == "" {
		return nil, nil
	}

	cfg := &gorm.Config{
		Logger: gLogger.Default.LogMode(gLogger.Warn),
	}

	var dialector gorm.Dialector
	switch c.Database.Driver {
	case "", "mysql":
		dialector = mysql.Open(c.Database.Source)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.Database.Driver)
	}

	db, err := gorm.Open(dialector, cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if c.Database.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(int(c.Database.MaxIdleConns))
	}
	if c.Database.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(int(c.Database.MaxOpenConns))
	}
	if c.Database.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(c.Database.ConnMaxLifetime) * time.Second)
	}

	return db, nil
}

// NewData 创建 Data，并返回资源清理函数。
func NewData(db *gorm.DB, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "data"))

	if db == nil {
		helper.Warn("database not configured, running without persistence")
		return &Data{db: nil, log: helper}, func() {}, nil
	}

	cleanup := func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		if err := sqlDB.Close(); err != nil {
			helper.Errorf("close database error: %v", err)
		}
	}

	return &Data{
		db:  db,
		log: helper,
	}, cleanup, nil
}

// DB 返回底层 GORM DB。
func (d *Data) DB() *gorm.DB {
	return d.db
}

func (d *Data) logger() *log.Helper {
	return d.log
}
