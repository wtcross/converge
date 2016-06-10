// Copyright © 2016 Asteris, LLC
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

package resource

import "fmt"

// ValidationError is the type returned by each resource's Validate method. It
// describes both what went wrong and which stanza caused the problem.
type ValidationError struct {
	Location string
	Err      error
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Location, v.Err)
}