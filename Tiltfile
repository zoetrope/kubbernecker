load('ext://restart_process', 'docker_build_with_restart')

CONTROLLER_DOCKERFILE = '''FROM golang:alpine
WORKDIR /
COPY ./bin/kubbernecker /
CMD ["/kubbernecker"]
'''

# Generate manifests
local_resource('make manifests', "make manifests", deps=["internal"], ignore=['*/*/zz_generated.deepcopy.go'])

# Deploy manager
watch_file('./config/')
k8s_yaml(kustomize('./config/dev'))

local_resource(
    'Watch & Compile', "make build", deps=['cmd', 'internal', 'pkg'],
    ignore=['*/*/zz_generated.deepcopy.go'])

docker_build_with_restart(
    'kubbernecker:dev', '.',
    dockerfile_contents=CONTROLLER_DOCKERFILE,
    entrypoint=['/kubbernecker'],
    only=['./bin/kubbernecker'],
    live_update=[
        sync('./bin/kubbernecker', '/kubbernecker'),
    ]
)
