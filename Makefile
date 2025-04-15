swagger:
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g master/master.go --parseDependency -o swag
