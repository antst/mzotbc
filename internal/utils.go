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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

var zeroTS time.Time

func init() {
	zeroTS = time.UnixMicro(0)
}

func extractF64PlainOrJson(message mqtt.Message, JSONEntry *string) (float64, error) {
	if JSONEntry == nil {
		return strconv.ParseFloat(string(message.Payload()), 64)
	}

	var valMap map[string]interface{}
	if err := json.Unmarshal(message.Payload(), &valMap); err != nil {
		return 0, errors.Wrapf(err, "json unmarshal error with : %v : %v", message.Topic(), string(message.Payload()))
	}

	v, ok := valMap[*JSONEntry]
	if !ok {
		return 0, fmt.Errorf("not found: `%v` in `%v`: %v", *JSONEntry, message.Topic(), string(message.Payload()))
	}

	t0, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("cannot cast `%v` to float64 in : %v : %v", v, message.Topic(), string(message.Payload()))
	}

	return t0, nil
}

//func mean(vals []float64) float64 {
//	mea
//}
