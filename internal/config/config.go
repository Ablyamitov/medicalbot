package config

import "github.com/ilyakaznacheev/cleanenv"

type (
	Config struct {
		DB DBConfig `yaml:"db" env-prefix:"DB_" env-required:"true"`
	}

	DBConfig struct {
		Host    string `yaml:"host" env:"HOST" env-required:"true"`
		Port    int64  `yaml:"port" env:"PORT" env-required:"true"`
		Name    string `yaml:"name" env:"NAME" env-required:"true"`
		User    string `yaml:"user" env:"USER" env-required:"true"`
		Pass    string `yaml:"pass" env:"PASS" env-required:"true"`
		SslMode bool   `yaml:"ssl_mode" env:"SSL_MODE" env-default:"false"`
	}
)

func MustLoad() *Config {
	var cfg Config
	err := cleanenv.ReadEnv(&cfg)

	if err != nil {
		panic(err)
	}
	return &cfg
}
