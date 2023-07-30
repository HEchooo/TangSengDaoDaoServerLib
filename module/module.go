package module

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	migrate "github.com/rubenv/sql-migrate"
)

func Setup(r *wkhttp.WKHttp, ctx *config.Context) error {

	// 获取所有模块
	ms := register.GetModules()

	// 初始化SQL
	var sqlfss []register.SQLFS
	for _, m := range ms {
		sqlfss = append(sqlfss, m.SQLDir)
	}
	err := executeSQL(sqlfss, ctx.DB())
	if err != nil {
		return err
	}
	// 注册api
	for _, m := range ms {
		a := m.SetupAPI(ctx)
		a.Route(r)
	}
	return nil

}

// 执行sql
func executeSQL(sqlfss []register.SQLFS, session *dbr.Session) error {
	migrations := &FileDirMigrationSource{
		sqlfss: sqlfss,
	}
	_, err := migrate.Exec(session.DB, "mysql", migrations, migrate.Up)
	if err != nil {
		return err
	}
	return nil
}

type byID []*migrate.Migration

func (b byID) Len() int           { return len(b) }
func (b byID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byID) Less(i, j int) bool { return b[i].Less(b[j]) }

// FileDirMigrationSource 文件目录源 遇到目录进行递归获取
type FileDirMigrationSource struct {
	sqlfss []register.SQLFS
}

// FindMigrations FindMigrations
func (f FileDirMigrationSource) FindMigrations() ([]*migrate.Migration, error) {

	if len(f.sqlfss) == 0 {
		return nil, nil
	}
	migrations := make([]*migrate.Migration, 0, 100)

	for _, sqlfs := range f.sqlfss {
		err := f.findMigrations(sqlfs, &migrations)
		if err != nil {
			return nil, err
		}
	}

	// Make sure migrations are sorted
	sort.Sort(byID(migrations))

	return migrations, nil
}

func (f FileDirMigrationSource) findMigrations(fs register.SQLFS, migrations *[]*migrate.Migration) error {

	files, err := fs.ReadDir("sql")
	if err != nil {
		return err
	}

	for _, info := range files {

		if strings.HasSuffix(info.Name(), ".sql") {
			file, err := fs.Open(info.Name())
			if err != nil {
				return fmt.Errorf("error while opening %s: %s", info.Name(), err)
			}

			migration, err := migrate.ParseMigration(info.Name(), file.(io.ReadSeeker))
			if err != nil {
				return fmt.Errorf("error while parsing %s: %s", info.Name(), err)
			}
			*migrations = append(*migrations, migration)

		}
	}

	return nil
}