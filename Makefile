SERVICE_NAME=subscription-service
NAMESPACE=k8sdeploy
GIT_COMMIT=`git rev-parse --short HEAD`
-include .env
export

.PHONY: setup
setup: ## Get linting stuffs
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	go install golang.org/x/tools/cmd/goimports
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc


.PHONY: build-push-latest
build-push-latest:
	nerdctl build --platform=amd64,arm64 --tag containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:latest --build-arg VERSION=0.1 --build-arg BUILD=${GIT_COMMIT} --build-arg SERVICE_NAME=${SERVICE_NAME} -f ./k8s/Containerfile .
	nerdctl push containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:latest --all-platforms

.PHONY: build-deploy-latest
build-deploy-latest: build-push-latest deploy-latest

.PHONY: build-images
build-images: ## Build the images
	nerdctl build --platform=amd64,arm64 --tag containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:${GIT_COMMIT} --build-arg VERSION=0.1 --build-arg BUILD=${GIT_COMMIT} --build-arg SERVICE_NAME=${SERVICE_NAME} -f ./k8s/Containerfile .
	nerdctl tag containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:${GIT_COMMIT} containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:latest

.PHONY: publish-images
publish-images:
	nerdctl push containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:${GIT_COMMIT} --all-platforms
	nerdctl push containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:latest --all-platforms

.PHONY: build
build: build-images

.PHONY: deploy
deploy:
	kubectl set image deployment/${SERVICE_NAME} ${SERVICE_NAME}=containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:${GIT_COMMIT} --namespace=${NAMESPACE}

.PHONY: deploy-latest
deploy-latest:
	kubectl set image deployment/${SERVICE_NAME} ${SERVICE_NAME}=containers.chewedfeed.com/${NAMESPACE}/${SERVICE_NAME}:latest --namespace=${NAMESPACE}
	kubectl rollout restart deployment/${SERVICE_NAME} --namespace=${NAMESPACE}

.PHONY: build-deploy
build-deploy: build publish-images deploy

.PHONY: lint-build-deploy
lint-build-deploy: build publish-images deploy

.PHONY: test
test: ## Test the app
	go test \
		-v \
		-race \
		-bench=./... \
		-benchmem \
		-timeout=120s \
		-cover \
		-coverprofile=./test_coverage.txt \
		-bench=./... ./...

.PHONY: mocks
mocks: ## Generate the mocks
	go generate ./...

.PHONY: full
full: clean build fmt  test ## Clean, build, make sure its formatted, linted, and test it

.PHONY: fmt
fmt: ## Formatting
	gofmt -w -s .
	goimports -w .
	go clean ./...

.PHONY: pre-commit
pre-commit: fmt ## Do formatting and linting

.PHONY: clean
clean: ## Clean
	go clean ./...
	rm -rf bin/${SERVICE_NAME}
