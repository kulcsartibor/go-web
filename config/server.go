package config

type ServerConfig struct {
	BindAddress string
	ApiExposeUrl string
	ApiExposePort int
	Port int
	ReadTimeout int
	WriteTimeout int
	DocumentManagementService bool
}
