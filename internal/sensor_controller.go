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

const (
	epsilon             = 1e-10
	sensorControlPrefix = "otbs-sensor-"
	sensorControlSuffix = "/sensors/"
)

type SensorController struct {
	name        string
	lock        sync.RWMutex
	cfg         *config.SensorConfig
	mqtt        safe_mqtt.MqttClient
	queries     *db.Queries
	value       float64
	timestamp   time.Time
	controlChan chan<- bool
}

func NewSensorController(
	_name string, _cfg *config.SensorConfig, _mqttCfg *config.MQTTConfig, _q *db.Queries, _controlChan chan<- bool,
) *SensorController {
	s := &SensorController{
		name:        _name,
		cfg:         _cfg,
		queries:     _q,
		timestamp:   zeroTS,
		controlChan: _controlChan,
	}

	if s.readState() {
		logger.L().Debugf("Loaded previous state from DB for sensor %v: %v", s.name, s.value)
		s.timestamp = time.Now()
	}

	s.mqtt = safe_mqtt.InitMQTTClient(_mqttCfg.URL, sensorControlPrefix+s.name+"-"+uuid.New().String())
	s.mqtt.SafeSubscribe(_cfg.Topic, mqttQoS, s.ValueUpdateHandler)
	zoneMQTTgroup := _mqttCfg.ControlTopic + sensorControlSuffix + s.name + "/"
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"offset", mqttQoS, s.controlUpdateHandler)
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"weight", mqttQoS, s.controlUpdateHandler)
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"scale", mqttQoS, s.controlUpdateHandler)

	return s
}

func (s *SensorController) ValueUpdateHandler(client mqtt.Client, message mqtt.Message) {
	t0, err := extractF64PlainOrJson(message, s.cfg.JSONEntry)
	if err != nil {
		logger.L().Error(err)
		return
	}
	s.lock.Lock()
	oldValue := s.value
	s.value = t0*(*s.cfg.Scale) + (*s.cfg.Offset)
	s.lock.Unlock()
	if err := s.writeState(); err != nil {
		logger.L().Error(err)
	}
	logger.L().Debugf("Got value for sensor %s : %f", s.name, s.value)
	if oldValue != s.value {
		s.controlChan <- true
	}
}

func (s *SensorController) writeState() error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.queries.UpsertSensorValue(
		context.Background(), db.UpsertSensorValueParams{SensorName: s.name, Value: s.value},
	)
}

func (s *SensorController) readState() bool {
	val, err := s.queries.GetSensorValue(context.Background(), s.name)
	if err != nil {
		return false
	}
	s.value = val
	return true
}

func (s *SensorController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	logger.L().Infof("Sensor %v got MQTT control request: %v : %v", s.name, topic, string(message.Payload()))

	value, err := strconv.ParseFloat(string(message.Payload()), 64)
	if err != nil {
		logger.L().Error(err)
		return
	}

	switch topic {
	case "weight":
		s.cfg.Weight = &value
	case "offset":
		s.cfg.Offset = &value
	case "scale":
		s.cfg.Scale = &value
	default:
		logger.L().Errorf("Unknown control topic: %s", topic)
		return
	}

	logger.L().Infof("Updated %s for sensor `%v` to %v", topic, s.name, value)
}

func sensorsMean(sensors []*SensorController) (float64, time.Time) {
	var v, wt float64

	for _, sensor := range sensors {
		sensor.lock.RLock()
		if sensor.timestamp.After(zeroTS) {
			weight := *sensor.cfg.Weight
			v += sensor.value * weight
			wt += weight
		}
		sensor.lock.RUnlock()
	}

	if wt < epsilon {
		return 0, zeroTS
	}

	return v / wt, time.Now()
}
