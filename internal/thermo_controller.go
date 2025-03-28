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
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/antst/mzotbc/internal/config"
	"github.com/antst/mzotbc/internal/logger"
	"github.com/antst/mzotbc/internal/safe_mqtt"
	"github.com/antst/mzotbc/internal/thermo_model"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	"github.com/antst/mzotbc/internal/db"
)

const (
	timerDuration  = 50 * time.Millisecond
	tickerDuration = 30 * time.Second
	minValidTemp   = -100.0
	defaultTSet    = 10.0
	minEnableTemp  = 18.0
	maxTSet        = 75.0
	minTSet        = 20.0
	fallbackTSet   = 10.0
)

type ThermoController struct {
	cfg         *config.Config
	queries     *db.Queries
	mqtt        safe_mqtt.MqttClient
	zones       map[string]*ZoneController
	outside     *OutsideController
	boiler      *BoilerController
	outsideChan chan float64
	zoneChan    chan *ZoneController
	updateMap   map[*ZoneController]bool
	zoneTRs     map[*ZoneController]float64
	enabled     bool
	forceChan   chan bool
}

type thermoState struct {
	OT          float64
	tSet        float64
	chEnable    bool
	forceUpdate bool
}

func NewThermoController() *ThermoController {
	c := &ThermoController{
		cfg:         config.Get(),
		forceChan:   make(chan bool, 2),
		outsideChan: make(chan float64, 3),
		zoneChan:    make(chan *ZoneController, 100),
		zones:       make(map[string]*ZoneController),
		zoneTRs:     make(map[*ZoneController]float64),
		updateMap:   make(map[*ZoneController]bool),
	}

	c.mqtt = safe_mqtt.InitMQTTClient(c.cfg.MQTTConfig.URL, "otbs-"+uuid.New().String())
	c.setupMQTTSubscriptions()
	c.queries = db.OpenDatabase(c.cfg.DBFile)
	c.outside = NewOutsideController(c.cfg.Outside, c.cfg.MQTTConfig, c.queries, c.outsideChan)
	c.boiler = NewBoilerController(c.cfg.Boiler, c.cfg.MQTTConfig, c.queries)
	c.initializeZones()
	c.setEnabled(c.readValueWithDefault("enabled", "true"))
	return c
}

func (c *ThermoController) setupMQTTSubscriptions() {
	controlTopic := c.cfg.MQTTConfig.ControlTopic
	c.mqtt.SafeSubscribe(controlTopic+"/default_heating_parameter", 1, c.controlUpdateHandler)
	c.mqtt.SafeSubscribe(controlTopic+"/log_level", 1, c.controlUpdateHandler)
	c.mqtt.SafeSubscribe(controlTopic+"/enable", 1, c.controlUpdateHandler)
}

func (c *ThermoController) initializeZones() {
	for s, cfg := range c.cfg.Zones {
		zone := newZoneController(s, cfg, c.cfg.MQTTConfig, c.queries, c.zoneChan)
		c.zones[s] = zone
		c.zoneTRs[zone] = 0.0
		c.updateMap[zone] = false
	}
}

func (c *ThermoController) Run() {
	state := &thermoState{OT: minValidTemp, tSet: defaultTSet}
	timer := time.NewTimer(timerDuration)
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()

	for {
		select {
		case <-c.forceChan:
			state.forceUpdate = true
			c.resetTimer(timer)
		case newOT := <-c.outsideChan:
			if newOT != state.OT {
				state.OT = newOT
				state.forceUpdate = true
				c.resetTimer(timer)
			}
		case zone := <-c.zoneChan:
			c.updateMap[zone] = true
			c.resetTimer(timer)
		case <-timer.C:
			c.handleUpdate(state)
		case <-ticker.C:
			c.update(state.tSet, state.chEnable)
		}
	}
}

func (c *ThermoController) resetTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timerDuration)
}

func (c *ThermoController) handleUpdate(state *thermoState) {
	needTRupdate := false

	if state.forceUpdate {
		for _, zone := range c.zones {
			c.updateMap[zone] = true
		}
	}

	for zone, needsUpdate := range c.updateMap {
		if !needsUpdate {
			continue
		}

		c.updateMap[zone] = false
		if newTR, ok := c.calculateSetpoint(zone, state.OT); ok && newTR != c.zoneTRs[zone] {
			c.zoneTRs[zone] = newTR
			needTRupdate = true
		}
	}

	if needTRupdate || state.forceUpdate {
		newTSet, newChEnable := c.averageSetpoints()
		if newTSet != state.tSet || newChEnable != state.chEnable || state.forceUpdate {
			logger.L().Infof(
				"Updated values: Tset: %.2f -> %.2f, chEnable: %v -> %v",
				state.tSet, newTSet, state.chEnable, newChEnable,
			)
			state.tSet = newTSet
			state.chEnable = newChEnable
			c.update(newTSet, newChEnable)
		}
		logger.L().Info("Update completed")
	}

	state.forceUpdate = false
}

func (c *ThermoController) update(tSet float64, chEnable bool) {
	if !c.enabled {
		c.boiler.Update(defaultTSet, false)
		return
	}
	c.boiler.Update(tSet, chEnable)
}

func (c *ThermoController) controlUpdateHandler(client mqtt.Client, message mqtt.Message) {
	topic := message.Topic()[strings.LastIndex(message.Topic(), "/")+1:]
	logger.L().Infof("main: Got MQTT control request: %v : %v", topic, string(message.Payload()))
	switch topic {
	case "default_heating_parameter":
		if hp, err := strconv.ParseFloat(string(message.Payload()), 64); err == nil {
			c.cfg.DefaultHeatingParameter = &hp
			logger.L().Infof("Updated default heating parameter to %v", hp)
		} else {
			logger.L().Error(err)
		}
	case "log_level":
		if err := c.cfg.LogLevel.Set(string(message.Payload())); err != nil {
			logger.L().Errorf("Wrong log level `%v`", string(message.Payload()))
		} else {
			logger.L().Infof("Updated loglevel to `%v`", c.cfg.LogLevel.String())
		}
	case "enable":
		c.setEnabled(string(message.Payload()))
	}
}

func (c *ThermoController) setEnabled(val string) {
	switch strings.ToLower(val) {
	case "true", "on":
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/active", 1, true, "ON")
		c.enabled = true
	case "false", "off":
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/active", 1, true, "OFF")
		c.enabled = false
	default:
		logger.L().Warnf("Invalid value for enabled_heating: %v", val)
		return
	}
	c.writeValue("enabled", strconv.FormatBool(c.enabled))
	c.forceChan <- true
}

func (c *ThermoController) averageSetpoints() (float64, bool) {
	logger.L().Debug("Calculate effective boiler Tset")
	maxT, minT, maxDiff := -1000.0, 1000.0, -10000.0
	var maxZone, minZone, maxDiffZone *ZoneController

	for zone, f := range c.zoneTRs {
		if f > maxT {
			maxT, maxZone = f, zone
		}
		if f < minT && f > 10.0 {
			minT, minZone = f, zone
		}
		md := zone.setpoint - zone.averageTemperature
		if md > maxDiff {
			maxDiffZone, maxDiff = zone, md
		}
	}

	tCut := minT + 0.7*(maxT-minT)
	tAvg, weight := 0.0, 0.0
	pwr := 3.0
	for zone, f := range c.zoneTRs {
		if f >= tCut {
			tAvg += *zone.cfg.Weight * math.Pow(f, pwr)
			weight += *zone.cfg.Weight
		}
	}

	tSet := defaultTSet
	if weight > 0 {
		tAvg /= weight
		tSet = boundTset(math.Round(math.Pow(tAvg, 1.0/pwr)*2) / 2)
	}

	chEnable := tSet >= minEnableTemp

	c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/maxdiff", 1, false, ThermoMarshalHelper(maxDiff, maxDiffZone))

	if maxZone != nil {
		logger.L().Infof(
			"Zone with MAX demand `%s`: SP=%.2f, T=%.2f Tset=%.2f", maxZone.name, maxZone.setpoint,
			maxZone.averageTemperature, maxT,
		)
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/maxcs", 1, false, ThermoMarshalHelper(maxT, maxZone))
	}
	if minZone != nil {
		logger.L().Infof(
			"Zone with MIN demand `%s`: SP=%.2f, T=%.2f Tset=%.2f", minZone.name, minZone.setpoint,
			minZone.averageTemperature, minT,
		)
		c.mqtt.SafePublish(c.cfg.MQTTConfig.ControlTopic+"/mincs", 1, false, ThermoMarshalHelper(minT, minZone))
	}
	logger.L().Debugf("New boiler parameters: Tset=%.2f, chEnable=%v", tSet, chEnable)
	return tSet, chEnable
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
	if err != nil {
		logger.L().Error(err)
	}
	return ret
}

func (c *ThermoController) calculateSetpoint(zone *ZoneController, OT float64) (float64, bool) {
	sp, rt, ok := zone.getPair()
	hp := c.getHeatingParameter(zone)
	dT := (rt - sp) * 1.5
	hp -= dT

	if ok && OT > minValidTemp {
		tset := thermo_model.CalculateSetpoint(hp, zone.setpoint, OT, zone.averageTemperature)
		tset = boundTset(tset)
		if OT > sp-3.0 || rt > sp+2.0 {
			tset = fallbackTSet
		}
		logger.L().Debugf("Update TSet for zone \"%s\" with SP=%.2f, T=%.2f : %.2f", zone.name, sp, rt, tset)
		return tset, true
	}
	return 0.0, false
}

func (c *ThermoController) getHeatingParameter(zone *ZoneController) float64 {
	if zone.cfg.HeatingParameter != nil {
		return *zone.cfg.HeatingParameter
	}
	return *c.cfg.DefaultHeatingParameter
}

func (s *ThermoController) writeValue(name, value string) error {
	return s.queries.UpsertControllerValue(
		context.Background(),
		db.UpsertControllerValueParams{Name: name, Value: value},
	)
}

func (s *ThermoController) readValue(name string) (string, error) {
	return s.queries.GetControllerValue(context.Background(), name)
}

func (s *ThermoController) readValueWithDefault(name string, defValue string) string {
	val, err := s.queries.GetControllerValue(context.Background(), name)
	if err != nil {
		val = defValue
	}
	return val
}

func boundTset(tset float64) float64 {
	if tset > maxTSet {
		return maxTSet
	}
	if tset < minTSet {
		return fallbackTSet
	}
	return tset
}
