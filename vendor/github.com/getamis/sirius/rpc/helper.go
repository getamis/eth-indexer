// Copyright 2017 AMIS Technologies
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

package rpc

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// Keys used for retrieving value from gRPC metadata
const (
	MetadataKeyTrackingID         string = "tracking-id"
	MetadataKeyURI                string = "uri"
	MetadataCustomWebhookEndpoint string = "custom-webhook-endpoint"
)

func GetTrackingIDFromContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)

	if vals, ok := md[MetadataKeyTrackingID]; ok && len(vals) > 0 {
		return vals[0]
	}

	return "unknown"
}

func GetURIFromContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)

	if vals, ok := md[MetadataKeyURI]; ok && len(vals) > 0 {
		return vals[0]
	}

	return ""
}

func GetCustomWebhookEndpointFromContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)

	if vals, ok := md[MetadataCustomWebhookEndpoint]; ok && len(vals) > 0 {
		return vals[0]
	}

	return ""
}
