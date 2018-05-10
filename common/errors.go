// Copyright 2018 AMIS Technologies
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

package common

import (
	"errors"

	"github.com/jinzhu/gorm"
)

var (
	// ErrWrongSigner returns if it's a wrong signer
	ErrWrongSigner = errors.New("wrong signer")
)

// NotFoundError checks whether it's a not found error
func NotFoundError(err error) bool {
	return err == gorm.ErrRecordNotFound
}
