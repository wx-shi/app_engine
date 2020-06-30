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
func NewEngine() *Engine {
	return &Engine{
		log:       zap.NewExample(),
		exit:      make(chan bool),
		servers:   make([]Server, 0),
		loadFns:   make([]func() error, 0),
		deferFns:  make([]func(chan bool) error, 0),
		cancelFns: make([]func(), 0),
	}
}

//CancelFunc 注入关闭方法
func (e *Engine) CancelFunc(fns ...func()) {
	e.cancelFns = append(e.cancelFns, fns...)
}

//Logger 设置日志
func (e *Engine) Logger(log *zap.Logger) {
	e.log = log
}

//LoadFunc 注入初始化方法
func (e *Engine) LoadFunc(fns ...func() error) {
	e.loadFns = append(e.loadFns, fns...)
}

//DeferFunc 注入初始化完成之后执行的方法
func (e *Engine) DeferFunc(fns ...func(chan bool) error) {
	e.deferFns = append(e.deferFns, fns...)
}

//Startup 注入服务
func (e *Engine) Server(fns ...func() (Server, error)) error {
	for _, fn := range fns {
		s, err := fn()
		if err != nil {
			return err
		}
		e.servers = append(e.servers, s)
	}

	return nil
}

//Run 运行
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
