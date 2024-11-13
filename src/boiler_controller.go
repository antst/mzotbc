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
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"sync"
	"time"
)

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

type BoilerController struct {
	lock sync.Mutex
	cfg  *BoilerConfig
	mqtt MqttClient
	db   *sqlx.DB
}

func NewBoilerController(_cfg *BoilerConfig, _mqttCfg *MQTTConfig, _db *sqlx.DB) *BoilerController {
	b := &BoilerController{cfg: _cfg, db: _db}
	b.mqtt = InitMQTTClient(_mqttCfg.URL, "otbs-boiler-"+uuid.New().String())
	//b.mqtt.SafeSubscribe(_cfg.Topic, 1, b.TemperatureUpdateHandler)

	return b
}

func (b *BoilerController) Update(Tset float64, chEnable bool) {

	if token := b.mqtt.SafePublish(
		b.cfg.TSetTopic, 1, true, fmt.Sprintf("%.1f", Tset),
	); token.Wait() && token.Error() != nil {
		L().Error(token.Error())
	}

	SchEnable := "0"
	if chEnable {
		SchEnable = "1"
	}

	if token := b.mqtt.SafePublish(
		b.cfg.CHEnableTopic, 1, true, SchEnable,
	); token.Wait() && token.Error() != nil {
		L().Error(token.Error())
	}
}
