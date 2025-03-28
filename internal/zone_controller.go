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
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/antst/mzotbc/internal/config"
	"github.com/antst/mzotbc/internal/logger"
	"github.com/antst/mzotbc/internal/safe_mqtt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	"github.com/antst/mzotbc/internal/db"
)

type ZoneController struct {
	name               string
	mu                 sync.RWMutex
	cfg                *config.ZoneConfig
	mqtt               safe_mqtt.MqttClient
	sensors            []*SensorController
	queries            *db.Queries
	setpoint           float64
	setpointTimestamp  time.Time
	averageTemperature float64
	averageTimestamp   time.Time
	tSet               float64
	tSetTimestamp      time.Time
	averageFunc        func([]*SensorController) (float64, time.Time)
	controlChan        chan<- *ZoneController
	childChan          chan bool
}

func (z *ZoneController) getPair() (float64, float64, bool) {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.setpoint, z.averageTemperature, z.setpointTimestamp.After(zeroTS) && z.averageTimestamp.After(zeroTS)
}

func (z *ZoneController) childProcessor() {
	for range z.childChan {
		z.updateAverage()
	}
}

func (z *ZoneController) LinkAverageFun() {
	if z.cfg.SensorsAverageType == config.DefaultAverageType {
		z.averageFunc = sensorsMean
	} else {
		logger.L().Errorf("Unknown average function type: %v", z.cfg.SensorsAverageType)
		logger.L().Error("Reverting to the `mean`")
		z.cfg.SensorsAverageType = config.DefaultAverageType
		z.averageFunc = sensorsMean
	}
}

func newZoneController(
	_name string, _cfg *config.ZoneConfig, _mqttCfg *config.MQTTConfig, _q *db.Queries,
	_controlChan chan<- *ZoneController,
) *ZoneController {
	z := &ZoneController{
		name:              _name,
		cfg:               _cfg,
		queries:           _q,
		setpointTimestamp: zeroTS,
		averageTimestamp:  zeroTS,
		controlChan:       _controlChan,
		childChan:         make(chan bool, childChanBuffer),
	}

	z.LinkAverageFun()
	if err := z.readState(); err == nil {
		logger.L().Debugf("Loaded previous state from DB for zone %v: %v", z.name, z.setpoint)
		z.setpointTimestamp = time.Now()
	}
	z.mqtt = safe_mqtt.InitMQTTClient(_mqttCfg.URL, "otbs-zone-"+z.name+"-"+uuid.New().String())

	z.mqtt.SafeSubscribe(_cfg.Setpoint.Topic, mqttQoS, z.setpointUpdateHandler)

	zoneMQTTgroup := _mqttCfg.ControlTopic + "/zone/" + z.name + "/"
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"sensors_average_type", mqttQoS, z.controlUpdateHandler)
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"weight", mqttQoS, z.controlUpdateHandler)
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"heating_parameter", mqttQoS, z.controlUpdateHandler)

	z.sensors = make([]*SensorController, len(z.cfg.Sensors))
	for i, sensor := range z.cfg.Sensors {
		sName := "zone-" + z.name + "-"
		if sensor.Name == "" {
			sName += strconv.Itoa(i + 1)
		} else {
			sName += sensor.Name
		}

		z.sensors[i] = NewSensorController(sName, sensor, _mqttCfg, z.queries, z.childChan)
	}
	go z.childProcessor()
	z.updateAverage()

	return z
}

func (z *ZoneController) updateAverage() {
	v, t := z.averageFunc(z.sensors)
	if t.After(zeroTS) {
		z.mu.Lock()
		z.averageTimestamp = t
		z.averageTemperature = v
		z.mu.Unlock()
		z.controlChan <- z
	}
}

func (z *ZoneController) setpointUpdateHandler(client mqtt.Client, message mqtt.Message) {
	t0, err := extractF64PlainOrJson(message, z.cfg.Setpoint.JSONEntry)
	if err != nil {
		logger.L().Error(err)
		return
	}

	z.mu.Lock()
	oldSP := z.setpoint
	z.setpoint = t0*(*z.cfg.Setpoint.Scale) + (*z.cfg.Setpoint.Offset)
	z.setpointTimestamp = time.Now()
	logger.L().Debugf("Got setpoint for zone %s : %f", z.name, z.setpoint)
	z.mu.Unlock()

	if err := z.writeState(); err != nil {
		logger.L().Error(err)
	}
	if t0 != oldSP {
		z.childChan <- true
	}
}

func (z *ZoneController) writeState() error {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return z.queries.UpsertZoneSetpoint(
		context.Background(), db.UpsertZoneSetpointParams{Setpoint: z.setpoint, ZoneName: z.name},
	)
}

func (z *ZoneController) readState() error {
	val, err := z.queries.GetZoneSetpoint(context.Background(), z.name)
	if err != nil {
		return err
	}
	z.setpoint = val
	return nil
}

func (z *ZoneController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	logger.L().Infof("Zone %v got MQTT control request: %v : %v", z.name, topic, string(message.Payload()))

	switch topic {
	case "weight", "heating_parameter":
		value, err := strconv.ParseFloat(string(message.Payload()), 64)
		if err != nil {
			logger.L().Error(err)
			return
		}
		if topic == "weight" {
			z.cfg.Weight = &value
		} else {
			z.cfg.HeatingParameter = &value
		}
		logger.L().Infof("Updated %s for zone `%v` to %v", topic, z.name, value)
	case "sensors_average_type":
		z.cfg.SensorsAverageType = string(message.Payload())
		z.LinkAverageFun()
		logger.L().Infof("Updated sensors average type to `%v`", z.cfg.SensorsAverageType)
	default:
		logger.L().Errorf("Unknown control topic: %s", topic)
	}
}
