// Copyright 2022 The Happy Authors
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

package happy

import "fmt"

// String returns a lower-case ASCII representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelSystemDebug:
		return "system"
	case LevelDebug:
		return "debug"
	case LevelVerbose:
		return "info"

	case LevelNotice:
		return "notice"
	case LevelOk:
		return "ok"
	case LevelIssue:
		return "issue"

	case LevelTask:
		return "task"

	case LevelWarn:
		return "warning"
	case LevelDeprecated:
		return "deprecated"
	case LevelNotImplemented:
		return "not-impl"

	case LevelError:
		return "error"
	case LevelCritical:
		return "critical"
	case LevelAlert:
		return "alert"
	case LevelEmergency:
		return "emergency"

	case LevelOut:
		return "out"
	case LevelQuiet:
		return "quiet"

	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// ShortString returns an short representation of the log level.
func (l LogLevel) ShortString() string {
	// Printing levels in all-caps is common enough that we should export this
	// functionality.
	switch l {
	case LevelSystemDebug:
		return "system"
	case LevelDebug:
		return "debug"
	case LevelVerbose:
		return "info"

	case LevelTask:
		return "task"

	case LevelNotice:
		return "notice"
	case LevelOk:
		return "ok"
	case LevelIssue:
		return "issue"

	case LevelWarn:
		return "warn"
	case LevelDeprecated:
		return "depr"
	case LevelNotImplemented:
		return "notimpl"

	case LevelError:
		return "error"
	case LevelCritical:
		return "crit"
	case LevelAlert:
		return "alert"
	case LevelEmergency:
		return "emerg"

	case LevelOut:
		return "out"
	case LevelQuiet:
		return "quiet"

	default:
		return fmt.Sprintf("LEVEL(%d)", l)
	}
}
