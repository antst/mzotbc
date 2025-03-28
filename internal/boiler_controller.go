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
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/antst/mzotbc/internal/config"
	"github.com/antst/mzotbc/internal/logger"
	"github.com/antst/mzotbc/internal/safe_mqtt"

	"github.com/antst/mzotbc/internal/db"
)

type BoilerController struct {
	lock    sync.Mutex
	cfg     *config.BoilerConfig
	mqtt    safe_mqtt.MqttClient
	queries *db.Queries
}

func NewBoilerController(_cfg *config.BoilerConfig, _mqttCfg *config.MQTTConfig, _q *db.Queries) *BoilerController {
	b := &BoilerController{cfg: _cfg, queries: _q}
	b.mqtt = safe_mqtt.InitMQTTClient(_mqttCfg.URL, "otbs-boiler-"+uuid.New().String())
	//b.mqtt.SafeSubscribe(_cfg.Topic, 1, b.TemperatureUpdateHandler)

	return b
}

func (b *BoilerController) Update(Tset float64, chEnable bool) {

	if token := b.mqtt.SafePublish(
		b.cfg.TSetTopic, 1, true, fmt.Sprintf("%.1f", Tset),
	); token.Wait() && token.Error() != nil {
		logger.L().Error(token.Error())
	}

	SchEnable := "0"
	if chEnable {
		SchEnable = "1"
	}

	if token := b.mqtt.SafePublish(
		b.cfg.CHEnableTopic, 1, true, SchEnable,
	); token.Wait() && token.Error() != nil {
		logger.L().Error(token.Error())
	}
}
