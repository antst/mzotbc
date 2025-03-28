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

type ZoneConfig struct {
	HeatingParameter   *float64            `yaml:"heating_parameter,omitempty"`
	RoomCompensation   *float64            `yaml:"room_compensation"`
	SensorsAverageType string              `yaml:"sensors_average_type"`
	Weight             *float64            `yaml:"weight"`
	Setpoint           *SetpointConfig     `yaml:"setpoint"`
	Sensors            []*SensorConfig     `yaml:"sensors"`
	HeatDemand         []*HeatDemandConfig `yaml:"heat_demand,omitempty"`
	Valves             []*ValveConfig      `yaml:"valves"`
}

func (z *ZoneConfig) FillDefaults() {
	if z.SensorsAverageType == "" {
		z.SensorsAverageType = DefaultAverageType
	}
	if z.Weight == nil {
		z.Weight = GetPTR(zoneDefaultWeight)
	}
	if z.RoomCompensation == nil {
		z.RoomCompensation = GetPTR(zoneDefaultCompensation)
	}

	z.Setpoint.FillDefaults()
	for _, s := range z.Sensors {
		s.FillDefaults()
	}
}

func NewZoneConfig() *ZoneConfig {
	return &ZoneConfig{
		Sensors:  make([]*SensorConfig, 0),
		Setpoint: NewSetpointConfig(),
	}
}
