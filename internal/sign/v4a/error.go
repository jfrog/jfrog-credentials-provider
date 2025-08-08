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

package controllers

import "fmt"

// SigningError indicates an error condition occurred while performing SigV4a signing
type SigningError struct {
	Err error
}

func (e *SigningError) Error() string {
	return fmt.Sprintf("failed to sign request: %v", e.Err)
}

// Unwrap returns the underlying error cause
func (e *SigningError) Unwrap() error {
	return e.Err
}
