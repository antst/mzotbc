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
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

func (s *SensorConfig) FillDefaults() SensorConfig {
	if s.Offset == nil {
		s.Offset = GetPTR(0.0)
	}
	if s.Scale == nil {
		s.Scale = GetPTR(1.0)
	}
	if s.Weight == nil {
		s.Weight = GetPTR(1.0)
	}
	return *s
}

type SensorController struct {
	name        string
	lock        sync.Mutex
	cfg         *SensorConfig
	mqtt        MqttClient
	db          *sqlx.DB
	value       float64
	timestamp   time.Time
	controlChan chan<- bool
}

func NewSensorController(
	_name string, _cfg *SensorConfig, _mqttCfg *MQTTConfig, _db *sqlx.DB, _controlChan chan<- bool,
) *SensorController {
	s := &SensorController{name: _name, cfg: _cfg, db: _db, timestamp: zeroTS, controlChan: _controlChan}

	if s.readState() {
		L().Debugf("Loaded previous state from DB for sensor %v: %v", s.name, s.value)
		s.timestamp = time.Now()
	}

	s.mqtt = InitMQTTClient(_mqttCfg.URL, "otbs-sensor-"+s.name+"-"+uuid.New().String())
	s.mqtt.SafeSubscribe(_cfg.Topic, 1, s.ValueUpdateHandler)
	zoneMQTTgroup := _mqttCfg.ControlTopic + "/sensors/" + s.name + "/"
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"offset", 1, s.controlUpdateHandler)
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"weight", 1, s.controlUpdateHandler)
	s.mqtt.SafeSubscribe(zoneMQTTgroup+"scale", 1, s.controlUpdateHandler)

	return s
}

func (s *SensorController) ValueUpdateHandler(client mqtt.Client, message mqtt.Message) {
	t0, err := extractF64PlainOrJson(client, message, s.cfg.JSONEntry)
	if checkError(err) {
		L().Error(err)
		return
	}
	s.lock.Lock()
	oldValue := s.value
	s.value = t0*(*s.cfg.Scale) + (*s.cfg.Offset)
	s.lock.Unlock()
	s.writeState()
	L().Debugf("Got value for sensor %s : %f", s.name, s.value)
	if oldValue != s.value {
		s.controlChan <- true
	}
}

func (s *SensorController) writeState() error {
	const QUERY = `
		INSERT INTO sensor(sensor_name,value,updated_at) 
  		VALUES($1,$2,$3)
  		ON CONFLICT(sensor_name) DO UPDATE SET
    	value=excluded.value,
    	updated_at=excluded.updated_at;`
	s.lock.Lock()
	_, err := s.db.Exec(QUERY, s.name, s.value, time.Now())
	s.lock.Unlock()
	return err
}

func (s *SensorController) readState() bool {
	const QUERY = `
		SELECT value from sensor where sensor_name=$1;`
	res, err := s.db.Query(QUERY, s.name)
	if checkError(err) || !res.Next() {
		return false
	}
	defer res.Close()
	return res.Scan(&s.value) == nil
}

func (s *SensorController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	L().Infof("Sensor %v got MQTT control request: %v : %v", s.name, topic, string(message.Payload()))
	switch topic {
	case "weight":
		{
			w, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			s.cfg.Weight = GetPTR(w)
			L().Infof("Uptaded weight for sensor `%v` to %v", s.name, w)
		}
	case "offset":
		{
			o, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			s.cfg.Offset = GetPTR(o)
			L().Infof("Uptaded offset for sensor `%v` to %v", s.name, o)
		}
	case "scale":
		{
			o, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			s.cfg.Scale = GetPTR(o)
			L().Infof("Uptaded scale for sensor `%v` to %v", s.name, o)
		}
	}
}

func sensorsMean(sensors []*SensorController) (float64, time.Time) {

	v := 0.0
	wt := 0.0
	for _, sensor := range sensors {
		sensor.lock.Lock()
		if sensor.timestamp.After(zeroTS) {
			v += sensor.value * *(sensor.cfg.Weight)
			wt += *(sensor.cfg.Weight)
		}
		sensor.lock.Unlock()
	}

	if wt < 1e-10 {
		return 0, zeroTS

	}

	return v / wt, time.Now()
}
