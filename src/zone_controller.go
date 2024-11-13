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
		z.SensorsAverageType = "mean"
	}
	if z.Weight == nil {
		z.Weight = GetPTR(1.0)
	}
	if z.RoomCompensation == nil {
		z.RoomCompensation = GetPTR(1.5)
	}

	z.Setpoint.FillDefaults()
	for _, s := range z.Sensors {
		s.FillDefaults()
	}
}

func NewZoneConfig() *ZoneConfig {
	cfg := &ZoneConfig{}
	cfg.Sensors = make([]*SensorConfig, 0)
	cfg.Setpoint = NewSetpointConfig()
	//cfg.SensorsAverageType = "mean"
	//cfg.Weight = GetPTR(1.0)
	return cfg
}

type ZoneController struct {
	name               string
	lock               sync.Mutex
	cfg                *ZoneConfig
	mqtt               MqttClient
	sensors            []*SensorController
	db                 *sqlx.DB
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
	z.lock.Lock()
	defer z.lock.Unlock()
	return z.setpoint, z.averageTemperature, z.setpointTimestamp.After(zeroTS) && z.averageTimestamp.After(zeroTS)
}

func (z *ZoneController) childProcessor() {
	for {
		select {
		case <-z.childChan:
			{
				for len(z.childChan) > 0 {
					<-z.childChan
				}
				z.updateAverage()
			}
		}
	}
}

func (z *ZoneController) LinkAverageFun() {
	switch z.cfg.SensorsAverageType {
	case "mean":
		z.averageFunc = sensorsMean
		break
	default:
		{
			L().Errorf("Unknown average function type :%v", z.cfg.SensorsAverageType)
			L().Error("Reverting to the `mean`")
			z.cfg.SensorsAverageType = "mean"
			z.LinkAverageFun()
		}
	}
}

func newZoneController(
	_name string, _cfg *ZoneConfig, _mqttCfg *MQTTConfig, _db *sqlx.DB, _controlChan chan<- *ZoneController,
) *ZoneController {
	z := &ZoneController{name: _name, cfg: _cfg, db: _db, setpointTimestamp: zeroTS, averageTimestamp: zeroTS, controlChan: _controlChan}
	z.childChan = make(chan bool, 10)

	z.LinkAverageFun()
	if z.readState() {
		L().Debugf("Loaded previous state from DB for zone %v: %v", z.name, z.setpoint)
		z.setpointTimestamp = time.Now()
	}
	z.mqtt = InitMQTTClient(_mqttCfg.URL, "otbs-zone-"+z.name+"-"+uuid.New().String())

	z.mqtt.SafeSubscribe(_cfg.Setpoint.Topic, 1, z.setpointUpdateHandler)

	zoneMQTTgroup := _mqttCfg.ControlTopic + "/zone/" + z.name + "/"
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"sensors_average_type", 1, z.controlUpdateHandler)
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"weight", 1, z.controlUpdateHandler)
	z.mqtt.SafeSubscribe(zoneMQTTgroup+"heating_parameter", 1, z.controlUpdateHandler)

	z.sensors = make([]*SensorController, len(z.cfg.Sensors))
	for i, sensor := range z.cfg.Sensors {
		sName := "zone-" + z.name + "-"
		if sensor.Name == "" {
			sName += strconv.Itoa(i + 1)
		} else {
			sName += sensor.Name
		}

		z.sensors[i] = NewSensorController(sName, sensor, _mqttCfg, z.db, z.childChan)
	}
	go z.childProcessor()
	z.updateAverage()

	return z
}

func (z *ZoneController) updateAverage() {
	v, t := z.averageFunc(z.sensors)
	if t.After(zeroTS) {
		z.lock.Lock()
		z.averageTimestamp = t
		z.averageTemperature = v
		z.lock.Unlock()
		z.controlChan <- z
	}
}

func (z *ZoneController) setpointUpdateHandler(client mqtt.Client, message mqtt.Message) {
	t0, err := extractF64PlainOrJson(client, message, z.cfg.Setpoint.JSONEntry)
	if checkError(err) {
		L().Error(err)
		return
	}

	z.lock.Lock()
	oldSP := z.setpoint
	z.setpoint = t0*(*z.cfg.Setpoint.Scale) + (*z.cfg.Setpoint.Offset)
	z.setpointTimestamp = time.Now()
	L().Debugf("Got setpoint for zone %s : %f", z.name, z.setpoint)
	z.lock.Unlock()

	if err := z.writeState(); checkError(err) {
		L().Error(err)
	}
	if t0 != oldSP {
		z.childChan <- true
	}

}

func (z *ZoneController) writeState() error {

	const QUERY = `
		INSERT INTO zone(zone_name,setpoint,updated_at) 
  		VALUES($1,$2,$3)
  		ON CONFLICT(zone_name) DO UPDATE SET
    	setpoint=excluded.setpoint,
    	updated_at=excluded.updated_at;`
	z.lock.Lock()
	_, err := z.db.Exec(QUERY, z.name, z.setpoint, time.Now())
	z.lock.Unlock()

	return err
}

func (z *ZoneController) readState() bool {
	const QUERY = `
		SELECT setpoint from zone where zone_name=$1;`
	res, err := z.db.Query(QUERY, z.name)
	if checkError(err) || !res.Next() {
		return false
	}
	defer res.Close()
	//z.lock.Lock()
	//defer z.lock.Unlock()
	return res.Scan(&z.setpoint) == nil
}

func (z *ZoneController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	L().Infof("Zone %v got MQTT control request: %v : %v", z.name, topic, string(message.Payload()))
	switch topic {
	case "weight":
		{
			w, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			z.cfg.Weight = GetPTR(w)
			L().Infof("Uptaded weight for zone `%v` to %v", z.name, w)
		}
	case "heating_parameter":
		{
			hp, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			z.cfg.HeatingParameter = GetPTR(hp)
			L().Infof("Uptaded heating parameter for zone `%v` to %v", z.name, hp)
		}
	case "sensors_average_type":
		{
			aType := string(message.Payload())
			z.cfg.SensorsAverageType = aType
			z.LinkAverageFun()
			L().Infof("Uptaded loglevel to `%v`", z.cfg.SensorsAverageType)

		}
	}
}
