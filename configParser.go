package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func parseConfigFile(curNode string, configFile string, nodeToUrl map[string]string) (int, error) {
	var totalNodes = 0
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		event := scanner.Text()
		connectionDetails := strings.Fields(event)
		nodeName := connectionDetails[0]
		hostname := connectionDetails[1]
		port := connectionDetails[2]
		nodeToUrl[nodeName] = hostname + ":" + port
		totalNodes++
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
		return 0, err
	}
	return totalNodes, nil
}
