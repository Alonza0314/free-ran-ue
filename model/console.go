package model

import "time"

type ConsoleConfig struct {
	Console ConsoleIE `yaml:"console"`
	Logger  LoggerIE  `yaml:"logger"`
}

type ConsoleIE struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	Port int `yaml:"port"`

	JWT JWTIE `yaml:"jwt"`
}

type JWTIE struct {
	Secret    string        `yaml:"secret"`
	ExpiresIn time.Duration `yaml:"expiresIn"`
}
