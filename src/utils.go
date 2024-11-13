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
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

var zeroTS time.Time

func init() {
	zeroTS = time.UnixMicro(0)
}

func checkError(err error) bool {
	if err != nil {
		L().Error(err)
		return true
	}
	return false
}

func GetPTR[T any](v T) *T {
	return &v
}

func extractF64PlainOrJson(client mqtt.Client, message mqtt.Message, JSONEntry *string) (float64, error) {
	var t0 float64
	var err error
	if JSONEntry != nil {
		var valMap map[string]any
		err = json.Unmarshal(message.Payload(), &valMap)
		if checkError(err) {
			L().Error(err)
			return 0, errors.WithMessagef(
				err, "json unmarshal error with : %v : %v", message.Topic(), string(message.Payload()),
			)

		}
		v, ok := valMap[*JSONEntry]
		if !ok {
			return 0, fmt.Errorf("not found: `%v` in `%v`: %v", *JSONEntry, message.Topic(), string(message.Payload()))
		}
		t0, ok = v.(float64)
		if !ok {
			return 0, fmt.Errorf("can not cast `%v` to float64 in : %v", v, message.Topic(), string(message.Payload()))
		}

	} else {
		t0, err = strconv.ParseFloat(string(message.Payload()), 64)
		if checkError(err) {
			L().Error(err)
			return 0, err
		}
	}

	return t0, nil
}

//func mean(vals []float64) float64 {
//	mea
//}
