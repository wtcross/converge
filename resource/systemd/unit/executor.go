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

package unit

type SystemdExecutor interface {
	// ListUnits will return a Unit slice
	ListUnits() ([]*Unit, error)

	// QueryUnit will construct a Unit from the given unit name.  If verify is
	// true, the name will be compared against the currently loaded units by
	// calling ListUnits.  This is slower but offers some additional guarantees
	// since the underlying dbus API will return a result even for nonexistant
	// unit names.
	QueryUnit(unitName string, verify bool) (*Unit, error)
	StartUnit(*Unit) error
	StopUnit(*Unit) error
	RestartUnit(*Unit) error
	ReloadUnit(*Unit) error
	SendSignal(u *Unit, signal Signal)
}