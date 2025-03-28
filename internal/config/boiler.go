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

import "time"

type BoilerConfig struct {
	TSetTopic      string        `yaml:"tset_topic"`
	CHEnableTopic  string        `yaml:"ch_enable_topic,omitempty"`
	UpdateInterval time.Duration `yaml:"update_interval"`
}

func NewBoilerConfig() *BoilerConfig {
	cfg := &BoilerConfig{}
	cfg.TSetTopic = "test_OTGW/set/otgw/tset"
	cfg.CHEnableTopic = "test_OTGW/set/otgw/ch_enable"
	return cfg
}
