// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crypto

import "fmt"

// ConstantTimeByteCompare is a constant-time byte comparison of x and y. This function performs an absolute comparison
// if the two byte slices assuming they represent a big-endian number.
//
//		 error if len(x) != len(y)
//	  -1 if x <  y
//	   0 if x == y
//	  +1 if x >  y
func ConstantTimeByteCompare(x, y []byte) (int, error) {
	if len(x) != len(y) {
		return 0, fmt.Errorf("slice lengths do not match")
	}

	xLarger, yLarger := 0, 0

	for i := 0; i < len(x); i++ {
		xByte, yByte := int(x[i]), int(y[i])

		x := ((yByte - xByte) >> 8) & 1
		y := ((xByte - yByte) >> 8) & 1

		xLarger |= x &^ yLarger
		yLarger |= y &^ xLarger
	}

	return xLarger - yLarger, nil
}
