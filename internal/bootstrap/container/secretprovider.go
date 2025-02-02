//
// Copyright (c) 2020 Intel Corporation
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

package container

import (
	"github.com/edgexfoundry/go-mod-bootstrap/di"

	"github.com/epcom-hdxt/app-functions-sdk-go/internal/security"
)

// SecretProviderName contains the name of the security.SecretProvider implementation in the DIC.
var SecretProviderName = di.TypeInstanceToName((*security.SecretProvider)(nil))

// SecretProviderFrom helper function queries the DIC and returns the security.SecretProvider implementation.
func SecretProviderFrom(get di.Get) security.SecretProvider {
	return get(SecretProviderName).(security.SecretProvider)
}
