package config

// 节点自身[静态配置]
type ServerConfig struct {
	ClusterName    string // 当前节点名
	IsRecoverModel bool   // 是否recover panic
	LogFilename    string // log文件, 空字符串写入stderr
	LogLevel       int    // 详见log/defines.go
	MeshConfig     string // 节点间路由表配置文件[动态配置]
}

var (
	ServerConf  ServerConfig
	ClusterConf map[string]string
)
