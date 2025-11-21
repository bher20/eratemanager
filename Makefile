# --------------------------------------------
# Project variables
# --------------------------------------------
APP_NAME        := eratemanager
IMAGE_REGISTRY  := ghcr.io
IMAGE_OWNER     := bher20
IMAGE_NAME      := $(APP_NAME)
IMAGE_TAG       := 0.1.0
IMAGE_URI       := $(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE_NAME):$(IMAGE_TAG)

CHART_DIR       := helm/eratemanager
CHART_PACKAGE   := $(APP_NAME)-chart-$(IMAGE_TAG).tgz
CHART_REPO      := oci://$(IMAGE_REGISTRY)/$(IMAGE_OWNER)/charts
CHART_VERSION   := $(IMAGE_TAG)

PYTHON          := python3


# --------------------------------------------
# Container image build/push
# --------------------------------------------

# Auto-detect podman/buildah/docker
ENGINE := $(shell command -v buildah 2>/dev/null || command -v podman 2>/dev/null || command -v docker 2>/dev/null)

.PHONY: image
image:
	@echo ">>> Building image with: $(ENGINE)"
	$(ENGINE) build -t $(IMAGE_URI) -f Containerfile .

.PHONY: image-push
image-push:
	@echo ">>> Pushing image: $(IMAGE_URI)"
	$(ENGINE) push $(IMAGE_URI)


# --------------------------------------------
# Helm packaging & publishing
# --------------------------------------------

.PHONY: helm-package
helm-package:
	@echo ">>> Packaging Helm chart"
	helm package $(CHART_DIR) --version $(CHART_VERSION) --app-version $(IMAGE_TAG) -d ./helm

.PHONY: helm-push
helm-push: helm-package
	@echo ">>> Logging into Helm registry"
	echo $$GITHUB_TOKEN | helm registry login $(IMAGE_REGISTRY) --username $(IMAGE_OWNER) --password-stdin

	@echo ">>> Pushing chart to: $(CHART_REPO)"
	helm push ./helm/$(APP_NAME)-$(CHART_VERSION).tgz $(CHART_REPO)


# --------------------------------------------
# Python commands
# --------------------------------------------

.PHONY: install
install:
	$(PYTHON) -m pip install -e ".[dev]"

.PHONY: test
test:
	pytest -v

.PHONY: run
run:
	uvicorn cemc_rates.api:app --host 0.0.0.0 --port 8000 --reload


# --------------------------------------------
# Kubernetes deployment
# --------------------------------------------

.PHONY: deploy
deploy:
	helm upgrade --install $(APP_NAME) $(CHART_DIR) \
		--set image.repository=$(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE_NAME) \
		--set image.tag=$(IMAGE_TAG)

.PHONY: undeploy
undeploy:
	helm uninstall $(APP_NAME)


# --------------------------------------------
# Formatting / Linting
# --------------------------------------------

.PHONY: fmt
fmt:
	black cemc_rates tests tools

.PHONY: lint
lint:
	flake8 cemc_rates tests tools || true


# --------------------------------------------
# Cleanup
# --------------------------------------------

.PHONY: clean
clean:
	rm -rf build dist *.egg-info .pytest_cache
	find . -name "__pycache__" -type d -exec rm -rf {} +

.PHONY: clean-helm
clean-helm:
	rm -f ./helm/*.tgz

