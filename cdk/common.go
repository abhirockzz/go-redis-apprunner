package main

import (
	"log"
	"os"
)

func getAppRunnerServiceName() string {
	appRunnerServiceName := os.Getenv("APPRUNNER_SERVICE_NAME")
	if appRunnerServiceName == "" {
		log.Fatal("missing env var APPRUNNER_SERVICE_NAME")
	}

	return appRunnerServiceName
}

func getAppRunnerServicePort() string {
	appRunnerServicePort := os.Getenv("APPRUNNER_SERVICE_PORT")
	if appRunnerServicePort == "" {
		log.Fatal("missing env var APPRUNNER_SERVICE_PORT")
	}

	return appRunnerServicePort
}

func getMemoryDBPassword() string {
	memorydbPassword := os.Getenv("MEMORYDB_PASSWORD")
	if memorydbPassword == "" {
		log.Fatal("missing env var MEMORYDB_PASSWORD")
	}

	return memorydbPassword
}
