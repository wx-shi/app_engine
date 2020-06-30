package app_engine

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

//Engine ...
type Engine struct {
	log       *zap.Logger
	servers   []Server                //服务
	loadFns   []func() error          //初始化加载方法
	deferFns  []func(chan bool) error //初始化完成之后执行的方法
	cancelFns []func()                //退出方法
	exit      chan bool               //退出管道
}

//NewEngine ...
func NewEngine(ops ...Option) *Engine {
	eng := defaultEngine()

	for _, op := range ops {
		op(eng)
	}

	return eng
}

func defaultEngine() *Engine {
	return &Engine{
		log:       zap.NewExample(),
		exit:      make(chan bool),
		servers:   make([]Server, 0),
		loadFns:   make([]func() error, 0),
		deferFns:  make([]func(chan bool) error, 0),
		cancelFns: make([]func(), 0),
	}
}

// Option ...
type Option func(e *Engine)

// WithLog 注入log.
func WithLog(log *zap.Logger) Option {
	return func(e *Engine) {
		e.log = log
	}
}

// WithCancelFunc 注入程序结束执行方法.
func WithCancelFunc(fns ...func()) Option {
	return func(e *Engine) {
		e.cancelFns = append(e.cancelFns, fns...)
	}
}

// WithLoadFunc 注入初始化加载方法.
func WithLoadFunc(fns ...func() error) Option {
	return func(e *Engine) {
		e.loadFns = append(e.loadFns, fns...)
	}
}

// WithDeferFunc 注入初始化完成后执行的方法.
func WithDeferFunc(fns ...func(chan bool) error) Option {
	return func(e *Engine) {
		e.deferFns = append(e.deferFns, fns...)
	}
}

// WithServer 注入server.
func WithServer(ss ...Server) Option {
	return func(e *Engine) {
		e.servers = append(e.servers, ss...)
	}
}

//Run 运行.
func (e *Engine) Run() error {
	for _, f := range e.loadFns {
		if err := f(); err != nil {
			return err
		}
	}

	for _, f := range e.deferFns {
		if err := f(e.exit); err != nil {
			return err
		}
	}

	for _, s := range e.servers {
		if err := s.Start(); err != nil {
			return err
		}
	}

	e.log.Debug("application run success")

	e.wait()
	return nil
}

//wait 等待退出命令
func (e *Engine) wait() {
	e.hookSignals()
}

//Stop 停止
func (e *Engine) stop() {
	for _, s := range e.servers {
		s.GracefulStop()
	}

	for _, cancelFunc := range e.cancelFns {
		cancelFunc()
	}
}

//退出信号
func (e *Engine) hookSignals() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	for sig := range sigChan {

		e.log.Debug("get a signal", zap.String("signal", sig.String()))
		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			e.stop()
			close(e.exit)
			time.Sleep(time.Second * 2)
			return
		case syscall.SIGHUP:
			//app.Stop()
		default:
			return
		}
	}
}
