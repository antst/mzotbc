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

// SetpointConfig config
type SetpointConfig struct {
	Topic     string   `yaml:"topic"`
	JSONEntry *string  `yaml:"json_entry,omitempty"`
	Offset    *float64 `yaml:"offset"`
	Scale     *float64 `yaml:"scale"`
}

func NewSetpointConfig() *SetpointConfig {
	cfg := &SetpointConfig{}
	cfg.FillDefaults()
	return cfg
}

func (c *SetpointConfig) FillDefaults() SetpointConfig {
	if c.Offset == nil {
		c.Offset = GetPTR(0.0)
	}
	if c.Scale == nil {
		c.Scale = GetPTR(1.0)
	}
	return *c
}
