package app_engine

//Server ...
type Server interface {
	Start() error  //启动服务
	GracefulStop() //优雅退出
}
