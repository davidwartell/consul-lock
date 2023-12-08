DOCKER=docker
GO=go
GOARCH=$(shell go env | grep GOARCH | cut -c 9-13)

.PHONY: run
run:
	$(GO) run .

########################################################################################################################
########################################################################################################################
# consul
########################################################################################################################
########################################################################################################################

.PHONY: pull-consul
pull-consul: rm-consul
	$(DOCKER) pull hashicorp/consul:latest --platform linux/$(GOARCH)

.PHONY: start-consul
start-consul:
	if [[ $$($(DOCKER) volume ls -q | grep consul-data | wc -l) -ne 1 ]]; then $(DOCKER) volume create consul-data; fi
	if [[ $$($(DOCKER) ps -a -f name=consul -q | wc -l) -ne 1 ]]; then $(DOCKER) run \
		 --no-healthcheck \
		 -p 127.0.0.1:8600:8600 \
		 -p 127.0.0.1:8600:8600/udp \
		 -p 127.0.0.1:8500:8500 \
		 -p 127.0.0.1:8301:8301 \
		 -p 127.0.0.1:8302:8302 \
		 -p 127.0.0.1:8300:8300 \
		 --mount type=volume,src=consul-data,target=/consul/data \
		 --name consul -d hashicorp/consul:latest; \
		 else $(DOCKER) start consul; fi

.PHONY: stop-consul
stop-consul:
	-$(DOCKER) stop consul 2>/dev/null || true

.PHONY: rm-consul
rm-consul: stop-consul
	-$(DOCKER) rm consul  2>/dev/null || true
