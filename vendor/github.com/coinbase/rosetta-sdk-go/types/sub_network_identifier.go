// Copyright 2021 Coinbase, Inc.
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

// Generated by: OpenAPI Generator (https://openapi-generator.tech)

package types

// SubNetworkIdentifier In blockchains with sharded state, the SubNetworkIdentifier is required to
// query some object on a specific shard. This identifier is optional for all non-sharded
// blockchains.
type SubNetworkIdentifier struct {
	Network  string                 `json:"network"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}