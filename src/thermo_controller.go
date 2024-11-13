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
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"math"
	"strconv"
	"strings"
	"time"
)

type ThermoController struct {
	cfg         *Config
	db          *sqlx.DB
	mqtt        MqttClient
	zones       map[string]*ZoneController
	outside     *OutsideController
	boiler      *BoilerController
	outsideChan chan float64         // chan to receive temperature updates from OutsideController
	zoneChan    chan *ZoneController // chan to receive notifications about change in ZoneControllers
	updateMap   map[*ZoneController]bool
	zoneTRs     map[*ZoneController]float64
}

func NewThermoController() *ThermoController {
	c := &ThermoController{}
	// read config from file/command line
	c.cfg, _ = getConfig()

	// initialize MQTT client
	c.mqtt = InitMQTTClient(c.cfg.MQTTConfig.URL, "otbs-"+uuid.New().String())

	// subscribe for control topics used by ThermoController

	c.mqtt.SafeSubscribe(c.cfg.MQTTConfig.ControlTopic+"/default_heating_parameter", 1, c.controlUpdateHandler)
	c.mqtt.SafeSubscribe(c.cfg.MQTTConfig.ControlTopic+"/log_level", 1, c.controlUpdateHandler)

	c.db = OpenDatabase(c.cfg.DBFile)

	c.outsideChan = make(chan float64, 3)
	c.zoneChan = make(chan *ZoneController, 100)

	c.outside = NewOutsideController(c.cfg.Outside, c.cfg.MQTTConfig, c.db, c.outsideChan)

	c.boiler = NewBoilerController(c.cfg.Boiler, c.cfg.MQTTConfig, c.db)

	c.zones = make(map[string]*ZoneController)
	for s, config := range c.cfg.Zones {
		c.zones[s] = newZoneController(s, config, c.cfg.MQTTConfig, c.db, c.zoneChan)
	}

	nZones := len(c.zones)
	c.zoneTRs = make(map[*ZoneController]float64, nZones)
	c.updateMap = make(map[*ZoneController]bool, nZones)

	for _, zone := range c.zones {
		c.zoneTRs[zone] = 0.0
		c.updateMap[zone] = false
	}
	return c
}

func (c *ThermoController) run() {
	OT := -10000.0
	forceUpdate := false

	timer := time.NewTimer(time.Millisecond * 50)
	timer.Stop()
	newOT := 0.0
	hasTimer := false
	tSet := 0.0
	chEnable := false
	ticker := time.NewTicker(time.Second * 30)
	for {
		select {
		case newOT = <-c.outsideChan:
			if newOT != OT {
				OT = newOT
				forceUpdate = true
				if !hasTimer {
					timer = time.NewTimer(time.Millisecond * 50)
					hasTimer = true
				}
			}
		case zone := <-c.zoneChan:
			c.updateMap[zone] = true
			if !hasTimer {
				timer = time.NewTimer(time.Millisecond * 50)
				hasTimer = true
			}
		case <-timer.C:
			{
				hasTimer = false
				needTRupdate := false
				if forceUpdate {
					forceUpdate = false
					for _, controller := range c.zones {
						c.updateMap[controller] = true
					}
				}
				for zone, flag := range c.updateMap {
					if flag {
						c.updateMap[zone] = false
						if nt, ok := c.calculateSetpoint(zone, OT); ok {
							if nt != c.zoneTRs[zone] {
								c.zoneTRs[zone] = nt
								needTRupdate = true
							}
						}
					}
				}
				if needTRupdate {
					tSet1, chEnable1 := c.averageSetpoints()
					if tSet1 != tSet || chEnable1 != chEnable {
						L().Infof(
							"We got updated values: %v -> %v, %v->%v, pushing to boiler", tSet, tSet1, chEnable,
							chEnable1,
						)
						tSet = tSet1
						chEnable = chEnable1
						c.boiler.Update(tSet, chEnable)
					}
					L().Infof("-----------------------------------------------------------")
				}

			}
		case <-ticker.C:
			L().Debugf("Pushung values to boiler on ticker")
			c.boiler.Update(tSet, chEnable)
		}
	}

}

func (c *ThermoController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	L().Infof("main: Got MQTT control request: %v : %v", topic, string(message.Payload()))
	switch topic {
	case "default_heating_parameter":
		{
			hp, err := strconv.ParseFloat(string(message.Payload()), 64)
			if checkError(err) {
				L().Error(err)
				return
			}
			c.cfg.DefaultHeatingParameter = GetPTR(hp)
			L().Infof("Uotaded default heating parameter to %v", hp)
		}
	case "log_level":
		{
			err := c.cfg.LogLevel.Set(string(message.Payload()))
			if checkError(err) {
				L().Errorf("Wrong log level `%v`", string(message.Payload()))
			}
			L().Infof("Uotaded loglevel to `%v`", c.cfg.LogLevel.String())

		}
	}
}

func (c *ThermoController) averageSetpoints() (float64, bool) {
	L().Debug("Calculate effective boiler Tset")
	maxT := -1000.0
	minT := 1000.0
	maxDiff := -10000.0
	var maxZone *ZoneController
	var minZone *ZoneController
	var maxDiffZone *ZoneController
	for zone, f := range c.zoneTRs {
		if f > maxT {
			maxT = f
			maxZone = zone
		}
		if f < minT && f > 10.0 {
			minT = f
			minZone = zone
		}
		md := (zone.setpoint - zone.averageTemperature)
		if md > maxDiff {
			maxDiffZone = zone
			maxDiff = md
		}

	}

	tCut := minT + .7*(maxT-minT)
	tAvg := 0.0
	weight := 0.0
	pwr := 3.0
	for zone, f := range c.zoneTRs {
		if f >= tCut {
			tAvg += *zone.cfg.Weight * math.Pow(f, pwr)
			weight += *zone.cfg.Weight
		}
	}
	tSet := 10.0
	if weight > 0 {
		tAvg /= weight
		tSet = boundTset(math.Round(math.Pow(tAvg, 1.0/pwr)*2) / 2)
	}

	chEnable := tSet >= 18

	c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/maxdiff", 1, false, ThermoMarshalHelper(maxDiff, maxDiffZone))

	if maxZone != nil {
		L().Infof(
			"Zone with MAX demand `%s`: SP=%.2f, T=%.2f Tset=%.2f", maxZone.name, maxZone.setpoint,
			maxZone.averageTemperature, maxT,
		)
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/maxcs", 1, false, ThermoMarshalHelper(maxT, maxZone))
	}
	if minZone != nil {
		L().Infof(
			"Zone with MIN demand `%s`: SP=%.2f, T=%.2f Tset=%.2f", minZone.name, minZone.setpoint,
			minZone.averageTemperature, minT,
		)
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/mincs", 1, false, ThermoMarshalHelper(minT, minZone))
	}
	L().Debugf("New boiler parameters: Tset=%.2f, chEnable=%v", tSet, chEnable)
	return tSet, tSet >= 18

}

func ThermoMarshalHelper(t float64, z *ZoneController) []byte {
	report := struct {
		Zone        string  `json:"zone"`
		Setpoint    float64 `json:"setpoint"`
		Temperature float64 `json:"temperature"`
		CS          float64 `json:"CS"`
	}{
		CS:          t,
		Setpoint:    z.setpoint,
		Temperature: z.averageTemperature,
		Zone:        z.name,
	}

	ret, err := json.Marshal(report)
	if checkError(err) {
		L().Error(err)
	}
	return ret

}

func (c *ThermoController) calculateSetpoint(zone *ZoneController, OT float64) (float64, bool) {
	sp, rt, ok := zone.getPair()
	_ = sp
	hp := *c.cfg.DefaultHeatingParameter
	if zone.cfg.HeatingParameter != nil {
		hp = *zone.cfg.HeatingParameter
	}
	dT := (rt - sp) * 1.5
	hp -= dT

	if ok && OT > -100.0 {
		tset := estimateTset(hp, zone.setpoint, OT, zone.averageTemperature)
		tset = boundTset(tset)
		if OT > sp-3.0 || rt > sp+2.0 {
			tset = 10.0
		}
		L().Debugf("Update TSet for zone \"%s\" with SP=%.2f, T=%.2f : %.2f", zone.name, sp, rt, tset)
		return tset, true
	}
	return 0.0, false
}

func boundTset(tset float64) float64 {
	if tset > 75.0 {
		tset = 75
	}
	if tset < 20 {
		tset = 10
	}
	return tset
}
