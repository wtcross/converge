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

package unlock_test

import (
	"testing"

	"github.com/asteris-llc/converge/helpers/fakerenderer"
	"github.com/asteris-llc/converge/resource"
	"github.com/asteris-llc/converge/resource/lock/unlock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnlockInterface tests that Lock properly implements the Task interface
func TestUnlockInterface(t *testing.T) {
	t.Parallel()

	assert.Implements(t, (*resource.Task)(nil), new(unlock.Unlock))
}

// TestCheck tests the Check function
func TestCheck(t *testing.T) {
	t.Parallel()

	unlck := &unlock.Unlock{}
	taskStatus, err := unlck.Check(fakerenderer.New())
	require.NoError(t, err)
	assert.NotNil(t, taskStatus)
}

// TestApply tests the Apply function
func TestApply(t *testing.T) {
	t.Parallel()

	unlck := &unlock.Unlock{}
	taskStatus, err := unlck.Apply()
	require.NoError(t, err)
	assert.NotNil(t, taskStatus)
}