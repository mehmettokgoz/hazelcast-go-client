/*
 * Copyright (c) 2008-2021, Hazelcast, Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License")
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package it

import (
	"context"

	"github.com/hazelcast/hazelcast-go-client"
	inearcache "github.com/hazelcast/hazelcast-go-client/internal/nearcache"
)

type DataStructureAdapter interface {
	Get(ctx context.Context, key interface{}) (interface{}, error)
	Put(ctx context.Context, key, value interface{}) (interface{}, error)
	LocalMapStats() hazelcast.LocalMapStats
}

type NearCacheAdapter interface {
	Size() int
	Get(key interface{}) (interface{}, error)
	GetFromNearCache(key interface{}) (interface{}, error)
	GetRecord(key interface{}) (*inearcache.Record, bool)
	InvalidationRequests() int64
	ToNearCacheKey(key interface{}) interface{}
}
