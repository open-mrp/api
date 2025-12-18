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
    './services/auth-service/templates',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
    sync('./services/auth-service/templates', '/app/services/auth-service/templates'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/auth-service.yaml')
k8s_resource(
  'auth-service',
  port_forwards='9092:9093',
  resource_deps=['auth-service-compile', 'rabbitmq'],
  labels='services',
)
### End of Auth Service ###

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

logging_service_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/logging-service ./services/logging-service/cmd'

local_resource(
  'logging-service-compile',
  logging_service_compile_cmd,
  deps=['./services/logging-service', './shared'], labels="compiles")

docker_build_with_restart(
  'augno-api/logging-service',
  '.',
  entrypoint=['/app/build/logging-service'],
  dockerfile='./infra/development/docker/logging-service.Dockerfile',
  only=[
    './build/logging-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/kubernetes/apps/logging-service.yaml')
k8s_resource(
  'logging-service',
  resource_deps=['logging-service-compile', 'rabbitmq'],
  labels='services',
)
### End of Logging Service ###
