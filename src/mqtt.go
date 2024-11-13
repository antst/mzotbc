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
	"sync"
	"time"
)

// Client is bridge between our app and MQTT
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
	connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
		// TODO: use proper logger
		or := client.OptionsReader()
		L().Warnln("Connected to MQTT broker:", or.Servers(), "as", or.ClientID())
	}
	connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		// TODO: use proper logger
		L().Warnf("Connection to MQTT broker lost: %v", err)
		for {
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				L().Warnf("Connection was not a success, will retry in 2 sec\n")
				time.Sleep(time.Second * 2)
			} else {
				break
			}
		}
		client.Connect()
	}
)

func InitMQTTClient(url, clientID string) MqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID(clientID)
	//opts.SetUsername(cfg.USER)
	//opts.SetPassword(cfg.PASSWORD)
	// opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(2 * time.Second)

	client := mqtt.NewClient(opts)
	for {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			L().Warnf("Connection was not a success, will retry in 2 sec\n")
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}

	return &mqttClient{
		mutex: sync.Mutex{},
		mqtt:  client,
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
