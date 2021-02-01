// Copyright 2019 Laszlo Fogas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sql

const Dummy = "dummy"
const SelectUserByLogin = "select-user-by-login"
const SelectAllUser = "select-all-user"
const DeleteUser = "deleteUser"
const SelectArtifactsSinceUntil = "select-artifacts-since-until"
const SelectArtifactsLimitOffset = "select-artifacts-limit-offset"

const artifactFields = "id, repository, branch, pr, source_branch, created, blob"

var queries = map[string]map[string]string{
	"sqlite3": {
		Dummy: `
SELECT 1;
`,
		SelectUserByLogin: `
SELECT id, login, secret, admin
FROM users
WHERE login = ?;
`,
		SelectAllUser: `
SELECT id, login, secret, admin
FROM users;
`, 		DeleteUser: `
DELETE FROM users where login = ?;
`,		SelectArtifactsLimitOffset: `
SELECT ` + artifactFields + `
FROM artifacts
ORDER BY created desc
LIMIT ? OFFSET ?;
`, SelectArtifactsSinceUntil: `
SELECT ` + artifactFields + `
FROM artifacts
WHERE created > since
  AND created < until
ORDER BY created desc;
`,
	},
	"postgres": {},
	"mysql":    {},
}
