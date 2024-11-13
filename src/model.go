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

import "math"

var mCoeff = [...]float64{1.665451e-04, -6.595542e-04, 1.326216e-03, 1.243637e-01, -6.933021e-04, -1.242895e-01, 6.221294e-02, -7.483894e-05, 5.947953e-03, -4.194166e-03, 8.534580e-01, 2.610415e-03, 7.909183e-02, -1.989083e+00}

// Heating parameter
var mTermsHP = []int{2, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0}

// Setpoint
var mTermsSP = []int{0, 2, 1, 1, 0, 0, 0, 3, 2, 1, 1, 0, 0, 0}

// Outside T
var mTermsOT = []int{0, 0, 1, 0, 2, 1, 0, 0, 0, 1, 0, 2, 1, 0}

const maxPower = 3

func estimateTset(hp, sp, ot, rt float64) float64 {
	//L().Debugf("model estimate: %f, %f, %f, %f", hp, sp, ot, rt)
	var hpPwr [maxPower + 1]float64
	var spPwr [maxPower + 1]float64
	var otPwr [maxPower + 1]float64
	hpPwr[0] = 1.0
	spPwr[0] = 1.0
	otPwr[0] = 1.0

	for i := 1; i <= maxPower; i++ {
		hpPwr[i] = math.Pow(hp, float64(i))
		spPwr[i] = math.Pow(sp, float64(i))
		otPwr[i] = math.Pow(ot, float64(i))

	}
	tset := 0.0
	nel := len(mCoeff)

	for i := 0; i < nel; i++ {
		tset += mCoeff[i] * hpPwr[mTermsHP[i]] * spPwr[mTermsSP[i]] * otPwr[mTermsOT[i]]
	}
	return tset
}
