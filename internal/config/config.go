/*
 * Copyright (c) 2023. Anton Starikov -- All Rights Reserved
 *
 * This file is part of MZOTBC project.
 *
 * MZOTBC is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as the Free Software Foundation,
 * either version 3 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package config

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/antst/mzotbc/internal/logger"

	"github.com/pborman/getopt/v2"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

const (
	defaultMQTTURL          = "tcp://127.0.0.1:1883"
	defaultControlTopic     = "mzotbc/control"
	defaultDBFile           = "~/.mzotbc.db"
	defaultConfigFile       = "config.yaml"
	DefaultAverageType      = "mean"
	zoneDefaultWeight       = 1.0
	zoneDefaultCompensation = 1.5
)

var defaultHeatingParam = 15.0

type HeatDemandConfig struct {
	Topic     string   `yaml:"topic"`
	JSONEntry *string  `yaml:"json_entry,omitempty"`
	Weight    *float64 `yaml:"weight"`
	Scale     *float64 `yaml:"scale"`
}

type Config struct {
	LogLevel                zapcore.Level          `yaml:"log_level"`
	MQTTConfig              *MQTTConfig            `yaml:"mqtt"`
	HAIntegration           bool                   `yaml:"ha_integration"`
	DefaultHeatingParameter *float64               `yaml:"default_heating_parameter"`
	DBFile                  string                 `yaml:"db_file"`
	Boiler                  *BoilerConfig          `yaml:"boiler"`
	Outside                 *OutsideConfig         `yaml:"outside"`
	Zones                   map[string]*ZoneConfig `yaml:"zones"`
}

func defConfig() *Config {
	return &Config{
		Zones:                   make(map[string]*ZoneConfig),
		Boiler:                  NewBoilerConfig(),
		Outside:                 NewOutsideConfig(),
		MQTTConfig:              NewMQTTConfig(),
		DefaultHeatingParameter: &defaultHeatingParam,
		HAIntegration:           true,
		DBFile:                  defaultDBFile,
	}
}

func prettyPrint(cfg *Config) {
	d, err := yaml.Marshal(cfg)
	if err != nil {
		logger.L().Error("Failed to marshal config for pretty print", err)
		return
	}
	logger.L().Debugf("--- Config ---\n%s\n\n", string(d))
}

func (cfg *Config) FillDefaults() {
	for _, v := range cfg.Zones {
		v.FillDefaults()
	}
	cfg.Outside.FillDefaults()

	if cfg.DefaultHeatingParameter == nil {
		cfg.DefaultHeatingParameter = &defaultHeatingParam
	}
}

func Get() *Config {
	cfg := defConfig()
	logLevel := getopt.StringLong("log-level", 'l', "", "log levels: debug, info, warn, error, dpanic, panic, fatal")
	configFile := getopt.StringLong("config", 'c', defaultConfigFile, "config file pathname")

	getopt.Parse()

	if err := readFile(cfg, *configFile); err != nil {
		log.Panicf("GetConfig: %v", err)
	}

	logger.L().Infof("Using config file `%v`", *configFile)
	dbFile := getopt.StringLong("db", 'd', cfg.DBFile, "DB file pathname")
	logger.L().Infof("Using DB file `%v`", cfg.DBFile)

	if *dbFile != "" {
		cfg.DBFile = *dbFile
	}
	logger.L().Infof("Using DB file `%v`", cfg.DBFile)

	cfg.FillDefaults()

	if *logLevel != "" {
		if err := cfg.LogLevel.Set(*logLevel); err != nil {
			logger.L().Errorf("Wrong log level `%v`: %v", *logLevel, err)
		}
	}
	logger.SetLogLevel(cfg.LogLevel)

	prettyPrint(cfg)

	return cfg
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func readFile(cfg *Config, configFileName string) error {
	if !fileExists(configFileName) {
		return nil
	}

	f, err := os.Open(configFileName)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) > 0 {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	return nil
}
