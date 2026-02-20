# Load the restart_process extension
load('ext://restart_process', 'docker_build_with_restart')

### K8s Config ###
k8s_yaml('./infra/development/k8s/app-config.yaml')

### End of K8s Config ###

### Trip Service ###

trip_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/trip-service ./services/trip-service/cmd/main.go'
if os.name == 'nt':
  trip_compile_cmd = './infra/development/docker/trip-build.bat'

local_resource(
  'trip-service-compile',
  trip_compile_cmd,
  deps=['./services/trip-service', './shared'],
  labels="compiles"
)

docker_build_with_restart(
  'go-echoride/trip-service',
  '.',
  entrypoint=['/app/build/trip-service'],
  dockerfile='./infra/development/docker/trip-service.Dockerfile',
  only=[
    './build/trip-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

# Apply standard deployment and service
k8s_yaml('./infra/development/k8s/trip-service-deployment.yaml')

# Apply APISIX routing rule
k8s_yaml('./infra/development/k8s/apisix-route.yaml')

k8s_resource('trip-service', resource_deps=['trip-service-compile'], labels="services")

### End of Trip Service ###