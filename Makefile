DB_HOST ?= localhost
DB_CONNECTION = postgres://prophet:password@${DB_HOST}:5432/prophet?sslmode=disable

all: setup build

setup:
	migrate -source file://db/migrations -database ${DB_CONNECTION} up
	jet -dsn=${DB_CONNECTION} -path=./.gen

build:
	go build -o prophet ./service.go ./fetchnodes.go ./dao.go

clean:
	rm -rf .gen prophet
