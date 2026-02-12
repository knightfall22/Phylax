# START: begin
CONFIG_PATH=${HOME}/.phylax/

.PHONY: init
init:
	mkdir -p "${CONFIG_PATH}"

.PHONY: gencert
gencert:
	cfssl gencert \
		-initca certs/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=server \
		certs/server-csr.json | cfssljson -bare server

# START: client
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=client \
		certs/client-csr.json | cfssljson -bare client
# END: client

	mv *.pem *.csr "${CONFIG_PATH}"

.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

up:
	goose -dir ./db/migration up

down:
	goose -dir ./db/migration down

db:
	goose -dir ./db/migration status

migration:
	goose -dir db/migration create $(name) sql 