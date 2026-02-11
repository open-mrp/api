# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###

k8s_yaml('./infra/development/kubernetes/config/secrets.yaml')
k8s_yaml('./infra/development/kubernetes/config/app-config.yaml')

### Observability ###
k8s_yaml('./infra/development/kubernetes/platform/opentelemetry-collector.yaml')
k8s_yaml('./infra/development/kubernetes/platform/jaeger.yaml')
k8s_resource('otel-collector', port_forwards=['4317', '4318', '13133'], labels='tooling')
k8s_resource('jaeger', port_forwards='16686', labels='tooling')
### End Observability ###

### End of K8s Config ###
### RabbitMQ ###
k8s_yaml('./infra/development/kubernetes/platform/rabbitmq.yaml')
k8s_resource('rabbitmq', port_forwards=['5672', '15672'], labels='tooling')
### End RabbitMQ ###
### API Gateway ###

gateway_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway/cmd'

local_resource(
  'api-gateway-compile',
  gateway_compile_cmd,
  deps=['./services/api-gateway', './shared'], labels="compiles")


docker_build_with_restart(
  'augno-api/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/api-gateway.yaml')
k8s_resource('api-gateway', port_forwards=8081,
             resource_deps=['api-gateway-compile', 'rabbitmq'], labels="services")
### End of API Gateway ###

### Auth Service ###

auth_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/auth-service ./services/auth-service/cmd'

local_resource(
  'auth-service-compile',
  auth_service_compile_cmd,
  deps=['./services/auth-service', './shared'], labels="compiles")

docker_build_with_restart(
  'augno-api/auth-service',
  '.',
  entrypoint=['/app/build/auth-service'],
  dockerfile='./infra/development/docker/auth-service.Dockerfile',
  only=[
    './build/auth-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/auth-service.yaml')
k8s_resource(
  'auth-service',
  port_forwards='9092:9092',
  resource_deps=['auth-service-compile', 'rabbitmq', 'core-service'],
  labels='services',
)
### End of Auth Service ###

### Core Service ###

core_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/core-service ./services/core-service/cmd'

local_resource(
  'core-service-compile',
  core_service_compile_cmd,
  deps=['./services/core-service', './shared'], labels="compiles")

docker_build_with_restart(
  'augno-api/core-service',
  '.',
  entrypoint=['/app/build/core-service'],
  dockerfile='./infra/development/docker/core-service.Dockerfile',
  only=[
    './build/core-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/core-service.yaml')
k8s_resource(
  'core-service',
  port_forwards='9094:9092',
  resource_deps=['core-service-compile'],
  labels='services',
)
### End of Core Service ###

### Notification Service ###

notification_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/notification-service ./services/notification-service/cmd'

local_resource(
  'notification-service-compile',
  notification_service_compile_cmd,
  deps=['./services/notification-service', './shared'], labels="compiles")

docker_build_with_restart(
  'augno-api/notification-service',
  '.',
  entrypoint=['/app/build/notification-service'],
  dockerfile='./infra/development/docker/notification-service.Dockerfile',
  only=[
    './build/notification-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/notification-service.yaml')
k8s_resource(
  'notification-service',
  port_forwards='9093:9092',
  resource_deps=['notification-service-compile', 'rabbitmq'],
  labels='services',
)
### End of Notification Service ###

### Logging Service ###

platform_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/platform-service ./services/platform-service/cmd'

local_resource(
  'platform-service-compile',
  platform_service_compile_cmd,
  deps=['./services/platform-service', './shared'], labels="compiles")

docker_build_with_restart(
  'augno-api/platform-service',
  '.',
  entrypoint=['/app/build/platform-service'],
  dockerfile='./infra/development/docker/platform-service.Dockerfile',
  only=[
    './build/platform-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/platform-service.yaml')
k8s_resource(
  'platform-service',
  resource_deps=['platform-service-compile', 'rabbitmq'],
  labels='services',
)
### End of Logging Service ###

### Payment Service ###

# payment_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/billing-service ./services/billing-service/cmd'

# local_resource(
#   'billing-service-compile',
#   payment_service_compile_cmd,
#   deps=['./services/billing-service', './shared'], labels="compiles")

# docker_build_with_restart(
#   'augno-api/billing-service',
#   '.',
#   entrypoint=['/app/build/billing-service'],
#   dockerfile='./infra/development/docker/billing-service.Dockerfile',
#   only=[
#     './build/billing-service',
#     './shared',
#   ],
#   live_update=[
#     sync('./build', '/app/build'),
#     sync('./shared', '/app/shared'),
#   ],
# )

# k8s_yaml('./infra/development/kubernetes/apps/billing-service.yaml')
# k8s_resource(
#   'billing-service',
#   port_forwards='9095:9092',
#   resource_deps=['billing-service-compile'],
#   labels='services',
# )
### End of Payment Service ###
