/*******************************************************************************
 * Copyright 2019 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package store

import (
	"github.com/epcom-hdxt/app-functions-sdk-go/internal/store/db"
	"github.com/epcom-hdxt/app-functions-sdk-go/internal/store/db/interfaces"
	"github.com/epcom-hdxt/app-functions-sdk-go/internal/store/db/mongo"
	"github.com/epcom-hdxt/app-functions-sdk-go/internal/store/db/redis"
)

func NewStoreClient(config db.DatabaseInfo) (interfaces.StoreClient, error) {
	switch config.Type {
	case db.MongoDB:
		return mongo.NewClient(config)
	case db.RedisDB:
		return redis.NewClient(config)
	default:
		return nil, db.ErrUnsupportedDatabase
	}
}
