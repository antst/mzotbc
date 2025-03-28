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

func (s *SensorConfig) FillDefaults() {
	if s.Offset == nil {
		s.Offset = GetPTR(0.0)
	}
	if s.Scale == nil {
		s.Scale = GetPTR(1.0)
	}
	if s.Weight == nil {
		s.Weight = GetPTR(1.0)
	}
}

// SensorConfig: Configuration
type SensorConfig struct {
	Name      string   `yaml:"Name,omitempty"`
	Topic     string   `yaml:"topic"`
	JSONEntry *string  `yaml:"json_entry,omitempty"`
	Offset    *float64 `yaml:"offset"`
	Scale     *float64 `yaml:"scale"`
	Weight    *float64 `yaml:"weight"`
}

func NewSensorConfig() *SensorConfig {
	cfg := &SensorConfig{}
	cfg.FillDefaults()
	return cfg
}

type sensorGroupConfig struct {
	AverageFuncType string          `yaml:"average_type,omitempty"`
	items           []*SensorConfig `yaml:"items"`
}
