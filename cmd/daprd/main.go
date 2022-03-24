/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/valyala/fasthttp"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/dapr/dapr/pkg/runtime"
	"github.com/dapr/kit/logger"

	// Included components in compiled daprd.

	// Secret stores.
	"github.com/dapr/components-contrib/secretstores"
	secretstore_kubernetes "github.com/dapr/components-contrib/secretstores/kubernetes"
	secretstore_env "github.com/dapr/components-contrib/secretstores/local/env"
	secretstore_file "github.com/dapr/components-contrib/secretstores/local/file"

	secretstores_loader "github.com/dapr/dapr/pkg/components/secretstores"

	// State Stores.
	"github.com/dapr/components-contrib/state"
	state_redis "github.com/dapr/components-contrib/state/redis"

	state_loader "github.com/dapr/dapr/pkg/components/state"

	// Pub/Sub.
	pubs "github.com/dapr/components-contrib/pubsub"
	pubsub_redis "github.com/dapr/components-contrib/pubsub/redis"

	configuration_loader "github.com/dapr/dapr/pkg/components/configuration"
	pubsub_loader "github.com/dapr/dapr/pkg/components/pubsub"

	// Name resolutions.
	nr "github.com/dapr/components-contrib/nameresolution"
	nr_consul "github.com/dapr/components-contrib/nameresolution/consul"
	nr_kubernetes "github.com/dapr/components-contrib/nameresolution/kubernetes"
	nr_mdns "github.com/dapr/components-contrib/nameresolution/mdns"

	nr_loader "github.com/dapr/dapr/pkg/components/nameresolution"

	// Bindings.
	"github.com/dapr/components-contrib/bindings"
	"github.com/dapr/components-contrib/bindings/redis"

	bindings_loader "github.com/dapr/dapr/pkg/components/bindings"

	// HTTP Middleware.

	middleware "github.com/dapr/components-contrib/middleware"
	"github.com/dapr/components-contrib/middleware/http/bearer"
	"github.com/dapr/components-contrib/middleware/http/oauth2"
	"github.com/dapr/components-contrib/middleware/http/oauth2clientcredentials"
	"github.com/dapr/components-contrib/middleware/http/ratelimit"

	http_middleware_loader "github.com/dapr/dapr/pkg/components/middleware/http"
	http_middleware "github.com/dapr/dapr/pkg/middleware/http"

	"github.com/dapr/components-contrib/configuration"
	configuration_redis "github.com/dapr/components-contrib/configuration/redis"
)

var (
	log        = logger.NewLogger("dapr.runtime")
	logContrib = logger.NewLogger("dapr.contrib")
)

func main() {
	// set GOMAXPROCS
	_, _ = maxprocs.Set()

	rt, err := runtime.FromFlags()
	if err != nil {
		log.Fatal(err)
	}

	err = rt.Run(
		runtime.WithSecretStores(
			secretstores_loader.New("kubernetes", func() secretstores.SecretStore {
				return secretstore_kubernetes.NewKubernetesSecretStore(logContrib)
			}),
			secretstores_loader.New("local.file", func() secretstores.SecretStore {
				return secretstore_file.NewLocalSecretStore(logContrib)
			}),
			secretstores_loader.New("local.env", func() secretstores.SecretStore {
				return secretstore_env.NewEnvSecretStore(logContrib)
			}),
		),
		runtime.WithStates(
			state_loader.New("redis", func() state.Store {
				return state_redis.NewRedisStateStore(logContrib)
			}),
		),
		runtime.WithConfigurations(
			configuration_loader.New("redis", func() configuration.Store {
				return configuration_redis.NewRedisConfigurationStore(logContrib)
			}),
		),
		runtime.WithPubSubs(
			pubsub_loader.New("redis", func() pubs.PubSub {
				return pubsub_redis.NewRedisStreams(logContrib)
			}),
		),
		runtime.WithNameResolutions(
			nr_loader.New("mdns", func() nr.Resolver {
				return nr_mdns.NewResolver(logContrib)
			}),
			nr_loader.New("kubernetes", func() nr.Resolver {
				return nr_kubernetes.NewResolver(logContrib)
			}),
			nr_loader.New("consul", func() nr.Resolver {
				return nr_consul.NewResolver(logContrib)
			}),
		),
		runtime.WithInputBindings(),
		runtime.WithOutputBindings(
			bindings_loader.NewOutput("redis", func() bindings.OutputBinding {
				return redis.NewRedis(logContrib)
			}),
		),
		runtime.WithHTTPMiddleware(
			http_middleware_loader.New("uppercase", func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
				return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
					return func(ctx *fasthttp.RequestCtx) {
						body := string(ctx.PostBody())
						ctx.Request.SetBody([]byte(strings.ToUpper(body)))
						h(ctx)
					}
				}, nil
			}),
			http_middleware_loader.New("oauth2", func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
				return oauth2.NewOAuth2Middleware().GetHandler(metadata)
			}),
			http_middleware_loader.New("oauth2clientcredentials", func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
				return oauth2clientcredentials.NewOAuth2ClientCredentialsMiddleware(log).GetHandler(metadata)
			}),
			http_middleware_loader.New("ratelimit", func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
				return ratelimit.NewRateLimitMiddleware(log).GetHandler(metadata)
			}),
			http_middleware_loader.New("bearer", func(metadata middleware.Metadata) (http_middleware.Middleware, error) {
				return bearer.NewBearerMiddleware(log).GetHandler(metadata)
			}),
		),
	)
	if err != nil {
		log.Fatalf("fatal error from runtime: %s", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	<-stop
	rt.ShutdownWithWait()
}
