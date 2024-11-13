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

package main

import (
	"fmt"
	"github.com/pborman/getopt/v2"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
)

type HeatDemandConfig struct {
	Topic     string   `yaml:"topic"`
	JSONEntry *string  `yaml:"json_entry,omitempty"`
	Weight    *float64 `yaml:"weight"`
	Scale     *float64 `yaml:"scale"`
}

func NewMQTTConfig() *MQTTConfig {
	cfg := &MQTTConfig{}
	cfg.URL = "tcp://127.0.0.1:1883"
	cfg.ControlTopic = "mzotbc/control"
	return cfg
}

type MQTTConfig struct {
	URL          string `yaml:"url"`
	ControlTopic string `yaml:"control_topic"`
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
	cfg := new(Config)
	cfg.Zones = make(map[string]*ZoneConfig)
	cfg.Boiler = NewBoilerConfig()
	cfg.Outside = NewOutsideConfig()
	cfg.MQTTConfig = NewMQTTConfig()
	cfg.DefaultHeatingParameter = GetPTR(15.0)
	cfg.HAIntegration = true
	cfg.DBFile = "~./mzotbc.db"
	return cfg
}

func prettyPrint(cfg *Config) {
	d, err := yaml.Marshal(cfg)
	checkError(err)
	L().Debugf("--- Config ---\n%s\n\n", string(d))
}

func (cfg *Config) FillDefaults() {
	for _, v := range cfg.Zones {
		v.FillDefaults()
	}
	cfg.Outside.FillDefaults()

	if cfg.DefaultHeatingParameter == nil {
		cfg.DefaultHeatingParameter = GetPTR(15.0)
	}

}
func getConfig() (*Config, error) {
	cfg := defConfig()
	logLevel := getopt.String(
		'l', "", "log levels:\"debug\", \"info\", \"warn\", \"error\", \"dpanic\", \"panic\", and \"fatal\"",
	)
	db_file := getopt.String('d', cfg.DBFile, "DB file pathname")
	cfg_file := getopt.String('c', "config.yaml", "config file pathname")

	parseOpts()

	if err := readFile(cfg, *cfg_file); err != nil {
		log.Panicf("getConfig: %w", err)
	}
	if db_file != nil && *db_file != "" {
		cfg.DBFile = *db_file
	} else {
		L().Errorf("Wrong DB pathname `%v`", db_file)
	}

	cfg.FillDefaults()
	if *logLevel != "" {
		err := cfg.LogLevel.Set(*logLevel)
		if checkError(err) {
			L().Errorf("Wrong log level `%v`", *logLevel)
		}
	}
	setLogLevel(zapcore.Level(cfg.LogLevel))

	prettyPrint(cfg)

	return cfg, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
func parseOpts() {
	helpFlag := false
	getopt.Flag(&helpFlag, 'h', "display help")
	getopt.Parse()
	if helpFlag {
		getopt.Usage()
		os.Exit(0)
	}
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
	if err != nil {
		if err != io.EOF {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		return nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to unmarhal config: %w", err)
	}

	return nil
}
