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
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strconv"
	"sync"
	"time"
)

// Outside config
type OutsideConfig struct {
	TemperatureSensors     []*SensorConfig `yaml:"temperature_sensors"`
	TemperatureAverageType string          `yaml:"temperature_average_type"`
	WindSpeedSensors       []*SensorConfig `yaml:"wind_speed_sensors,omitempty"`
	WindSpeedAverageType   string          `yaml:"wind_speed_average_type,omitempty"`
	HumiditySensors        []*SensorConfig `yaml:"humidity_sensors,omitempty"`
	HumidityAverageType    string          `yaml:"humidity_average_type,omitempty"`
}

type OutsideController struct {
	lock                        sync.Mutex
	cfg                         *OutsideConfig
	mqtt                        MqttClient
	db                          *sqlx.DB
	temperatureSensors          []*SensorController
	humiditySensors             []*SensorController
	windSpeedSensors            []*SensorController
	controlChan                 chan<- float64
	childChan                   chan bool
	averageTemperature          float64
	averageTemperatureTimestamp time.Time
	averageTemperatureFunc      func([]*SensorController) (float64, time.Time)
}

func NewOutsideConfig() *OutsideConfig {
	cfg := &OutsideConfig{}
	cfg.FillDefaults()
	return cfg
}

func (c *OutsideConfig) FillDefaults() OutsideConfig {
	for _, s := range c.TemperatureSensors {
		s.FillDefaults()
	}
	if c.TemperatureAverageType == "" {
		c.TemperatureAverageType = "mean"
	}
	if len(c.HumiditySensors) > 0 {
		for _, s := range c.HumiditySensors {
			s.FillDefaults()
		}
		if c.HumidityAverageType == "" {
			c.HumidityAverageType = "mean"
		}
	}
	if len(c.WindSpeedSensors) > 0 {
		for _, s := range c.WindSpeedSensors {
			s.FillDefaults()
		}
		if c.WindSpeedAverageType == "" {
			c.WindSpeedAverageType = "mean"
		}
	}
	return *c
}

func (o *OutsideController) childProcessor() {
	for {
		select {
		case <-o.childChan:
			for len(o.childChan) > 0 {
				<-o.childChan
			}
			o.updateTemperatureAverage()
		}
	}
}

func (o *OutsideController) updateTemperatureAverage() {
	v, t := o.averageTemperatureFunc(o.temperatureSensors)
	if t.After(zeroTS) {
		o.lock.Lock()
		o.averageTemperatureTimestamp = t
		o.averageTemperature = v
		o.lock.Unlock()
		o.controlChan <- v
	}
}

func (o *OutsideController) LinkAverageFun() {
	switch o.cfg.TemperatureAverageType {
	case "mean":
		o.averageTemperatureFunc = sensorsMean
		break
	default:
		{
			L().Errorf("Unknown average function type :%v", o.cfg.TemperatureAverageType)
			L().Error("Reverting to the `mean`")
			o.cfg.TemperatureAverageType = "mean"
			o.LinkAverageFun()
		}
	}
}

func NewOutsideController(
	_cfg *OutsideConfig, _mqttCfg *MQTTConfig, _db *sqlx.DB, _controlChan chan<- float64,
) *OutsideController {
	o := &OutsideController{cfg: _cfg, db: _db, controlChan: _controlChan, averageTemperatureTimestamp: zeroTS}
	o.childChan = make(chan bool, 10)
	o.LinkAverageFun()

	o.temperatureSensors = make([]*SensorController, len(o.cfg.TemperatureSensors))
	for i, sensor := range o.cfg.TemperatureSensors {
		sName := "outside-temperature-"

		if sensor.Name == "" {
			sName += strconv.Itoa(i + 1)
		} else {
			sName += sensor.Name
		}
		o.temperatureSensors[i] = NewSensorController(sName, sensor, _mqttCfg, o.db, o.childChan)
	}
	go o.childProcessor()
	o.updateTemperatureAverage()
	o.mqtt = InitMQTTClient(_mqttCfg.URL, "otbs-outside-"+uuid.New().String())
	return o
}
