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

// OutsideConfig represents the configuration for outside sensors
type OutsideConfig struct {
	TemperatureSensors     []*SensorConfig `yaml:"temperature_sensors"`
	TemperatureAverageType string          `yaml:"temperature_average_type"`
	WindSpeedSensors       []*SensorConfig `yaml:"wind_speed_sensors,omitempty"`
	WindSpeedAverageType   string          `yaml:"wind_speed_average_type,omitempty"`
	HumiditySensors        []*SensorConfig `yaml:"humidity_sensors,omitempty"`
	HumidityAverageType    string          `yaml:"humidity_average_type,omitempty"`
}

// NewOutsideConfig creates a new OutsideConfig with default values
func NewOutsideConfig() *OutsideConfig {
	cfg := &OutsideConfig{}
	cfg.FillDefaults()
	return cfg
}

// FillDefaults sets default values for the OutsideConfig
func (c *OutsideConfig) FillDefaults() {
	for _, s := range c.TemperatureSensors {
		s.FillDefaults()
	}
	if c.TemperatureAverageType == "" {
		c.TemperatureAverageType = DefaultAverageType
	}
	fillSensorDefaults(c.HumiditySensors, &c.HumidityAverageType)
	fillSensorDefaults(c.WindSpeedSensors, &c.WindSpeedAverageType)
}

func fillSensorDefaults(sensors []*SensorConfig, avgType *string) {
	if len(sensors) > 0 {
		for _, s := range sensors {
			s.FillDefaults()
		}
		if *avgType == "" {
			*avgType = DefaultAverageType
		}
	}
}
