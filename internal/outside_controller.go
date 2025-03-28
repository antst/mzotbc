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

package internal

import (
	"strconv"
	"sync"
	"time"

	"github.com/antst/mzotbc/internal/config"
	"github.com/antst/mzotbc/internal/logger"
	"github.com/antst/mzotbc/internal/safe_mqtt"

	"github.com/google/uuid"

	"github.com/antst/mzotbc/internal/db"
)

const (
	outsidePrefix     = "outside-temperature-"
	mqttOutsidePrefix = "otbs-outside-"
)

// OutsideController manages outside sensors and their data
type OutsideController struct {
	mu                          sync.RWMutex
	cfg                         *config.OutsideConfig
	mqtt                        safe_mqtt.MqttClient
	queries                     *db.Queries
	temperatureSensors          []*SensorController
	humiditySensors             []*SensorController
	windSpeedSensors            []*SensorController
	controlChan                 chan<- float64
	childChan                   chan bool
	averageTemperature          float64
	averageTemperatureTimestamp time.Time
	averageTemperatureFunc      func([]*SensorController) (float64, time.Time)
}

func (o *OutsideController) childProcessor() {
	for range o.childChan {
		o.updateTemperatureAverage()
	}
}

func (o *OutsideController) updateTemperatureAverage() {
	v, t := o.averageTemperatureFunc(o.temperatureSensors)
	if t.After(zeroTS) {
		o.mu.Lock()
		o.averageTemperatureTimestamp = t
		o.averageTemperature = v
		o.mu.Unlock()
		o.controlChan <- v
	}
}

func (o *OutsideController) LinkAverageFun() {
	if o.cfg.TemperatureAverageType == "mean" {
		o.averageTemperatureFunc = sensorsMean
	} else {
		logger.L().Errorf("Unknown average function type: %v", o.cfg.TemperatureAverageType)
		logger.L().Error("Reverting to the `mean`")
		o.cfg.TemperatureAverageType = config.DefaultAverageType
		o.averageTemperatureFunc = sensorsMean
	}
}

func NewOutsideController(
	_cfg *config.OutsideConfig, _mqttCfg *config.MQTTConfig, _q *db.Queries, _controlChan chan<- float64,
) *OutsideController {
	o := &OutsideController{
		cfg:                         _cfg,
		queries:                     _q,
		controlChan:                 _controlChan,
		averageTemperatureTimestamp: zeroTS,
		childChan:                   make(chan bool, childChanBuffer),
	}
	o.LinkAverageFun()

	o.temperatureSensors = make([]*SensorController, len(o.cfg.TemperatureSensors))
	for i, sensor := range o.cfg.TemperatureSensors {
		sName := outsidePrefix
		if sensor.Name == "" {
			sName += strconv.Itoa(i + 1)
		} else {
			sName += sensor.Name
		}
		o.temperatureSensors[i] = NewSensorController(sName, sensor, _mqttCfg, o.queries, o.childChan)
	}

	go o.childProcessor()
	o.updateTemperatureAverage()
	o.mqtt = safe_mqtt.InitMQTTClient(_mqttCfg.URL, mqttOutsidePrefix+uuid.New().String())
	return o
}
