FROM golang:1.26rc2 AS build

ENV GOOS=linux

WORKDIR /app

COPY go.mod go.sum main.go ./
RUN go mod download && go mod verify

RUN go build

# ---

FROM golang:1.26rc2

COPY --from=build /app/k8s-keda-kafka-demo /usr/bin/k8s-keda-kafka-demo

ENTRYPOINT ["k8s-keda-kafka-demo"]
