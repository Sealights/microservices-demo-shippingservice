# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
FROM sealights/golang-builder as builder
WORKDIR /src

# restore dependencies
COPY go.mod go.sum ./
RUN go mod download

ARG RM_DEV_SL_TOKEN=local
ENV RM_DEV_SL_TOKEN ${RM_DEV_SL_TOKEN}

COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS

RUN wget https://agents.sealights.co/slcli/latest/slcli-linux-amd64.tar.gz \
    && tar -xzvf slcli-linux-amd64.tar.gz \
    && chmod +x ./slcli
RUN wget https://agents.sealights.co/slgoagent/latest/slgoagent-linux-amd64.tar.gz \
    && tar -xzvf slgoagent-linux-amd64.tar.gz \
    && chmod +x ./slgoagent
	
RUN ./slcli config init --lang go --token $RM_DEV_SL_TOKEN
RUN BUILD_NAME=$(date +%F_%T) && ./slcli config create-bsid --app "shippingservice" --build "$BUILD_NAME" --branch "master"
RUN ./slcli scan  --bsid buildSessionId.txt --path-to-scanner ./slgoagent --workspacepath ./ --scm git --scmProvider github
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /go/bin/shippingservice .
RUN go test ./... -v	
	
FROM alpine as release
RUN apk add --no-cache ca-certificates
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.7 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /src
COPY --from=builder /go/bin/shippingservice /src/shippingservice
ENV APP_PORT=50051

# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

EXPOSE 50051
ENTRYPOINT ["/src/shippingservice"]
