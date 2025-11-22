APP_NAME        := eratemanager
IMAGE_REPO      := ghcr.io/bher20/eratemanager
IMAGE_TAG       := latest

HELM_CHART_DIR  := helm/eratemanager

BUILDER := $(shell command -v buildah >/dev/null 2>&1 && echo buildah || echo docker)

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME) ./cmd/$(APP_NAME)

.PHONY: test
test:
	go test ./...

.PHONY: build-image
build-image:
	@echo "==> Building container image using $(BUILDER)..."
ifeq ($(BUILDER),buildah)
	buildah bud -f Containerfile -t $(IMAGE_REPO):$(IMAGE_TAG) .
else
	docker build -f Containerfile -t $(IMAGE_REPO):$(IMAGE_TAG) .
endif

.PHONY: push-image
push-image:
ifeq ($(BUILDER),buildah)
	buildah push $(IMAGE_REPO):$(IMAGE_TAG)
else
	docker push $(IMAGE_REPO):$(IMAGE_TAG)
endif

.PHONY: helm-upgrade
helm-upgrade:
	helm upgrade --install $(APP_NAME) $(HELM_CHART_DIR) \
		--set image.repository=$(IMAGE_REPO) \
		--set image.tag=$(IMAGE_TAG) \
		$(EXTRA_ARGS)
