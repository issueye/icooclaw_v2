package adapter

// ChannelInfo 渠道信息结构体
type ChannelInfo struct {
	Type   string
	Config string
}

// StorageInterface 渠道配置存储器接口
type StorageInterface interface {
	List() ([]ChannelInfo, error)
}
