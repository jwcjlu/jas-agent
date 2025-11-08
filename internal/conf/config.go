package conf

type Bootstrap struct {
	Server *Server `yaml:"server" json:"server" mapstructure:"server"`
	Data   *Data   `yaml:"data" json:"data" mapstructure:"data"`
	LLM    *LLM    `yaml:"llm" json:"llm" mapstructure:"llm"`
}

type Server struct {
	HTTP *ServerHTTP `yaml:"http" json:"http" mapstructure:"http"`
	GRPC *ServerGRPC `yaml:"grpc" json:"grpc" mapstructure:"grpc"`
}

type ServerHTTP struct {
	Addr string `yaml:"addr" json:"addr" mapstructure:"addr"`
}

type ServerGRPC struct {
	Addr string `yaml:"addr" json:"addr" mapstructure:"addr"`
}

type Data struct {
	Database *Database `yaml:"database" json:"database" mapstructure:"database"`
}

type Database struct {
	Driver          string `yaml:"driver" json:"driver" mapstructure:"driver"`
	Source          string `yaml:"source" json:"source" mapstructure:"source"`
	MaxIdleConns    int32  `yaml:"max_idle_conns" json:"max_idle_conns" mapstructure:"max_idle_conns"`
	MaxOpenConns    int32  `yaml:"max_open_conns" json:"max_open_conns" mapstructure:"max_open_conns"`
	ConnMaxLifetime int32  `yaml:"conn_max_lifetime" json:"conn_max_lifetime" mapstructure:"conn_max_lifetime"` // seconds
}

type LLM struct {
	APIKey  string `yaml:"api_key" json:"api_key" mapstructure:"api_key"`
	BaseURL string `yaml:"base_url" json:"base_url" mapstructure:"base_url"`
	Model   string `yaml:"model" json:"model" mapstructure:"model"`
}
