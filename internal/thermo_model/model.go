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

package thermo_model

const (
	maxPower = 3
	nelCoeff = 14
)

var (
	mCoeff = [nelCoeff]float64{
		1.665451e-04, -6.595542e-04, 1.326216e-03, 1.243637e-01,
		-6.933021e-04, -1.242895e-01, 6.221294e-02, -7.483894e-05,
		5.947953e-03, -4.194166e-03, 8.534580e-01, 2.610415e-03,
		7.909183e-02, -1.989083e+00,
	}

	mTermsHP = [nelCoeff]int{2, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0}
	mTermsSP = [nelCoeff]int{0, 2, 1, 1, 0, 0, 0, 3, 2, 1, 1, 0, 0, 0}
	mTermsOT = [nelCoeff]int{0, 0, 1, 0, 2, 1, 0, 0, 0, 1, 0, 2, 1, 0}
)

func CalculateSetpoint(hp, sp, ot, rt float64) float64 {
	var hpPwr, spPwr, otPwr [maxPower + 1]float64
	hpPwr[0], spPwr[0], otPwr[0] = 1.0, 1.0, 1.0
	hpPwr[1], spPwr[1], otPwr[1] = hp, sp, ot

	for i := 2; i <= maxPower; i++ {
		hpPwr[i] = hpPwr[i-1] * hp
		spPwr[i] = spPwr[i-1] * sp
		otPwr[i] = otPwr[i-1] * ot
	}

	tset := 0.0
	for i := 0; i < nelCoeff; i++ {
		tset += mCoeff[i] * hpPwr[mTermsHP[i]] * spPwr[mTermsSP[i]] * otPwr[mTermsOT[i]]
	}

	return tset
}
