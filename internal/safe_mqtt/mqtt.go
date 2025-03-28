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

package safe_mqtt

import (
	"github.com/antst/mzotbc/internal/logger"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	reconnectInterval = 2 * time.Second
)

// MqttClient is bridge between our app and MQTT
type MqttClient interface {
	SafePublish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
	SafeSubscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token
	SafeUnsubscribe(topics ...string) mqtt.Token
}

type mqttClient struct {
	mutex sync.Mutex
	mqtt  mqtt.Client
}

var (
	connectHandler = func(client mqtt.Client) {
		or := client.OptionsReader()
		logger.L().Infof("Connected to MQTT broker: %v as %s", or.Servers(), or.ClientID())
	}

	connectLostHandler = func(client mqtt.Client, err error) {
		logger.L().Warnf("Connection to MQTT broker lost: %v", err)
		reconnectMQTT(client)
	}
)

func InitMQTTClient(url, clientID string) MqttClient {
	opts := mqtt.NewClientOptions().
		AddBroker(url).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(reconnectInterval)

	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	//opts.SetUsername(cfg.USER)
	//opts.SetPassword(cfg.PASSWORD)
	// opts.SetDefaultPublishHandler(messagePubHandler)

	client := mqtt.NewClient(opts)
	reconnectMQTT(client)

	return &mqttClient{
		mqtt: client,
	}
}

func reconnectMQTT(client mqtt.Client) {
	for {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			logger.L().Warnf("Connection failed, retrying in %v: %v", reconnectInterval, token.Error())
			time.Sleep(reconnectInterval)
		} else {
			break
		}
	}
}

func (m *mqttClient) SafePublish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.mqtt.Publish(topic, qos, retained, payload)
}

func (m *mqttClient) SafeSubscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.mqtt.Subscribe(topic, qos, callback)
}

func (m *mqttClient) SafeUnsubscribe(topics ...string) mqtt.Token {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.mqtt.Unsubscribe(topics...)
}
