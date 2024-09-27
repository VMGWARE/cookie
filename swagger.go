// Copyright 2023 Woodpecker Authors
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

package main

import (
	"github.com/discuitnet/discuit/server/docs"
)

// Generate docs/swagger.json via:
//
//	@contact.name	Camphouse Community
//	@contact.url	https://camphouse.org/
//
//go:generate go run camphouse_docs_gen.go swagger.go
//go:generate go run github.com/getkin/kin-openapi/cmd/validate@latest docs/swagger.json
func setupSwaggerStaticConfig() {
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.InfoInstanceName = "api"
	docs.SwaggerInfo.Title = "Camphouse API"
	docs.SwaggerInfo.Description = ""
	docs.SwaggerInfo.Version = "next"
}
