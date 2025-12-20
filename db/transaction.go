package db

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

var (
	ctxKeySession = "__db_session__"
)

var ErrNoTransaction = errors.New("no transaction found")

// TxReadOnly 只读事务
var TxReadOnly = &sql.TxOptions{
	Isolation: sql.LevelDefault, // 默认
	ReadOnly:  true,
}

// Transaction 开启事务
func Transaction[T any](ctx context.Context, f func(ctx context.Context) (T, error), options ...*sql.TxOptions) (result T, err error) {
	tx := Get().Begin(options...)

	if tx.Error != nil {
		return result, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}

		if err != nil {
			// 回滚事务
			if rollBackErr := tx.Rollback().Error; rollBackErr != nil {
				err = errors.Join(err, rollBackErr)
			}
			return
		} else {
			// 提交事务
			err = tx.Commit().Error
		}
	}()

	result, err = f(context.WithValue(ctx, ctxKeySession, tx))
	return
}

// TransactionWithoutResult 开启事务
func TransactionWithoutResult(ctx context.Context, f func(ctx context.Context) error, options ...*sql.TxOptions) (err error) {
	_, err = Transaction[any](ctx, func(ctx context.Context) (any, error) {
		return nil, f(ctx)
	}, options...)
	return
}

// Tx 获取当前上下文中的事务
func Tx(ctx context.Context) (*gorm.DB, error) {
	val := ctx.Value(ctxKeySession)
	if val == nil {
		return nil, ErrNoTransaction
	}
	session, ok := val.(*gorm.DB)
	if !ok {
		return nil, ErrNoTransaction
	}
	return session, nil
}
